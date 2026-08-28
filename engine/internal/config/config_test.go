// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", opts.Port)
	}
	if opts.DataDir != "data" {
		t.Errorf("expected default data-dir 'data', got %s", opts.DataDir)
	}
	if opts.DatabasePath() != "data/triage.db" {
		t.Errorf("expected default db path 'data/triage.db', got %s", opts.DatabasePath())
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
