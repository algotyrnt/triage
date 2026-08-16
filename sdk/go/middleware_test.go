// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package triage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMiddlewarePanicRecovery(t *testing.T) {
	telemetryChan := make(chan []byte, 1)

	// Isolated mock telemetry engine server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer ts.Close()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated test crash")
	})

	mw := Middleware("test_key", ts.URL)
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

	select {
	case payload := <-telemetryChan:
		if len(payload) == 0 {
			t.Errorf("expected non-empty telemetry payload")
		}
		if !strings.Contains(string(payload), "test_key") || !strings.Contains(string(payload), "stack_trace") {
			t.Errorf("expected telemetry payload to contain api_key and stack_trace, got: %s", string(payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for async telemetry payload to arrive at telemetry engine")
	}
}

func TestMiddlewareSelfHosted(t *testing.T) {
	telemetryChan := make(chan []byte, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated self-hosted crash")
	})

	mw := Middleware("test_selfhosted_key", ts.URL)
	handler := mw(panicHandler)

	req := httptest.NewRequest("GET", "/self-hosted", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	select {
	case payload := <-telemetryChan:
		if !strings.Contains(string(payload), "test_selfhosted_key") || !strings.Contains(string(payload), "stack_trace") {
			t.Errorf("expected payload to contain test_selfhosted_key and stack_trace, got %s", string(payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for telemetry payload on custom Gateway URL")
	}
}

func TestMiddlewarePreservesInboundTraceparent(t *testing.T) {
	telemetryChan := make(chan []byte, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("traceparent panic test")
	})

	mw := Middleware("test_key", ts.URL)
	handler := mw(panicHandler)

	req := httptest.NewRequest("GET", "/traceparent-test", nil)
	expectedTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	req.Header.Set("traceparent", "00-"+expectedTraceID+"-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	if traceIDHeader := rec.Header().Get("X-Triage-Trace-ID"); traceIDHeader != expectedTraceID {
		t.Errorf("expected response header X-Triage-Trace-ID to be %s, got %s", expectedTraceID, traceIDHeader)
	}

	select {
	case payload := <-telemetryChan:
		if !strings.Contains(string(payload), expectedTraceID) {
			t.Errorf("expected telemetry payload to contain trace ID %s, got: %s", expectedTraceID, string(payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for telemetry payload")
	}
}
