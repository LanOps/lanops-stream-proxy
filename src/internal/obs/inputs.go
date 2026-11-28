package obs

import (
	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"

	"github.com/andreykaipov/goobs"
)

func (client *Client) deleteInput(inputUuid string) error {
	params := goobRequestInputs.NewRemoveInputParams().
		WithInputUuid(inputUuid)
	return client.withObs(func(obs *goobs.Client) error {
		_, err := obs.Inputs.RemoveInput(params)
		return err
	})
}
