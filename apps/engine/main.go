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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
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
	_ = godotenv.Load(".env.local", ".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")

	http.HandleFunc("/health", handleHealthRoute)
	http.HandleFunc("/api/v1/telemetry", handleTelemetryRoute)
	http.HandleFunc("/api/v1/github/webhook", github.WebhookHandler(webhookSecret))

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Triage Engine server listening on :%s ...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server stopped unexpectedly: %v", err)
		}
	}()

	<-stopCtx.Done()
	log.Println("Received termination signal. Shutting down Triage Engine gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Engine server forced shutdown error: %v", err)
	} else {
		log.Println("Engine server exited cleanly.")
	}
}

func handleHealthRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","component":"triage-engine","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

func handleTelemetryRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	handleTelemetry(w, r)
}

func isValidAPIKey(key string) bool {
	if key == "" {
		return false
	}
	expected := os.Getenv("TRIAGE_API_KEY")
	if expected != "" {
		return key == expected
	}
	return true
}

func validateAndResolveFilePath(reqFile string) (string, error) {
	if reqFile == "" {
		return "", fmt.Errorf("file path is empty")
	}
	if strings.Contains(reqFile, "..") {
		return "", fmt.Errorf("path traversal segments '..' are not allowed: %s", reqFile)
	}

	root := os.Getenv("AST_WORKSPACE_ROOT")
	if root == "" {
		root = "."
	}
	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		cleanRoot = filepath.Clean(root)
	}

	cleanReq := filepath.Clean(reqFile)
	var targetPath string

	if filepath.IsAbs(cleanReq) {
		rel, relErr := filepath.Rel(cleanRoot, cleanReq)
		if relErr == nil && !strings.HasPrefix(rel, "..") {
			targetPath = filepath.Join(cleanRoot, rel)
		} else {
			parts := strings.Split(filepath.ToSlash(cleanReq), "/")
			found := false
			for i := 0; i < len(parts); i++ {
				subPath := filepath.Join(parts[i:]...)
				candidate := filepath.Join(cleanRoot, subPath)
				if _, statErr := os.Stat(candidate); statErr == nil {
					targetPath = candidate
					found = true
					break
				}
			}
			if !found {
				return "", fmt.Errorf("absolute path cannot be mapped to workspace root: %s", reqFile)
			}
		}
	} else {
		targetPath = filepath.Join(cleanRoot, cleanReq)
	}

	rel, err := filepath.Rel(cleanRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("resolved path outside project root: %s", reqFile)
	}
	return targetPath, nil
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req TelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Invalid request payload: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TelemetryResponse{
			Status:       "error",
			ErrorMessage: "invalid JSON body or body size limit exceeded",
		})
		return
	}

	// Validate API Key before file reads, Gemini calls, or issue creation
	if !isValidAPIKey(req.APIKey) {
		log.Printf("[WARNING] Unauthorized telemetry request attempt")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(TelemetryResponse{
			Status:       "error",
			ErrorMessage: "unauthorized: missing or invalid API key",
		})
		return
	}

	log.Printf("==================================================")
	log.Printf("[TELEMETRY RECEIVED] File: %s | Line: %d", req.File, req.Line)
	log.Printf("Stack Trace:\n%s", req.StackTrace)

	var astSnippet string
	var astErr error
	if req.File != "" && req.Line > 0 {
		resolvedPath, valErr := validateAndResolveFilePath(req.File)
		if valErr != nil {
			log.Printf("[WARNING] AST extraction file validation failed: %v", valErr)
		} else {
			astSnippet, astErr = ast.ExtractFuncAST(resolvedPath, req.Line)
			if astErr != nil {
				log.Printf("[WARNING] AST extraction failed: %v", astErr)
			} else {
				log.Printf("[AST EXTRACTED] Surrounding Function Node:\n%s", astSnippet)
			}
		}
	} else {
		log.Printf("[WARNING] File or Line not provided in telemetry")
	}

	// Separate context for LLM analysis
	analysisCtx, cancelAnalysis := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancelAnalysis()

	analysis, llmErr := llm.AnalyzeCrash(analysisCtx, req.StackTrace, astSnippet)
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

		// Dedicated context for GitHub issue creation
		ghCtx, cancelGH := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancelGH()

		issue, err := ghClient.CreateIssue(ghCtx, req.GithubOwner, req.GithubRepo, req.InstallationID, issueReq)
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
