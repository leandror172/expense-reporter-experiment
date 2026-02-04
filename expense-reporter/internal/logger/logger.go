// Package logger wraps log/slog to provide a single configurable logger for the app.
// Levels: Debug (verbose tracing), Info (normal ops), Warn, Error.
// Call SetDebug(true) to enable Debug-level output (e.g. from a --debug CLI flag).
package logger

import (
	"log/slog"
	"os"
)

var current *slog.Logger

func init() {
	// Default: Info level, no debug noise
	current = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SetDebug switches the logger to Debug level if enabled, Info otherwise.
func SetDebug(enabled bool) {
	level := slog.LevelInfo
	if enabled {
		level = slog.LevelDebug
	}
	current = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// Logger returns the current *slog.Logger so callers can do logger.Logger().Debug(...).
func Logger() *slog.Logger { return current }

// Convenience wrappers so callers can write logger.Debug("msg", "key", val) directly.
func Debug(msg string, args ...any) { current.Debug(msg, args...) }
func Info(msg string, args ...any)  { current.Info(msg, args...) }
func Warn(msg string, args ...any)  { current.Warn(msg, args...) }
func Error(msg string, args ...any) { current.Error(msg, args...) }
