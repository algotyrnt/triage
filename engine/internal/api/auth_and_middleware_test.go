// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triage/engine/internal/config"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/llm"
)

func TestHandleAuthGitHub_And_Callback_EdgeCases(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. AuthGitHub when OAuth is unconfigured -> redirects with error
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleAuthGitHub(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "reason=oauth_not_configured") {
		t.Errorf("expected oauth_not_configured in redirect: %s", rec.Header().Get("Location"))
	}

	// 2. AuthGitHub when OAuth is configured -> redirects to GitHub authorize URL with CSRF cookie
	_ = env.ConfigStore.SaveGitHubOAuth(ctx, "mock_client_id", "mock_client_secret")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/github", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleAuthGitHub(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "https://github.com/login/oauth/authorize") || !strings.Contains(loc, "client_id=mock_client_id") {
		t.Errorf("unexpected authorize redirect location: %s", loc)
	}
	// Verify CSRF cookie set
	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "triage_oauth_state" {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil || stateCookie.Value == "" {
		t.Fatalf("expected triage_oauth_state cookie to be set")
	}

	// 3. Callback without state cookie -> redirects with invalid_state
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=abc&state="+stateCookie.Value, nil)
	rec = httptest.NewRecorder()
	env.Server.HandleAuthGitHubCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "reason=invalid_state") {
		t.Errorf("expected invalid_state redirect, got %d, loc=%s", rec.Code, rec.Header().Get("Location"))
	}

	// 4. Callback with mismatched state -> redirects with invalid_state
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=abc&state=wrong_state", nil)
	req.AddCookie(stateCookie)
	rec = httptest.NewRecorder()
	env.Server.HandleAuthGitHubCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "reason=invalid_state") {
		t.Errorf("expected invalid_state redirect for wrong state, got %d, loc=%s", rec.Code, rec.Header().Get("Location"))
	}

	// 5. Callback with valid state but missing code -> redirects with missing_code
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?state="+stateCookie.Value, nil)
	req.AddCookie(stateCookie)
	rec = httptest.NewRecorder()
	env.Server.HandleAuthGitHubCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "reason=missing_code") {
		t.Errorf("expected missing_code redirect, got %d, loc=%s", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHandleAuthGitHubCallback_Success(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	_ = env.ConfigStore.SaveGitHubOAuth(ctx, "mock_client_id", "mock_client_secret")

	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.String(), "access_token") {
			body := `{"access_token":"gho_mock_token_123"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}
		if strings.Contains(req.URL.String(), "api.github.com/user") {
			body := `{"id":882233,"login":"octonewbie","email":"newbie@example.com","avatar_url":"https://avatar.png"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=mock_code&state=good_state", nil)
	req.AddCookie(&http.Cookie{
		Name:  "triage_oauth_state",
		Value: "good_state",
	})
	rec := httptest.NewRecorder()
	env.Server.HandleAuthGitHubCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "auth=success") {
		t.Errorf("expected auth=success redirect, got %s", loc)
	}

	// Verify session cookie was issued
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "triage_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Errorf("expected session cookie to be set")
	}

	// Verify user exists in database
	user, err := env.DB.GetUserByID(ctx, "usr_882233")
	if err != nil || user == nil || user.Username != "octonewbie" {
		t.Errorf("expected user in DB, got %+v, err=%v", user, err)
	}
}

func TestHandleAuthLogout(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Method not allowed (GET)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleAuthLogout(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET logout, got %d", rec.Code)
	}

	// 2. POST logout invalidates session cookie
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	env.Server.HandleAuthLogout(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for logout, got %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "triage_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.MaxAge != -1 || !sessionCookie.Secure {
		t.Errorf("expected expired secure session cookie: %+v", sessionCookie)
	}
}

func TestHandleAuthMe_Variations(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Method not allowed (POST)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleAuthMe(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}

	// 2. Unauthenticated -> 401
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleAuthMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated auth/me, got %d", rec.Code)
	}

	// 3. Authenticated via session cookie -> 200
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  "triage_session",
		Value: env.OwnerToken,
	})
	rec = httptest.NewRecorder()
	env.Server.HandleAuthMe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for cookie-authenticated auth/me, got %d", rec.Code)
	}
	var res map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	user, ok := res["user"].(map[string]interface{})
	if !ok || user["username"] != "owner_user" || user["role"] != "Owner" {
		t.Errorf("unexpected auth me user data: %+v", res)
	}
}

