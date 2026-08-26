// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"triage/engine/internal/config"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/llm"
)

type SetupRepoItem struct {
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	Name          string `json:"name"`
	Installed     bool   `json:"installed"`
	DefaultBranch string `json:"branch"`
	Language      string `json:"lang"`
	Visibility    string `json:"visibility"`
	Private       bool   `json:"private"`
}

func (s *Server) HandleSetupStatus(w http.ResponseWriter, r *http.Request) {
	appID := ""
	slug := s.appSlug
	oauthID := ""
	hasLLM := false
	var hasInstallation bool

	if s.configStore != nil {
		if app, _ := s.configStore.GetGitHubApp(r.Context()); app != nil {
			appID = "configured"
		}
		if slug == "" {
			slug = s.configStore.GetGitHubAppSlug(r.Context())
		}
		oauthID, _ = s.configStore.GetGitHubOAuth(r.Context())
		llmCfg := s.configStore.GetLLM(r.Context())
		hasLLM = llmCfg.APIKey != "" || llmCfg.Provider == "ollama" || llmCfg.Provider == "custom"
	}
	if s.db != nil {
		inst, _ := s.db.GetInstallation(r.Context())
		hasInstallation = inst != nil
	}

	if appID == "" && s.githubApp != nil {
		appID = "configured"
	}

	hasApp := appID != "" || s.githubApp != nil || slug != ""
	hasOauth := oauthID != ""

	result := map[string]interface{}{
		"github_app":   hasApp,
		"installation": hasInstallation,
		"oauth":        hasOauth,
		"llm":          hasLLM,
		"configured":   hasApp && hasInstallation && hasOauth && hasLLM,
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleSetupManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InstanceURL string `json:"instance_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.InstanceURL != "" {
		if s.configStore != nil {
			_ = s.configStore.SaveInstanceURL(r.Context(), req.InstanceURL)
		}
	} else {
		req.InstanceURL = s.ResolveAppURL(r.Context(), r)
	}

	engineURL := s.ResolveEngineURL(r)

	manifest := map[string]interface{}{
		"name":         "Triage",
		"url":          req.InstanceURL,
		"redirect_url": engineURL + "/api/v1/setup/callback",
		"setup_url":    engineURL + "/api/v1/setup/install/callback",
		"callback_urls": []string{
			engineURL + "/api/v1/auth/github/callback",
		},
		"public": true,
		"default_permissions": map[string]string{
			"contents":      "write",
			"issues":        "write",
			"pull_requests": "write",
			"metadata":      "read",
		},
	}

	if !strings.Contains(engineURL, "localhost") {
		manifest["hook_attributes"] = map[string]string{
			"url": engineURL + "/api/v1/webhooks/github",
		}
		manifest["default_events"] = []string{"push", "installation", "installation_repositories"}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"manifest": manifest,
		"url":      "https://github.com/settings/apps/new",
	})
}

func (s *Server) HandleSetupCallback(w http.ResponseWriter, r *http.Request) {
	targetAppURL := s.ResolveAppURL(r.Context(), r)
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, targetAppURL+"?setup_error=missing_code", http.StatusFound)
		return
	}

	convURL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, convURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	github.SetDefaultHeaders(req)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("manifest conversion request failed", "error", err)
		http.Redirect(w, r, targetAppURL+"?setup_error=conversion_failed", http.StatusFound)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("manifest conversion returned non-201 status", "status_code", resp.StatusCode, "body", string(body))
		http.Redirect(w, r, targetAppURL+"?setup_error=conversion_failed", http.StatusFound)
		return
	}

	var result struct {
		ID            int64  `json:"id"`
		Slug          string `json:"slug"`
		PEM           string `json:"pem"`
		WebhookSecret string `json:"webhook_secret"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Error("failed to decode manifest response JSON", "error", err)
		http.Redirect(w, r, targetAppURL+"?setup_error=decode_failed", http.StatusFound)
		return
	}

	s.appSlug = result.Slug

	if s.configStore != nil {
		_ = s.configStore.SaveGitHubApp(r.Context(), config.GitHubAppParams{
			ID:            result.ID,
			Slug:          result.Slug,
			PEM:           result.PEM,
			WebhookSecret: result.WebhookSecret,
			ClientID:      result.ClientID,
			ClientSecret:  result.ClientSecret,
		})
	}

	if result.ID > 0 && result.PEM != "" {
		if cfg, cfgErr := github.LoadAppConfig(result.ID, []byte(result.PEM), result.WebhookSecret, result.ClientID, result.ClientSecret); cfgErr == nil {
			s.githubApp = cfg
		}
	}

	s.LoadGitHubAppConfig(r.Context())

	slog.Info("GitHub App created successfully", "app_id", result.ID, "slug", result.Slug)
	http.Redirect(w, r, targetAppURL+"?setup_step=2&app_created=true", http.StatusFound)
}

