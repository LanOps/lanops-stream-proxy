package obs

import (
	"fmt"

	"github.com/andreykaipov/goobs"
)

// textKindPreference is ordered so the current behaviour keeps working: the
// FreeType2 source is what this proxy has always used, and the GDI+ sources are
// only a fallback for OBS builds that do not ship FreeType2.
var textKindPreference = []string{
	"text_ft2_source_v2",
	"text_ft2_source",
	"text_gdiplus_v3",
	"text_gdiplus_v2",
	"text_gdiplus",
}

func pickTextKind(available []string) string {
	for _, want := range textKindPreference {
		for _, have := range available {
			if have == want {
				return want
			}
		}
	}
	return ""
}

// getTextInputKind asks OBS which text source plugins it actually has. The
// source id differs between builds (FreeType2 on Linux, GDI+ on older Windows
// installs), and CreateInput fails outright for an id OBS does not know, which
// leaves streams invisible with only a sync error to show for it.
func (client *Client) getTextInputKind() (string, error) {
	if client.textKind != "" {
		return client.textKind, nil
	}

	var available []string
	err := client.withObs(func(obs *goobs.Client) error {
		resp, err := obs.Inputs.GetInputKindList()
		if err != nil {
			return err
		}
		available = resp.InputKinds
		return nil
	})
	if err != nil {
		return "", err
	}

	kind := pickTextKind(available)
	if kind == "" {
		return "", fmt.Errorf("no supported text source found in OBS, a text source is needed for the stream name overlay")
	}
	client.textKind = kind
	return kind, nil
}
