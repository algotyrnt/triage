// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"triage/engine/internal/db"
	"triage/engine/internal/llm"
)

func generateTestRSAPEM(t *testing.T) string {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(privKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	return string(pem.EncodeToMemory(block))
}

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

	optsEmpty := &Options{DataDir: ""}
	if optsEmpty.DatabasePath() != "data/triage.db" {
		t.Errorf("expected 'data/triage.db' for empty DataDir, got %s", optsEmpty.DatabasePath())
	}

	optsCustom := &Options{DataDir: "/var/triage"}
	if optsCustom.DatabasePath() != "/var/triage/triage.db" {
		t.Errorf("expected '/var/triage/triage.db', got %s", optsCustom.DatabasePath())
	}
}

func TestStore_NilDB(t *testing.T) {
	s := NewStore(nil)
	ctx := context.Background()

	llmCfg := s.GetLLM(ctx)
	if llmCfg.Provider != "gemini" {
		t.Errorf("expected default gemini provider, got %s", llmCfg.Provider)
	}
	if err := s.SaveLLM(ctx, llm.Config{}); err != nil {
		t.Errorf("expected nil error on nil db SaveLLM: %v", err)
	}

	if url := s.GetInstanceURL(ctx); url != "" {
		t.Errorf("expected empty instance url for nil db, got %s", url)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.dev/")
	if url := s.GetInstanceURL(ctx, req); url != "https://app.dev" {
		t.Errorf("expected fallback to Origin header, got %s", url)
	}

	if err := s.SaveInstanceURL(ctx, "https://triage.dev"); err != nil {
		t.Errorf("expected nil error on nil db SaveInstanceURL: %v", err)
	}

	app, err := s.GetGitHubApp(ctx)
	if err != nil || app != nil {
		t.Errorf("expected nil app config for nil db, got %v, err=%v", app, err)
	}
	if err := s.SaveGitHubApp(ctx, GitHubAppParams{}); err != nil {
		t.Errorf("expected nil error on nil db SaveGitHubApp: %v", err)
	}

	id, sec := s.GetGitHubOAuth(ctx)
	if id != "" || sec != "" {
		t.Errorf("expected empty OAuth on nil db")
	}
	if err := s.SaveGitHubOAuth(ctx, "id", "sec"); err != nil {
		t.Errorf("expected nil error on nil db SaveGitHubOAuth: %v", err)
	}

	if slug := s.GetGitHubAppSlug(ctx); slug != "" {
		t.Errorf("expected empty slug on nil db, got %s", slug)
	}
	if err := s.SaveGitHubAppSlug(ctx, "my-app"); err != nil {
		t.Errorf("expected nil error on nil db SaveGitHubAppSlug: %v", err)
	}

	secret, err := s.EnsureSessionSecret(ctx)
	if err != nil || secret == "" {
		t.Errorf("expected fallback session secret for nil db, got %s, err=%v", secret, err)
	}
	secret2, err := s.GetSessionSecret(ctx)
	if err != nil || secret2 != secret {
		t.Errorf("expected cached session secret %s, got %s", secret, secret2)
	}

	completed, err := s.IsSetupCompleted(ctx)
	if err != nil || completed {
		t.Errorf("expected false for IsSetupCompleted on nil db, got %v", completed)
	}

	// nil ctx branches
	if s.GetGitHubAppSlug(nil) != "" {
		t.Errorf("expected empty slug for nil ctx")
	}
	oID, oSec := s.GetGitHubOAuth(nil)
	if oID != "" || oSec != "" {
		t.Errorf("expected empty oauth for nil ctx")
	}
	gApp, gErr := s.GetGitHubApp(nil)
	if gErr != nil || gApp != nil {
		t.Errorf("expected nil app for nil ctx")
	}
}

func TestStore_WithDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "store_test.db")
	ctx := context.Background()

	database, err := db.NewDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}
	defer database.Close()

	s := NewStore(database)

	// 1. LLM configuration
	initialLLM := s.GetLLM(ctx)
	if initialLLM.Provider != "gemini" {
		t.Errorf("expected default provider gemini, got %s", initialLLM.Provider)
	}

	// Save LLM with specific values
	err = s.SaveLLM(ctx, llm.Config{
		Provider: "openai",
		APIKey:   "sk-test-12345",
		Model:    "gpt-4o",
		BaseURL:  "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("SaveLLM failed: %v", err)
	}

	savedLLM := s.GetLLM(ctx)
	if savedLLM.Provider != "openai" || savedLLM.APIKey != "sk-test-12345" || savedLLM.Model != "gpt-4o" || savedLLM.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("unexpected saved LLM config: %+v", savedLLM)
	}

	// Save LLM with empty provider (defaults to gemini)
	err = s.SaveLLM(ctx, llm.Config{
		Provider: "",
		APIKey:   "gemini-key",
	})
	if err != nil {
		t.Fatalf("SaveLLM with empty provider failed: %v", err)
	}
	geminiLLM := s.GetLLM(ctx)
	if geminiLLM.Provider != "gemini" {
		t.Errorf("expected provider gemini, got %s", geminiLLM.Provider)
	}

	// 2. Instance URL
	if u := s.GetInstanceURL(ctx); u != "" {
		t.Errorf("expected empty instance url initially, got %s", u)
	}
	// Fallback to app_url
	_ = database.SaveInstanceConfig(ctx, KeyAppURL, "https://fallback-app.dev/")
	if u := s.GetInstanceURL(ctx); u != "https://fallback-app.dev" {
		t.Errorf("expected fallback app_url, got %s", u)
	}

	// Save instance_url explicitly
	err = s.SaveInstanceURL(ctx, "https://triage.production.dev/")
	if err != nil {
		t.Fatalf("SaveInstanceURL failed: %v", err)
	}
	if u := s.GetInstanceURL(ctx); u != "https://triage.production.dev" {
		t.Errorf("expected https://triage.production.dev, got %s", u)
	}

	// 3. GitHub App Configuration
	appNone, err := s.GetGitHubApp(ctx)
	if err != nil || appNone != nil {
		t.Errorf("expected nil app config initially, got %v, err=%v", appNone, err)
	}

	// Invalid app ID string
	_ = database.SaveInstanceConfig(ctx, KeyGitHubAppID, "not-a-number")
	_ = database.SaveInstanceConfig(ctx, KeyGitHubAppPrivateKey, "dummy-pem")
	_, err = s.GetGitHubApp(ctx)
	if err == nil {
		t.Errorf("expected error for non-numeric app ID")
	}

	// Save complete valid GitHub App
	pemStr := generateTestRSAPEM(t)
	appParams := GitHubAppParams{
		ID:            123456,
		Slug:          "triage-bot",
		PEM:           pemStr,
		WebhookSecret: "whsec_123456",
		ClientID:      "Iv1.123456",
		ClientSecret:  "client_secret_xyz",
	}
	err = s.SaveGitHubApp(ctx, appParams)
	if err != nil {
		t.Fatalf("SaveGitHubApp failed: %v", err)
	}

	loadedApp, err := s.GetGitHubApp(ctx)
	if err != nil || loadedApp == nil {
		t.Fatalf("GetGitHubApp failed: %v", err)
	}
	if loadedApp.AppID != 123456 || loadedApp.ClientID != "Iv1.123456" {
		t.Errorf("unexpected loaded app: %+v", loadedApp)
	}

	// 4. GitHub OAuth
	oClientID, oClientSecret := s.GetGitHubOAuth(ctx)
	if oClientID != "Iv1.123456" || oClientSecret != "client_secret_xyz" {
		t.Errorf("unexpected OAuth config: (%s, %s)", oClientID, oClientSecret)
	}

	err = s.SaveGitHubOAuth(ctx, "new_client_id", "new_client_secret")
	if err != nil {
		t.Fatalf("SaveGitHubOAuth failed: %v", err)
	}
	newID, newSec := s.GetGitHubOAuth(ctx)
	if newID != "new_client_id" || newSec != "new_client_secret" {
		t.Errorf("unexpected updated OAuth: (%s, %s)", newID, newSec)
	}

	// 5. GitHub App Slug
	slug := s.GetGitHubAppSlug(ctx)
	if slug != "triage-bot" {
		t.Errorf("expected triage-bot slug, got %s", slug)
	}
	err = s.SaveGitHubAppSlug(ctx, "triage-bot-v2")
	if err != nil {
		t.Fatalf("SaveGitHubAppSlug failed: %v", err)
	}
	if s.GetGitHubAppSlug(ctx) != "triage-bot-v2" {
		t.Errorf("expected updated slug triage-bot-v2")
	}

	// 6. Session Secret
	// Insecure default gets replaced
	_ = database.SaveInstanceConfig(ctx, KeySessionSecret, InsecureDefaultSecret)
	sec1, err := s.EnsureSessionSecret(ctx)
	if err != nil || sec1 == InsecureDefaultSecret || len(sec1) < 32 {
		t.Fatalf("expected insecure secret to be replaced with secure secret, got %s, err=%v", sec1, err)
	}

	// Second call retrieves persisted secret without regenerating
	sec2, err := s.GetSessionSecret(ctx)
	if err != nil || sec2 != sec1 {
		t.Errorf("expected matching session secret %s, got %s", sec1, sec2)
	}

	// 7. Setup completed
	isCompleted, err := s.IsSetupCompleted(ctx)
	if err != nil || !isCompleted {
		t.Errorf("expected IsSetupCompleted=true after saving github app and oauth, got %v, err=%v", isCompleted, err)
	}

	// 8. Canceled context error branches
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.SaveLLM(canceledCtx, llm.Config{Provider: "openai"}); err == nil {
		t.Errorf("expected error from SaveLLM with canceled context")
	}
	if err := s.SaveGitHubApp(canceledCtx, appParams); err == nil {
		t.Errorf("expected error from SaveGitHubApp with canceled context")
	}
	if err := s.SaveGitHubOAuth(canceledCtx, "id", "sec"); err == nil {
		t.Errorf("expected error from SaveGitHubOAuth with canceled context")
	}
	if _, err := s.EnsureSessionSecret(canceledCtx); err == nil {
		t.Errorf("expected error from EnsureSessionSecret with canceled context")
	}
}
