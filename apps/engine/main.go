// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/llm"
	"triage/engine/internal/session"

	"github.com/joho/godotenv"
)

// Version is the current build version of the Triage Engine, overridable via ldflags at build time.
var Version = "v0.1.0-dev"

type TelemetryRequest struct {
	APIKey     string `json:"api_key"`
	Owner      string `json:"owner,omitempty"`
	Repo       string `json:"repo,omitempty"`
	Commit     string `json:"commit,omitempty"`
	RootDir    string `json:"root_dir,omitempty"`
	RootPath   string `json:"root_path,omitempty"`
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
	APIKey        string `json:"api_key,omitempty"`
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	Commit        string `json:"commit"`
	WorkspacePath string `json:"workspace_path"`
	RootDir       string `json:"root_dir,omitempty"`
	RootPath      string `json:"root_path,omitempty"`
}

var (
	githubNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)

	database   *db.DB
	astManager *ast.Manager
	astCache   = ast.NewASTCache()
	astFetcher = ast.NewOnDemandFetcher()

	githubApp     *github.AppConfig
	sessionSecret string
	appURL        string
	port          string
	engineURL     string
)

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func loadGitHubAppConfig(ctx context.Context) {
	if database == nil {
		return
	}
	appIDStr, _ := database.GetInstanceConfig(ctx, "github_app_id")
	pemKey, _ := database.GetInstanceConfig(ctx, "github_app_private_key")
	webhookSecret, _ := database.GetInstanceConfig(ctx, "github_app_webhook_secret")
	clientID, _ := database.GetInstanceConfig(ctx, "github_app_client_id")
	clientSecret, _ := database.GetInstanceConfig(ctx, "github_app_client_secret")

	if appIDStr == "" || pemKey == "" {
		return
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		log.Printf("[WARNING] Invalid GITHUB_APP_ID: %v", err)
		return
	}
	cfg, err := github.LoadAppConfig(appID, []byte(pemKey), webhookSecret, clientID, clientSecret)
	if err != nil {
		log.Printf("[WARNING] Failed to load GitHub App config: %v", err)
		return
	}
	githubApp = cfg
	log.Println("[GITHUB APP] Loaded GitHub App configuration")
}

func loadEnvLocal() {
	_ = godotenv.Load(".env.local", ".env")
}

