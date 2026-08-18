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
)

func TestIsValidAPIKey(t *testing.T) {
	ctx := context.Background()

	// 1. Unset TRIAGE_API_KEY should fail closed
	_ = os.Unsetenv("TRIAGE_API_KEY")
	if isValidAPIKey(ctx, "any_key") {
		t.Errorf("expected isValidAPIKey to fail closed when TRIAGE_API_KEY is unset")
	}

	// 2. Empty input key should return false
	_ = os.Setenv("TRIAGE_API_KEY", "tr_valid_key")
	defer os.Unsetenv("TRIAGE_API_KEY")

	if isValidAPIKey(ctx, "") {
		t.Errorf("expected isValidAPIKey to return false for empty key")
	}

	// 3. Valid matching key should return true
	if !isValidAPIKey(ctx, "tr_valid_key") {
		t.Errorf("expected isValidAPIKey to return true for matching key")
	}

	// 4. Mismatched key should return false
	if isValidAPIKey(ctx, "tr_wrong_key") {
		t.Errorf("expected isValidAPIKey to return false for wrong key")
	}
}

func TestValidateAndResolveFilePath_Symlink(t *testing.T) {
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

	_ = os.Setenv("AST_WORKSPACE_ROOT", workspaceDir)
	defer os.Unsetenv("AST_WORKSPACE_ROOT")

	// Expect validation to reject target because its resolved symlink path lies outside workspace root
	_, err := validateAndResolveFilePath("symlink.go")
	if err == nil {
		t.Errorf("expected error for symlink pointing outside workspace root, got nil")
	}
}

func TestValidateAndResolveFilePath_Monorepo(t *testing.T) {
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

	_ = os.Setenv("AST_WORKSPACE_ROOT", workspaceDir)
	defer os.Unsetenv("AST_WORKSPACE_ROOT")

	// 1. Resolve relative file "pkg/handler/user.go" with rootDir "backend"
	resolved, err := validateAndResolveFilePath("pkg/handler/user.go", "backend")
	if err != nil {
		t.Fatalf("expected no error resolving monorepo path, got: %v", err)
	}
	if resolved != expectedPath {
		t.Errorf("expected resolved path %s, got %s", expectedPath, resolved)
	}

	// 2. Resolve already prefixed file "backend/pkg/handler/user.go" with rootDir "backend"
	resolved2, err := validateAndResolveFilePath("backend/pkg/handler/user.go", "backend")
	if err != nil {
		t.Fatalf("expected no error resolving already-prefixed path, got: %v", err)
	}
	if resolved2 != expectedPath {
		t.Errorf("expected resolved path %s, got %s", expectedPath, resolved2)
	}
}

func TestProjectKeysRoutes(t *testing.T) {
	// 1. Test GET /api/v1/projects/keys (empty fallback)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/keys?owner=algotyrnt&repo=triage", nil)
	rec := httptest.NewRecorder()
	handleProjectKeysRoute(rec, req)

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
	handleProjectKeysRoute(createRec, createReq)

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
	handleRevokeProjectKeyRoute(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for key revoke, got %d", revokeRec.Code)
	}
}

func TestCreateIncidentIssueRoute(t *testing.T) {
	// 1. Test missing incident_id
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	handleCreateIncidentIssueRoute(rec, req)

	// Since database is nil during this test, expect service unavailable (503) or bad request (400)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 400 or 503, got %d", rec.Code)
	}

	// 2. Test invalid method GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/create-issue", nil)
	getRec := httptest.NewRecorder()
	handleCreateIncidentIssueRoute(getRec, getReq)

	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", getRec.Code)
	}
}
