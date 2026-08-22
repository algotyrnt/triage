// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
