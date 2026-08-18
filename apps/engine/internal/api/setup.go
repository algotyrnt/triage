// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/session"
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
	result := map[string]interface{}{
		"configured":   false,
		"github_app":   false,
		"installation": false,
		"oauth":        false,
		"llm":          false,
	}

	if s.db != nil {
		appID, _ := s.db.GetInstanceConfig(r.Context(), "github_app_id")
		result["github_app"] = appID != ""

		inst, _ := s.db.GetInstallation(r.Context())
		result["installation"] = inst != nil

		oauthID, _ := s.db.GetInstanceConfig(r.Context(), "github_oauth_client_id")
		result["oauth"] = oauthID != ""

		apiKey, _ := s.db.GetInstanceConfig(r.Context(), "gemini_api_key")
		modelName, _ := s.db.GetInstanceConfig(r.Context(), "gemini_model")
		result["llm"] = apiKey != "" && modelName != ""

		if appID != "" && inst != nil && oauthID != "" && apiKey != "" && modelName != "" {
			result["configured"] = true
		}
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
	if req.InstanceURL == "" {
		req.InstanceURL = s.appURL
	}

	manifest := map[string]interface{}{
		"name":         "Triage",
		"url":          req.InstanceURL,
		"redirect_url": s.engineURL + "/api/v1/setup/callback",
		"setup_url":    s.engineURL + "/api/v1/setup/install/callback",
		"callback_urls": []string{
			s.engineURL + "/api/v1/auth/github/callback",
		},
		"public": false,
		"default_permissions": map[string]string{
			"contents":      "write",
			"issues":        "write",
			"pull_requests": "write",
			"metadata":      "read",
		},
	}

	if !strings.Contains(s.engineURL, "localhost") {
		manifest["hook_attributes"] = map[string]string{
			"url": s.engineURL + "/api/v1/webhooks/github",
		}
		manifest["default_events"] = []string{"push", "installation", "installation_repositories"}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"manifest": manifest,
		"url":      "https://github.com/settings/apps/new",
	})
}

func (s *Server) HandleSetupCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	convURL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, convURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("manifest conversion request failed", "error", err)
		http.Redirect(w, r, s.appURL+"?setup_error=conversion_failed", http.StatusFound)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("manifest conversion returned non-201 status", "status_code", resp.StatusCode, "body", string(body))
		http.Redirect(w, r, s.appURL+"?setup_error=conversion_failed", http.StatusFound)
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
		http.Redirect(w, r, s.appURL+"?setup_error=decode_failed", http.StatusFound)
		return
	}

	if s.db != nil {
		s.db.SaveInstanceConfig(r.Context(), "github_app_id", strconv.FormatInt(result.ID, 10))
		s.db.SaveInstanceConfig(r.Context(), "github_app_slug", result.Slug)
		s.db.SaveInstanceConfig(r.Context(), "github_app_private_key", result.PEM)
		s.db.SaveInstanceConfig(r.Context(), "github_app_webhook_secret", result.WebhookSecret)
		s.db.SaveInstanceConfig(r.Context(), "github_app_client_id", result.ClientID)
		s.db.SaveInstanceConfig(r.Context(), "github_app_client_secret", result.ClientSecret)
		s.db.SaveInstanceConfig(r.Context(), "github_oauth_client_id", result.ClientID)
		s.db.SaveInstanceConfig(r.Context(), "github_oauth_client_secret", result.ClientSecret)
	}

	s.LoadGitHubAppConfig(r.Context())

	slog.Info("GitHub App created successfully", "app_id", result.ID, "slug", result.Slug)
	http.Redirect(w, r, s.appURL+"?setup_step=2&app_created=true", http.StatusFound)
}

func (s *Server) HandleSetupInstall(w http.ResponseWriter, r *http.Request) {
	slug := ""
	if s.db != nil {
		slug, _ = s.db.GetInstanceConfig(r.Context(), "github_app_slug")
	}
	if slug == "" {
		http.Error(w, "GitHub App not configured yet", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"url": fmt.Sprintf("https://github.com/apps/%s/installations/new", slug),
	})
}

func (s *Server) HandleSetupInstallCallback(w http.ResponseWriter, r *http.Request) {
	installIDStr := r.URL.Query().Get("installation_id")
	if installIDStr == "" {
		http.Redirect(w, r, s.appURL+"?setup_error=missing_installation_id", http.StatusFound)
		return
	}

	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, s.appURL+"?setup_error=invalid_installation_id", http.StatusFound)
		return
	}

	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	if s.githubApp != nil && s.db != nil {
		jwt, err := s.githubApp.SignAppJWT()
		if err == nil {
			client := &http.Client{Timeout: 15 * time.Second}
			reqURL := fmt.Sprintf("https://api.github.com/app/installations/%d", installID)
			req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Accept", "application/vnd.github+json")

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

				s.db.SaveInstallation(r.Context(), installID, instData.Account.Login, instData.Account.ID, instData.Account.Type)

				repoInfos, repoErr := s.githubApp.ListInstallationRepositories(r.Context(), installID)
				if repoErr == nil {
					var repos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						repos = append(repos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
					}
					s.db.SaveInstallationRepos(r.Context(), installID, repos)
					slog.Info("stored installation repositories", "count", len(repos), "installation_id", installID, "org", instData.Account.Login)
				}
			}
		}
	}

	http.Redirect(w, r, s.appURL+"?setup_step=3&installed=true", http.StatusFound)
}

