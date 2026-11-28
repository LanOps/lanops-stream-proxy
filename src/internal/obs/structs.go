package obs

import (
	"sync"

	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/andreykaipov/goobs"
)

type Client struct {
	cfg   config.Config
	db    *dbstreams.Client
	msgCh chan<- channels.MsgCh

	// mu guards conn. Every OBS call goes through withObs so two loops
	// (stream sync and rotation) never talk to the socket at the same time.
	mu   sync.Mutex
	conn *goobs.Client

	// textKind is resolved once from OBS rather than hardcoded, because the
	// available text source plugin differs between OBS builds.
	textKind string

	obsStreams struct {
		current  *dbstreams.Stream
		previous *dbstreams.Stream
	}
}

type ClientError struct {
	Err     error
	Message string
}

type CreateStreamMediaSourceInputOutput struct {
	InputUuid   string
	SceneItemId int64
}
