// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package triage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewarePanicRecovery(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated test crash")
	})

	mw := Middleware("test_key", "http://localhost:8080/api/v1/telemetry")
	handler := mw(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != "Internal Server Error" {
		t.Errorf("expected generic body 'Internal Server Error', got '%s'", body)
	}
}
