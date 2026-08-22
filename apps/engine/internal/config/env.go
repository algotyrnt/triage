// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// EnvConfig contains immutable startup configuration loaded from the environment.
type EnvConfig struct {
	DatabaseURL string
	Port        string
	LogLevel    string
}

// LoadEnv reads environment variables (loading .env.local / .env if present)
// and returns a validated EnvConfig. Returns an error if required variables are missing.
func LoadEnv() (*EnvConfig, error) {
	_ = godotenv.Load(".env.local", ".env")

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return nil, fmt.Errorf("missing required environment variable: DATABASE_URL")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	logLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "info"
	}

	return &EnvConfig{
		DatabaseURL: dbURL,
		Port:        port,
		LogLevel:    logLevel,
	}, nil
}
