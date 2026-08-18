// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
)

var githubNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)

// Config holds runtime dependencies and configurations for the API server.
type Config struct {
	DB            *db.DB
	GitHubApp     *github.AppConfig
	ASTManager    *ast.Manager
	ASTCache      *ast.ASTCache
	ASTFetcher    *ast.OnDemandFetcher
	SessionSecret string
	AppURL        string
	EngineURL     string
	Version       string
}

// Server encapsulates HTTP routes and middleware for the Triage Engine.
type Server struct {
	db            *db.DB
	githubApp     *github.AppConfig
	astManager    *ast.Manager
	astCache      *ast.ASTCache
	astFetcher    *ast.OnDemandFetcher
	sessionSecret string
	appURL        string
	engineURL     string
	version       string
}

// NewServer initializes a new API server with the provided dependencies.
func NewServer(cfg Config) *Server {
	if cfg.ASTCache == nil {
		cfg.ASTCache = ast.NewASTCache()
	}
	if cfg.ASTFetcher == nil {
		cfg.ASTFetcher = ast.NewOnDemandFetcher()
	}
	if cfg.AppURL == "" {
		cfg.AppURL = "http://localhost:3000"
	}
	if cfg.EngineURL == "" {
		cfg.EngineURL = "http://localhost:8080"
	}
	if cfg.Version == "" {
		cfg.Version = "v0.1.0-dev"
	}

	return &Server{
		db:            cfg.DB,
		githubApp:     cfg.GitHubApp,
		astManager:    cfg.ASTManager,
		astCache:      cfg.ASTCache,
		astFetcher:    cfg.ASTFetcher,
		sessionSecret: cfg.SessionSecret,
		appURL:        cfg.AppURL,
		engineURL:     cfg.EngineURL,
		version:       cfg.Version,
	}
}

// RegisterRoutes registers all API routes onto the given ServeMux with standard middleware.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Core routes
	mux.HandleFunc("/health", s.withMiddleware(s.HandleHealth))
	mux.HandleFunc("/api/v1/telemetry", s.withMiddleware(s.HandleTelemetry))
	mux.HandleFunc("/api/v1/ast/index", s.withMiddleware(s.HandleASTIndex))
	mux.HandleFunc("/api/v1/incidents", s.withMiddleware(s.HandleIncidents))
	mux.HandleFunc("/api/v1/incidents/create-issue", s.withMiddleware(s.HandleCreateIncidentIssue))
	mux.HandleFunc("/api/v1/incidents/create-pr", s.withMiddleware(s.HandleCreateIncidentPR))
	mux.HandleFunc("/api/v1/projects", s.withMiddleware(s.HandleProjects))
	mux.HandleFunc("/api/v1/stats", s.withMiddleware(s.HandleStats))
	mux.HandleFunc("/api/v1/repos/detect-modules", s.withMiddleware(s.HandleDetectModules))

	// Setup wizard routes
	mux.HandleFunc("/api/v1/setup/status", s.withMiddleware(s.HandleSetupStatus))
	mux.HandleFunc("/api/v1/setup/manifest", s.withMiddleware(s.HandleSetupManifest))
	mux.HandleFunc("/api/v1/setup/callback", s.withMiddleware(s.HandleSetupCallback))
	mux.HandleFunc("/api/v1/setup/install", s.withMiddleware(s.HandleSetupInstall))
	mux.HandleFunc("/api/v1/setup/install/callback", s.withMiddleware(s.HandleSetupInstallCallback))
	mux.HandleFunc("/api/v1/setup/oauth", s.withMiddleware(s.HandleSetupOAuth))
	mux.HandleFunc("/api/v1/setup/llm", s.withMiddleware(s.HandleSetupLLM))
	mux.HandleFunc("/api/v1/setup/test", s.withMiddleware(s.HandleSetupTest))
	mux.HandleFunc("/api/v1/setup/repos", s.withMiddleware(s.HandleSetupRepos))
	mux.HandleFunc("/api/v1/setup/check-repo", s.withMiddleware(s.HandleCheckRepo))

	// Settings & Key management routes
	mux.HandleFunc("/api/v1/settings/llm", s.withMiddleware(s.HandleSettingsLLM))
	mux.HandleFunc("/api/v1/projects/keys", s.withMiddleware(s.HandleProjectKeys))
	mux.HandleFunc("/api/v1/projects/keys/revoke", s.withMiddleware(s.HandleRevokeProjectKey))
	mux.HandleFunc("/api/v1/gemini/analyze-panic", s.withMiddleware(s.HandleGeminiAnalyzePanic))
	mux.HandleFunc("/api/v1/gemini/generate-patch", s.withMiddleware(s.HandleGeminiGeneratePatch))

	// GitHub OAuth authentication routes
	mux.HandleFunc("/api/v1/auth/github", s.withMiddleware(s.HandleGitHubAuth))
	mux.HandleFunc("/api/v1/auth/github/callback", s.withMiddleware(s.HandleGitHubCallback))
	mux.HandleFunc("/api/v1/auth/me", s.withMiddleware(s.HandleAuthMe))
}

