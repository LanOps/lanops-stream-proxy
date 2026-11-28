package dbstreams

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// QuietLogger drops GORM's "record not found" trace.
//
// The first sync for every new stream misses on the lookup by name, and the
// rotation asks for a stream after the last one every time it wraps, so a
// normal run prints that trace next to real output every 30 seconds. Whoever
// is watching the console at an event reads it as a fault.
type QuietLogger struct {
	logger.Interface
}

func (l QuietLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	l.Interface.Trace(ctx, begin, fc, err)
}

// NewQuietLogger keeps everything except the not-found trace, so real database
// errors still show up.
func NewQuietLogger() logger.Interface {
	return QuietLogger{Interface: logger.Default}
}