func main() {
	loadEnvLocal()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var err error
		database, err = db.NewDB(context.Background(), dbURL)
		if err != nil {
			log.Printf("[WARNING] Failed to connect Database Pool: %v", err)
		} else {
			defer database.Close()
			log.Println("[DATABASE] Connected Engine directly to PostgreSQL Database Pool")
		}

		astManager, err = ast.NewManager(context.Background(), dbURL)
		if err != nil {
			log.Printf("[WARNING] Failed to connect AST Manager to PostgreSQL: %v", err)
		} else {
			defer astManager.Close()
			log.Println("[AST MANAGER] Connected Engine to PostgreSQL Database Pool")
		}
	}

	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	engineURL = fmt.Sprintf("http://localhost:%s", port)

	if database != nil {
		storedSecret, _ := database.GetInstanceConfig(context.Background(), "session_secret")
		if storedSecret != "" {
			sessionSecret = storedSecret
		} else {
			// Auto-generate a secure random 32-byte secret
			b := make([]byte, 32)
			rand.Read(b)
			sessionSecret = hex.EncodeToString(b)
			err := database.SaveInstanceConfig(context.Background(), "session_secret", sessionSecret)
			if err != nil {
				log.Printf("[WARNING] Failed to persist generated session_secret: %v", err)
			} else {
				log.Println("[AUTH] Auto-generated and persisted secure session_secret")
			}
		}
	} else {
		log.Println("[WARNING] Database not connected, using insecure fallback session secret")
		sessionSecret = "dev-secret-change-me-in-production"
	}

	appURL = os.Getenv("NEXT_PUBLIC_APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}

	if database != nil {
		loadGitHubAppConfig(context.Background())
	}

	// Core routes
	http.HandleFunc("/health", corsMiddleware(handleHealthRoute))
	http.HandleFunc("/api/v1/telemetry", corsMiddleware(handleTelemetryRoute))
	http.HandleFunc("/api/v1/ast/index", corsMiddleware(handleASTIndexRoute))
	http.HandleFunc("/api/v1/incidents", corsMiddleware(handleIncidentsRoute))
	http.HandleFunc("/api/v1/projects", corsMiddleware(handleProjectsRoute))
	http.HandleFunc("/api/v1/stats", corsMiddleware(handleStatsRoute))
	http.HandleFunc("/api/v1/repos/detect-modules", corsMiddleware(handleDetectModulesRoute))

	// Setup wizard routes (first-run configuration)
	http.HandleFunc("/api/v1/setup/status", corsMiddleware(handleSetupStatus))
	http.HandleFunc("/api/v1/setup/manifest", corsMiddleware(handleSetupManifest))
	http.HandleFunc("/api/v1/setup/callback", corsMiddleware(handleSetupCallback))
	http.HandleFunc("/api/v1/setup/install", corsMiddleware(handleSetupInstall))
	http.HandleFunc("/api/v1/setup/install/callback", corsMiddleware(handleSetupInstallCallback))
	http.HandleFunc("/api/v1/setup/oauth", corsMiddleware(handleSetupOAuth))
	http.HandleFunc("/api/v1/setup/llm", corsMiddleware(handleSetupLLMRoute))
	http.HandleFunc("/api/v1/setup/test", corsMiddleware(handleSetupTest))
	http.HandleFunc("/api/v1/setup/repos", corsMiddleware(handleSetupRepos))
	http.HandleFunc("/api/v1/setup/check-repo", corsMiddleware(handleCheckRepoRoute))

	// Post-setup settings routes
	http.HandleFunc("/api/v1/settings/llm", corsMiddleware(handleSettingsLLMRoute))

	// GitHub OAuth authentication routes
	http.HandleFunc("/api/v1/auth/github", corsMiddleware(handleGitHubAuthRoute))
	http.HandleFunc("/api/v1/auth/github/callback", corsMiddleware(handleGitHubCallbackRoute))
	http.HandleFunc("/api/v1/auth/me", corsMiddleware(handleAuthMe))

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

	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = r.Header.Get("X-Triage-API-Key")
	}
	if !isValidAPIKey(r.Context(), apiKey) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "error", "error": "unauthorized: missing or invalid API key"})
		return
	}

	if astManager == nil {
		http.Error(w, "Database AST Manager uninitialized", http.StatusInternalServerError)
		return
	}

	workspacePath := req.WorkspacePath
	if workspacePath != "" {
		resolvedPath, valErr := validateAndResolveFilePath(workspacePath)
		if valErr != nil {
			log.Printf("[WARNING] WorkspacePath validation failed: %v", valErr)
			workspacePath = os.Getenv("AST_WORKSPACE_ROOT")
		} else {
			workspacePath = resolvedPath
		}
	}
	if workspacePath == "" {
		workspacePath = os.Getenv("AST_WORKSPACE_ROOT")
	}

	rootDir := req.RootDir
	if rootDir == "" {
		rootDir = req.RootPath
	}

	count, err := astManager.IndexRepositoryAST(r.Context(), req.Owner, req.Repo, req.Commit, workspacePath, rootDir)
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

func generateIncidentID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func isValidAPIKey(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}
	if database != nil && database.VerifyAPIKey(ctx, key) {
		return true
	}
	expected := os.Getenv("TRIAGE_API_KEY")
	if expected == "" {
		// Fail closed if TRIAGE_API_KEY is unset
		return false
	}
	return subtle.ConstantTimeCompare([]byte(key), []byte(expected)) == 1
}

