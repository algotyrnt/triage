// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"triage/engine/internal/llm"
)

func TestFormatIssueBody(t *testing.T) {
	req := &IssueRequest{
		File:       "/path/to/main.go",
		Line:       42,
		StackTrace: "goroutine 1 [running]:\nmain.main()",
		ASTSnippet: "func main() { panic(\"boom\") }",
		Analysis: &llm.AnalysisResult{
			RootCause:    "Explicit panic call",
			SuggestedFix: "Remove panic call",
		},
	}

	body := formatIssueBody(req)
	if body == "" {
		t.Fatalf("expected non-empty body")
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "test-webhook-secret"
	payload := []byte(`{"action":"deleted"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sigHex := hex.EncodeToString(mac.Sum(nil))
	sigHeader := "sha256=" + sigHex

	if !VerifySignature(payload, sigHeader, secret) {
		t.Errorf("expected valid signature verification")
	}

	if VerifySignature(payload, "sha256=invalid", secret) {
		t.Errorf("expected invalid signature verification to fail")
	}
}
