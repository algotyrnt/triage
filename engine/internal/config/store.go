// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/llm"
)

// InsecureDefaultSecret defines the rejected insecure legacy default key.
const InsecureDefaultSecret = "dev-secret-change-me-in-production"

// Keys used in the PostgreSQL instance_config table.
const (
	KeyInstanceURL          = "instance_url"
	KeyAppURL               = "app_url"
	KeyLLMProvider          = "llm_provider"
	KeyLLMAPIKey            = "llm_api_key"
	KeyLLMModel             = "llm_model"
	KeyLLMBaseURL           = "llm_base_url"
	KeyGitHubAppID          = "github_app_id"
	KeyGitHubAppSlug        = "github_app_slug"
	KeyGitHubAppPrivateKey  = "github_app_private_key"
	KeyGitHubAppWebhookSec  = "github_app_webhook_secret"
	KeyGitHubAppClientID    = "github_app_client_id"
	KeyGitHubAppClientSec   = "github_app_client_secret"
	KeyGitHubOAuthClientID  = "github_oauth_client_id"
	KeyGitHubOAuthClientSec = "github_oauth_client_secret"
	KeySessionSecret        = "session_secret"
)

// GitHubAppParams encapsulates registration payload for the GitHub App.
type GitHubAppParams struct {
	ID            int64
	Slug          string
	PEM           string
	WebhookSecret string
	ClientID      string
	ClientSecret  string
}

// Store manages reading and writing dynamic instance configuration from PostgreSQL.
type Store struct {
	db               *db.DB
	memMu            sync.Mutex
	memSessionSecret string
}

// NewStore initializes a new configuration store with the given database instance.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// GetLLM retrieves configured LLM provider configuration with database and environment variable fallbacks.
func (s *Store) GetLLM(ctx context.Context) llm.Config {
	var cfg llm.Config

	if s.db != nil && ctx != nil {
		cfg.Provider, _ = s.db.GetInstanceConfig(ctx, KeyLLMProvider)
		cfg.APIKey, _ = s.db.GetInstanceConfig(ctx, KeyLLMAPIKey)
		cfg.Model, _ = s.db.GetInstanceConfig(ctx, KeyLLMModel)
		cfg.BaseURL, _ = s.db.GetInstanceConfig(ctx, KeyLLMBaseURL)
	}

	if cfg.Provider == "" {
		cfg.Provider = "gemini"
	}

	return llm.Config{
		Provider: strings.TrimSpace(cfg.Provider),
		APIKey:   strings.TrimSpace(cfg.APIKey),
		Model:    strings.TrimSpace(cfg.Model),
		BaseURL:  strings.TrimSpace(cfg.BaseURL),
	}
}

// SaveLLM persists the LLM provider configuration.
func (s *Store) SaveLLM(ctx context.Context, cfg llm.Config) error {
	if s.db == nil {
		return nil
	}

	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = "gemini"
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyLLMProvider, provider); err != nil {
		return err
	}

	if cfg.APIKey != "" {
		if err := s.db.SaveInstanceConfig(ctx, KeyLLMAPIKey, strings.TrimSpace(cfg.APIKey)); err != nil {
			return err
		}
	}

	if cfg.Model != "" {
		if err := s.db.SaveInstanceConfig(ctx, KeyLLMModel, strings.TrimSpace(cfg.Model)); err != nil {
			return err
		}
	}

	if err := s.db.SaveInstanceConfig(ctx, KeyLLMBaseURL, strings.TrimSpace(cfg.BaseURL)); err != nil {
		return err
	}

	return nil
}

// GetInstanceURL returns the public dashboard URL configured in the database,
// or falls back to the HTTP request Origin header during initial setup.
func (s *Store) GetInstanceURL(ctx context.Context, r ...*http.Request) string {
	if s.db != nil && ctx != nil {
		if u, _ := s.db.GetInstanceConfig(ctx, KeyInstanceURL); u != "" {
			return strings.TrimRight(strings.TrimSpace(u), "/")
		}
		if u, _ := s.db.GetInstanceConfig(ctx, KeyAppURL); u != "" {
			return strings.TrimRight(strings.TrimSpace(u), "/")
		}
	}
	if len(r) > 0 && r[0] != nil {
		if origin := r[0].Header.Get("Origin"); origin != "" {
			return strings.TrimRight(strings.TrimSpace(origin), "/")
		}
	}
	return ""
}

// SaveInstanceURL persists the public dashboard URL.
func (s *Store) SaveInstanceURL(ctx context.Context, url string) error {
	if s.db == nil {
		return nil
	}
	return s.db.SaveInstanceConfig(ctx, KeyInstanceURL, strings.TrimRight(strings.TrimSpace(url), "/"))
}

