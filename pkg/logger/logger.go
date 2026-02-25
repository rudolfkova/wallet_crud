// Package logger ...
package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/phsym/console-slog"
)

// NewLogger ...
func NewLogger(logLevel string) *slog.Logger {
	var lvl slog.Level

	if err := lvl.UnmarshalText([]byte(logLevel)); err != nil {
		lvl = slog.LevelInfo
	}

	handler := console.NewHandler(
		os.Stderr,
		&console.HandlerOptions{
			Level:      lvl,
			TimeFormat: time.TimeOnly,
		},
	)

	return slog.New(handler)
}
