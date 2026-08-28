// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if len(createRes.Key.RawKey) != 32 {
		t.Errorf("expected raw_key to be 32 hex chars, got: %s", createRes.Key.RawKey)
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
