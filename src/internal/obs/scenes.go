package obs

import (
	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/typedefs"
)

func (client *Client) getScenes() ([]*typedefs.Scene, error) {
	var scenes []*typedefs.Scene
	err := client.withObs(func(obs *goobs.Client) error {
		items, err := obs.Scenes.GetSceneList()
		if err != nil {
			return err
		}
		scenes = items.Scenes
		return nil
	})
	if err != nil {
		return nil, err
	}
	return scenes, nil
}
