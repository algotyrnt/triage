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

	srv := &Server{
		db:          cfg.DB,
		configStore: cfg.ConfigStore,
		githubApp:   cfg.GitHubApp,
		astManager:  cfg.ASTManager,
		astCache:    cfg.ASTCache,
		astFetcher:  cfg.ASTFetcher,
		eventBroker: cfg.EventBroker,
		version:     cfg.Version,
	}

	if srv.astFetcher != nil {
		srv.astFetcher.GitHubApp = srv.githubApp
		srv.astFetcher.GetInstallationID = func(ctx context.Context, owner, repo string) (int64, error) {
			return srv.ResolveInstallationID(ctx, owner, repo)
		}
	}

	return srv
}

// ResolveEngineURL dynamically retrieves the base engine URL from the incoming HTTP request.
func (s *Server) ResolveEngineURL(r *http.Request) string {
	if r == nil {
		return "http://localhost:8080"
	}
	proto := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s", proto, r.Host)
}

// Routes initializes and returns the primary HTTP router multiplexer.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	return mux
}

// RegisterRoutes registers all API routes onto the given ServeMux with centralized authentication and RBAC middleware.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// 1. Core Public Routes
	mux.HandleFunc("/health", s.public(s.HandleHealth))
	mux.HandleFunc("/api/v1/telemetry", s.public(s.HandleTelemetry)) // Enforces API key verification internally
	mux.HandleFunc("/api/v1/webhooks/github", s.public(s.HandleGitHubWebhook))

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
	mux.HandleFunc("/api/v1/setup/llm/test", s.public(s.HandleTestLLM))
	mux.HandleFunc("/api/v1/setup/test", s.public(s.HandleSetupTest))
	mux.HandleFunc("/api/v1/setup/repos", s.public(s.HandleSetupRepos))
	mux.HandleFunc("/api/v1/setup/installed-repos", s.public(s.HandleInstalledRepos))
	mux.HandleFunc("/api/v1/setup/check-repo", s.public(s.HandleCheckRepo))

	// 4. Authenticated Core Routes (Viewer, Developer, Admin, Owner)
	mux.HandleFunc("/api/v1/events/stream", s.withAuth(s.HandleEventsStream))
	mux.HandleFunc("/api/v1/incidents", s.withAuth(s.HandleIncidents))
	mux.HandleFunc("/api/v1/projects", s.withAuth(s.HandleProjects))
	mux.HandleFunc("/api/v1/stats", s.withAuth(s.HandleStats))
	mux.HandleFunc("/api/v1/repos/detect-modules", s.withAuth(s.HandleDetectModules))
	mux.HandleFunc("/api/v1/team/members", s.withAuth(s.HandleTeamMembers))

	// 5. Developer+ Protected Mutation Routes (Developer, Admin, Owner)
	mux.HandleFunc("/api/v1/incidents/resolve", s.withAuthRole(s.HandleResolveIncident, "Developer", "Admin", "Owner"))
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
		if s.astFetcher != nil {
			s.astFetcher.GitHubApp = cfg
			s.astFetcher.GetInstallationID = func(ctx context.Context, owner, repo string) (int64, error) {
				return s.ResolveInstallationID(ctx, owner, repo)
			}
		}
	}
}

// ResolveInstallationID retrieves the active GitHub App installation ID for an owner/repo.
func (s *Server) ResolveInstallationID(ctx context.Context, owner, repo string) (int64, error) {
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(ctx)
	}
	if s.githubApp == nil {
		return 0, fmt.Errorf("github app not configured on this instance")
	}

	// 1. Direct verified on-demand query to GitHub App for this specific repository
	if owner != "" && repo != "" {
		if liveID, err := s.githubApp.GetRepoInstallation(ctx, owner, repo); err == nil && liveID > 0 {
			if s.db != nil {
				_ = s.db.SaveInstallation(ctx, liveID, owner, 0, "Organization")
				_ = s.db.SaveInstallationRepo(ctx, liveID, owner, repo)
				_ = s.db.UpdateRepositoryInstallation(ctx, owner, repo, liveID)
			}
			return liveID, nil
		}
	}

	// 2. Database lookup
	if s.db != nil && owner != "" && repo != "" {
		if instID, err := s.db.GetInstallationForRepo(ctx, owner, repo); err == nil && instID > 0 {
			return instID, nil
		}
	}

	// 3. Query all installations for the GitHub App
	appInstalls, err := s.githubApp.ListAppInstallations(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query github installations: %w", err)
	}
	if len(appInstalls) == 0 {
		return 0, fmt.Errorf("github app is not installed on any account")
	}

	instID := appInstalls[0].ID
	for _, inst := range appInstalls {
		if strings.EqualFold(inst.AccountLogin, owner) {
			instID = inst.ID
			break
		}
	}

	if s.db != nil {
		for _, inst := range appInstalls {
			_ = s.db.SaveInstallation(ctx, inst.ID, inst.AccountLogin, inst.AccountID, inst.AccountType)
		}
		if owner != "" && repo != "" && instID > 0 {
			_ = s.db.SaveInstallationRepo(ctx, instID, owner, repo)
			_ = s.db.UpdateRepositoryInstallation(ctx, owner, repo, instID)
		}
	}

	return instID, nil
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
