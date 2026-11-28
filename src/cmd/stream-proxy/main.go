package main

import (
	"lanops/obs-proxy-bridge/api"
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"lanops/obs-proxy-bridge/internal/mediamtx"
	"lanops/obs-proxy-bridge/internal/obs"

	"os"
	"runtime"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const (
	streamSyncInterval = 7 * time.Second
	rotationInterval   = 30 * time.Second
	obsRetryInterval   = 5 * time.Second
)

var (
	logger          zerolog.Logger
	cfg             config.Config
	msgCh           = make(chan channels.MsgCh, 20)
	dbStreamsClient *dbstreams.Client
	mediamtxClient  *mediamtx.Client
	obsClient       *obs.Client
)

func main() {
	logger = zerolog.New(
		// cmd.exe does not interpret the ANSI escapes ConsoleWriter emits.
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: runtime.GOOS == "windows"},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	logger.Info().Msg("Initializing Stream Proxy")

	// Start draining the message channel before anything can write to it.
	go func() {
		for msg := range msgCh {
			logger.Info().Msg(msg.Message)
		}
	}()

	logger.Info().Msg("Loading Config")
	cfg = config.Load()

	db, err := gorm.Open(sqlite.Open(cfg.DbPath), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed Connecting to DB")
	}
	if err := db.AutoMigrate(dbstreams.Stream{}); err != nil {
		logger.Fatal().Err(err).Msg("Failed to migrate DB")
	}

	dbStreamsClient, err = dbstreams.New(cfg, db)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create DB Streams Client")
	}

	mediamtxClient, err = mediamtx.New(cfg, dbStreamsClient, msgCh)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create MediaMTX Client")
	}

	// The OBS connection is opened lazily by the obs package, so nothing here
	// depends on OBS being up yet.
	obsClient, err = obs.New(cfg, dbStreamsClient, nil, msgCh)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to OBS Client")
	}

	// OBS is normally started by hand, often after the rest of the stack, so
	// wait for it instead of exiting. Exiting here is what put the container
	// into a restart loop when OBS was opened a minute late.
	logger.Info().Msg("Waiting for OBS")
	for attempt := 1; ; attempt++ {
		err := obsClient.InitProxyScene()
		if err == nil {
			break
		}
		logger.Error().Err(err.Err).Msgf("%s (attempt %d, retrying in %s)", err.Message, attempt, obsRetryInterval)
		time.Sleep(obsRetryInterval)
	}

	logger.Info().Msg("Starting Proxy Stream")

	logger.Info().Msg("Starting API")
	go api.Run()

	logger.Info().Msg("Starting Stream Syncs")
	go func() {
		c := time.NewTicker(streamSyncInterval)
		defer c.Stop()
		for range c.C {
			if err := mediamtxClient.SyncStreams(); err != nil {
				logger.Error().Err(err.Err).Msg(err.Message)
			}
			if err := obsClient.SyncStreams(); err != nil {
				logger.Error().Err(err.Err).Msg(err.Message)
			}
		}
	}()

	logger.Info().Msg("Starting Active Stream Rotation")
	c := time.NewTicker(rotationInterval)
	defer c.Stop()
	for range c.C {
		if err := obsClient.RotateActiveStream(); err != nil {
			logger.Error().Err(err.Err).Msg(err.Message)
		}
	}
}
