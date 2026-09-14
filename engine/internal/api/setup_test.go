// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triage/engine/internal/github"
	"triage/engine/internal/llm"
)

func TestHandleSetupStatus_Variations(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Initial status: unconfigured
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res["configured"] == true {
		t.Errorf("expected configured: false initially")
	}

	// 2. Configure OAuth, LLM, App, Installation -> configured: true
	_ = env.ConfigStore.SaveGitHubOAuth(ctx, "client_id_1", "client_secret_1")
	_ = env.ConfigStore.SaveLLM(ctx, llm.Config{Provider: "ollama"})
	_ = env.ConfigStore.SaveGitHubAppSlug(ctx, "my-triage-app")
	_ = env.DB.SaveInstallation(ctx, 12345, "orglogin", 100, "Organization")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupStatus(rec, req)
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res["configured"] != true {
		t.Errorf("expected configured: true when all steps complete: %+v", res)
	}
}

func TestHandleSetupManifest_Options(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Method not allowed (GET)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/manifest", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupManifest(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rec.Code)
	}

	// 2. POST with custom instance_url on remote engine host
	manifestPayload, _ := json.Marshal(map[string]string{
		"instance_url": "https://dashboard.example.com",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/manifest", bytes.NewReader(manifestPayload))
	req.Host = "engine.example.com"
	rec = httptest.NewRecorder()
	env.Server.HandleSetupManifest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res struct {
		Manifest map[string]interface{} `json:"manifest"`
		URL      string                 `json:"url"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res.Manifest["url"] != "https://dashboard.example.com" {
		t.Errorf("expected custom instance url, got %v", res.Manifest["url"])
	}
	hookAttrs, ok := res.Manifest["hook_attributes"].(map[string]interface{})
	if !ok || !strings.Contains(hookAttrs["url"].(string), "engine.example.com") {
		t.Errorf("expected webhook url for remote engine: %+v", res.Manifest)
	}
}

func TestHandleSetupCallback_Redirects(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Missing code parameter redirects with setup_error=missing_code
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/callback", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupCallback(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "setup_error=missing_code") {
		t.Errorf("expected missing_code error in redirect, got %s", location)
	}
}

func TestHandleSetupCallback_Success(t *testing.T) {
	env := setupAPITestEnv(t)

	mockHandler := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.String(), "conversions") {
			body := `{
				"id": 12345,
				"slug": "awesome-app",
				"pem": "fake-pem-key",
				"webhook_secret": "whsec_abc",
				"client_id": "cid_xyz",
				"client_secret": "csec_xyz"
			}`
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})

	origTransport := http.DefaultTransport
	http.DefaultTransport = mockHandler
	defer func() { http.DefaultTransport = origTransport }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/callback?code=mock_manifest_code", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "app_created=true") {
		t.Errorf("expected app_created=true redirect, got %s", loc)
	}
}

func TestHandleSetupInstall_And_Callback(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. SetupInstall when slug is not found -> 400
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/install", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupInstall(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when slug not configured, got %d", rec.Code)
	}

	// 2. SetupInstall when slug is configured in ConfigStore
	_ = env.ConfigStore.SaveGitHubAppSlug(ctx, "cool-triage-app")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/install", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupInstall(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	var installRes map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&installRes)
	if !strings.Contains(installRes["url"], "cool-triage-app") {
		t.Errorf("expected slug in install URL: %s", installRes["url"])
	}

	// 3. SetupInstallCallback missing installation_id -> 302
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/install/callback", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupInstallCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "missing_installation_id") {
		t.Errorf("expected missing_installation_id redirect, got %d, loc=%s", rec.Code, rec.Header().Get("Location"))
	}

	// 4. SetupInstallCallback invalid installation_id -> 302
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/install/callback?installation_id=invalid", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupInstallCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "invalid_installation_id") {
		t.Errorf("expected invalid_installation_id redirect, got %d, loc=%s", rec.Code, rec.Header().Get("Location"))
	}

	// 5. SetupInstallCallback success with GitHub App
	ghApp := setupTestGitHubApp(t)
	env.Server.githubApp = ghApp

	mockAppHandler := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path
		if strings.Contains(path, "/installations/12345") {
			body := `{"account":{"login":"octoorg","id":999,"type":"Organization"}}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}
		if path == "/installation/repositories" {
			body := `{"total_count":1,"repositories":[{"name":"core","owner":{"login":"octoorg"}}]}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}
		if path == "/app" {
			body := `{"slug":"my-app-slug"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})

	mockHTTP := mockClient(mockAppHandler)
	restoreGH := github.SetHTTPClient(mockHTTP)
	defer restoreGH()
	origTransport := http.DefaultTransport
	http.DefaultTransport = mockAppHandler
	defer func() { http.DefaultTransport = origTransport }()

	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/install/callback?installation_id=12345", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupInstallCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "installed=true") {
		t.Errorf("expected installed=true redirect, got %d, loc=%s", rec.Code, rec.Header().Get("Location"))
	}

	// 6. SetupInstall when slug is empty but GitHub App is present -> calls /app to get slug
	env.Server.appSlug = ""
	_ = env.DB.SaveInstanceConfig(ctx, "github_app_slug", "")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/install", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupInstall(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for slug from GitHub App, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSetupOAuth(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. GET initial OAuth credentials (empty)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/oauth", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupOAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// 2. Method not allowed (PUT)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/setup/oauth", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupOAuth(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}

	// 3. POST with empty credentials -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/oauth", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	env.Server.HandleSetupOAuth(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty OAuth credentials, got %d", rec.Code)
	}

	// 4. POST with valid credentials -> 200
	oauthBody, _ := json.Marshal(map[string]string{
		"client_id":     "gh_client_123",
		"client_secret": "gh_secret_456",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/oauth", bytes.NewReader(oauthBody))
	rec = httptest.NewRecorder()
	env.Server.HandleSetupOAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for OAuth config, got %d", rec.Code)
	}

	// 5. GET returns updated credentials
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/oauth", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupOAuth(rec, req)
	var getRes map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&getRes)
	if getRes["client_id"] != "gh_client_123" || getRes["client_secret"] != "gh_secret_456" {
		t.Errorf("unexpected OAuth creds returned: %+v", getRes)
	}
}

