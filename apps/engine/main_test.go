// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"triage/engine/internal/api"
)

func newTestServer() *api.Server {
	return api.NewServer(api.Config{})
}

func TestIsValidAPIKey(t *testing.T) {
	s := newTestServer()
	ctx := context.Background()

	// 1. Without database, IsValidAPIKey fails closed
	if s.IsValidAPIKey(ctx, "any_key") {
		t.Errorf("expected IsValidAPIKey to fail closed when database is nil")
	}

	// 2. Empty input key should return false
	if s.IsValidAPIKey(ctx, "") {
		t.Errorf("expected IsValidAPIKey to return false for empty key")
	}
}

func TestValidateAndResolveFilePath_Symlink(t *testing.T) {
	s := newTestServer()
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	externalDir := filepath.Join(tmpDir, "external")
	if err := os.MkdirAll(externalDir, 0755); err != nil {
		t.Fatalf("failed to create external dir: %v", err)
	}

	externalFile := filepath.Join(externalDir, "secret.go")
	if err := os.WriteFile(externalFile, []byte("package external\n"), 0644); err != nil {
		t.Fatalf("failed to create external file: %v", err)
	}

	// Create symlink inside workspace pointing to external file outside workspace
	symlinkPath := filepath.Join(workspaceDir, "symlink.go")
	if err := os.Symlink(externalFile, symlinkPath); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	_ = os.Setenv("TRIAGE_WORKSPACE_ROOT", workspaceDir)
	defer os.Unsetenv("TRIAGE_WORKSPACE_ROOT")

	// Expect validation to reject target because its resolved symlink path lies outside workspace root
	_, err := s.ValidateAndResolveFilePath("symlink.go")
	if err == nil {
		t.Errorf("expected error for symlink pointing outside workspace root, got nil")
	}
}

func TestValidateAndResolveFilePath_Monorepo(t *testing.T) {
	s := newTestServer()
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "monorepo")
	backendDir := filepath.Join(workspaceDir, "backend", "pkg", "handler")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("failed to create backend dir: %v", err)
	}

	userGo := filepath.Join(backendDir, "user.go")
	if err := os.WriteFile(userGo, []byte("package handler\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	expectedPath := userGo
	if evalExpected, err := filepath.EvalSymlinks(userGo); err == nil {
		expectedPath = evalExpected
	}

	_ = os.Setenv("TRIAGE_WORKSPACE_ROOT", workspaceDir)
	defer os.Unsetenv("TRIAGE_WORKSPACE_ROOT")

	// 1. Resolve relative file "pkg/handler/user.go" with rootDir "backend"
	resolved, err := s.ValidateAndResolveFilePath("pkg/handler/user.go", "backend")
	if err != nil {
		t.Fatalf("expected no error resolving monorepo path, got: %v", err)
	}
	if resolved != expectedPath {
		t.Errorf("expected resolved path %s, got %s", expectedPath, resolved)
	}

	// 2. Resolve already prefixed file "backend/pkg/handler/user.go" with rootDir "backend"
	resolved2, err := s.ValidateAndResolveFilePath("backend/pkg/handler/user.go", "backend")
	if err != nil {
		t.Fatalf("expected no error resolving already-prefixed path, got: %v", err)
	}
	if resolved2 != expectedPath {
		t.Errorf("expected resolved path %s, got %s", expectedPath, resolved2)
	}
}

func TestValidateAndResolveFilePath_AbsoluteServicePath(t *testing.T) {
	s := newTestServer()
	tmpDir := t.TempDir()

	serviceDir := filepath.Join(tmpDir, "test-service")
	engineDir := filepath.Join(tmpDir, "apps", "engine")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatalf("failed to create test-service dir: %v", err)
	}
	if err := os.MkdirAll(engineDir, 0755); err != nil {
		t.Fatalf("failed to create apps/engine dir: %v", err)
	}

	serviceMain := filepath.Join(serviceDir, "main.go")
	engineMain := filepath.Join(engineDir, "main.go")
	_ = os.WriteFile(serviceMain, []byte("package main\nfunc testService() {}\n"), 0644)
	_ = os.WriteFile(engineMain, []byte("package main\nfunc engineMain() {}\n"), 0644)

	_ = os.Setenv("TRIAGE_WORKSPACE_ROOT", tmpDir)
	defer os.Unsetenv("TRIAGE_WORKSPACE_ROOT")

	// Resolving absolute path to test-service/main.go must return test-service/main.go, never apps/engine/main.go
	resolved, err := s.ValidateAndResolveFilePath(serviceMain)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evalServiceMain, _ := filepath.EvalSymlinks(serviceMain)
	if resolved != evalServiceMain {
		t.Errorf("expected %s, got %s (did it incorrectly resolve to engine main?)", evalServiceMain, resolved)
	}
}

