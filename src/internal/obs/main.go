package obs

import (
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/andreykaipov/goobs"
)

func New(cfg config.Config, dbStreamsClient *dbstreams.Client, obsClient *goobs.Client, msgCh chan<- channels.MsgCh) (*Client, error) {
	client := &Client{
		cfg:   cfg,
		conn:  obsClient,
		db:    dbStreamsClient,
		msgCh: msgCh,
	}
	return client, nil
}

// withObs runs fn against a live OBS WebSocket connection.
//
// goobs cannot reconnect, so the connection is opened lazily and discarded as
// soon as it stops answering. The next call dials again. Without that, an OBS
// restart leaves the proxy logging errors until someone restarts it by hand,
// which is not something to do halfway through an event.
func (client *Client) withObs(fn func(obs *goobs.Client) error) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.conn == nil {
		conn, err := goobs.New(
			client.cfg.ObsWebSocketAddress,
			goobs.WithPassword(client.cfg.ObsWebSocketPassword),
		)
		if err != nil {
			return err
		}
		client.conn = conn
		// Scene UUIDs belong to the scene collection that was loaded when
		// they were read, so re-resolve after every new connection.
		if err := client.resolveProxyScene(conn); err != nil {
			_ = conn.Disconnect()
			client.conn = nil
			return err
		}
	}

	err := fn(client.conn)
	if err == nil {
		return nil
	}

	// A failed request means either OBS rejected it or the socket is dead.
	// Probing with GetVersion separates the two without matching on goobs'
	// error strings, which are plain fmt.Errorf values.
	if _, probeErr := client.conn.General.GetVersion(); probeErr != nil {
		_ = client.conn.Disconnect()
		client.conn = nil
	}
	return err
}
