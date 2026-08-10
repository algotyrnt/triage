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

	"triage/engine/internal/ast"
	"triage/engine/internal/llm"

	"github.com/joho/godotenv"
)

type TelemetryRequest struct {
	APIKey     string `json:"api_key"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	StackTrace string `json:"stack_trace"`
	ASTSnippet string `json:"ast_snippet,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

type TelemetryResponse struct {
	Status       string              `json:"status"`
	TraceID      string              `json:"trace_id,omitempty"`
	AST          string              `json:"ast,omitempty"`
	Analysis     *llm.AnalysisResult `json:"analysis,omitempty"`
	ErrorMessage string              `json:"error,omitempty"`
}

type IndexRequest struct {
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	Commit        string `json:"commit"`
	WorkspacePath string `json:"workspace_path"`
}

var astManager *ast.Manager

func loadEnvLocal() {
	_ = godotenv.Load(".env.local", ".env")
}

func main() {
	loadEnvLocal()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var err error
		astManager, err = ast.NewManager(context.Background(), dbURL)
		if err != nil {
			log.Printf("[WARNING] Failed to connect AST Manager to PostgreSQL: %v", err)
		} else {
			defer astManager.Close()
			log.Println("[AST MANAGER] Connected Engine to PostgreSQL Database Pool")
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", handleHealthRoute)
	http.HandleFunc("/api/v1/telemetry", handleTelemetryRoute)
	http.HandleFunc("/api/v1/ast/index", handleASTIndexRoute)

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
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

func handleASTIndexRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if astManager == nil {
		http.Error(w, "Database AST Manager uninitialized", http.StatusInternalServerError)
		return
	}

	count, err := astManager.IndexRepositoryAST(r.Context(), req.Owner, req.Repo, req.Commit, req.WorkspacePath)
	if err != nil {
		log.Printf("[ERROR] AST indexing failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "error", "error": err.Error()})
		return
	}

	log.Printf("[AST INDEXING COMPLETE] Indexed %d function nodes into PostgreSQL for %s/%s", count, req.Owner, req.Repo)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "indexed_count": count})
}

func isValidAPIKey(key string) bool {
	if key == "" {
		return false
	}
	expected := os.Getenv("TRIAGE_API_KEY")
	if expected == "" {
		// Fail closed if TRIAGE_API_KEY is unset
		return false
	}
	return key == expected
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
	if evalRoot, evalErr := filepath.EvalSymlinks(cleanRoot); evalErr == nil {
		cleanRoot = evalRoot
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

	if evalTarget, evalErr := filepath.EvalSymlinks(targetPath); evalErr == nil {
		targetPath = evalTarget
	}

	rel, err := filepath.Rel(cleanRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("resolved path outside project root: %s", reqFile)
	}
	return targetPath, nil
}

func ExtractASTContext(ctx context.Context, reqFile string, line int) (string, error) {
	if astManager != nil {
		node, err := astManager.GetASTNode(ctx, reqFile, line)
		if err == nil && node != nil && node.Snippet != "" {
			log.Printf("[AST DB HIT] %s:%d", reqFile, line)
			return node.Snippet, nil
		}
		log.Printf("[AST DB MISS] %s:%d - %v", reqFile, line, err)
	}

	resolvedPath, valErr := validateAndResolveFilePath(reqFile)
	if valErr != nil {
		return "", valErr
	}

	return ast.ExtractFuncAST(resolvedPath, line)
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

	traceID := req.TraceID
	if traceID == "" {
		traceID = r.Header.Get("X-Triage-Trace-ID")
	}
	if traceID != "" {
		w.Header().Set("X-Triage-Trace-ID", traceID)
	}

	// Validate API Key before file reads or Gemini calls
	if !isValidAPIKey(req.APIKey) {
		log.Printf("[WARNING] [TRACE %s] Unauthorized telemetry request attempt", traceID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(TelemetryResponse{
			Status:       "error",
			TraceID:      traceID,
			ErrorMessage: "unauthorized: missing or invalid API key",
		})
		return
	}

	log.Printf("==================================================")
	log.Printf("[TELEMETRY RECEIVED] Trace: %s | File: %s | Line: %d", traceID, req.File, req.Line)
	log.Printf("Stack Trace:\n%s", req.StackTrace)

	var astSnippet string
	var astErr error
	if req.ASTSnippet != "" {
		astSnippet = req.ASTSnippet
		log.Printf("[AST SNIPPET PROVIDED IN PAYLOAD]\n%s", astSnippet)
	} else if req.File != "" && req.Line > 0 {
		astSnippet, astErr = ExtractASTContext(r.Context(), req.File, req.Line)
		if astErr != nil {
			log.Printf("[WARNING] [TRACE %s] AST extraction failed: %v", traceID, astErr)
		} else {
			log.Printf("[AST RESOLVED] [TRACE %s] Surrounding Function Node:\n%s", traceID, astSnippet)
		}
	} else {
		log.Printf("[WARNING] [TRACE %s] File or Line not provided in telemetry", traceID)
	}

	// Context for LLM analysis
	analysisCtx, cancelAnalysis := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancelAnalysis()

	analysis, llmErr := llm.AnalyzeCrash(analysisCtx, req.StackTrace, astSnippet)
	if llmErr != nil {
		log.Printf("[ERROR] [TRACE %s] Gemini analysis failed: %v", traceID, llmErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TelemetryResponse{
			Status:       "partial_success",
			TraceID:      traceID,
			AST:          astSnippet,
			ErrorMessage: fmt.Sprintf("LLM analysis error: %v", llmErr),
		})
		return
	}

	log.Printf("[ANALYSIS COMPLETE] [TRACE %s]", traceID)
	log.Printf("  Root Cause: %s", analysis.RootCause)
	log.Printf("  Suggested Fix: %s", analysis.SuggestedFix)
	log.Printf("==================================================")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TelemetryResponse{
		Status:   "success",
		TraceID:  traceID,
		AST:      astSnippet,
		Analysis: analysis,
	})
}
