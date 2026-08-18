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
	if engineURL == "" {
		engineURL = "http://localhost:8080/api/v1/telemetry"
	}

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
		val := 42
		ptr := &val
		fmt.Fprintf(w, "Value: %d\n", *ptr)
	})

	wrappedHandler := triage.Middleware(apiKey, engineURL)(mux)

	log.Printf("Starting test-service on :%s ...", port)
	if err := http.ListenAndServe(":"+port, wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}