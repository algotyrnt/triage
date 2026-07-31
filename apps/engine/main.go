// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

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
	"triage/engine/internal/github"
	"triage/engine/internal/llm"
)

type TelemetryRequest struct {
	APIKey         string `json:"api_key"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	StackTrace     string `json:"stack_trace"`
	GithubOwner    string `json:"github_owner,omitempty"`
	GithubRepo     string `json:"github_repo,omitempty"`
	InstallationID int64  `json:"installation_id,omitempty"`
}

type TelemetryResponse struct {
	Status       string                `json:"status"`
	AST          string                `json:"ast,omitempty"`
	Analysis     *llm.AnalysisResult   `json:"analysis,omitempty"`
	GithubIssue  *github.IssueResponse `json:"github_issue,omitempty"`
	ErrorMessage string                `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")

	http.HandleFunc("/api/v1/telemetry", handleTelemetryRoute)
	http.HandleFunc("/api/v1/github/webhook", github.WebhookHandler(webhookSecret))

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

	var githubIssueResp *github.IssueResponse
	appID := os.Getenv("GITHUB_APP_ID")
	privateKeyPem := os.Getenv("GITHUB_PRIVATE_KEY")

	if appID != "" && privateKeyPem != "" && req.GithubOwner != "" && req.GithubRepo != "" && req.InstallationID > 0 {
		ghClient := github.NewClient(appID, []byte(privateKeyPem))
		issueReq := &github.IssueRequest{
			File:       req.File,
			Line:       req.Line,
			StackTrace: req.StackTrace,
			ASTSnippet: astSnippet,
			Analysis:   analysis,
		}

		issue, err := ghClient.CreateIssue(ctx, req.GithubOwner, req.GithubRepo, req.InstallationID, issueReq)
		if err != nil {
			log.Printf("[WARNING] Failed to post GitHub issue: %v", err)
		} else {
			githubIssueResp = issue
			log.Printf("[GITHUB ISSUE CREATED] #%d: %s", issue.Number, issue.HTMLURL)
		}
	}

	log.Printf("==================================================")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TelemetryResponse{
		Status:      "success",
		AST:         astSnippet,
		Analysis:    analysis,
		GithubIssue: githubIssueResp,
	})
}
