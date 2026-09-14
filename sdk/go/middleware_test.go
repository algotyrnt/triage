// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package triage

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockHTTPClient(fn func(req *http.Request) (*http.Response, error)) func() {
	origClient := telemetryHTTPClient
	SetTelemetryHTTPClient(&http.Client{
		Transport: roundTripperFunc(fn),
	})
	return func() {
		SetTelemetryHTTPClient(origClient)
	}
}

func TestMiddlewarePanicRecovery(t *testing.T) {
	telemetryChan := make(chan []byte, 1)
	cleanup := mockHTTPClient(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
			Header:     make(http.Header),
		}, nil
	})
	defer cleanup()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated test crash")
	})

	mw := Middleware("test_key", "http://telemetry.mock/api/v1/telemetry")
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
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for async telemetry payload to arrive at telemetry engine")
	}
}

func TestMiddlewareSelfHosted(t *testing.T) {
	telemetryChan := make(chan []byte, 1)
	cleanup := mockHTTPClient(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}, nil
	})
	defer cleanup()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated self-hosted crash")
	})

	mw := Middleware("test_selfhosted_key", "http://custom-gateway.local/v1/telemetry")
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
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for telemetry payload on custom Gateway URL")
	}
}

func TestMiddlewarePreservesInboundTraceparent(t *testing.T) {
	telemetryChan := make(chan []byte, 1)
	cleanup := mockHTTPClient(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}, nil
	})
	defer cleanup()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("traceparent panic test")
	})

	mw := Middleware("test_key", "http://telemetry.mock/api/v1/telemetry")
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
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for telemetry payload")
	}
}

func TestMiddlewareNormalRequestPassThrough(t *testing.T) {
	mw := Middleware("test_key", "http://mock")
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Errorf("expected underlying handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
}

func TestMiddlewareInboundXTriageTraceID(t *testing.T) {
	telemetryChan := make(chan []byte, 1)
	cleanup := mockHTTPClient(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		telemetryChan <- body
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`ok`)),
			Header:     make(http.Header),
		}, nil
	})
	defer cleanup()

	mw := Middleware("test_key", "http://mock")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("crash with triage header")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	customTrace := "my-custom-trace-id-12345"
	req.Header.Set("X-Triage-Trace-ID", customTrace)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Triage-Trace-ID") != customTrace {
		t.Errorf("expected trace header %s, got %s", customTrace, rec.Header().Get("X-Triage-Trace-ID"))
	}

	select {
	case payload := <-telemetryChan:
		if !strings.Contains(string(payload), customTrace) {
			t.Errorf("expected payload to contain %s, got %s", customTrace, string(payload))
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for telemetry")
	}
}

func TestGetTelemetryMetrics(t *testing.T) {
	queued, processed, dropped := GetTelemetryMetrics()
	if queued < processed {
		t.Logf("metrics: queued=%d, processed=%d, dropped=%d", queued, processed, dropped)
	}
}

func TestIsValidTraceID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"", false},
		{"abc123DEF", true},
		{"trace-id_1.0", true},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
		{"invalid char space", false},
		{"invalid$char", false},
	}

	for _, tt := range tests {
		got := isValidTraceID(tt.id)
		if got != tt.valid {
			t.Errorf("isValidTraceID(%q) = %v; want %v", tt.id, got, tt.valid)
		}
	}
}

func TestParseTopApplicationFrame(t *testing.T) {
	trace := `goroutine 1 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x65
net/http.HandlerFunc.ServeHTTP(0x1234)
	/usr/local/go/src/net/http/server.go:2084 +0x2f
github.com/algotyrnt/triage/sdk/go.Middleware.func1.1()
	/path/sdk/go/middleware.go:150 +0x85
main.crashHandler()
	/app/handlers/crash.go:42 +0x1a
`
	file, line := parseTopApplicationFrame(trace)
	if file != "/app/handlers/crash.go" || line != 42 {
		t.Errorf("parseTopApplicationFrame() = (%s, %d); want (/app/handlers/crash.go, 42)", file, line)
	}

	fileEmpty, lineEmpty := parseTopApplicationFrame("")
	if fileEmpty != "" || lineEmpty != 0 {
		t.Errorf("expected empty result for empty trace, got %s:%d", fileEmpty, lineEmpty)
	}
}

func TestSetTelemetryHTTPClient(t *testing.T) {
	orig := telemetryHTTPClient
	SetTelemetryHTTPClient(nil)
	if telemetryHTTPClient != orig {
		t.Errorf("nil client should not override")
	}

	custom := &http.Client{}
	SetTelemetryHTTPClient(custom)
	if telemetryHTTPClient != custom {
		t.Errorf("custom client was not set")
	}
	SetTelemetryHTTPClient(orig)
}

func TestSendTelemetry_ErrorHandling(t *testing.T) {
	cleanup := mockHTTPClient(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("network error")
	})
	defer cleanup()

	sendTelemetry("http://invalid.mock", "key", "rev", "main.go", 10, "panic", "stack", "trace-id")
	sendTelemetry(":%invalid-url", "key", "rev", "main.go", 10, "panic", "stack", "trace-id")
}
