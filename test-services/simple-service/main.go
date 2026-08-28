// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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
	port := getEnv("PORT", "8081")

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Simple Test Service running. Call /crash to trigger a panic.")
	})

	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Triggering nil pointer dereference panic...")
		val := 42
		ptr := &val
		_ = ptr
	})

	wrappedHandler := triage.Middleware(apiKey, engineURL)(mux)

	log.Printf("Starting simple-service on :%s ...", port)
	if err := http.ListenAndServe(":"+port, wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
