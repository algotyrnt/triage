// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

// VerifySignature verifies the X-Hub-Signature-256 header sent by GitHub using HMAC-SHA256.
func VerifySignature(payload []byte, signatureHeader string, secret string) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}

	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	expectedSigHex := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	actualSig := mac.Sum(nil)

	return hmac.Equal(actualSig, expectedSig)
}

// WebhookHandler handles incoming GitHub App webhooks (e.g. installation.deleted, push)
func WebhookHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		sig := r.Header.Get("X-Hub-Signature-256")
		if secret != "" && !VerifySignature(body, sig, secret) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		eventType := r.Header.Get("X-GitHub-Event")
		switch eventType {
		case "installation":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"processed","event":"installation"}`))
		case "push":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"processed","event":"push"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ignored"}`))
		}
	}
}
