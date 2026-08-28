// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEventBroker_PublishAndSubscribe(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	ch1 := make(chan []byte, 10)
	ch2 := make(chan []byte, 10)

	broker.register <- ch1
	broker.register <- ch2

	// Wait for registration
	time.Sleep(50 * time.Millisecond)
	if count := broker.ClientCount(); count != 2 {
		t.Fatalf("expected 2 clients, got %d", count)
	}

	testData := map[string]string{"file": "main.go", "line": "42"}
	broker.Publish("test_event", testData)

	select {
	case msg := <-ch1:
		var sseMsg SSEMessage
		if err := json.Unmarshal(msg, &sseMsg); err != nil {
			t.Fatalf("failed to unmarshal msg on ch1: %v", err)
		}
		if sseMsg.Type != "test_event" {
			t.Fatalf("expected test_event, got %s", sseMsg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for message on ch1")
	}

	select {
	case msg := <-ch2:
		var sseMsg SSEMessage
		if err := json.Unmarshal(msg, &sseMsg); err != nil {
			t.Fatalf("failed to unmarshal msg on ch2: %v", err)
		}
		if sseMsg.Type != "test_event" {
			t.Fatalf("expected test_event, got %s", sseMsg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for message on ch2")
	}

	broker.unregister <- ch1
	time.Sleep(50 * time.Millisecond)
	if count := broker.ClientCount(); count != 1 {
		t.Fatalf("expected 1 client after unregister, got %d", count)
	}
}

type sseResponseWriter struct {
	header http.Header
	writer *io.PipeWriter
	code   int
}

func (w *sseResponseWriter) Header() http.Header {
	return w.header
}

func (w *sseResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *sseResponseWriter) WriteHeader(code int) {
	w.code = code
}

func (w *sseResponseWriter) Flush() {}

func TestHandleEventsStream(t *testing.T) {
	broker := NewEventBroker()
	defer broker.Close()

	server := NewServer(Config{
		EventBroker: broker,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/events/stream", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	rw := &sseResponseWriter{
		header: make(http.Header),
		writer: pw,
	}

	go func() {
		server.HandleEventsStream(rw, req)
		pw.Close()
	}()

	reader := bufio.NewReader(pr)

	// Read initial connected event
	var firstEventLine string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("error reading stream: %v", err)
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			firstEventLine = strings.TrimPrefix(line, "data:")
			firstEventLine = strings.TrimSpace(firstEventLine)
			break
		}
	}

	var initMsg SSEMessage
	if err := json.Unmarshal([]byte(firstEventLine), &initMsg); err != nil {
		t.Fatalf("failed to parse init event: %v (raw: %s)", err, firstEventLine)
	}
	if initMsg.Type != "connected" {
		t.Fatalf("expected type connected, got %s", initMsg.Type)
	}

	// Publish an incident_created event
	broker.Publish("incident_created", map[string]string{
		"id":    "INC-TEST-01",
		"title": "nil pointer dereference",
	})

	// Read incident_created event
	var incidentLine string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("error reading stream: %v", err)
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			incidentLine = strings.TrimPrefix(line, "data:")
			incidentLine = strings.TrimSpace(incidentLine)
			break
		}
	}

	var incMsg SSEMessage
	if err := json.Unmarshal([]byte(incidentLine), &incMsg); err != nil {
		t.Fatalf("failed to parse incident event: %v (raw: %s)", err, incidentLine)
	}
	if incMsg.Type != "incident_created" {
		t.Fatalf("expected type incident_created, got %s", incMsg.Type)
	}

	if contentType := rw.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %s", contentType)
	}

	// Cancel context to test disconnect
	cancel()
	time.Sleep(50 * time.Millisecond)
}