func TestHandleSetupTest(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Without GitHub App -> 400
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/test", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupTest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfigured GitHub App, got %d", rec.Code)
	}
}

func TestHandleInstalledRepos_And_CheckRepo(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Setup installation and repository in DB
	_ = env.DB.SaveInstallation(ctx, 443322, "acmeorg", 100, "Organization")
	_ = env.DB.SaveInstallationRepo(ctx, 443322, "acmeorg", "payments-engine")

	// 2. HandleInstalledRepos returns slugs
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/installed-repos", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleInstalledRepos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for installed-repos, got %d", rec.Code)
	}
	var res struct {
		InstalledRepos []string `json:"installed_repos"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if len(res.InstalledRepos) != 1 || res.InstalledRepos[0] != "acmeorg/payments-engine" {
		t.Errorf("unexpected installed repos list: %+v", res.InstalledRepos)
	}

	// 3. HandleCheckRepo - Missing params -> 400
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/check-repo", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleCheckRepo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing check-repo params, got %d", rec.Code)
	}

	// 4. HandleCheckRepo - Installed repo
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/check-repo?repo=acmeorg/payments-engine", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleCheckRepo(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var checkRes struct {
		Installed      bool  `json:"installed"`
		InstallationID int64 `json:"installation_id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&checkRes)
	if !checkRes.Installed || checkRes.InstallationID != 443322 {
		t.Errorf("expected installed repo check to pass: %+v", checkRes)
	}

	// 5. HandleCheckRepo - Without DB / uninstalled repo
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/check-repo?owner=acmeorg&repo=not-installed", nil)
	rec = httptest.NewRecorder()
	noDB.HandleCheckRepo(rec, req)
	_ = json.NewDecoder(rec.Body).Decode(&checkRes)
	if checkRes.Installed {
		t.Errorf("expected uninstalled repo to report installed=false")
	}
}

func TestHandleSetupLLM_And_SettingsLLM(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. HandleSetupLLM - Method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/llm", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupLLM(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET setup/llm, got %d", rec.Code)
	}

	// 2. HandleSetupLLM - Invalid body
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/llm", strings.NewReader(`not json`))
	rec = httptest.NewRecorder()
	env.Server.HandleSetupLLM(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in setup/llm, got %d", rec.Code)
	}

	// 3. HandleSetupLLM - Success
	payload, _ := json.Marshal(map[string]string{
		"provider": "anthropic",
		"api_key":  "claude-secret-key",
		"model":    "claude-3-5-sonnet",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/llm", bytes.NewReader(payload))
	rec = httptest.NewRecorder()
	env.Server.HandleSetupLLM(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for setup/llm, got %d", rec.Code)
	}

	// 4. HandleSettingsLLM - GET
	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/llm", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSettingsLLM(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET settings/llm, got %d", rec.Code)
	}
	var settingsRes map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&settingsRes)
	if settingsRes["provider"] != "anthropic" || settingsRes["model"] != "claude-3-5-sonnet" {
		t.Errorf("unexpected settings returned: %+v", settingsRes)
	}

	// 5. HandleSettingsLLM - Method not allowed (DELETE)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/settings/llm", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSettingsLLM(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE settings/llm, got %d", rec.Code)
	}

	// 6. HandleSettingsLLM - POST with bad json
	req = httptest.NewRequest(http.MethodPost, "/api/v1/settings/llm", strings.NewReader(`bad json`))
	rec = httptest.NewRecorder()
	env.Server.HandleSettingsLLM(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", rec.Code)
	}

	// 7. HandleSettingsLLM - POST success
	updatePayload, _ := json.Marshal(map[string]string{
		"provider": "openai",
		"api_key":  "sk-new-key",
		"model":    "gpt-4o-mini",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/settings/llm", bytes.NewReader(updatePayload))
	rec = httptest.NewRecorder()
	env.Server.HandleSettingsLLM(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST settings/llm, got %d", rec.Code)
	}

	// 8. HandleTestLLM - Method not allowed (GET)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/llm/test", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleTestLLM(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET test llm, got %d", rec.Code)
	}

	// 9. HandleTestLLM - Invalid body
	req = httptest.NewRequest(http.MethodPost, "/api/v1/settings/llm/test", strings.NewReader(`invalid`))
	rec = httptest.NewRecorder()
	env.Server.HandleTestLLM(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", rec.Code)
	}
}
