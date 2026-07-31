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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"triage/engine/internal/ast"
	"triage/engine/internal/llm"
)

type TelemetryRequest struct {
	APIKey     string `json:"api_key"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	StackTrace string `json:"stack_trace"`
}

type TelemetryResponse struct {
	Status       string              `json:"status"`
	AST          string              `json:"ast,omitempty"`
	Analysis     *llm.AnalysisResult `json:"analysis,omitempty"`
	ErrorMessage string              `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/v1/telemetry", handleTelemetryRoute)

	log.Printf("Triage Engine server listening on :%s ...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

func handleTelemetryRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	handleTelemetry(w, r)
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	var req TelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Invalid request payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TelemetryResponse{
			Status:       "error",
			ErrorMessage: "invalid JSON body",
		})
		return
	}

	log.Printf("==================================================")
	log.Printf("[TELEMETRY RECEIVED] API Key: %s | File: %s | Line: %d", req.APIKey, req.File, req.Line)
	log.Printf("Stack Trace:\n%s", req.StackTrace)

	var astSnippet string
	var astErr error
	if req.File != "" && req.Line > 0 {
		astSnippet, astErr = ast.ExtractFuncAST(req.File, req.Line)
		if astErr != nil {
			log.Printf("[WARNING] AST extraction failed: %v", astErr)
		} else {
			log.Printf("[AST EXTRACTED] Surrounding Function Node:\n%s", astSnippet)
		}
	} else {
		log.Printf("[WARNING] File or Line not provided in telemetry")
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	analysis, llmErr := llm.AnalyzeCrash(ctx, req.StackTrace, astSnippet)
	if llmErr != nil {
		log.Printf("[ERROR] Gemini analysis failed: %v", llmErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TelemetryResponse{
			Status:       "partial_success",
			AST:          astSnippet,
			ErrorMessage: fmt.Sprintf("LLM analysis error: %v", llmErr),
		})
		return
	}

	log.Printf("[ANALYSIS COMPLETE]")
	log.Printf("  Root Cause: %s", analysis.RootCause)
	log.Printf("  Suggested Fix: %s", analysis.SuggestedFix)
	log.Printf("==================================================")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TelemetryResponse{
		Status:   "success",
		AST:      astSnippet,
		Analysis: analysis,
	})
}