// LoadGitHubAppConfig refreshes the GitHub App credentials from PostgreSQL instance configuration.
func (s *Server) LoadGitHubAppConfig(ctx context.Context) {
	if s.db == nil {
		return
	}
	appIDStr, _ := s.db.GetInstanceConfig(ctx, "github_app_id")
	privateKey, _ := s.db.GetInstanceConfig(ctx, "github_app_private_key")
	clientID, _ := s.db.GetInstanceConfig(ctx, "github_client_id")
	clientSecret, _ := s.db.GetInstanceConfig(ctx, "github_client_secret")
	webhookSecret, _ := s.db.GetInstanceConfig(ctx, "github_webhook_secret")

	if appIDStr != "" && privateKey != "" {
		var appID int64
		for _, c := range appIDStr {
			if c >= '0' && c <= '9' {
				appID = appID*10 + int64(c-'0')
			}
		}
		if appID > 0 {
			cfg, err := github.LoadAppConfig(appID, []byte(privateKey), webhookSecret, clientID, clientSecret)
			if err == nil {
				s.githubApp = cfg
			}
		}
	}
}

// ResolveAppURL dynamically determines the public self-hosted Dashboard/App URL.
// ResolveAppURL dynamically determines the public self-hosted Dashboard URL.
// Priority:
// 1. Stored DB configuration ("instance_url", "app_url")
// 2. TRIAGE_DASHBOARD_URL environment variable
// 3. Server configured appURL
// 4. Request Host header (if r != nil)
// 5. Default "http://localhost:3000"
func (s *Server) ResolveAppURL(ctx context.Context, r ...*http.Request) string {
	if s.db != nil && ctx != nil {
		if u, _ := s.db.GetInstanceConfig(ctx, "instance_url"); u != "" {
			return strings.TrimRight(u, "/")
		}
		if u, _ := s.db.GetInstanceConfig(ctx, "app_url"); u != "" {
			return strings.TrimRight(u, "/")
		}
	}

	if envURL := os.Getenv("TRIAGE_DASHBOARD_URL"); envURL != "" {
		return strings.TrimRight(envURL, "/")
	}

	if s.appURL != "" && s.appURL != "http://localhost:3000" {
		return strings.TrimRight(s.appURL, "/")
	}

	if len(r) > 0 && r[0] != nil {
		proto := "https"
		if r[0].TLS == nil && r[0].Header.Get("X-Forwarded-Proto") != "https" {
			proto = "http"
		}
		host := r[0].Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r[0].Host
		}
		if host != "" {
			return fmt.Sprintf("%s://%s", proto, host)
		}
	}

	if s.appURL != "" {
		return strings.TrimRight(s.appURL, "/")
	}

	return "http://localhost:3000"
}

// Helper to respond with JSON
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
