// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SSEMessage represents a payload streamed to dashboard clients.
type SSEMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// EventBroker manages active SSE client subscribers and event fan-out.
type EventBroker struct {
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
	stop       chan struct{}
	mu         sync.RWMutex
}

// NewEventBroker initializes and starts a new EventBroker.
func NewEventBroker() *EventBroker {
	b := &EventBroker{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 256),
		stop:       make(chan struct{}),
	}
	go b.run()
	return b
}

func (b *EventBroker) run() {
	for {
		select {
		case <-b.stop:
			b.mu.Lock()
			for ch := range b.clients {
				close(ch)
				delete(b.clients, ch)
			}
			b.mu.Unlock()
			return

		case ch := <-b.register:
			b.mu.Lock()
			b.clients[ch] = true
			b.mu.Unlock()
			slog.Debug("SSE client connected", "active_clients", b.ClientCount())

		case ch := <-b.unregister:
			b.mu.Lock()
			if _, exists := b.clients[ch]; exists {
				delete(b.clients, ch)
				close(ch)
			}
			b.mu.Unlock()
			slog.Debug("SSE client disconnected", "active_clients", b.ClientCount())

		case msg := <-b.broadcast:
			b.mu.RLock()
			for ch := range b.clients {
				select {
				case ch <- msg:
				default:
					slog.Warn("dropping SSE message for slow client")
				}
			}
			b.mu.RUnlock()
		}
	}
}

// Close stops the broker and cleans up all active client channels.
func (b *EventBroker) Close() {
	select {
	case <-b.stop:
	default:
		close(b.stop)
	}
}

// ClientCount returns the number of currently active SSE subscribers.
func (b *EventBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Publish broadcasts an event with the given type and data to all connected clients.
func (b *EventBroker) Publish(eventType string, data interface{}) {
	msg := SSEMessage{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to serialize SSE event payload", "error", err, "type", eventType)
		return
	}

	select {
	case b.broadcast <- payload:
	default:
		slog.Warn("SSE broadcast buffer full, dropping message", "type", eventType)
	}
}

// HandleEventsStream handles incoming SSE connections from the dashboard.
func (s *Server) HandleEventsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported by response writer", http.StatusInternalServerError)
		return
	}

	if s.eventBroker == nil {
		http.Error(w, "Event broker unavailable", http.StatusServiceUnavailable)
		return
	}

	// Disable write deadline for long-lived streaming connection if supported
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	// Set required SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	clientChan := make(chan []byte, 32)
	s.eventBroker.register <- clientChan
	defer func() {
		s.eventBroker.unregister <- clientChan
	}()

	// Send initial handshake message
	initMsg := SSEMessage{
		Type:      "connected",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if initBytes, err := json.Marshal(initMsg); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", string(initBytes))
		flusher.Flush()
	}

	// Heartbeat ticker to prevent proxy or load balancer timeouts
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-ticker.C:
			// SSE comment as keep-alive heartbeat
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", string(msg)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
