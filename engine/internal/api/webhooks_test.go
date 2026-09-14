// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"triage/engine/internal/github"
)

func computeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "my-webhook-secret"
	payload := []byte(`{"action":"created"}`)
	validSig := computeSignature(payload, secret)

	if !verifyWebhookSignature(payload, validSig, secret) {
		t.Errorf("expected signature to verify successfully")
	}

	if verifyWebhookSignature(payload, "sha256=invalidhash", secret) {
		t.Errorf("expected invalid signature to fail")
	}

	if verifyWebhookSignature(payload, validSig, "wrong-secret") {
		t.Errorf("expected wrong secret to fail")
	}
}

func TestHandleGitHubWebhook(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Method Not Allowed (GET)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/github", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rec.Code)
	}

	// 2. Invalid JSON payload
	req = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader([]byte(`not json`)))
	rec = httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rec.Code)
	}

	// 3. Ping event
	req = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-GitHub-Event", "ping")
	rec = httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ping, got %d", rec.Code)
	}
	var pingRes struct {
		Status string `json:"status"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&pingRes)
	if pingRes.Status != "pong" {
		t.Errorf("expected status 'pong', got %s", pingRes.Status)
	}

	// 4. Webhook with signature verification
	env.Server.githubApp = &github.AppConfig{
		AppID:         12345,
		WebhookSecret: "secret-key-123",
	}

	rawPayload := []byte(`{"action":"created"}`)
	validSig := computeSignature(rawPayload, "secret-key-123")

	// Invalid signature -> 401
	req = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(rawPayload))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	rec = httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid signature, got %d", rec.Code)
	}

	// Valid signature -> 200
	req = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(rawPayload))
	req.Header.Set("X-Hub-Signature-256", validSig)
	rec = httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid signature, got %d", rec.Code)
	}

	// 5. Form URL-encoded webhook payload
	formBody := "payload=" + url.QueryEscape(`{"action":"form_action"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader([]byte(formBody)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for form urlencoded payload, got %d", rec.Code)
	}

	// 6. Installation event with repository mapping
	installPayload := map[string]interface{}{
		"action": "created",
		"installation": map[string]interface{}{
			"id": 998877,
		},
		"repository": map[string]interface{}{
			"name": "core-api",
			"owner": map[string]interface{}{
				"login": "octocat",
			},
		},
	}
	installBytes, _ := json.Marshal(installPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(installBytes))
	req.Header.Set("X-GitHub-Event", "installation")
	rec = httptest.NewRecorder()
	env.Server.HandleGitHubWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for installation event, got %d", rec.Code)
	}

	// Verify installation recorded in DB
	instID, err := env.DB.GetInstallationForRepo(req.Context(), "octocat", "core-api")
	if err != nil || instID != 998877 {
		t.Errorf("expected installation ID 998877 saved in DB, got %d, err=%v", instID, err)
	}
}
