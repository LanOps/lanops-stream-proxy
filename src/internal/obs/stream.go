package obs

import (
	"fmt"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"time"

	"github.com/andreykaipov/goobs"
	goobRequestFilters "github.com/andreykaipov/goobs/api/requests/filters"
	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"
	goobRequestSceneItems "github.com/andreykaipov/goobs/api/requests/sceneitems"

	"github.com/andreykaipov/goobs/api/typedefs"
)

func (client *Client) setStreamSceneItemsVisibility(stream dbstreams.Stream, enabled bool) error {
	return client.withObs(func(obs *goobs.Client) error {
		if _, err := obs.SceneItems.SetSceneItemEnabled(&goobRequestSceneItems.SetSceneItemEnabledParams{
			SceneUuid:        &client.cfg.ObsProxySceneUuid,
			SceneItemId:      &stream.ObsTextId,
			SceneItemEnabled: &enabled,
		}); err != nil {
			return err
		}

		if _, err := obs.SceneItems.SetSceneItemEnabled(&goobRequestSceneItems.SetSceneItemEnabledParams{
			SceneUuid:        &client.cfg.ObsProxySceneUuid,
			SceneItemId:      &stream.ObsStreamId,
			SceneItemEnabled: &enabled,
		}); err != nil {
			return err
		}
		return nil
	})
}

func (client *Client) createStreamSceneItemFadeSourceFilter(sourceUuid string) error {
	filterSettings := map[string]interface{}{
		"opacity": 100.0,
	}
	filterName := "FadeFilter"
	filterKind := "color_filter"
	return client.withObs(func(obs *goobs.Client) error {
		_, err := obs.Filters.CreateSourceFilter(&goobRequestFilters.CreateSourceFilterParams{
			SourceUuid:     &sourceUuid,
			FilterName:     &filterName,
			FilterKind:     &filterKind,
			FilterSettings: filterSettings,
		})
		return err
	})
}

func (client *Client) setStreamSceneItemsFade(stream dbstreams.Stream, start, end float64) error {
	steps := 40                                        // how many fade steps
	interval := 5 * time.Second / time.Duration(steps) // delay between updates
	for i := 0; i <= steps; i++ {
		opacity := start + (end-start)*(float64(i)/float64(steps))
		filterSettings := map[string]interface{}{
			"opacity": opacity,
		}
		filterName := "FadeFilter"

		// One withObs per step so an incoming fade can run alongside an
		// outgoing one instead of queueing behind all 40 steps.
		err := client.withObs(func(obs *goobs.Client) error {
			// Media Source
			mediaSourceName := fmt.Sprintf("media_source-%s", stream.Name)
			if _, err := obs.Filters.SetSourceFilterSettings(&goobRequestFilters.SetSourceFilterSettingsParams{
				SourceName:     &mediaSourceName,
				FilterName:     &filterName,
				FilterSettings: filterSettings,
			}); err != nil {
				return err
			}
			// Text
			textName := fmt.Sprintf("text-%s", stream.Name)
			_, err := obs.Filters.SetSourceFilterSettings(&goobRequestFilters.SetSourceFilterSettingsParams{
				SourceName:     &textName,
				FilterName:     &filterName,
				FilterSettings: filterSettings,
			})
			return err
		})
		if err != nil {
			return err
		}

		time.Sleep(interval)
	}
	return nil
}

func (client *Client) createStreamMediaSourceInput(stream dbstreams.Stream) (dbstreams.Stream, error) {
	inputSettings := map[string]interface{}{
		"input":               fmt.Sprintf("rtmp://%s/%s", client.cfg.MediaMtxRtmpAddress, stream.Name),
		"is_local_file":       false,
		"looping":             false,
		"restart_on_activate": false,
		"reconnect":           true,
		"buffering_mb":        4,
		"enabled":             false,
	}
	params := goobRequestInputs.NewCreateInputParams().
		WithSceneUuid(client.cfg.ObsProxySceneUuid).
		WithInputName(fmt.Sprintf("media_source-%s", stream.Name)).
		WithInputKind("ffmpeg_source").
		WithInputSettings(inputSettings)
	var resp *goobRequestInputs.CreateInputResponse
	err := client.withObs(func(obs *goobs.Client) error {
		var err error
		resp, err = obs.Inputs.CreateInput(params)
		return err
	})
	if err != nil {
		return stream, err
	}
	updateStreamParams := map[string]interface{}{
		"obs_stream_uuid": resp.InputUuid,
		"obs_stream_id":   resp.SceneItemId,
	}
	if stream, err = client.db.UpdateStream(stream, updateStreamParams); err != nil {
		return stream, err
	}

	// Add Fade Source Filter
	err = client.createStreamSceneItemFadeSourceFilter(stream.ObsStreamUuid)
	if err != nil {
		return stream, err
	}

	// Fit to screen - To fix people who don't know how to output at 1080p
	// First get all the current Item Transforms as we need to pass ALL in the object
	transformResp, err := client.getSceneItemTransform(resp.SceneItemId)
	if err != nil {
		return stream, err
	}
	transformResp.SceneItemTransform.BoundsType = "OBS_BOUNDS_SCALE_INNER" // fit within screen
	transformResp.SceneItemTransform.BoundsAlignment = 0                   // center
	transformResp.SceneItemTransform.BoundsWidth = 1920                    // match your canvas width
	transformResp.SceneItemTransform.BoundsHeight = 1080                   // match your canvas height
	if err := client.setSceneItemTransform(resp.SceneItemId, transformResp.SceneItemTransform); err != nil {
		return stream, err
	}

	return stream, nil
}

