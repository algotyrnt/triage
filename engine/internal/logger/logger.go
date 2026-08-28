// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger initializes and configures the default global structured slog logger.
func InitLogger() *slog.Logger {
	levelStr := strings.ToUpper(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	var level slog.Level
	switch levelStr {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	if format == "json" || env == "production" || env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
