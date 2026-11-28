package dbstreams

import (
	"time"

	"lanops/obs-proxy-bridge/internal/config"

	"gorm.io/gorm"
)

type Client struct {
	cfg config.Config
	db  *gorm.DB
}

type Stream struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"unique" json:"name"`
	Enabled bool   `gorm:"default:false" json:"enabled"`
	// Live is whether MediaMTX currently has a publisher for this stream.
	// Rows outlive the publisher so a reconnecting encoder keeps its enabled
	// state, its place in the rotation and its OBS sources.
	Live     bool      `gorm:"default:true" json:"live"`
	LastSeen time.Time `json:"lastSeen"`
	// We need both UUID and ID due to imitations with the OBS WebSockets
	ObsStreamUuid string `json:"obsStreamUuid"`
	ObsStreamId   int    `json:"obsStreamId"`
	ObsTextUuid   string `json:"obsTextUuid"`
	ObsTextId     int    `json:"obsTextId"`
}