func TestProjectKeysRoutes(t *testing.T) {
	s := newTestServer()

	// 1. Test GET /api/v1/projects/keys (empty fallback)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/keys?owner=algotyrnt&repo=triage", nil)
	rec := httptest.NewRecorder()
	s.HandleProjectKeys(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var getRes struct {
		Keys []interface{} `json:"keys"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&getRes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 2. Test POST /api/v1/projects/keys (create key)
	createPayload := map[string]string{
		"owner": "algotyrnt",
		"repo":  "test-repo",
		"name":  "Test Ingestion Key",
	}
	body, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys", bytes.NewBuffer(body))
	createRec := httptest.NewRecorder()
	s.HandleProjectKeys(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for key creation, got %d", createRec.Code)
	}

	var createRes struct {
		Success bool `json:"success"`
		Key     struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			RawKey    string `json:"raw_key"`
			KeyMasked string `json:"key_masked"`
			Status    string `json:"status"`
		} `json:"key"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&createRes); err != nil {
		t.Fatalf("failed to decode create key response: %v", err)
	}
	if !createRes.Success {
		t.Errorf("expected success=true in create key response")
	}
	if !strings.HasPrefix(createRes.Key.RawKey, "tr_live_") {
		t.Errorf("expected raw_key to have prefix 'tr_live_', got: %s", createRes.Key.RawKey)
	}

	// 3. Test POST /api/v1/projects/keys/revoke
	revokePayload := map[string]string{
		"key_id": createRes.Key.ID,
	}
	revokeBody, _ := json.Marshal(revokePayload)
	revokeReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys/revoke", bytes.NewBuffer(revokeBody))
	revokeRec := httptest.NewRecorder()
	s.HandleRevokeProjectKey(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for key revoke, got %d", revokeRec.Code)
	}
}

func TestCreateIncidentIssueRoute(t *testing.T) {
	s := newTestServer()

	// 1. Test missing incident_id
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	s.HandleCreateIncidentIssue(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 400 or 503, got %d", rec.Code)
	}

	// 2. Test invalid method GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/create-issue", nil)
	getRec := httptest.NewRecorder()
	s.HandleCreateIncidentIssue(getRec, getReq)

	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", getRec.Code)
	}
}

func TestLLMRoutes(t *testing.T) {
	s := newTestServer()

	// 1. Test POST /api/v1/llm/analyze-panic missing api key or model
	body, _ := json.Marshal(map[string]string{
		"panicMessage":   "nil pointer dereference",
		"triggeringFile": "main.go",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/llm/analyze-panic", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	s.HandleLLMAnalyzePanic(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when AI is unconfigured, got %d", rec.Code)
	}

	// 2. Test POST /api/v1/llm/generate-patch missing api key
	patchBody, _ := json.Marshal(map[string]string{
		"triggeringFile": "main.go",
		"panicMessage":   "nil pointer dereference",
	})
	patchReq := httptest.NewRequest(http.MethodPost, "/api/v1/llm/generate-patch", bytes.NewBuffer(patchBody))
	patchRec := httptest.NewRecorder()
	s.HandleLLMGeneratePatch(patchRec, patchReq)

	if patchRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when LLM key missing, got %d", patchRec.Code)
	}

	// 3. Test method not allowed
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/llm/generate-patch", nil)
	getRec := httptest.NewRecorder()
	s.HandleLLMGeneratePatch(getRec, getReq)

	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", getRec.Code)
	}
}

func TestCreateIncidentPRRoute(t *testing.T) {
	s := newTestServer()

	// 1. Test missing incident_id
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	s.HandleCreateIncidentPR(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 400 or 503, got %d", rec.Code)
	}

	// 2. Test invalid method GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/create-pr", nil)
	getRec := httptest.NewRecorder()
	s.HandleCreateIncidentPR(getRec, getReq)

	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", getRec.Code)
	}
}
