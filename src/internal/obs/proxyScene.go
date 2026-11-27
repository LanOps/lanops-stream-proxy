package obs

import (
	"fmt"
	"lanops/obs-proxy-bridge/internal/channels"

	goobRequestSceneItems "github.com/andreykaipov/goobs/api/requests/sceneitems"
	goobRequestScenes "github.com/andreykaipov/goobs/api/requests/scenes"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/typedefs"
)

func (client *Client) InitProxyScene() *ClientError {
	err := client.withObs(client.resolveProxyScene)
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot Create OBS Scene",
		}
	}

	client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("OBS Scene Name: %s", client.cfg.ObsProxySceneName), Level: "INFO"}
	client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("OBS Scene UUID: %s", client.cfg.ObsProxySceneUuid), Level: "INFO"}

	return nil
}

// resolveProxyScene looks the proxy scene up by name and creates it if it is
// missing. It runs on the first connection and again after every reconnect,
// because OBS scene UUIDs are only valid for the scene collection that was
// loaded when they were read.
func (client *Client) resolveProxyScene(obs *goobs.Client) error {
	scenes, err := obs.Scenes.GetSceneList()
	if err != nil {
		return err
	}
	for _, scene := range scenes.Scenes {
		if scene.SceneName == client.cfg.ObsProxySceneName {
			client.cfg.ObsProxySceneUuid = scene.SceneUuid
			return nil
		}
	}

	client.notify("OBS Scene doesn't exist. Creating...")
	resp, err := obs.Scenes.CreateScene(&goobRequestScenes.CreateSceneParams{
		SceneName: &client.cfg.ObsProxySceneName,
	})
	if err != nil {
		return err
	}
	client.cfg.ObsProxySceneUuid = resp.SceneUuid
	return nil
}

// notify reports progress without ever blocking a caller that holds mu.
func (client *Client) notify(message string) {
	select {
	case client.msgCh <- channels.MsgCh{Message: message, Level: "INFO"}:
	default:
	}
}

func (client *Client) getProxySceneItems() ([]*typedefs.SceneItem, error) {
	var sceneItems []*typedefs.SceneItem
	err := client.withObs(func(obs *goobs.Client) error {
		items, err := obs.SceneItems.GetSceneItemList(&goobRequestSceneItems.GetSceneItemListParams{
			SceneUuid: &client.cfg.ObsProxySceneUuid,
		})
		if err != nil {
			return err
		}
		sceneItems = items.SceneItems
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sceneItems, nil
}