func (s *Server) HandleSetupOAuth(w http.ResponseWriter, r *http.Request) {
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
	if s.db != nil {
		s.db.SaveInstanceConfig(r.Context(), "github_oauth_client_id", req.ClientID)
		s.db.SaveInstanceConfig(r.Context(), "github_oauth_client_secret", req.ClientSecret)
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
	if s.db != nil {
		appName, _ = s.db.GetInstanceConfig(r.Context(), "github_app_slug")
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "app_name": appName})
}

func (s *Server) HandleSetupRepos(w http.ResponseWriter, r *http.Request) {
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	repoItemsMap := make(map[string]SetupRepoItem)

	if s.githubApp != nil {
		if appInstalls, err := s.githubApp.ListAppInstallations(r.Context()); err == nil {
			for _, inst := range appInstalls {
				if s.db != nil {
					_ = s.db.SaveInstallation(r.Context(), inst.ID, inst.AccountLogin, inst.AccountID, inst.AccountType)
				}
				if repoInfos, rErr := s.githubApp.ListInstallationRepositories(r.Context(), inst.ID); rErr == nil {
					var repos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						repos = append(repos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
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
						_ = s.db.SaveInstallationRepos(r.Context(), inst.ID, repos)
					}
				}
			}
		}
	}

	var installedRepos []db.InstallationRepo
	if s.db != nil {
		if list, err := s.db.GetAllInstallationRepos(r.Context()); err == nil {
			installedRepos = list
		}
	}

	installedMap := make(map[string]bool)
	for _, ir := range installedRepos {
		key := strings.ToLower(fmt.Sprintf("%s/%s", ir.Owner, ir.Repo))
		installedMap[key] = true
	}

	username := r.URL.Query().Get("username")
	var userAccessToken string
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") && s.sessionSecret != "" {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, cErr := session.ValidateSessionJWT(tokenStr, s.sessionSecret); cErr == nil && claims != nil {
			if username == "" {
				username = claims.Username
			}
			userAccessToken = claims.GitHubToken
		}
	}

	if username != "" || userAccessToken != "" {
		if userRepos, uErr := github.FetchUserRepositories(r.Context(), username, userAccessToken); uErr == nil {
			for _, ur := range userRepos {
				key := strings.ToLower(fmt.Sprintf("%s/%s", ur.Owner, ur.Repo))
				isInstalled := installedMap[key]
				branch := ur.DefaultBranch
				if branch == "" {
					branch = "main"
				}
				lang := ur.Language
				if lang == "" {
					lang = "Go"
				}
				vis := "Public"
				if ur.Private {
					vis = "Private"
				}
				repoItemsMap[key] = SetupRepoItem{
					Owner:         ur.Owner,
					Repo:          ur.Repo,
					Name:          fmt.Sprintf("%s/%s", ur.Owner, ur.Repo),
					Installed:     isInstalled,
					DefaultBranch: branch,
					Language:      lang,
					Visibility:    vis,
					Private:       ur.Private,
				}
			}
		} else {
			slog.Warn("failed to fetch user repositories from GitHub", "username", username, "error", uErr)
		}
	}

	for _, ir := range installedRepos {
		key := strings.ToLower(fmt.Sprintf("%s/%s", ir.Owner, ir.Repo))
		if existing, ok := repoItemsMap[key]; ok {
			existing.Installed = true
			repoItemsMap[key] = existing
		} else {
			repoItemsMap[key] = SetupRepoItem{
				Owner:         ir.Owner,
				Repo:          ir.Repo,
				Name:          fmt.Sprintf("%s/%s", ir.Owner, ir.Repo),
				Installed:     true,
				DefaultBranch: "main",
				Language:      "Go",
				Visibility:    "Installed",
				Private:       false,
			}
		}
	}

	var sortedRepos []SetupRepoItem
	for _, item := range repoItemsMap {
		sortedRepos = append(sortedRepos, item)
	}
	sort.Slice(sortedRepos, func(i, j int) bool {
		if sortedRepos[i].Installed != sortedRepos[j].Installed {
			return sortedRepos[i].Installed
		}
		return sortedRepos[i].Name < sortedRepos[j].Name
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repos": sortedRepos,
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
		GeminiAPIKey string `json:"gemini_api_key"`
		GeminiModel  string `json:"gemini_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if s.db != nil {
		if req.GeminiAPIKey != "" {
			s.db.SaveInstanceConfig(r.Context(), "gemini_api_key", req.GeminiAPIKey)
		}
		if req.GeminiModel != "" {
			s.db.SaveInstanceConfig(r.Context(), "gemini_model", req.GeminiModel)
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) HandleSettingsLLM(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		api := ""
		model := ""
		if s.db != nil {
			api, _ = s.db.GetInstanceConfig(r.Context(), "gemini_api_key")
			model, _ = s.db.GetInstanceConfig(r.Context(), "gemini_model")
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"gemini_api_key": api,
			"gemini_model":   model,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			GeminiAPIKey string `json:"gemini_api_key"`
			GeminiModel  string `json:"gemini_model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if s.db != nil {
			if req.GeminiAPIKey != "" {
				s.db.SaveInstanceConfig(r.Context(), "gemini_api_key", req.GeminiAPIKey)
			}
			if req.GeminiModel != "" {
				s.db.SaveInstanceConfig(r.Context(), "gemini_model", req.GeminiModel)
			}
		}

		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}
