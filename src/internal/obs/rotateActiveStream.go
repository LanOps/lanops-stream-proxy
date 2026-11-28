package obs

import (
	"lanops/obs-proxy-bridge/internal/channels"
	"time"
)

// rotationHold is how long a stream stays on screen before the rotation moves
// on. RotateActiveStream is called more often than this, so a stream whose
// publisher has gone can be replaced straight away instead of holding an empty
// scene until the next scheduled changeover.
const rotationHold = 30 * time.Second

func (client *Client) RotateActiveStream() *ClientError {
	streamCount, err := client.db.GetAvailableStreamsCount()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot pull DB Streams",
		}
	}

	if streamCount == 0 {
		client.msgCh <- channels.MsgCh{Err: nil, Message: "No Enabled Streams found", Level: "INFO"}
		client.obsStreams.previous = nil
		client.obsStreams.current = nil
		client.visibleSince = time.Time{}
		return nil
	}

	// Check the visible stream is still there and still has a publisher.
	// Either way it has to come off screen, so whoever is next goes up
	// immediately rather than at the next scheduled changeover.
	if client.obsStreams.current != nil {
		streamLive, err := client.db.IsStreamLive(client.obsStreams.current.Name)
		if err != nil || !streamLive {
			client.obsStreams.current = nil
			client.visibleSince = time.Time{}
		}
	}

	nextStream, err := client.db.GetNextEnabledStream(client.obsStreams.current)
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot pull next DB Stream",
		}
	}

	// Nothing on screen: startup, or the visible stream just vanished.
	if client.obsStreams.current == nil {
		if err := client.setStreamSceneItemsVisibility(*nextStream, true); err != nil {
			return &ClientError{
				Err:     err,
				Message: "Cannot Set Stream Visibility",
			}
		}
		if err := client.setStreamSceneItemsFade(*nextStream, 0, 100); err != nil {
			return &ClientError{
				Err:     err,
				Message: "Cannot Set Stream Visibility",
			}
		}
		client.obsStreams.current = nextStream
		client.visibleSince = time.Now()
		return nil
	}

	// Something is on screen and still live. Leave it there until it has had
	// its turn.
	if nextStream.ID == client.obsStreams.current.ID || time.Since(client.visibleSince) < rotationHold {
		return nil
	}

	previous := client.obsStreams.current
	incoming := nextStream
	go func() {
		if err := client.setStreamSceneItemsVisibility(*incoming, true); err != nil {
			return
		}
		_ = client.setStreamSceneItemsFade(*incoming, 0, 100)
	}()
	if err := client.setStreamSceneItemsFade(*previous, 100, 0); err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot Set Stream Fade",
		}
	}
	if err := client.setStreamSceneItemsVisibility(*previous, false); err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot Set Stream Visibility",
		}
	}
	client.obsStreams.previous = previous
	client.obsStreams.current = incoming
	client.visibleSince = time.Now()
	return nil
}
