// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"auth-service/pkg/auth"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func requireEnv(key string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		log.Fatalf("Fatal: missing required environment variable %q", key)
	}
	return val
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func main() {
	apiKey := requireEnv("TRIAGE_API_KEY")
	engineURL := getEnv("TRIAGE_ENGINE_URL", "http://localhost:8080/api/v1/telemetry")
	port := getEnv("PORT", "8084")

	manager := auth.NewSessionManager("auth.corp.local")

	mux := http.NewServeMux()

	// Landing status endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service": "Auth & Identity Service", "port": "%s", "endpoints": ["/auth/profile-nil-claims", "/auth/token-slice-bounds", "/auth/closed-channel"]}`+"\n", port)
	})

	// 1. Crash Route 1: Deep Nested Nil Pointer Dereference (session.Claims.User.Profile.Email)
	mux.HandleFunc("/auth/profile-nil-claims", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating user profile extraction on session with nil claims...")
		session := &auth.Session{
			SessionID: "sess_9901_abc",
			Claims:    nil, // Intentionally nil
			IPAddress: "127.0.0.1",
		}

		email, err := manager.GetUserEmail(session)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"email": email})
	})

	// 2. Crash Route 2: Slice Bounds Out of Range (authHeader[7:] when header is short)
	mux.HandleFunc("/auth/token-slice-bounds", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating token extraction from short header...")
		shortHeader := "JWT" // length is 3, which causes slice panic authHeader[7:]
		token := auth.ExtractBearerToken(shortHeader)
		fmt.Fprintf(w, "Extracted token: %s\n", token)
	})

	// 3. Crash Route 3: Panic on Send to Closed Channel
	mux.HandleFunc("/auth/closed-channel", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating broadcast to closed channel...")
		ch := make(chan string, 1)
		close(ch) // Closed channel
		manager.BroadcastAuditLog(ch, "USER_LOGIN_EVENT")
	})

	wrappedHandler := triage.Middleware(apiKey, engineURL)(mux)

	log.Printf("Starting auth-service on :%s ...", port)
	if err := http.ListenAndServe(":"+port, wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
