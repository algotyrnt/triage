// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"triage/engine/internal/db"
)

func newTestAPIServer() *Server {
	return NewServer(Config{})
}

func TestServerHealthAndStats(t *testing.T) {
	s := newTestAPIServer()

	// 1. Test /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for health, got %d", rec.Code)
	}

	var healthRes struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&healthRes); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if healthRes.Status != "ok" {
		t.Errorf("expected health status 'ok', got %s", healthRes.Status)
	}

	// 2. Test /api/v1/stats
	statsReq := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	statsRec := httptest.NewRecorder()
	s.HandleStats(statsRec, statsReq)

	if statsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for stats, got %d", statsRec.Code)
	}
}

func TestServerSetupStatus(t *testing.T) {
	s := newTestAPIServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec := httptest.NewRecorder()
	s.HandleSetupStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var statusRes map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&statusRes); err != nil {
		t.Fatalf("failed to decode setup status response: %v", err)
	}
	if statusRes["configured"] == nil {
		t.Errorf("expected configured key in setup status")
	}
}

func TestServerTelemetry_Unauthorized(t *testing.T) {
	s := newTestAPIServer()

	// 1. Invalid key without database fails with 401
	body, _ := json.Marshal(TelemetryRequest{
		APIKey: "invalid_key",
		File:   "main.go",
		Line:   10,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	s.HandleTelemetry(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized for invalid key, got %d", rec.Code)
	}

	// 2. Empty key fails with 401
	emptyBody, _ := json.Marshal(TelemetryRequest{
		APIKey: "",
		File:   "main.go",
		Line:   10,
	})
	emptyReq := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewBuffer(emptyBody))
	emptyRec := httptest.NewRecorder()
	s.HandleTelemetry(emptyRec, emptyReq)

	if emptyRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized for empty key, got %d", emptyRec.Code)
	}
}

func TestServerDetectModules(t *testing.T) {
	s := newTestAPIServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repos/detect-modules", nil)
	rec := httptest.NewRecorder()
	s.HandleDetectModules(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res struct {
		Modules []DetectedModule `json:"modules"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode detect modules response: %v", err)
	}
}

func TestServerSetupRepos(t *testing.T) {
	s := newTestAPIServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/repos", nil)
	rec := httptest.NewRecorder()
	s.HandleSetupRepos(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for repos, got %d", rec.Code)
	}

	var res struct {
		Repos []SetupRepoItem `json:"repos"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode setup repos response: %v", err)
	}
}

func TestCORSResolution(t *testing.T) {
	s := newTestAPIServer()

	// 1. Initial setup phase (unconfigured DB) reflects caller origin
	setupReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	setupReq.Header.Set("Origin", "http://setup-domain.com:3000")
	setupRec := httptest.NewRecorder()
	handler := s.withMiddleware(s.HandleHealth)
	handler(setupRec, setupReq)

	if setupRec.Header().Get("Access-Control-Allow-Origin") != "http://setup-domain.com:3000" {
		t.Errorf("expected reflected origin during setup, got: %s", setupRec.Header().Get("Access-Control-Allow-Origin"))
	}
	if setupRec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected Allow-Credentials true, got: %s", setupRec.Header().Get("Access-Control-Allow-Credentials"))
	}

	// 2. Preflight OPTIONS request
	optionsReq := httptest.NewRequest(http.MethodOptions, "/api/v1/telemetry", nil)
	optionsReq.Header.Set("Origin", "http://localhost:3000")
	optionsRec := httptest.NewRecorder()
	handler(optionsRec, optionsReq)

	if optionsRec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for OPTIONS preflight, got %d", optionsRec.Code)
	}
	if optionsRec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected localhost origin allowed, got: %s", optionsRec.Header().Get("Access-Control-Allow-Origin"))
	}

	// 3. Localhost loopback check helper
	if !isLocalhostOrigin("http://localhost:3000") {
		t.Errorf("expected localhost:3000 to be localhost origin")
	}
	if !isLocalhostOrigin("http://127.0.0.1:8080") {
		t.Errorf("expected 127.0.0.1:8080 to be localhost origin")
	}
	if isLocalhostOrigin("https://evil-site.com") {
		t.Errorf("expected evil-site.com to not be localhost origin")
	}
}

func TestJWTAuthAndRBAC(t *testing.T) {
	s := newTestAPIServer()
	secret := s.getSessionSecret(context.Background())

	// 1. Generate JWT
	user := &db.User{
		ID:        "usr_123",
		GitHubID:  "123",
		Username:  "testdev",
		Email:     "dev@example.com",
		AvatarURL: "https://example.com/avatar.png",
		Role:      "Developer",
	}

	tokenStr, err := GenerateUserJWT(user, secret)
	if err != nil {
		t.Fatalf("failed to generate user JWT: %v", err)
	}

	// 2. Verify JWT
	claims, err := ParseAndVerifyUserJWT(tokenStr, secret)
	if err != nil {
		t.Fatalf("failed to parse user JWT: %v", err)
	}
	if claims.Username != "testdev" || claims.Role != "Developer" {
		t.Errorf("unexpected claims: %+v", claims)
	}

	// 3. Test /api/v1/auth/me
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+tokenStr)
	meRec := httptest.NewRecorder()

	s.HandleAuthMe(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/v1/auth/me, got %d", meRec.Code)
	}

	// 4. Test withAuth middleware with role check
	protectedHandler := s.withAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"authorized"}`))
	}, "Owner", "Admin")

	// Developer should be forbidden (403)
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+tokenStr)
	forbiddenRec := httptest.NewRecorder()
	protectedHandler(forbiddenRec, forbiddenReq)

	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Developer role, got %d", forbiddenRec.Code)
	}

	// Owner token should pass (200)
	ownerUser := &db.User{
		ID:       "usr_owner",
		Username: "ownerdev",
		Role:     "Owner",
	}
	ownerToken, _ := GenerateUserJWT(ownerUser, secret)

	allowedReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	allowedReq.Header.Set("Authorization", "Bearer "+ownerToken)
	allowedRec := httptest.NewRecorder()
	protectedHandler(allowedRec, allowedReq)

	if allowedRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for Owner role, got %d", allowedRec.Code)
	}
}

func TestProjectsWithContext(t *testing.T) {
	s := newTestAPIServer()

	// 1. Test POST /api/v1/projects with context
	body, _ := json.Marshal(map[string]interface{}{
		"owner":    "myorg",
		"repo":     "payments-service",
		"root_dir": "backend",
		"context":  "High-throughput payment gateway processing Stripe webhooks.",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.HandleProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var createRes struct {
		Success bool   `json:"success"`
		Repo    string `json:"repo"`
		Context string `json:"context"`
		ApiKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&createRes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !createRes.Success || createRes.Context != "High-throughput payment gateway processing Stripe webhooks." {
		t.Errorf("unexpected create response: %+v", createRes)
	}

	// 2. Test PUT /api/v1/projects/context
	updateBody, _ := json.Marshal(map[string]interface{}{
		"owner":    "myorg",
		"repo":     "payments-service",
		"root_dir": "backend",
		"context":  "Updated architectural context for payment service.",
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/projects/context", bytes.NewReader(updateBody))
	updateRec := httptest.NewRecorder()
	s.HandleUpdateProjectContext(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for context update, got %d", updateRec.Code)
	}
}
