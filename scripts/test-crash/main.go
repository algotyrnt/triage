// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"

	"triage/sdk"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Test Crash Server running. Call /crash to trigger panic.")
	})

	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Triggering nil pointer dereference panic...")
		var ptr *int
		*ptr = 42 // Nil pointer dereference panic
	})

	// Wrap server multiplexer in Triage middleware
	wrappedHandler := triage.Middleware("test_key", "http://localhost:8080/api/v1/telemetry")(mux)

	log.Println("Starting test-crash server on :8081 ...")
	if err := http.ListenAndServe(":8081", wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