func TestExtractBearerOrAPIKey(t *testing.T) {
	// 1. Nil request
	if key := ExtractBearerOrAPIKey(nil); key != "" {
		t.Errorf("expected empty string for nil request")
	}

	// 2. X-Triage-API-Key header
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Triage-API-Key", "my_custom_api_key")
	if key := ExtractBearerOrAPIKey(req1); key != "my_custom_api_key" {
		t.Errorf("expected custom API key, got %s", key)
	}

	// 3. Authorization Bearer
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer my_jwt_token")
	if key := ExtractBearerOrAPIKey(req2); key != "my_jwt_token" {
		t.Errorf("expected jwt token, got %s", key)
	}

	// 4. Session cookie
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(&http.Cookie{Name: "triage_session", Value: "session_token_123"})
	if key := ExtractBearerOrAPIKey(req3); key != "session_token_123" {
		t.Errorf("expected session cookie token, got %s", key)
	}

	// 5. Empty
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	if key := ExtractBearerOrAPIKey(req4); key != "" {
		t.Errorf("expected empty string, got %s", key)
	}
}

func TestServerRoutes_And_Helpers(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Routes() returns initialized mux handler
	routes := env.Server.Routes()
	if routes == nil {
		t.Fatalf("expected non-nil routes")
	}

	// 2. ResolveEngineURL
	if url := env.Server.ResolveEngineURL(nil); url != "http://localhost:8080" {
		t.Errorf("expected localhost fallback for nil request: %s", url)
	}
	tlsReq := httptest.NewRequest(http.MethodGet, "https://triage.local:9000/test", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	if url := env.Server.ResolveEngineURL(tlsReq); !strings.HasPrefix(url, "https://") {
		t.Errorf("expected https prefix for TLS request: %s", url)
	}

	// 3. ResolveAppURL without config store
	noCfgServer := NewServer(Config{})
	noCfgServer.configStore = nil
	if appURL := noCfgServer.ResolveAppURL(ctx); appURL != "" {
		t.Errorf("expected empty string when configStore is nil")
	}

	// 4. GetLLMProvider
	_ = env.ConfigStore.SaveLLM(ctx, llm.Config{Provider: "ollama"})
	prov, err := env.Server.GetLLMProvider(ctx)
	if err != nil || prov == nil {
		t.Fatalf("expected ollama provider: %v", err)
	}

	// 5. ResolveInstallationID
	// Without GitHub App
	_, err = env.Server.ResolveInstallationID(ctx, "myorg", "myrepo")
	if err == nil {
		t.Errorf("expected error when GitHub App not configured")
	}

	// With DB installation repo
	env.Server.githubApp = &github.AppConfig{AppID: 101}
	_ = env.DB.SaveInstallation(ctx, 8888, "myorg", 1, "Organization")
	_ = env.DB.SaveInstallationRepo(ctx, 8888, "myorg", "myrepo")
	id, err := env.Server.ResolveInstallationID(ctx, "myorg", "myrepo")
	if err != nil || id != 8888 {
		t.Errorf("expected installation ID 8888 from DB, got %d, err=%v", id, err)
	}
}

func TestGetSessionSecret_Errors(t *testing.T) {
	// Server without configStore
	s := NewServer(Config{})
	s.configStore = nil
	ctx := context.Background()
	_, err := s.getSessionSecret(ctx)
	if err == nil {
		t.Errorf("expected error when configStore is nil")
	}

	// Server with DB closed (cannot ensure secret)
	testDB, _ := db.NewDB(ctx, t.TempDir()+"/secret_test.db")
	store := config.NewStore(testDB)
	testDB.Close()
	s2 := NewServer(Config{DB: testDB, ConfigStore: store})
	_, err = s2.getSessionSecret(ctx)
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}
}
