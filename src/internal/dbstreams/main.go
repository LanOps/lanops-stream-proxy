package dbstreams

import (
	"errors"
	"lanops/obs-proxy-bridge/internal/config"
	"time"

	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB) (*Client, error) {
	client := &Client{
		cfg: cfg,
		db:  db,
	}
	return client, nil
}

func (client *Client) GetStreams() (dbStreams []Stream, err error) {
	result := client.db.Find(&dbStreams)
	if result.Error != nil {
		return dbStreams, result.Error
	}
	return dbStreams, nil
}

func (client *Client) GetStreamByName(streamName string) (stream Stream, err error) {
	err = client.db.First(&stream, "name = ?", streamName).Error
	return stream, err
}

func (client *Client) GetNextEnabledStream(currentStream *Stream) (stream *Stream, err error) {
	if currentStream != nil {
		err = client.db.Where("id > ? AND enabled = ? AND live = ?", currentStream.ID, true, true).Order("id ASC").First(&stream).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || currentStream == nil {
		err = client.db.Where("enabled = ? AND live = ?", true, true).Order("id ASC").First(&stream).Error
	}
	return stream, err
}

func (client *Client) GetStreamsCount() (count int64, err error) {
	err = client.db.Model(&Stream{}).Count(&count).Error
	return count, err
}

func (client *Client) GetAvailableStreamsCount() (count int64, err error) {
	err = client.db.Model(&Stream{}).Where("enabled = ? AND live = ?", true, true).Count(&count).Error
	return count, err
}

func (client *Client) CreateStream(streamName string) (stream Stream, err error) {
	stream, err = client.GetStreamByName(streamName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		params := map[string]interface{}{
			"name":            streamName,
			"enabled":         true,
			"live":            true,
			"last_seen":       time.Now(),
			"obs_stream_id":   nil,
			"obs_stream_uuid": nil,
			"obs_text_uuid":   nil,
			"obs_text_id":     nil,
		}
		if err := client.db.Model(&stream).Create(params).Error; err != nil {
			return stream, err
		}
	}
	return stream, nil
}

func (client *Client) DeleteStream(streamName string) error {
	var stream Stream

	if err := client.db.First(&stream, "name = ?", streamName).Error; err != nil {
		return err
	}
	if err := client.db.Unscoped().Delete(&stream).Error; err != nil {
		return err
	}
	return nil
}

func (client *Client) CheckStreamExistsByName(streamName string) (bool, error) {
	var stream Stream
	var count int64
	err := client.db.Model(stream).Where("name = ?", streamName).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, err
}

func (client *Client) CheckStreamExistsByObsSceneItemUuid(sceneItemUuid string) (bool, error) {
	var stream Stream
	var count int64
	err := client.db.Model(stream).Where("obs_stream_uuid = ? OR obs_text_uuid = ?", sceneItemUuid, sceneItemUuid).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, err
}

func (client *Client) GetStreamByObsSceneItemUuid(sceneItemUuid string) (stream Stream, err error) {
	if err := client.db.First(&stream, "obs_stream_uuid = ? OR obs_text_uuid = ?", sceneItemUuid, sceneItemUuid).Error; err != nil {
		return stream, err
	}
	return stream, nil
}

func (client *Client) UpdateStream(stream Stream, params map[string]interface{}) (Stream, error) {
	if err := client.db.Model(&stream).Updates(params).Error; err != nil {
		return stream, err
	}
	return stream, nil
}

// MarkStreamLive records that MediaMTX has a publisher for this stream now.
func (client *Client) MarkStreamLive(streamName string) error {
	return client.db.Model(&Stream{}).Where("name = ?", streamName).
		Updates(map[string]interface{}{"live": true, "last_seen": time.Now()}).Error
}

// MarkStreamOffline records that the publisher has gone, without deleting the
// row. The stream keeps its enabled flag, its rotation position and its OBS
// sources so a reconnect is not a new stream.
func (client *Client) MarkStreamOffline(streamName string) error {
	return client.db.Model(&Stream{}).Where("name = ?", streamName).
		Updates(map[string]interface{}{"live": false}).Error
}

// PruneOfflineStreams deletes rows that have been offline since before cutoff.
func (client *Client) PruneOfflineStreams(cutoff time.Time) error {
	return client.db.Unscoped().
		Where("live = ? AND last_seen < ?", false, cutoff).
		Delete(&Stream{}).Error
}

// IsStreamLive reports whether the stream exists and currently has a publisher.
func (client *Client) IsStreamLive(streamName string) (bool, error) {
	stream, err := client.GetStreamByName(streamName)
	if err != nil {
		return false, err
	}
	return stream.Live, nil
}
