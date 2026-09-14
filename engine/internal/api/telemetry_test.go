// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triage/engine/internal/llm"
)

func TestIsValidTraceID(t *testing.T) {
	tests := []struct {
		traceID string
		want    bool
	}{
		{"", false},
		{"short", false},
		{"1234567", false},
		{"12345678", true},
		{"trace-uuid-1234-5678-abcdef", true},
		{"trace_id_valid_123", true},
		{"trace with spaces", false},
		{"trace@invalid#chars", false},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
	}

	for _, tt := range tests {
		got := isValidTraceID(tt.traceID)
		if got != tt.want {
			t.Errorf("isValidTraceID(%q) = %v; want %v", tt.traceID, got, tt.want)
		}
	}
}

func TestExtractASTContext(t *testing.T) {
	s := NewServer(Config{})
	ctx := context.Background()

	// 1. Cache hit
	s.astCache.Set("owner", "repo", "main", "service/pay.go", 42, "func ProcessPayment() { panic(\"nil\") }")
	snippet, err := s.ExtractASTContext(ctx, "owner", "repo", "main", "service/pay.go", 42)
	if err != nil || !strings.Contains(snippet, "ProcessPayment") {
		t.Fatalf("expected cache hit: %v, snippet: %s", err, snippet)
	}

	// 2. Cache miss without fetcher returns error
	_, err = s.ExtractASTContext(ctx, "owner", "repo", "main", "missing.go", 99)
	if err == nil {
		t.Errorf("expected error for unretrievable AST")
	}
}

func TestHandleTelemetry_Flows(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Method Not Allowed (GET)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleTelemetry(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rec.Code)
	}

	// 2. Invalid JSON payload
	req = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", strings.NewReader(`malformed json`))
	rec = httptest.NewRecorder()
	env.Server.HandleTelemetry(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rec.Code)
	}

	// 3. Create a project to get a valid API Key
	rawKey, repoID, err := env.DB.CreateProject(ctx, "testorg", "backend", "", "owner_user")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 4. Ingest Telemetry with valid API Key and Trace ID
	telemetryPayload := TelemetryRequest{
		APIKey:       rawKey,
		Owner:        "testorg",
		Repo:         "backend",
		File:         "internal/worker.go",
		Line:         42,
		PanicMessage: "runtime error: invalid memory address",
		StackTrace:   "goroutine 1 [running]:\nmain.go:42",
		ASTSnippet:   "func processJob() {\n    panic(\"nil\")\n}",
		TraceID:      "trace-valid-uuid-12345",
	}
	body, _ := json.Marshal(telemetryPayload)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	env.Server.HandleTelemetry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for telemetry intake, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp TelemetryResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "success" || resp.TraceID != "trace-valid-uuid-12345" {
		t.Errorf("unexpected telemetry response: %+v", resp)
	}

	// 5. Ingest duplicate telemetry (same fingerprint) -> increments occurrence count
	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewReader(body))
	dupRec := httptest.NewRecorder()
	env.Server.HandleTelemetry(dupRec, dupReq)
	if dupRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for duplicate telemetry intake, got %d", dupRec.Code)
	}

	// Verify incident in database has occurrence count = 2
	incidents, _ := env.DB.GetIncidents(ctx, 10)
	var matched bool
	for _, inc := range incidents {
		if inc.RepositoryID == repoID {
			matched = true
			if inc.OccurrenceCount != 2 {
				t.Errorf("expected occurrence count 2, got %d", inc.OccurrenceCount)
			}
			break
		}
	}
	if !matched {
		t.Errorf("expected to find ingested incident in DB for repo %s", repoID)
	}

	// 6. Ingest Telemetry with LLM Analysis enabled
	mockHTTP := mockClient(func(req *http.Request) (*http.Response, error) {
		reply := `{"root_cause":"nil pointer access","severity":"HIGH","suggested_fix":"add guard"}`
		respJSON := fmt.Sprintf(`{"choices":[{"message":{"content":%q},"finish_reason":"stop"}]}`, reply)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(respJSON)),
			Header:     make(http.Header),
		}, nil
	})

	_ = env.ConfigStore.SaveLLM(ctx, llm.Config{
		Provider:   "openai",
		APIKey:     "sk-test",
		Model:      "gpt-4o",
		HTTPClient: mockHTTP,
	})
	env.Server.llmClient = mockHTTP

	llmPayload := TelemetryRequest{
		APIKey:     rawKey,
		Owner:      "testorg",
		Repo:       "backend",
		File:       "internal/ai_panic.go",
		Line:       88,
		StackTrace: "goroutine 1 [running]:\npanic: nil pointer dereference\nai_panic.go:88",
		ASTSnippet: "func aiPanic() { panic(\"nil\") }",
	}
	llmBody, _ := json.Marshal(llmPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewReader(llmBody))
	rec = httptest.NewRecorder()
	env.Server.HandleTelemetry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for LLM-enabled telemetry intake, got %d: %s", rec.Code, rec.Body.String())
	}
	var llmResp TelemetryResponse
	_ = json.NewDecoder(rec.Body).Decode(&llmResp)
	if llmResp.Analysis == nil || llmResp.Analysis.RootCause != "nil pointer access" {
		t.Errorf("expected AI analysis in response: %+v", llmResp.Analysis)
	}
}
