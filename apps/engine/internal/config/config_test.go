// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"os"
	"testing"
)

func TestLoadEnv_MissingDatabaseURL(t *testing.T) {
	_ = os.Unsetenv("DATABASE_URL")
	_, err := LoadEnv()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
	}
}

func TestLoadEnv_ValidConfig(t *testing.T) {
	_ = os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	_ = os.Setenv("PORT", "9090")
	_ = os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := LoadEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("unexpected database URL: %s", cfg.DatabaseURL)
	}
	if cfg.Port != "9090" {
		t.Errorf("unexpected port: %s", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("unexpected log level: %s", cfg.LogLevel)
	}
}

func TestStore_NilDB(t *testing.T) {
	s := NewStore(nil)
	ctx := context.Background()

	llmCfg := s.GetLLM(ctx)
	if llmCfg.APIKey != "" || llmCfg.Model != "" {
		t.Errorf("expected empty LLM config for nil db, got (%s, %s)", llmCfg.APIKey, llmCfg.Model)
	}

	if url := s.GetInstanceURL(ctx); url != "" {
		t.Errorf("expected empty instance url for nil db, got %s", url)
	}

	app, err := s.GetGitHubApp(ctx)
	if err != nil || app != nil {
		t.Errorf("expected nil app config for nil db, got %v, err=%v", app, err)
	}

	secret, err := s.EnsureSessionSecret(ctx)
	if err != nil || secret == "" {
		t.Errorf("expected fallback session secret for nil db, got %s, err=%v", secret, err)
	}
}