func isValidTraceID(id string) bool {
	if id == "" {
		return true
	}
	if len(id) > 64 {
		return false
	}
	for _, ch := range id {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

func validateAndResolveFilePath(reqFile string, rootDir ...string) (string, error) {
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

	rd := ""
	if len(rootDir) > 0 {
		rd = rootDir[0]
	}
	normReq := ast.NormalizeMonorepoPath(reqFile, rd)

	candidatePaths := []string{normReq}
	if normReq != reqFile {
		candidatePaths = append(candidatePaths, reqFile)
	}

	for _, cand := range candidatePaths {
		cleanCand := filepath.Clean(cand)
		var targetPath string

		if filepath.IsAbs(cleanCand) {
			rel, relErr := filepath.Rel(cleanRoot, cleanCand)
			if relErr == nil && !strings.HasPrefix(rel, "..") {
				targetPath = filepath.Join(cleanRoot, rel)
			} else {
				parts := strings.Split(filepath.ToSlash(cleanCand), "/")
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
					continue
				}
			}
		} else {
			targetPath = filepath.Join(cleanRoot, cleanCand)
		}

		if evalTarget, evalErr := filepath.EvalSymlinks(targetPath); evalErr == nil {
			targetPath = evalTarget
		}

		rel, err := filepath.Rel(cleanRoot, targetPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			if _, statErr := os.Stat(targetPath); statErr == nil {
				return targetPath, nil
			}
		}
	}

	// Default fallback to first candidate target path
	targetPath := filepath.Join(cleanRoot, filepath.Clean(normReq))
	if evalTarget, evalErr := filepath.EvalSymlinks(targetPath); evalErr == nil {
		targetPath = evalTarget
	}
	rel, err := filepath.Rel(cleanRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("resolved path outside project root: %s", reqFile)
	}
	return targetPath, nil
}

func ExtractASTContext(ctx context.Context, owner, repo, commit, reqFile string, line int, rootDir ...string) (string, error) {
	rd := ""
	if len(rootDir) > 0 {
		rd = rootDir[0]
	}

	normPath := ast.NormalizeMonorepoPath(reqFile, rd)

	// 1. In-memory KV Cache check (< 2ms)
	if snippet, found := astCache.Get(owner, repo, commit, normPath, line); found {
		log.Printf("[AST CACHE HIT] %s/%s@%s %s:%d", owner, repo, commit, normPath, line)
		return snippet, nil
	}
	if normPath != reqFile {
		if snippet, found := astCache.Get(owner, repo, commit, reqFile, line); found {
			log.Printf("[AST CACHE HIT] %s/%s@%s %s:%d", owner, repo, commit, reqFile, line)
			return snippet, nil
		}
	}

	// 2. Pre-indexed PostgreSQL check fallback
	if astManager != nil {
		node, err := astManager.GetASTNode(ctx, owner, repo, commit, reqFile, line, rd)
		if err == nil && node != nil && node.Snippet != "" {
			log.Printf("[AST DB HIT] %s/%s@%s %s:%d", owner, repo, commit, normPath, line)
			astCache.Set(owner, repo, commit, normPath, line, node.Snippet)
			return node.Snippet, nil
		}
		log.Printf("[AST DB MISS] %s/%s@%s %s:%d - %v", owner, repo, commit, normPath, line, err)
	}

	// 3. On-demand fetch file source from GitHub or Local Workspace
	content, fetchErr := astFetcher.FetchFile(ctx, owner, repo, commit, reqFile, rd)
	if fetchErr == nil && len(content) > 0 {
		snippet, parseErr := ast.ExtractFuncASTFromBytes(content, line)
		if parseErr == nil && snippet != "" {
			log.Printf("[AST ON-DEMAND FETCHED] Extracted AST snippet for %s/%s@%s %s:%d", owner, repo, commit, normPath, line)
			astCache.Set(owner, repo, commit, normPath, line, snippet)
			return snippet, nil
		}
		log.Printf("[WARNING] On-demand AST byte parsing failed: %v", parseErr)
	} else {
		log.Printf("[WARNING] On-demand file fetch failed: %v", fetchErr)
	}

	// 4. Local workspace fallback
	resolvedPath, valErr := validateAndResolveFilePath(reqFile, rd)
	if valErr != nil {
		return "", valErr
	}

	snippet, err := ast.ExtractFuncAST(resolvedPath, line)
	if err == nil && snippet != "" {
		astCache.Set(owner, repo, commit, normPath, line, snippet)
	}
	return snippet, err
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
	if !isValidTraceID(traceID) {
		log.Printf("[WARNING] Invalid trace ID format received: %s, ignoring", traceID)
		traceID = ""
	}
	if traceID != "" {
		w.Header().Set("X-Triage-Trace-ID", traceID)
	}

	// Validate API Key before file reads or Gemini calls
	if !isValidAPIKey(r.Context(), req.APIKey) {
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

	rootDir := req.RootDir
	if rootDir == "" {
		rootDir = req.RootPath
	}

	// Auto-lookup repository from DB if API key belongs to a registered project
	if database != nil && req.APIKey != "" {
		if repoRecord, err := database.GetRepositoryByAPIKey(r.Context(), req.APIKey); err == nil && repoRecord != nil {
			if req.Owner == "" {
				req.Owner = repoRecord.Owner
			}
			if req.Repo == "" {
				req.Repo = repoRecord.Repo
			}
			if rootDir == "" {
				rootDir = repoRecord.RootDir
			}
		}
	}

	var astSnippet string
	var astErr error
	if req.ASTSnippet != "" {
		if len(req.ASTSnippet) > 8192 {
			astSnippet = req.ASTSnippet[:8192] + "\n... [truncated]"
		} else {
			astSnippet = req.ASTSnippet
		}
		log.Printf("[AST SNIPPET PROVIDED IN PAYLOAD]\n%s", astSnippet)
	} else if req.File != "" && req.Line > 0 {
		astSnippet, astErr = ExtractASTContext(r.Context(), req.Owner, req.Repo, req.Commit, req.File, req.Line, rootDir)
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

	delimitedSnippet := fmt.Sprintf("```go\n%s\n```", astSnippet)

	llmAPIKey := ""
	llmModelName := ""
	if database != nil {
		llmAPIKey, _ = database.GetInstanceConfig(r.Context(), "gemini_api_key")
		llmModelName, _ = database.GetInstanceConfig(r.Context(), "gemini_model")
	}

	analysis, llmErr := llm.AnalyzeCrash(analysisCtx, req.StackTrace, delimitedSnippet, llmAPIKey, llmModelName)
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

	// Save Incident to PostgreSQL database if connected
	if database != nil && analysis != nil {
		randID, idErr := generateIncidentID()
		if idErr != nil {
			log.Printf("[ERROR] Failed to generate incident ID: %v", idErr)
		} else {
			incidentID := fmt.Sprintf("INC-%s", randID)
			panicMsg := "Runtime panic"
			if req.StackTrace != "" {
				lines := strings.Split(req.StackTrace, "\n")
				if len(lines) > 0 {
					panicMsg = lines[0]
				}
			}

			_ = database.SaveIncident(r.Context(), &db.Incident{
				ID:           incidentID,
				Title:        fmt.Sprintf("Panic in %s:%d", req.File, req.Line),
				Status:       "CRITICAL",
				File:         req.File,
				Line:         req.Line,
				PanicMessage: panicMsg,
				StackTrace:   req.StackTrace,
				ASTSnippet:   astSnippet,
				RootCause:    analysis.RootCause,
				SuggestedFix: analysis.SuggestedFix,
			})
			log.Printf("[ENGINE DB SAVE] Persisted Incident %s in PostgreSQL", incidentID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TelemetryResponse{
		Status:   "success",
		TraceID:  traceID,
		AST:      astSnippet,
		Analysis: analysis,
	})
}

func handleIncidentsRoute(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"incidents": []db.Incident{}})
		return
	}

	incidents, err := database.GetIncidents(r.Context(), 50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch incidents: %v", err), http.StatusInternalServerError)
		return
	}
	if incidents == nil {
		incidents = []db.Incident{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"incidents": incidents,
	})
}

func handleProjectsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Repo          string `json:"repo"`
			Owner         string `json:"owner"`
			RootDir       string `json:"root_dir"`
			ServicePath   string `json:"service_path"`
			OwnerUsername string `json:"owner_username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}
		if req.Repo == "" {
			http.Error(w, "Field 'repo' is required", http.StatusBadRequest)
			return
		}
		owner := req.Owner
		repoName := req.Repo
		if strings.Contains(req.Repo, "/") {
			parts := strings.Split(req.Repo, "/")
			owner = parts[0]
			repoName = parts[1]
		}
		if owner == "" {
			owner = "algotyrnt"
		}
		rootDir := req.RootDir
		if rootDir == "" {
			rootDir = req.ServicePath
		}
		rootDir = strings.Trim(strings.TrimSpace(rootDir), "/")

		apiKey := fmt.Sprintf("tr_live_%s_%d", repoName, time.Now().UnixNano())
		if database != nil {
			k, _, err := database.CreateProject(r.Context(), owner, repoName, rootDir, req.OwnerUsername)
			if err == nil && k != "" {
				apiKey = k
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"repo":     fmt.Sprintf("%s/%s", owner, repoName),
			"root_dir": rootDir,
			"api_key":  apiKey,
		})
		return
	}

	if database == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"projects": []db.Repository{}})
		return
	}

	projects, err := database.GetProjects(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch projects: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"projects": projects})
}

func handleStatsRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := map[string]interface{}{
		"status":          "healthy",
		"engine_version":  Version,
		"total_incidents": 0,
		"funcs_indexed":   1420,
	}

	if database != nil {
		s, err := database.GetStats(r.Context(), Version)
		if err == nil {
			stats = s
		}
	}

	_ = json.NewEncoder(w).Encode(stats)
}

func handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"configured":   false,
		"github_app":   false,
		"installation": false,
		"oauth":        false,
		"llm":          false,
	}

	if database != nil {
		appID, _ := database.GetInstanceConfig(r.Context(), "github_app_id")
		result["github_app"] = appID != ""

		inst, _ := database.GetInstallation(r.Context())
		result["installation"] = inst != nil

		oauthID, _ := database.GetInstanceConfig(r.Context(), "github_oauth_client_id")
		result["oauth"] = oauthID != ""

		apiKey, _ := database.GetInstanceConfig(r.Context(), "gemini_api_key")
		modelName, _ := database.GetInstanceConfig(r.Context(), "gemini_model")
		result["llm"] = apiKey != "" && modelName != ""

		if appID != "" && inst != nil && oauthID != "" && apiKey != "" && modelName != "" {
			result["configured"] = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func handleSetupManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InstanceURL string `json:"instance_url"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.InstanceURL == "" {
		req.InstanceURL = appURL
	}

	manifest := map[string]interface{}{
		"name":         "Triage",
		"url":          req.InstanceURL,
		"redirect_url": engineURL + "/api/v1/setup/callback",
		"setup_url":    engineURL + "/api/v1/setup/install/callback",
		"callback_urls": []string{
			engineURL + "/api/v1/auth/github/callback",
		},
		"public": false,
		"default_permissions": map[string]string{
			"contents": "read",
			"issues":   "write",
			"metadata": "read",
		},
	}

	if !strings.Contains(engineURL, "localhost") {
		manifest["hook_attributes"] = map[string]string{
			"url": engineURL + "/api/v1/webhooks/github",
		}
		manifest["default_events"] = []string{"push", "installation", "installation_repositories"}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"manifest": manifest,
		"url":      "https://github.com/settings/apps/new",
	})
}

func handleSetupCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	convURL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, convURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Manifest conversion failed: %v", err)
		http.Redirect(w, r, appURL+"?setup_error=conversion_failed", http.StatusFound)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[ERROR] Manifest conversion returned %d: %s", resp.StatusCode, string(body))
		http.Redirect(w, r, appURL+"?setup_error=conversion_failed", http.StatusFound)
		return
	}

	var result struct {
		ID            int64  `json:"id"`
		Slug          string `json:"slug"`
		PEM           string `json:"pem"`
		WebhookSecret string `json:"webhook_secret"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[ERROR] Failed to decode manifest response: %v", err)
		http.Redirect(w, r, appURL+"?setup_error=decode_failed", http.StatusFound)
		return
	}

	if database != nil {
		database.SaveInstanceConfig(r.Context(), "github_app_id", strconv.FormatInt(result.ID, 10))
		database.SaveInstanceConfig(r.Context(), "github_app_slug", result.Slug)
		database.SaveInstanceConfig(r.Context(), "github_app_private_key", result.PEM)
		database.SaveInstanceConfig(r.Context(), "github_app_webhook_secret", result.WebhookSecret)
		database.SaveInstanceConfig(r.Context(), "github_app_client_id", result.ClientID)
		database.SaveInstanceConfig(r.Context(), "github_app_client_secret", result.ClientSecret)
		database.SaveInstanceConfig(r.Context(), "github_oauth_client_id", result.ClientID)
		database.SaveInstanceConfig(r.Context(), "github_oauth_client_secret", result.ClientSecret)
	}

	loadGitHubAppConfig(r.Context())

	log.Printf("[SETUP] GitHub App created: ID=%d, Slug=%s", result.ID, result.Slug)
	http.Redirect(w, r, appURL+"?setup_step=2&app_created=true", http.StatusFound)
}

func handleSetupInstall(w http.ResponseWriter, r *http.Request) {
	slug := ""
	if database != nil {
		slug, _ = database.GetInstanceConfig(r.Context(), "github_app_slug")
	}
	if slug == "" {
		http.Error(w, "GitHub App not configured yet", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"url": fmt.Sprintf("https://github.com/apps/%s/installations/new", slug),
	})
}

func handleSetupInstallCallback(w http.ResponseWriter, r *http.Request) {
	installIDStr := r.URL.Query().Get("installation_id")
	if installIDStr == "" {
		http.Redirect(w, r, appURL+"?setup_error=missing_installation_id", http.StatusFound)
		return
	}

	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, appURL+"?setup_error=invalid_installation_id", http.StatusFound)
		return
	}

	if githubApp == nil {
		loadGitHubAppConfig(r.Context())
	}

	if githubApp != nil && database != nil {
		jwt, err := githubApp.SignAppJWT()
		if err == nil {
			client := &http.Client{Timeout: 15 * time.Second}
			reqURL := fmt.Sprintf("https://api.github.com/app/installations/%d", installID)
			req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Accept", "application/vnd.github+json")

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var instData struct {
					Account struct {
						Login string `json:"login"`
						ID    int64  `json:"id"`
						Type  string `json:"type"`
					} `json:"account"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&instData)
				resp.Body.Close()

				database.SaveInstallation(r.Context(), installID, instData.Account.Login, instData.Account.ID, instData.Account.Type)

				// List all installation repositories using pagination
				repoInfos, repoErr := githubApp.ListInstallationRepositories(r.Context(), installID)
				if repoErr == nil {
					var repos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						repos = append(repos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
					}
					database.SaveInstallationRepos(r.Context(), installID, repos)
					log.Printf("[SETUP] Stored %d repos for installation %d (@%s)", len(repos), installID, instData.Account.Login)
				} else {
					log.Printf("[SETUP ERROR] Failed to list installation repos: %v", repoErr)
				}

				log.Printf("[SETUP] Installation saved: ID=%d, Account=%s", installID, instData.Account.Login)
			}
		}
	}

	http.Redirect(w, r, appURL+"?setup_step=3&installed=true", http.StatusFound)
}

func handleSetupOAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.ClientID == "" || req.ClientSecret == "" {
		http.Error(w, "client_id and client_secret are required", http.StatusBadRequest)
		return
	}
	if database != nil {
		database.SaveInstanceConfig(r.Context(), "github_oauth_client_id", req.ClientID)
		database.SaveInstanceConfig(r.Context(), "github_oauth_client_secret", req.ClientSecret)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleSetupTest(w http.ResponseWriter, r *http.Request) {
	if githubApp == nil {
		loadGitHubAppConfig(r.Context())
	}
	if githubApp == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "GitHub App not configured"})
		return
	}

	err := githubApp.VerifyApp(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	appName := ""
	if database != nil {
		appName, _ = database.GetInstanceConfig(r.Context(), "github_app_slug")
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "app_name": appName})
}

type SetupRepoItem struct {
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	Name          string `json:"name"`
	Installed     bool   `json:"installed"`
	DefaultBranch string `json:"branch"`
	Language      string `json:"lang"`
	Visibility    string `json:"visibility"`
}

func handleSetupRepos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if database == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"repos": []SetupRepoItem{}})
		return
	}

	if githubApp == nil {
		loadGitHubAppConfig(r.Context())
	}

	// 1. Live-sync all active installations and their repositories if githubApp is configured
	if githubApp != nil {
		if appInstalls, err := githubApp.ListAppInstallations(r.Context()); err == nil {
			for _, inst := range appInstalls {
				_ = database.SaveInstallation(r.Context(), inst.ID, inst.AccountLogin, inst.AccountID, inst.AccountType)
				if repoInfos, rErr := githubApp.ListInstallationRepositories(r.Context(), inst.ID); rErr == nil {
					var repos []db.InstallationRepo
					for _, rInfo := range repoInfos {
						repos = append(repos, db.InstallationRepo{Owner: rInfo.Owner, Repo: rInfo.Repo})
					}
					_ = database.SaveInstallationRepos(r.Context(), inst.ID, repos)
				}
			}
		}
	}

	// 2. Fetch all installed repos from database
	installedRepos, err := database.GetAllInstallationRepos(r.Context())
	if err != nil {
		installedRepos = []db.InstallationRepo{}
	}

	installedMap := make(map[string]bool)
	for _, ir := range installedRepos {
		key := strings.ToLower(fmt.Sprintf("%s/%s", ir.Owner, ir.Repo))
		installedMap[key] = true
	}

	// 3. Extract username from query param or auth session
	username := r.URL.Query().Get("username")
	if username == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") && sessionSecret != "" {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, cErr := session.ValidateSessionJWT(tokenStr, sessionSecret); cErr == nil && claims != nil {
				username = claims.Username
			}
		}
	}

	repoItemsMap := make(map[string]SetupRepoItem)

	// Fetch user's repositories from GitHub if username is specified
	if username != "" {
		if userRepos, uErr := github.FetchUserRepositories(r.Context(), username); uErr == nil {
			for _, ur := range userRepos {
				key := strings.ToLower(fmt.Sprintf("%s/%s", ur.Owner, ur.Repo))
				isInstalled := installedMap[key]
				branch := ur.DefaultBranch
				if branch == "" {
					branch = "main"
				}
				lang := ur.Language
				if lang == "" {
					lang = "Go"
				}
				vis := "Public"
				if ur.Private {
					vis = "Private"
				}
				if isInstalled {
					vis = "Installed"
				}
				repoItemsMap[key] = SetupRepoItem{
					Owner:         ur.Owner,
					Repo:          ur.Repo,
					Name:          fmt.Sprintf("%s/%s", ur.Owner, ur.Repo),
					Installed:     isInstalled,
					DefaultBranch: branch,
					Language:      lang,
					Visibility:    vis,
				}
			}
		}
	}

	// Add all installed repositories from database
	for _, ir := range installedRepos {
		key := strings.ToLower(fmt.Sprintf("%s/%s", ir.Owner, ir.Repo))
		if existing, ok := repoItemsMap[key]; ok {
			existing.Installed = true
			existing.Visibility = "Installed"
			repoItemsMap[key] = existing
		} else {
			repoItemsMap[key] = SetupRepoItem{
				Owner:         ir.Owner,
				Repo:          ir.Repo,
				Name:          fmt.Sprintf("%s/%s", ir.Owner, ir.Repo),
				Installed:     true,
				DefaultBranch: "main",
				Language:      "Go / TS",
				Visibility:    "Installed",
			}
		}
	}

	// Sort repos: Installed first, then alphabetically
	var sortedRepos []SetupRepoItem
	for _, item := range repoItemsMap {
		sortedRepos = append(sortedRepos, item)
	}
	sort.Slice(sortedRepos, func(i, j int) bool {
		if sortedRepos[i].Installed != sortedRepos[j].Installed {
			return sortedRepos[i].Installed
		}
		return sortedRepos[i].Name < sortedRepos[j].Name
	})

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"repos": sortedRepos,
	})
}

func handleCheckRepoRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner = parts[0]
			repo = parts[1]
		}
	}

	if owner == "" || repo == "" {
		http.Error(w, "owner and repo parameters are required", http.StatusBadRequest)
		return
	}

	if database == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"installed": false})
		return
	}

	instID, err := database.GetInstallationForRepo(r.Context(), owner, repo)
	installed := err == nil && instID > 0

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"installed":       installed,
		"installation_id": instID,
		"owner":           owner,
		"repo":            repo,
	})
}

func getGitHubOAuthCredentials(ctx context.Context) (clientID, clientSecret string) {
	if database != nil {
		clientID, _ = database.GetInstanceConfig(ctx, "github_oauth_client_id")
		clientSecret, _ = database.GetInstanceConfig(ctx, "github_oauth_client_secret")
	}
	if v := os.Getenv("GITHUB_OAUTH_CLIENT_ID"); v != "" {
		clientID = v
	} else if clientID == "" {
		clientID = os.Getenv("GITHUB_CLIENT_ID")
	}
	if v := os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"); v != "" {
		clientSecret = v
	} else if clientSecret == "" {
		clientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	}
	return clientID, clientSecret
}

func handleGitHubAuthRoute(w http.ResponseWriter, r *http.Request) {
	clientID, _ := getGitHubOAuthCredentials(r.Context())
	if clientID == "" {
		http.Redirect(w, r, appURL+"?user=algotyrnt&auth=dev", http.StatusFound)
		return
	}

	callbackURL := fmt.Sprintf("%s/api/v1/auth/github/callback", engineURL)
	redirectURI := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email,read:user,read:org", clientID, callbackURL)
	http.Redirect(w, r, redirectURI, http.StatusFound)
}

