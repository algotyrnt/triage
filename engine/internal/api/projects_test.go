// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleProjects_POST_And_GET(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Viewer cannot create project (Forbidden 403)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"repo":"test/repo"}`))
	req.Header.Set("Authorization", "Bearer "+env.ViewerToken)
	rec := httptest.NewRecorder()
	env.Server.HandleProjects(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Viewer, got %d", rec.Code)
	}

	// 2. Invalid JSON payload (400)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`invalid json`))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleProjects(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}

	// 3. Missing repo field (400)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"owner":"myorg"}`))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleProjects(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing repo, got %d", rec.Code)
	}

	// 4. Success creating project with owner/repo format
	createBody := map[string]interface{}{
		"repo":       "acme/analytics",
		"root_dir":   "services/worker",
		"context":    "Processes real-time clickstream events",
		"owner_user": "owner_user",
	}
	b, _ := json.Marshal(createBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleProjects(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for creating project, got %d: %s", rec.Code, rec.Body.String())
	}
	var createRes map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&createRes)
	if createRes["api_key"] == "" || createRes["key_masked"] == "" {
		t.Errorf("expected api_key in response: %+v", createRes)
	}

	// 5. Creating duplicate project returns existing masked key
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleProjects(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for duplicate project, got %d", rec.Code)
	}

	// 6. No DB mode generates secure API key
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(b))
	rec = httptest.NewRecorder()
	noDB.HandleProjects(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for no DB project creation, got %d", rec.Code)
	}

	// 7. GET /api/v1/projects without DB
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec = httptest.NewRecorder()
	noDB.HandleProjects(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for no DB GET projects, got %d", rec.Code)
	}

	// 8. GET /api/v1/projects with DB
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleProjects(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET projects, got %d", rec.Code)
	}
	var getRes struct {
		Projects []interface{} `json:"projects"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&getRes)
	if len(getRes.Projects) == 0 {
		t.Errorf("expected at least 1 project in response")
	}
}

func TestHandleUpdateProjectContext_EdgeCases(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/context", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleUpdateProjectContext(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rec.Code)
	}

	// 2. Viewer role forbidden
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/context", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+env.ViewerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleUpdateProjectContext(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer, got %d", rec.Code)
	}

	// 3. Invalid JSON
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/context", strings.NewReader(`invalid json`))
	req.Header.Set("Authorization", "Bearer "+env.DevToken)
	rec = httptest.NewRecorder()
	env.Server.HandleUpdateProjectContext(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}

	// 4. Success updating context with owner/repo format
	payload, _ := json.Marshal(map[string]string{
		"repo":     "testorg/payments",
		"root_dir": "",
		"context":  "Updated architectural context",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/context", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+env.DevToken)
	rec = httptest.NewRecorder()
	env.Server.HandleUpdateProjectContext(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for context update, got %d", rec.Code)
	}
}

func TestHandleProjectKeys_And_Revoke(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Invalid HTTP Method (PUT) -> 405
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/keys", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleProjectKeys(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}

	// 2. GET without DB
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/keys?owner=o&repo=r", nil)
	rec = httptest.NewRecorder()
	noDB.HandleProjectKeys(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for GET keys without DB, got %d", rec.Code)
	}

	// 3. POST without DB
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys", strings.NewReader(`{"owner":"o","repo":"r","name":"test-key"}`))
	rec = httptest.NewRecorder()
	noDB.HandleProjectKeys(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for POST keys without DB, got %d", rec.Code)
	}

	// 4. POST with invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys", strings.NewReader(`bad json`))
	rec = httptest.NewRecorder()
	env.Server.HandleProjectKeys(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rec.Code)
	}

	// 5. Create project and generate a new key with DB
	_, _, _ = env.DB.CreateProject(ctx, "acme", "billing", "", "owner_user")

	postKeyPayload, _ := json.Marshal(map[string]string{
		"owner":    "acme",
		"repo":     "billing",
		"root_dir": "",
		"name":     "Production Billing Key",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys", bytes.NewReader(postKeyPayload))
	rec = httptest.NewRecorder()
	env.Server.HandleProjectKeys(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for POST key, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdKeyRes struct {
		Success bool `json:"success"`
		Key     struct {
			ID string `json:"id"`
		} `json:"key"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&createdKeyRes)
	keyID := createdKeyRes.Key.ID
	if keyID == "" {
		t.Fatalf("expected non-empty key ID: %+v", createdKeyRes)
	}

	// 6. GET keys with DB
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/keys?owner=acme&repo=billing", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleProjectKeys(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET keys, got %d", rec.Code)
	}

	// 7. Revoke Key - Method not allowed (GET) -> 405
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/keys/revoke", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleRevokeProjectKey(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET revoke, got %d", rec.Code)
	}

	// 8. Revoke Key - Missing key_id -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys/revoke", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	env.Server.HandleRevokeProjectKey(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing key_id, got %d", rec.Code)
	}

	// 9. Revoke Key - Success via JSON body
	revokePayload, _ := json.Marshal(map[string]string{"key_id": keyID})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys/revoke", bytes.NewReader(revokePayload))
	rec = httptest.NewRecorder()
	env.Server.HandleRevokeProjectKey(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for revoke key, got %d", rec.Code)
	}

	// 10. Revoking already revoked key returns 500
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/keys/revoke", bytes.NewReader(revokePayload))
	rec = httptest.NewRecorder()
	env.Server.HandleRevokeProjectKey(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for already revoked key, got %d", rec.Code)
	}

	// 11. Revoke Key - Success via Query parameter (DELETE) on a fresh key
	keyRecord2, _ := env.DB.CreateAPIKey(ctx, "acme", "billing", "", "Second Key")
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/keys/revoke?key_id="+keyRecord2.ID, nil)
	rec = httptest.NewRecorder()
	env.Server.HandleRevokeProjectKey(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for DELETE revoke with query param, got %d", rec.Code)
	}
}
