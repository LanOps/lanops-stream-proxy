package mediamtx

import (
	"encoding/json"
	"fmt"
	"io"
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"net/http"
	"time"
)

const (
	// streamRetention is how long a stream that has stopped stays in the
	// database. Long enough that an encoder reconnecting is still the same
	// stream, with its enabled flag, its rotation position and its OBS
	// sources intact. Short enough that finished sessions tidy themselves up.
	streamRetention = 10 * time.Minute
	// lastSeenRefresh avoids rewriting last_seen on every sync tick.
	lastSeenRefresh = 60 * time.Second
)

func New(cfg config.Config, dbStreamsClient *dbstreams.Client, msgCh chan<- channels.MsgCh) (*Client, error) {
	client := &Client{
		cfg:   cfg,
		db:    dbStreamsClient,
		msgCh: msgCh,
	}
	return client, nil
}

func (client *Client) SyncStreams() *ClientError {
	// Read Streams from MediaMTX
	proxyStreamsList, err := client.GetStreams()
	if err != nil {
		// Name the address that was tried. The usual cause is mediamtx.yml and
		// .env disagreeing about the API port, which looks like a dead service
		// from here.
		return &ClientError{
			Err: err,
			Message: fmt.Sprintf(
				"Cannot reach the MediaMTX API at %s. Check apiAddress in mediamtx.yml matches MEDIAMTX_API_ADDRESS in .env",
				client.cfg.MediaMtxApiAddress,
			),
		}
	}

	dbStreamsList, err := client.db.GetStreams()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Error pulling DB Streams",
		}
	}

	// A stream MediaMTX lists now is either new or coming back. Either way its
	// row keeps its identity, so the enabled flag, the rotation position and
	// the OBS sources survive a reconnecting encoder.
	byName := make(map[string]dbstreams.Stream, len(dbStreamsList))
	for _, dbStream := range dbStreamsList {
		byName[dbStream.Name] = dbStream
	}
	live := make(map[string]bool, len(proxyStreamsList))
	for _, proxyStream := range proxyStreamsList {
		live[proxyStream.Name] = true

		existing, known := byName[proxyStream.Name]
		if !known {
			if _, err := client.db.CreateStream(proxyStream.Name); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error adding DB Stream",
				}
			}
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("Stream Found! Adding %s to the DB", proxyStream.Name), Level: "INFO"}
			continue
		}
		if !existing.Live {
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("Stream %s is live again", proxyStream.Name), Level: "INFO"}
		}
		if !existing.Live || time.Since(existing.LastSeen) > lastSeenRefresh {
			if err := client.db.MarkStreamLive(proxyStream.Name); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error marking DB Stream live",
				}
			}
		}
	}

	// Anything the database still thinks is live but MediaMTX no longer lists
	// has stopped. Mark it offline rather than deleting it: the row, its
	// enabled flag and its OBS sources stay, which is what stops a flapping
	// stream from being torn down and rebuilt on every reconnect.
	for _, dbStream := range dbStreamsList {
		if !dbStream.Live || live[dbStream.Name] {
			continue
		}
		if err := client.db.MarkStreamOffline(dbStream.Name); err != nil {
			return &ClientError{
				Err:     err,
				Message: "Error marking DB Stream offline",
			}
		}
		client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("Stream %s stopped, holding it for %s", dbStream.Name, streamRetention), Level: "INFO"}
	}

	// Rows offline for longer than the retention are a finished session. Drop
	// them, which is also what lets obs.SyncStreams remove their sources.
	if err := client.db.PruneOfflineStreams(time.Now().Add(-streamRetention)); err != nil {
		return &ClientError{
			Err:     err,
			Message: "Error pruning offline DB Streams",
		}
	}
	return nil
}

func (client *Client) GetStreams() ([]MediamtxListStreamsOutput, error) {
	var response MediamtxListStreamsResponse
	var streams []MediamtxListStreamsOutput
	url := fmt.Sprintf("http://%s/v3/paths/list", client.cfg.MediaMtxApiAddress)

	// Create HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return streams, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return streams, fmt.Errorf("unexpected HTTP %d from %s", resp.StatusCode, url)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return streams, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return streams, err
	}

	// Only paths with a publisher are streams. MediaMTX also lists paths that
	// exist in its config with nothing publishing to them, and they report
	// ready:false. Treating those as live creates a media source that never
	// connects, which the rotation then shows as a black frame.
	streams = make([]MediamtxListStreamsOutput, 0, len(response.Items))
	for _, item := range response.Items {
		if item.Ready {
			streams = append(streams, item)
		}
	}
	return streams, nil
}