func (s *Server) HandleSetupInstall(w http.ResponseWriter, r *http.Request) {
	slug := s.appSlug
	if slug == "" && s.configStore != nil {
		slug = s.configStore.GetGitHubAppSlug(r.Context())
	}
	if slug == "" {
		s.LoadGitHubAppConfig(r.Context())
		if s.appSlug != "" {
			slug = s.appSlug
		}
	}
	if slug == "" && s.githubApp != nil {
		jwt, err := s.githubApp.SignAppJWT()
		if err == nil {
			client := &http.Client{Timeout: 10 * time.Second}
			req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/app", nil)
			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Accept", "application/vnd.github+json")
			github.SetDefaultHeaders(req)

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var appData struct {
					Slug string `json:"slug"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&appData)
				resp.Body.Close()
				if appData.Slug != "" {
					slug = appData.Slug
					s.appSlug = slug
					if s.db != nil {
						_ = s.db.SaveInstanceConfig(r.Context(), "github_app_slug", slug)
					}
				}
			}
		}
	}

	if slug == "" {
		slog.Warn("HandleSetupInstall: GitHub App slug not found")
		http.Error(w, "GitHub App not configured yet", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"url": fmt.Sprintf("https://github.com/apps/%s/installations/new", slug),
	})
}

func (s *Server) HandleSetupInstallCallback(w http.ResponseWriter, r *http.Request) {
	targetAppURL := s.ResolveAppURL(r.Context(), r)
	installIDStr := r.URL.Query().Get("installation_id")
	if installIDStr == "" {
		http.Redirect(w, r, targetAppURL+"?setup_error=missing_installation_id", http.StatusFound)
		return
	}

	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, targetAppURL+"?setup_error=invalid_installation_id", http.StatusFound)
		return
	}

	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	if s.githubApp != nil {
		jwt, err := s.githubApp.SignAppJWT()
		if err == nil {
			client := &http.Client{Timeout: 15 * time.Second}
			reqURL := fmt.Sprintf("https://api.github.com/app/installations/%d", installID)
			req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Accept", "application/vnd.github+json")
			github.SetDefaultHeaders(req)

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var instData struct {
					Account struct {
						Login string `json:"login"`
						ID    int64  `json:"id"`
						Type  string `json:"type"`
					} `json:"account"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&instData)
				resp.Body.Close()

				if s.db != nil {
					_ = s.db.SaveInstallation(r.Context(), installID, instData.Account.Login, instData.Account.ID, instData.Account.Type)
				}

				repoInfos, repoErr := s.githubApp.ListInstallationRepositories(r.Context(), installID)
				if repoErr == nil {
					var repos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						repos = append(repos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
					}
					if s.db != nil {
						_ = s.db.SaveInstallationRepos(r.Context(), installID, repos)
					}
					slog.Info("stored installation repositories", "count", len(repos), "installation_id", installID, "org", instData.Account.Login)
				}
			}
		}
	}

	http.Redirect(w, r, targetAppURL+"?setup_step=3&installed=true", http.StatusFound)
}