func handleGitHubCallbackRoute(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, appURL+"?auth=error&reason=missing_code", http.StatusFound)
		return
	}

	clientID, clientSecret := getGitHubOAuthCredentials(r.Context())
	if clientID == "" || clientSecret == "" {
		http.Redirect(w, r, appURL+"?auth=error&reason=oauth_not_configured", http.StatusFound)
		return
	}

	tokenURL := "https://github.com/login/oauth/access_token"
	tokenBody := fmt.Sprintf(`{"client_id":"%s","client_secret":"%s","code":"%s"}`, clientID, clientSecret, code)
	tokenReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenURL, strings.NewReader(tokenBody))
	tokenReq.Header.Set("Accept", "application/json")
	tokenReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		log.Printf("[AUTH ERROR] Token exchange failed: %v", err)
		http.Redirect(w, r, appURL+"?auth=error&reason=token_exchange_failed", http.StatusFound)
		return
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	_ = json.NewDecoder(tokenResp.Body).Decode(&tokenData)
	if tokenData.AccessToken == "" {
		log.Printf("[AUTH ERROR] No access token returned: %s", tokenData.Error)
		http.Redirect(w, r, appURL+"?auth=error&reason=no_access_token", http.StatusFound)
		return
	}

	userReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := client.Do(userReq)
	if err != nil {
		log.Printf("[AUTH ERROR] User fetch failed: %v", err)
		http.Redirect(w, r, appURL+"?auth=error&reason=user_fetch_failed", http.StatusFound)
		return
	}
	defer userResp.Body.Close()

	var userData struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	_ = json.NewDecoder(userResp.Body).Decode(&userData)
	if userData.Login == "" {
		http.Redirect(w, r, appURL+"?auth=error&reason=invalid_user", http.StatusFound)
		return
	}

	githubIDStr := strconv.FormatInt(userData.ID, 10)
	if database != nil {
		database.UpsertUser(r.Context(), githubIDStr, userData.Login, userData.AvatarURL)
	}

	userID := fmt.Sprintf("usr_%s", githubIDStr)
	token, err := session.MintSessionJWT(userID, userData.Login, userData.AvatarURL, githubIDStr, sessionSecret)
	if err != nil {
		log.Printf("[AUTH ERROR] JWT minting failed: %v", err)
		http.Redirect(w, r, appURL+"?auth=error&reason=jwt_error", http.StatusFound)
		return
	}

	log.Printf("[AUTH] User authenticated: @%s (ID: %d)", userData.Login, userData.ID)
	http.Redirect(w, r, fmt.Sprintf("%s?token=%s&auth=success", appURL, token), http.StatusFound)
}

func handleAuthMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid authorization header"})
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := session.ValidateSessionJWT(tokenStr, sessionSecret)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired session"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         claims.UserID,
		"username":   claims.Username,
		"avatar_url": claims.AvatarURL,
		"github_id":  claims.GitHubID,
	})
}

func handleSettingsLLMRoute(w http.ResponseWriter, r *http.Request) {
	// Authenticate
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	_, err := session.ValidateSessionJWT(tokenStr, sessionSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		api, _ := database.GetInstanceConfig(r.Context(), "gemini_api_key")
		model, _ := database.GetInstanceConfig(r.Context(), "gemini_model")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"gemini_api_key": api,
			"gemini_model":   model,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			GeminiAPIKey string `json:"gemini_api_key"`
			GeminiModel  string `json:"gemini_model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.GeminiAPIKey != "" {
			database.SaveInstanceConfig(r.Context(), "gemini_api_key", req.GeminiAPIKey)
		}
		if req.GeminiModel != "" {
			database.SaveInstanceConfig(r.Context(), "gemini_model", req.GeminiModel)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func handleSetupLLMRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GeminiAPIKey string `json:"gemini_api_key"`
		GeminiModel  string `json:"gemini_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.GeminiAPIKey != "" {
		database.SaveInstanceConfig(r.Context(), "gemini_api_key", req.GeminiAPIKey)
	}
	if req.GeminiModel != "" {
		database.SaveInstanceConfig(r.Context(), "gemini_model", req.GeminiModel)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

type DetectedModule struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	IsRoot bool   `json:"is_root"`
}

func handleDetectModulesRoute(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Triage-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if !isValidAPIKey(r.Context(), apiKey) {
		if sessionSecret != "" && apiKey != "" {
			if _, err := session.ValidateSessionJWT(apiKey, sessionSecret); err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner = parts[0]
			repo = parts[1]
		}
	}

	if owner == "" || repo == "" || !githubNamePattern.MatchString(owner) || !githubNamePattern.MatchString(repo) {
		http.Error(w, "Invalid owner or repo parameter", http.StatusBadRequest)
		return
	}

	modules := []DetectedModule{
		{Path: "", Name: "Repository Root (/)", IsRoot: true},
	}
	seen := map[string]bool{"": true}

	// 1. Try detecting via GitHub App if configured
	if githubApp != nil && database != nil {
		installID, err := database.GetInstallationForRepo(r.Context(), owner, repo)
		if err == nil && installID > 0 {
			token, tokenErr := githubApp.GetInstallationToken(r.Context(), installID)
			if tokenErr == nil {
				// Query GitHub repo tree recursively
				treesURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo)
				treeReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, treesURL, nil)
				treeReq.Header.Set("Authorization", "Bearer "+token)
				treeReq.Header.Set("Accept", "application/vnd.github+json")

				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Do(treeReq)
				if err == nil && resp != nil {
					if resp.StatusCode == http.StatusOK {
						var treeData struct {
							Tree []struct {
								Path string `json:"path"`
								Type string `json:"type"`
							} `json:"tree"`
						}
						if jsonErr := json.NewDecoder(resp.Body).Decode(&treeData); jsonErr == nil {
							for _, item := range treeData.Tree {
								if item.Type == "blob" && strings.HasSuffix(item.Path, "go.mod") {
									dir := filepath.ToSlash(filepath.Dir(item.Path))
									if dir == "." {
										dir = ""
									}
									if !seen[dir] {
										seen[dir] = true
										displayName := fmt.Sprintf("%s/ (Go Module)", dir)
										modules = append(modules, DetectedModule{
											Path:   dir,
											Name:   displayName,
											IsRoot: false,
										})
									}
								}
							}
						}
					}
					_ = resp.Body.Close()
				}
			}
		}
	}

	// 2. Scan local workspace if present
	wsRoot := os.Getenv("AST_WORKSPACE_ROOT")
	if wsRoot == "" {
		wsRoot = "."
	}
	_ = filepath.Walk(wsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path == wsRoot {
				return nil
			}
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "go.mod" {
			rel, relErr := filepath.Rel(wsRoot, path)
			if relErr == nil {
				dir := filepath.ToSlash(filepath.Dir(rel))
				if dir == "." {
					dir = ""
				}
				if !seen[dir] {
					seen[dir] = true
					displayName := fmt.Sprintf("%s/ (Go Module)", dir)
					modules = append(modules, DetectedModule{
						Path:   dir,
						Name:   displayName,
						IsRoot: false,
					})
				}
			}
		}
		return nil
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"modules": modules,
	})
}
