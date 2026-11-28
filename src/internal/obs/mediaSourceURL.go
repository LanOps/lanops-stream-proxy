package obs

import (
	"fmt"

	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/andreykaipov/goobs"
	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"
)

// ensureStreamMediaSourceURL keeps an existing media source pointed at the
// address in MEDIAMTX_RTMP_ADDRESS.
//
// OBS bakes the URL into the input when it is created and never looks at it
// again. Change the MediaMTX port or host and every source that already exists
// keeps trying the old address, which shows as a black frame with no error
// anywhere obvious. Rewriting it here means a config change takes effect on
// the next sync instead of needing the database and the sources thrown away.
func (client *Client) ensureStreamMediaSourceURL(stream dbstreams.Stream) error {
	want := fmt.Sprintf("rtmp://%s/%s", client.cfg.MediaMtxRtmpAddress, stream.Name)
	sourceName := fmt.Sprintf("media_source-%s", stream.Name)

	var current string
	err := client.withObs(func(obs *goobs.Client) error {
		resp, err := obs.Inputs.GetInputSettings(&goobRequestInputs.GetInputSettingsParams{
			InputName: &sourceName,
		})
		if err != nil {
			return err
		}
		if value, ok := resp.InputSettings["input"].(string); ok {
			current = value
		}
		return nil
	})
	if err != nil {
		return err
	}
	if current == want {
		return nil
	}

	overlay := true
	err = client.withObs(func(obs *goobs.Client) error {
		_, err := obs.Inputs.SetInputSettings(&goobRequestInputs.SetInputSettingsParams{
			InputName:     &sourceName,
			InputSettings: map[string]any{"input": want},
			Overlay:       &overlay,
		})
		return err
	})
	if err != nil {
		return err
	}
	client.notify(fmt.Sprintf("Stream %s now reads from %s (was %s)", stream.Name, want, current))
	return nil
}