func (s *Server) HandleSetupOAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var clientID, clientSecret string
		if s.configStore != nil {
			clientID, clientSecret = s.configStore.GetGitHubOAuth(r.Context())
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"client_id":     clientID,
			"client_secret": clientSecret,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ClientID == "" || req.ClientSecret == "" {
		http.Error(w, "client_id and client_secret are required", http.StatusBadRequest)
		return
	}
	if s.configStore != nil {
		_ = s.configStore.SaveGitHubOAuth(r.Context(), req.ClientID, req.ClientSecret)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) HandleSetupTest(w http.ResponseWriter, r *http.Request) {
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}
	if s.githubApp == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "GitHub App not configured"})
		return
	}

	err := s.githubApp.VerifyApp(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	appName := ""
	if s.configStore != nil {
		appName = s.configStore.GetGitHubAppSlug(r.Context())
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "app_name": appName})
}

// HandleSetupRepos returns repositories that have the GitHub App installed.
// Uses only the GitHub App token — never the user's OAuth token.
// Used by the Setup Wizard (Step 2) to show which repos have the App installed.
func (s *Server) HandleSetupRepos(w http.ResponseWriter, r *http.Request) {
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	repoItemsMap := make(map[string]SetupRepoItem)

	// Fetch installed repos via GitHub App token only
	if s.githubApp != nil {
		if appInstalls, err := s.githubApp.ListAppInstallations(r.Context()); err == nil {
			for _, inst := range appInstalls {
				if s.db != nil {
					_ = s.db.SaveInstallation(r.Context(), inst.ID, inst.AccountLogin, inst.AccountID, inst.AccountType)
				}
				if repoInfos, rErr := s.githubApp.ListInstallationRepositories(r.Context(), inst.ID); rErr == nil {
					var dbRepos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						dbRepos = append(dbRepos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
						key := strings.ToLower(fmt.Sprintf("%s/%s", rInfo.Owner, rInfo.Repo))
						branch := rInfo.DefaultBranch
						if branch == "" {
							branch = "main"
						}
						lang := rInfo.Language
						if lang == "" {
							lang = "Go"
						}
						vis := "Public"
						if rInfo.Private {
							vis = "Private"
						}
						repoItemsMap[key] = SetupRepoItem{
							Owner:         rInfo.Owner,
							Repo:          rInfo.Repo,
							Name:          fmt.Sprintf("%s/%s", rInfo.Owner, rInfo.Repo),
							Installed:     true,
							DefaultBranch: branch,
							Language:      lang,
							Visibility:    vis,
							Private:       rInfo.Private,
						}
					}
					if s.db != nil {
						_ = s.db.SaveInstallationRepos(r.Context(), inst.ID, dbRepos)
					}
				}
			}
		}
	}

	// Merge with DB records (covers repos that may have been de-listed from App but are still in DB)
	if s.db != nil {
		if list, err := s.db.GetAllInstallationRepos(r.Context()); err == nil {
			for _, ir := range list {
				key := strings.ToLower(fmt.Sprintf("%s/%s", ir.Owner, ir.Repo))
				if _, ok := repoItemsMap[key]; !ok {
					repoItemsMap[key] = SetupRepoItem{
						Owner:         ir.Owner,
						Repo:          ir.Repo,
						Name:          fmt.Sprintf("%s/%s", ir.Owner, ir.Repo),
						Installed:     true,
						DefaultBranch: "main",
						Language:      "Go",
						Visibility:    "Private",
						Private:       false,
					}
				}
			}
		}
	}

	var sortedRepos []SetupRepoItem
	for _, item := range repoItemsMap {
		sortedRepos = append(sortedRepos, item)
	}
	sort.Slice(sortedRepos, func(i, j int) bool {
		return sortedRepos[i].Name < sortedRepos[j].Name
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repos": sortedRepos,
	})
}

