// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type gitHubWebhookPayload struct {
	Action string `json:"action"`
	Issue  *struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Title   string `json:"title"`
	} `json:"issue,omitempty"`
	Repository *struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository,omitempty"`
	Installation *struct {
		ID int64 `json:"id"`
	} `json:"installation,omitempty"`
}

// HandleGitHubWebhook processes inbound GitHub App Webhook deliveries.
func (s *Server) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "Failed to read request payload", http.StatusBadRequest)
		return
	}

	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	// Verify HMAC-SHA256 signature if webhook secret is configured and signature header present
	if s.githubApp != nil && strings.TrimSpace(s.githubApp.WebhookSecret) != "" {
		sigHeader := r.Header.Get("X-Hub-Signature-256")
		if sigHeader != "" {
			if !verifyWebhookSignature(body, sigHeader, strings.TrimSpace(s.githubApp.WebhookSecret)) {
				slog.Warn("github webhook signature verification failed", "signature", sigHeader)
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
		}
	}

	contentType := r.Header.Get("Content-Type")
	jsonBody := body
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		payloadStr := strings.TrimPrefix(string(body), "payload=")
		if unescaped, uErr := url.QueryUnescape(payloadStr); uErr == nil {
			jsonBody = []byte(unescaped)
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	var payload gitHubWebhookPayload
	if err := json.Unmarshal(jsonBody, &payload); err != nil {
		slog.Warn("failed to decode github webhook json", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	slog.Info("received github webhook event", "event", eventType, "action", payload.Action, "delivery_id", deliveryID)

	switch eventType {
	case "ping":
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "pong"})
		return

	case "installation", "installation_repositories":
		if payload.Installation != nil && payload.Installation.ID > 0 {
			if payload.Repository != nil {
				owner := payload.Repository.Owner.Login
				repo := payload.Repository.Name
				if s.db != nil && owner != "" && repo != "" {
					_ = s.db.SaveInstallation(r.Context(), payload.Installation.ID, owner, 0, "Organization")
					_ = s.db.SaveInstallationRepo(r.Context(), payload.Installation.ID, owner, repo)
					_ = s.db.UpdateRepositoryInstallation(r.Context(), owner, repo, payload.Installation.ID)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"event":   eventType,
		"action":  payload.Action,
	})
}

func verifyWebhookSignature(payload []byte, signatureHeader, secret string) bool {
	sig := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(payload)
	expectedSignature := hex.EncodeToString(expectedMAC.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expectedSignature))
}
