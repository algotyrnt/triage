// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newTestAPIServer() *Server {
	return NewServer(Config{
		AppURL:    "http://localhost:3000",
		EngineURL: "http://localhost:8080",
	})
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

	body, _ := json.Marshal(TelemetryRequest{
		APIKey: "invalid_key",
		File:   "main.go",
		Line:   10,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	s.HandleTelemetry(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestServerTelemetry_ValidKey(t *testing.T) {
	s := newTestAPIServer()
	_ = os.Setenv("TRIAGE_API_KEY", "tr_secret_test_key")
	defer os.Unsetenv("TRIAGE_API_KEY")

	body, _ := json.Marshal(TelemetryRequest{
		APIKey:     "tr_secret_test_key",
		File:       "main.go",
		Line:       10,
		StackTrace: "panic: runtime error\ngoroutine 1 [running]:",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	s.HandleTelemetry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}

	var res TelemetryResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode telemetry response: %v", err)
	}
	if res.Status != "success" {
		t.Errorf("expected status 'success', got: %s", res.Status)
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
