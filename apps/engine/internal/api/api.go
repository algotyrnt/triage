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
	"strconv"
	"strings"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
)

var githubNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)

// Config holds runtime dependencies and configurations for the API server.
type Config struct {
	DB         *db.DB
	GitHubApp  *github.AppConfig
	ASTManager *ast.Manager
	ASTCache   *ast.ASTCache
	ASTFetcher *ast.OnDemandFetcher
	AppURL     string
	EngineURL  string
}

// Server encapsulates HTTP routes and middleware for the Triage Engine.
type Server struct {
	db         *db.DB
	githubApp  *github.AppConfig
	astManager *ast.Manager
	astCache   *ast.ASTCache
	astFetcher *ast.OnDemandFetcher
	appURL     string
	engineURL  string
	appSlug    string
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

	return &Server{
		db:         cfg.DB,
		githubApp:  cfg.GitHubApp,
		astManager: cfg.ASTManager,
		astCache:   cfg.ASTCache,
		astFetcher: cfg.ASTFetcher,
		appURL:     cfg.AppURL,
		engineURL:  cfg.EngineURL,
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
	mux.HandleFunc("/api/v1/setup/installed-repos", s.withMiddleware(s.HandleInstalledRepos))
	mux.HandleFunc("/api/v1/setup/check-repo", s.withMiddleware(s.HandleCheckRepo))

	// Settings & Key management routes
	mux.HandleFunc("/api/v1/settings/llm", s.withMiddleware(s.HandleSettingsLLM))
	mux.HandleFunc("/api/v1/projects/keys", s.withMiddleware(s.HandleProjectKeys))
	mux.HandleFunc("/api/v1/projects/keys/revoke", s.withMiddleware(s.HandleRevokeProjectKey))
	mux.HandleFunc("/api/v1/gemini/analyze-panic", s.withMiddleware(s.HandleGeminiAnalyzePanic))
	mux.HandleFunc("/api/v1/gemini/generate-patch", s.withMiddleware(s.HandleGeminiGeneratePatch))

	// Root handler: seamlessly redirect visitors/setup queries from engine to dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		dashboardURL := s.ResolveAppURL(r.Context(), r)
		target := dashboardURL
		if r.URL.RawQuery != "" {
			target = fmt.Sprintf("%s?%s", dashboardURL, r.URL.RawQuery)
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
}

// LoadGitHubAppConfig refreshes the GitHub App credentials and client details from the instance configuration database.
func (s *Server) LoadGitHubAppConfig(ctx context.Context) {
	if s.db == nil || ctx == nil {
		return
	}

	appIDStr, _ := s.db.GetInstanceConfig(ctx, "github_app_id")
	privateKey, _ := s.db.GetInstanceConfig(ctx, "github_app_private_key")
	clientID, _ := s.db.GetInstanceConfig(ctx, "github_app_client_id")
	if clientID == "" {
		clientID, _ = s.db.GetInstanceConfig(ctx, "github_client_id")
	}
	clientSecret, _ := s.db.GetInstanceConfig(ctx, "github_app_client_secret")
	if clientSecret == "" {
		clientSecret, _ = s.db.GetInstanceConfig(ctx, "github_client_secret")
	}
	webhookSecret, _ := s.db.GetInstanceConfig(ctx, "github_app_webhook_secret")
	if webhookSecret == "" {
		webhookSecret, _ = s.db.GetInstanceConfig(ctx, "github_webhook_secret")
	}
	if dbSlug, _ := s.db.GetInstanceConfig(ctx, "github_app_slug"); dbSlug != "" {
		s.appSlug = dbSlug
	}

	if appIDStr != "" && privateKey != "" {
		appID, err := strconv.ParseInt(strings.TrimSpace(appIDStr), 10, 64)
		if err == nil && appID > 0 {
			cfg, err := github.LoadAppConfig(appID, []byte(privateKey), webhookSecret, clientID, clientSecret)
			if err == nil {
				s.githubApp = cfg
			}
		}
	}
}

// ResolveAppURL dynamically determines the public self-hosted Dashboard URL.
// Priority:
// 1. TRIAGE_DASHBOARD_URL environment variable
// 2. Stored DB configuration ("instance_url", "app_url")
// 3. Server configured appURL
// 4. Default "http://localhost:3000"
func (s *Server) ResolveAppURL(ctx context.Context, r ...*http.Request) string {
	if envURL := os.Getenv("TRIAGE_DASHBOARD_URL"); envURL != "" {
		return strings.TrimRight(envURL, "/")
	}

	if s.db != nil && ctx != nil {
		if u, _ := s.db.GetInstanceConfig(ctx, "instance_url"); u != "" {
			return strings.TrimRight(u, "/")
		}
		if u, _ := s.db.GetInstanceConfig(ctx, "app_url"); u != "" {
			return strings.TrimRight(u, "/")
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
