// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	triage "github.com/algotyrnt/triage/sdk/go"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env.local", ".env")

	apiKey := os.Getenv("TRIAGE_API_KEY")
	if apiKey == "" {
		apiKey = "tr_test_key_9042"
	}

	engineURL := os.Getenv("TRIAGE_ENGINE_URL")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Test Service running. Call /crash to trigger panic.")
	})

	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Triggering nil pointer dereference panic...")
		var ptr *int
		*ptr = 42 // Nil pointer dereference panic
	})

	// Wrap server multiplexer in triage middleware with WithGatewayURL option for local testing
	var wrappedHandler http.Handler
	if engineURL != "" {
		wrappedHandler = triage.Middleware(apiKey, triage.WithGatewayURL(engineURL))(mux)
	} else {
		wrappedHandler = triage.Middleware(apiKey)(mux)
	}

	log.Printf("Starting test-service on :%s ...", port)
	if err := http.ListenAndServe(":"+port, wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
