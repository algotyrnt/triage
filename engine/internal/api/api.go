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
	"triage/engine/internal/ui"
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

// RegisterRoutes registers all API routes onto the given ServeMux with centralized authentication and RBAC middleware.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// 1. Core Public Routes
	mux.HandleFunc("/health", s.public(s.HandleHealth))
	mux.HandleFunc("/api/v1/telemetry", s.public(s.HandleTelemetry)) // Enforces API key verification internally

	// 2. Auth & Session Routes
	mux.HandleFunc("/api/v1/auth/github", s.public(s.HandleAuthGitHub))
	mux.HandleFunc("/api/v1/auth/github/callback", s.public(s.HandleAuthGitHubCallback))
	mux.HandleFunc("/api/v1/auth/logout", s.public(s.HandleAuthLogout))
	mux.HandleFunc("/api/v1/auth/me", s.withAuth(s.HandleAuthMe))

	// 3. Setup Wizard Routes
	mux.HandleFunc("/api/v1/setup/status", s.public(s.HandleSetupStatus))
	mux.HandleFunc("/api/v1/setup/manifest", s.public(s.HandleSetupManifest))
	mux.HandleFunc("/api/v1/setup/callback", s.public(s.HandleSetupCallback))
	mux.HandleFunc("/api/v1/setup/install", s.public(s.HandleSetupInstall))
	mux.HandleFunc("/api/v1/setup/install/callback", s.public(s.HandleSetupInstallCallback))
	mux.HandleFunc("/api/v1/setup/oauth", s.public(s.HandleSetupOAuth))
	mux.HandleFunc("/api/v1/setup/llm", s.public(s.HandleSetupLLM))
	mux.HandleFunc("/api/v1/setup/test", s.public(s.HandleSetupTest))
	mux.HandleFunc("/api/v1/setup/repos", s.public(s.HandleSetupRepos))
	mux.HandleFunc("/api/v1/setup/installed-repos", s.public(s.HandleInstalledRepos))
	mux.HandleFunc("/api/v1/setup/check-repo", s.public(s.HandleCheckRepo))

	// 4. Authenticated Core Routes (Viewer, Developer, Admin, Owner)
	mux.HandleFunc("/api/v1/events/stream", s.withAuth(s.HandleEventsStream))
	mux.HandleFunc("/api/v1/incidents", s.withAuth(s.HandleIncidents))
	mux.HandleFunc("/api/v1/projects", s.withAuth(s.HandleProjects))
	mux.HandleFunc("/api/v1/stats", s.withAuth(s.HandleStats))
	mux.HandleFunc("/api/v1/ast/index", s.withAuth(s.HandleASTIndex))
	mux.HandleFunc("/api/v1/ast/tree", s.withAuth(s.HandleASTTree))
	mux.HandleFunc("/api/v1/repos/detect-modules", s.withAuth(s.HandleDetectModules))
	mux.HandleFunc("/api/v1/team/members", s.withAuth(s.HandleTeamMembers))

	// 5. Developer+ Protected Mutation Routes (Developer, Admin, Owner)
	mux.HandleFunc("/api/v1/incidents/create-issue", s.withAuthRole(s.HandleCreateIncidentIssue, "Developer", "Admin", "Owner"))
	mux.HandleFunc("/api/v1/incidents/create-pr", s.withAuthRole(s.HandleCreateIncidentPR, "Developer", "Admin", "Owner"))
	mux.HandleFunc("/api/v1/projects/context", s.withAuthRole(s.HandleUpdateProjectContext, "Developer", "Admin", "Owner"))
	mux.HandleFunc("/api/v1/llm/analyze-panic", s.withAuthRole(s.HandleLLMAnalyzePanic, "Developer", "Admin", "Owner"))
	mux.HandleFunc("/api/v1/llm/generate-patch", s.withAuthRole(s.HandleLLMGeneratePatch, "Developer", "Admin", "Owner"))

	// 6. Admin+ Management Routes (Admin, Owner)
	mux.HandleFunc("/api/v1/team/members/role", s.withAuthRole(s.HandleTeamMemberRole, "Admin", "Owner"))
	mux.HandleFunc("/api/v1/team/invites", s.withAuthRole(s.HandleTeamInvites, "Admin", "Owner"))
	mux.HandleFunc("/api/v1/settings/llm", s.withAuthRole(s.HandleSettingsLLM, "Admin", "Owner"))
	mux.HandleFunc("/api/v1/settings/llm/test", s.withAuthRole(s.HandleTestLLM, "Admin", "Owner"))
	mux.HandleFunc("/api/v1/projects/keys", s.withAuthRole(s.HandleProjectKeys, "Admin", "Owner"))
	mux.HandleFunc("/api/v1/projects/keys/revoke", s.withAuthRole(s.HandleRevokeProjectKey, "Admin", "Owner"))

	// 7. Embedded Studio Dashboard SPA Handler
	mux.Handle("/", ui.Handler())
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
