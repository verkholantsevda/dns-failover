package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string, format string) *slog.Logger {

	var logLevel slog.Level

	switch strings.ToLower(level) {

	case "debug":
		logLevel = slog.LevelDebug

	case "warn":
		logLevel = slog.LevelWarn

	case "error":
		logLevel = slog.LevelError

	default:
		logLevel = slog.LevelInfo
	}


	opts := &slog.HandlerOptions{
		Level: logLevel,
	}


	var handler slog.Handler

	switch strings.ToLower(format) {

	case "json":
		handler = slog.NewJSONHandler(
			os.Stdout,
			opts,
		)

	default:
		handler = slog.NewTextHandler(
			os.Stdout,
			opts,
		)
	}


	return slog.New(handler)
}