func (client *Client) createStreamTextInput(stream dbstreams.Stream) (dbstreams.Stream, error) {
	inputSettings := map[string]interface{}{
		"text": stream.Name, // the actual text content
		"font": map[string]interface{}{
			"face":  "Arial",
			"size":  50,
			"flags": 0, // optional font flags (bold/italic etc.)
		},
		"color1":     0xFFFFFFFF, // ARGB color
		"outline":    1,          // outline thickness
		"boundsType": "OBS_BOUNDS_NONE",
		"enabled":    false,
	}
	textKind, err := client.getTextInputKind()
	if err != nil {
		return stream, err
	}
	params := goobRequestInputs.NewCreateInputParams().
		WithSceneUuid(client.cfg.ObsProxySceneUuid).
		WithInputName(fmt.Sprintf("text-%s", stream.Name)).
		WithInputKind(textKind).
		WithInputSettings(inputSettings)
	var resp *goobRequestInputs.CreateInputResponse
	err = client.withObs(func(obs *goobs.Client) error {
		var err error
		resp, err = obs.Inputs.CreateInput(params)
		return err
	})
	if err != nil {
		return stream, err
	}

	updateStreamParams := map[string]interface{}{
		"obs_text_uuid": resp.InputUuid,
		"obs_text_id":   resp.SceneItemId,
	}
	if stream, err = client.db.UpdateStream(stream, updateStreamParams); err != nil {
		return stream, err
	}

	// Transform the text
	// First get all the current Item Transforms as we need to pass ALL in the object
	transformResp, err := client.getSceneItemTransform(resp.SceneItemId)
	if err != nil {
		return stream, err
	}
	// Set to Top Left with a small border
	transformResp.SceneItemTransform.PositionX = 102.0
	transformResp.SceneItemTransform.PositionY = 47.0
	transformResp.SceneItemTransform.BoundsWidth = 500.0
	transformResp.SceneItemTransform.BoundsHeight = 500.0
	if err := client.setSceneItemTransform(resp.SceneItemId, transformResp.SceneItemTransform); err != nil {
		return stream, err
	}

	// Add Fade Source Filter
	err = client.createStreamSceneItemFadeSourceFilter(stream.ObsTextUuid)
	if err != nil {
		return stream, err
	}
	return stream, nil
}

func (client *Client) getSceneItemTransform(sceneItemId int) (*goobRequestSceneItems.GetSceneItemTransformResponse, error) {
	var resp *goobRequestSceneItems.GetSceneItemTransformResponse
	err := client.withObs(func(obs *goobs.Client) error {
		var err error
		resp, err = obs.SceneItems.GetSceneItemTransform(&goobRequestSceneItems.GetSceneItemTransformParams{
			SceneItemId: &sceneItemId,
			SceneUuid:   &client.cfg.ObsProxySceneUuid,
		})
		return err
	})
	return resp, err
}

func (client *Client) setSceneItemTransform(sceneItemId int, transform *typedefs.SceneItemTransform) error {
	return client.withObs(func(obs *goobs.Client) error {
		_, err := obs.SceneItems.SetSceneItemTransform(&goobRequestSceneItems.SetSceneItemTransformParams{
			SceneItemId:        &sceneItemId,
			SceneUuid:          &client.cfg.ObsProxySceneUuid,
			SceneItemTransform: transform,
		})
		return err
	})
}
