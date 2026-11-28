package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func Load() Config {
	_ = godotenv.Load()

	// DB
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		log.Fatal("❌ DB_PATH not set in environment")
	}

	// OBS
	obsWebSocketAddress := os.Getenv("OBS_WEBSOCKET_ADDRESS")
	if obsWebSocketAddress == "" {
		log.Fatal("❌ OBS_WEBSOCKET_ADDRESS not set in environment")
	}
	// OBS WebSocket auth can be turned off, in which case there is no
	// password to supply. An empty value is valid here, unlike the rest.
	obsWebSocketPassword := os.Getenv("OBS_WEBSOCKET_PASSWORD")
	// Media sources should stop decoding while they are not visible. Without
	// this every stream the proxy knows about decodes at once, whether or not
	// the rotation is showing it, which is what makes OBS crawl at an event
	// with a lot of streamers. Optional, defaults to on.
	obsCloseWhenInactive := true
	if v := os.Getenv("OBS_CLOSE_WHEN_INACTIVE"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			obsCloseWhenInactive = parsed
		}
	}
	obsProxySceneName := os.Getenv("OBS_PROXY_SCENE_NAME")
	if obsProxySceneName == "" {
		log.Fatal("❌ OBS_PROXY_SCENE_NAME not set in environment")
	}

	// Media MTX
	mediaMtxApiAddress := os.Getenv("MEDIAMTX_API_ADDRESS")
	if mediaMtxApiAddress == "" {
		log.Fatal("❌ MEDIAMTX_API_ADDRESS not set in environment")
	}
	mediaMtxRtmpAddress := os.Getenv("MEDIAMTX_RTMP_ADDRESS")
	if mediaMtxRtmpAddress == "" {
		log.Fatal("❌ MEDIAMTX_RTMP_ADDRESS not set in environment")
	}

	// API
	apiAdminUsername := os.Getenv("API_ADMIN_USERNAME")
	if apiAdminUsername == "" {
		log.Fatal("❌ API_ADMIN_USERNAME not set in environment")
	}
	apiAdminPassword := os.Getenv("API_ADMIN_PASSWORD")
	if apiAdminPassword == "" {
		log.Fatal("❌ API_ADMIN_PASSWORD not set in environment")
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		log.Fatal("❌ API_PORT not set in environment")
	}

	return Config{
		DbPath:               dbPath,
		ObsWebSocketAddress:  obsWebSocketAddress,
		ObsWebSocketPassword: obsWebSocketPassword,
		ObsProxySceneName:    obsProxySceneName,
		ObsCloseWhenInactive: obsCloseWhenInactive,
		MediaMtxApiAddress:   mediaMtxApiAddress,
		MediaMtxRtmpAddress:  mediaMtxRtmpAddress,
		ApiAdminUsername:     apiAdminUsername,
		ApiAdminPassword:     apiAdminPassword,
		ApiPort:              apiPort,
	}
}
