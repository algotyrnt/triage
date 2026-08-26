// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"triage/engine/internal/ast"
	"triage/engine/internal/config"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/llm"
	"triage/engine/internal/version"
)

var githubNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)

// Config holds initialization dependencies for the Server.
type Config struct {
	DB          *db.DB
	ConfigStore *config.Store
	GitHubApp   *github.AppConfig
	ASTManager  *ast.Manager
	ASTCache    *ast.ASTCache
	ASTFetcher  *ast.OnDemandFetcher
	EventBroker *EventBroker
	Version     string
}

// Server encapsulates HTTP routes and middleware for the Triage Engine.
type Server struct {
	db          *db.DB
	configStore *config.Store
	githubApp   *github.AppConfig
	astManager  *ast.Manager
	astCache    *ast.ASTCache
	astFetcher  *ast.OnDemandFetcher
	eventBroker *EventBroker
	appSlug     string
	version     string
}

// NewServer initializes a new API server with the provided dependencies.
func NewServer(cfg Config) *Server {
	if cfg.ConfigStore == nil {
		cfg.ConfigStore = config.NewStore(cfg.DB)
	}
	if cfg.ASTCache == nil {
		cfg.ASTCache = ast.NewASTCache()
	}
	if cfg.ASTFetcher == nil {
		cfg.ASTFetcher = ast.NewOnDemandFetcher()
	}
	if cfg.EventBroker == nil {
		cfg.EventBroker = NewEventBroker()
	}
	if cfg.Version == "" {
		cfg.Version = version.Get()
	}

	return &Server{
		db:          cfg.DB,
		configStore: cfg.ConfigStore,
		githubApp:   cfg.GitHubApp,
		astManager:  cfg.ASTManager,
		astCache:    cfg.ASTCache,
		astFetcher:  cfg.ASTFetcher,
		eventBroker: cfg.EventBroker,
		version:     cfg.Version,
	}
}

// ResolveEngineURL dynamically retrieves the base engine URL from the incoming HTTP request.
func (s *Server) ResolveEngineURL(r *http.Request) string {
	scheme := "http"
	if r != nil {
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		if fHost := r.Header.Get("X-Forwarded-Host"); fHost != "" {
			return fmt.Sprintf("%s://%s", scheme, strings.TrimRight(strings.TrimSpace(fHost), "/"))
		}
		if r.Host != "" {
			return fmt.Sprintf("%s://%s", scheme, strings.TrimRight(strings.TrimSpace(r.Host), "/"))
		}
	}
	return "http://localhost:8080"
}

// RegisterRoutes registers all API routes onto the given ServeMux with standard middleware.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Core routes
	mux.HandleFunc("/health", s.withMiddleware(s.HandleHealth))
	mux.HandleFunc("/api/v1/events/stream", s.withMiddleware(s.HandleEventsStream))
	mux.HandleFunc("/api/v1/telemetry", s.withMiddleware(s.HandleTelemetry))
	mux.HandleFunc("/api/v1/ast/index", s.withMiddleware(s.HandleASTIndex))
	mux.HandleFunc("/api/v1/ast/tree", s.withMiddleware(s.HandleASTTree))
	mux.HandleFunc("/api/v1/incidents", s.withMiddleware(s.HandleIncidents))
	mux.HandleFunc("/api/v1/incidents/create-issue", s.withMiddleware(s.HandleCreateIncidentIssue))
	mux.HandleFunc("/api/v1/incidents/create-pr", s.withMiddleware(s.HandleCreateIncidentPR))
	mux.HandleFunc("/api/v1/projects", s.withMiddleware(s.HandleProjects))
	mux.HandleFunc("/api/v1/projects/context", s.withMiddleware(s.HandleUpdateProjectContext))
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

	// Auth & RBAC routes
	mux.HandleFunc("/api/v1/auth/github", s.withMiddleware(s.HandleAuthGitHub))
	mux.HandleFunc("/api/v1/auth/github/callback", s.withMiddleware(s.HandleAuthGitHubCallback))
	mux.HandleFunc("/api/v1/auth/me", s.withMiddleware(s.HandleAuthMe))

	// Team Management routes
	mux.HandleFunc("/api/v1/team/members", s.withMiddleware(s.HandleTeamMembers))
	mux.HandleFunc("/api/v1/team/members/role", s.withMiddleware(s.HandleTeamMemberRole))
	mux.HandleFunc("/api/v1/team/invites", s.withMiddleware(s.HandleTeamInvites))

	// Settings & Key management routes
	mux.HandleFunc("/api/v1/settings/llm", s.withMiddleware(s.HandleSettingsLLM))
	mux.HandleFunc("/api/v1/settings/llm/test", s.withMiddleware(s.HandleTestLLM))
	mux.HandleFunc("/api/v1/projects/keys", s.withMiddleware(s.HandleProjectKeys))
	mux.HandleFunc("/api/v1/projects/keys/revoke", s.withMiddleware(s.HandleRevokeProjectKey))
	mux.HandleFunc("/api/v1/llm/analyze-panic", s.withMiddleware(s.HandleLLMAnalyzePanic))
	mux.HandleFunc("/api/v1/llm/generate-patch", s.withMiddleware(s.HandleLLMGeneratePatch))

	// Root handler: redirect visitors to dashboard if configured, or return engine operational status
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		dashboardURL := s.ResolveAppURL(r.Context(), r)
		if dashboardURL == "" {
			writeJSON(w, http.StatusOK, map[string]string{
				"service": "triage-engine",
				"status":  "operational",
			})
			return
		}
		target := dashboardURL
		if r.URL.RawQuery != "" {
			target = fmt.Sprintf("%s?%s", dashboardURL, r.URL.RawQuery)
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
}

// LoadGitHubAppConfig refreshes the GitHub App credentials and client details from the instance configuration database.
func (s *Server) LoadGitHubAppConfig(ctx context.Context) {
	if s.configStore == nil || ctx == nil {
		return
	}

	if slug := s.configStore.GetGitHubAppSlug(ctx); slug != "" {
		s.appSlug = slug
	}

	if cfg, err := s.configStore.GetGitHubApp(ctx); err == nil && cfg != nil {
		s.githubApp = cfg
	}
}

// ResolveAppURL dynamically retrieves the public self-hosted Dashboard URL strictly from database instance configuration.
func (s *Server) ResolveAppURL(ctx context.Context, r ...*http.Request) string {
	if s.configStore == nil {
		return ""
	}
	return s.configStore.GetInstanceURL(ctx, r...)
}

// GetLLMConfig retrieves the configured LLM provider configuration.
func (s *Server) GetLLMConfig(ctx context.Context) llm.Config {
	if s.configStore == nil {
		return llm.Config{}
	}
	return s.configStore.GetLLM(ctx)
}

// GetLLMProvider initializes the active LLM provider from the stored configuration.
func (s *Server) GetLLMProvider(ctx context.Context) (llm.Provider, error) {
	cfg := s.GetLLMConfig(ctx)
	return llm.NewProvider(cfg)
}

// Helper to respond with JSON
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