// GetGitHubApp loads the GitHub App configuration from database settings.
func (s *Store) GetGitHubApp(ctx context.Context) (*github.AppConfig, error) {
	if s.db == nil || ctx == nil {
		return nil, nil
	}

	appIDStr, _ := s.db.GetInstanceConfig(ctx, KeyGitHubAppID)
	pemKey, _ := s.db.GetInstanceConfig(ctx, KeyGitHubAppPrivateKey)
	webhookSecret, _ := s.db.GetInstanceConfig(ctx, KeyGitHubAppWebhookSec)
	if webhookSecret == "" {
		webhookSecret, _ = s.db.GetInstanceConfig(ctx, "github_webhook_secret")
	}

	clientID, _ := s.db.GetInstanceConfig(ctx, KeyGitHubAppClientID)
	if clientID == "" {
		clientID, _ = s.db.GetInstanceConfig(ctx, "github_client_id")
	}

	clientSecret, _ := s.db.GetInstanceConfig(ctx, KeyGitHubAppClientSec)
	if clientSecret == "" {
		clientSecret, _ = s.db.GetInstanceConfig(ctx, "github_client_secret")
	}

	if appIDStr == "" || pemKey == "" {
		return nil, nil
	}

	appID, err := strconv.ParseInt(strings.TrimSpace(appIDStr), 10, 64)
	if err != nil || appID <= 0 {
		return nil, err
	}

	return github.LoadAppConfig(appID, []byte(pemKey), webhookSecret, clientID, clientSecret)
}

// SaveGitHubApp saves the complete GitHub App credential payload.
func (s *Store) SaveGitHubApp(ctx context.Context, params GitHubAppParams) error {
	if s.db == nil {
		return nil
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubAppID, strconv.FormatInt(params.ID, 10)); err != nil {
		return err
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubAppSlug, params.Slug); err != nil {
		return err
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubAppPrivateKey, params.PEM); err != nil {
		return err
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubAppWebhookSec, params.WebhookSecret); err != nil {
		return err
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubAppClientID, params.ClientID); err != nil {
		return err
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubAppClientSec, params.ClientSecret); err != nil {
		return err
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubOAuthClientID, params.ClientID); err != nil {
		return err
	}
	return s.db.SaveInstanceConfig(ctx, KeyGitHubOAuthClientSec, params.ClientSecret)
}

// GetGitHubOAuth retrieves the OAuth client ID and client secret.
func (s *Store) GetGitHubOAuth(ctx context.Context) (clientID, clientSecret string) {
	if s.db == nil || ctx == nil {
		return "", ""
	}
	clientID, _ = s.db.GetInstanceConfig(ctx, KeyGitHubOAuthClientID)
	clientSecret, _ = s.db.GetInstanceConfig(ctx, KeyGitHubOAuthClientSec)
	return strings.TrimSpace(clientID), strings.TrimSpace(clientSecret)
}

// SaveGitHubOAuth persists GitHub OAuth credentials.
func (s *Store) SaveGitHubOAuth(ctx context.Context, clientID, clientSecret string) error {
	if s.db == nil {
		return nil
	}
	if err := s.db.SaveInstanceConfig(ctx, KeyGitHubOAuthClientID, strings.TrimSpace(clientID)); err != nil {
		return err
	}
	return s.db.SaveInstanceConfig(ctx, KeyGitHubOAuthClientSec, strings.TrimSpace(clientSecret))
}

// GetGitHubAppSlug retrieves the configured GitHub App slug name.
func (s *Store) GetGitHubAppSlug(ctx context.Context) string {
	if s.db == nil || ctx == nil {
		return ""
	}
	slug, _ := s.db.GetInstanceConfig(ctx, KeyGitHubAppSlug)
	return strings.TrimSpace(slug)
}

// SaveGitHubAppSlug persists the GitHub App slug name.
func (s *Store) SaveGitHubAppSlug(ctx context.Context, slug string) error {
	if s.db == nil {
		return nil
	}
	return s.db.SaveInstanceConfig(ctx, KeyGitHubAppSlug, strings.TrimSpace(slug))
}

// EnsureSessionSecret returns the existing secure session secret or auto-generates and persists a new 256-bit crypto key.
func (s *Store) EnsureSessionSecret(ctx context.Context) (string, error) {
	if s.db == nil {
		s.memMu.Lock()
		defer s.memMu.Unlock()
		if s.memSessionSecret != "" {
			return s.memSessionSecret, nil
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random session secret: %w", err)
		}
		s.memSessionSecret = hex.EncodeToString(b)
		return s.memSessionSecret, nil
	}

	secret, err := s.db.GetInstanceConfig(ctx, KeySessionSecret)
	if err == nil && secret != "" && secret != InsecureDefaultSecret && len(secret) >= 32 {
		return secret, nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random session secret: %w", err)
	}
	newSecret := hex.EncodeToString(b)
	if err := s.db.SaveInstanceConfig(ctx, KeySessionSecret, newSecret); err != nil {
		return "", fmt.Errorf("failed to persist session secret: %w", err)
	}
	return newSecret, nil
}

// GetSessionSecret retrieves the active session signing key, ensuring it exists and is secure.
func (s *Store) GetSessionSecret(ctx context.Context) (string, error) {
	return s.EnsureSessionSecret(ctx)
}

// IsSetupCompleted checks whether core required setup items have been configured.
func (s *Store) IsSetupCompleted(ctx context.Context) (bool, error) {
	if s.db == nil {
		return false, nil
	}
	return s.db.IsInstanceConfigured(ctx)
}