// HandleInstalledRepos returns a lightweight list of installed repo slugs ("owner/repo").
// Used by the dashboard's Setup Project repo picker to mark which repos have the App installed.
// Refreshes from GitHub App, writes to DB, then returns DB records.
// Engine uses only GitHub App token — never the user's OAuth token.
func (s *Server) HandleInstalledRepos(w http.ResponseWriter, r *http.Request) {
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	// Refresh from GitHub App installations
	if s.githubApp != nil {
		if appInstalls, err := s.githubApp.ListAppInstallations(r.Context()); err == nil {
			for _, inst := range appInstalls {
				if s.db != nil {
					_ = s.db.SaveInstallation(r.Context(), inst.ID, inst.AccountLogin, inst.AccountID, inst.AccountType)
				}
				if repoInfos, rErr := s.githubApp.ListInstallationRepositories(r.Context(), inst.ID); rErr == nil {
					var dbRepos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						dbRepos = append(dbRepos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
					}
					if s.db != nil {
						_ = s.db.SaveInstallationRepos(r.Context(), inst.ID, dbRepos)
					}
				}
			}
		}
	}

	// Return slug list from DB
	var installedSlugs []string
	if s.db != nil {
		if list, err := s.db.GetAllInstallationRepos(r.Context()); err == nil {
			for _, ir := range list {
				installedSlugs = append(installedSlugs, strings.ToLower(fmt.Sprintf("%s/%s", ir.Owner, ir.Repo)))
			}
		}
	}
	if installedSlugs == nil {
		installedSlugs = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"installed_repos": installedSlugs,
	})
}

func (s *Server) HandleCheckRepo(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner = parts[0]
			repo = parts[1]
		}
	}

	if owner == "" || repo == "" {
		http.Error(w, "owner and repo parameters are required", http.StatusBadRequest)
		return
	}

	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"installed": false})
		return
	}

	instID, err := s.db.GetInstallationForRepo(r.Context(), owner, repo)
	installed := err == nil && instID > 0

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"installed":       installed,
		"installation_id": instID,
		"owner":           owner,
		"repo":            repo,
	})
}

func (s *Server) HandleSetupLLM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
		BaseURL  string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	provider := req.Provider
	if provider == "" {
		provider = "gemini"
	}

	if s.configStore != nil {
		_ = s.configStore.SaveLLM(r.Context(), llm.Config{
			Provider: provider,
			APIKey:   req.APIKey,
			Model:    req.Model,
			BaseURL:  req.BaseURL,
		})
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) HandleSettingsLLM(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var cfg llm.Config
		if s.configStore != nil {
			cfg = s.configStore.GetLLM(r.Context())
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"provider": cfg.Provider,
			"api_key":  cfg.APIKey,
			"model":    cfg.Model,
			"base_url": cfg.BaseURL,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Provider string `json:"provider"`
			APIKey   string `json:"api_key"`
			Model    string `json:"model"`
			BaseURL  string `json:"base_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		provider := req.Provider
		if provider == "" {
			provider = "gemini"
		}

		if s.configStore != nil {
			_ = s.configStore.SaveLLM(r.Context(), llm.Config{
				Provider: provider,
				APIKey:   req.APIKey,
				Model:    req.Model,
				BaseURL:  req.BaseURL,
			})
		}

		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) HandleTestLLM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
		BaseURL  string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	providerType := req.Provider
	if providerType == "" {
		providerType = "gemini"
	}

	start := time.Now()
	testCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	provider, err := llm.NewProvider(llm.Config{
		Provider: providerType,
		APIKey:   req.APIKey,
		Model:    req.Model,
		BaseURL:  req.BaseURL,
	})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to initialize provider: %v", err),
		})
		return
	}

	if err := provider.TestConnection(testCtx); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Connection test failed: %v", err),
		})
		return
	}

	latencyMs := time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"latency_ms": latencyMs,
		"provider":   providerType,
		"model":      req.Model,
	})
}
