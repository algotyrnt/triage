// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/llm"
)

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

type GitHubIssue struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
	Title   string `json:"title"`
}

type TelemetryResponse struct {
	Status       string              `json:"status"`
	TraceID      string              `json:"trace_id,omitempty"`
	AST          string              `json:"ast,omitempty"`
	Analysis     *llm.AnalysisResult `json:"analysis,omitempty"`
	GitHubIssue  *GitHubIssue        `json:"github_issue,omitempty"`
	ErrorMessage string              `json:"error,omitempty"`
}

var traceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

func isValidTraceID(traceID string) bool {
	if len(traceID) < 8 || len(traceID) > 64 {
		return false
	}
	return traceIDPattern.MatchString(traceID)
}

func detectWorkspaceRoot() string {
	if root := os.Getenv("TRIAGE_WORKSPACE_ROOT"); root != "" {
		return root
	}
	if root := os.Getenv("AST_WORKSPACE_ROOT"); root != "" {
		return root
	}

	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	curr := dir
	for {
		if _, err := os.Stat(filepath.Join(curr, ".git")); err == nil {
			return curr
		}
		if _, err := os.Stat(filepath.Join(curr, "go.work")); err == nil {
			return curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return dir
}

func (s *Server) ValidateAndResolveFilePath(reqFile string, rootDir ...string) (string, error) {
	if reqFile == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	rd := ""
	if len(rootDir) > 0 {
		rd = rootDir[0]
	}

	projectRoot := detectWorkspaceRoot()
	cleanRoot := filepath.Clean(projectRoot)
	if evalRoot, err := filepath.EvalSymlinks(cleanRoot); err == nil {
		cleanRoot = evalRoot
	}

	// Case 1: Absolute path on host machine
	if filepath.IsAbs(reqFile) {
		cleanAbs := filepath.Clean(reqFile)
		if evalAbs, err := filepath.EvalSymlinks(cleanAbs); err == nil {
			cleanAbs = evalAbs
		}
		rel, err := filepath.Rel(cleanRoot, cleanAbs)
		if err == nil && !strings.HasPrefix(rel, "..") {
			if _, statErr := os.Stat(cleanAbs); statErr == nil {
				return cleanAbs, nil
			}
		}
	}

	// Case 2: Normalized monorepo or relative candidate paths
	normReq := ast.NormalizeMonorepoPath(reqFile, rd)
	candidateRelPaths := []string{}
	if rd != "" && !strings.HasPrefix(reqFile, rd) && !filepath.IsAbs(reqFile) {
		candidateRelPaths = append(candidateRelPaths, filepath.Join(rd, reqFile))
	}
	candidateRelPaths = append(candidateRelPaths, normReq)
	if normReq != reqFile && !filepath.IsAbs(reqFile) {
		candidateRelPaths = append(candidateRelPaths, reqFile)
	}

	for _, relPath := range candidateRelPaths {
		targetPath := filepath.Join(cleanRoot, filepath.Clean(relPath))
		if evalTarget, err := filepath.EvalSymlinks(targetPath); err == nil {
			targetPath = evalTarget
		}
		rel, err := filepath.Rel(cleanRoot, targetPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			if _, statErr := os.Stat(targetPath); statErr == nil {
				return targetPath, nil
			}
		}
	}

	targetPath := filepath.Join(cleanRoot, filepath.Clean(normReq))
	if evalTarget, err := filepath.EvalSymlinks(targetPath); err == nil {
		targetPath = evalTarget
	}
	rel, err := filepath.Rel(cleanRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("resolved path outside project root: %s", reqFile)
	}
	return targetPath, nil
}

func (s *Server) ExtractASTContext(ctx context.Context, owner, repo, commit, reqFile string, line int, rootDir ...string) (string, error) {
	rd := ""
	if len(rootDir) > 0 {
		rd = rootDir[0]
	}

	normPath := ast.NormalizeMonorepoPath(reqFile, rd)

	// 1. In-memory KV Cache check (< 2ms)
	if s.astCache != nil {
		if snippet, found := s.astCache.Get(owner, repo, commit, normPath, line); found {
			slog.Debug("AST cache hit", "owner", owner, "repo", repo, "commit", commit, "path", normPath, "line", line)
			return snippet, nil
		}
		if normPath != reqFile {
			if snippet, found := s.astCache.Get(owner, repo, commit, reqFile, line); found {
				slog.Debug("AST cache hit", "owner", owner, "repo", repo, "commit", commit, "path", reqFile, "line", line)
				return snippet, nil
			}
		}
	}

	// 2. Pre-indexed PostgreSQL check fallback
	if s.astManager != nil {
		node, err := s.astManager.GetASTNode(ctx, owner, repo, commit, reqFile, line, rd)
		if err == nil && node != nil && node.Snippet != "" {
			slog.Debug("AST DB hit", "owner", owner, "repo", repo, "commit", commit, "path", normPath, "line", line)
			if s.astCache != nil {
				s.astCache.Set(owner, repo, commit, normPath, line, node.Snippet)
			}
			return node.Snippet, nil
		}
	}

	// 3. On-demand fetch file source from GitHub or Local Workspace
	if s.astFetcher != nil {
		content, fetchErr := s.astFetcher.FetchFile(ctx, owner, repo, commit, reqFile, rd)
		if fetchErr == nil && len(content) > 0 {
			snippet, parseErr := ast.ExtractFuncASTFromBytes(content, line)
			if parseErr == nil && snippet != "" {
				slog.Debug("AST on-demand fetched and extracted", "owner", owner, "repo", repo, "commit", commit, "path", normPath, "line", line)
				if s.astCache != nil {
					s.astCache.Set(owner, repo, commit, normPath, line, snippet)
				}
				return snippet, nil
			}
		}
	}

	// 4. Local workspace fallback
	resolvedPath, valErr := s.ValidateAndResolveFilePath(reqFile, rd)
	if valErr != nil {
		return "", valErr
	}

	snippet, err := ast.ExtractFuncAST(resolvedPath, line)
	if err == nil && snippet != "" && s.astCache != nil {
		s.astCache.Set(owner, repo, commit, normPath, line, snippet)
	}
	return snippet, err
}

func (s *Server) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req TelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("invalid telemetry JSON payload", "error", err)
		writeJSON(w, http.StatusBadRequest, TelemetryResponse{
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
		traceID = ""
	}
	if traceID != "" {
		w.Header().Set("X-Triage-Trace-ID", traceID)
	}

	if !s.IsValidAPIKey(r.Context(), req.APIKey) {
		slog.Warn("unauthorized telemetry request attempt", "trace_id", traceID, "remote_addr", r.RemoteAddr)
		writeJSON(w, http.StatusUnauthorized, TelemetryResponse{
			Status:       "error",
			TraceID:      traceID,
			ErrorMessage: "unauthorized: missing or invalid API key",
		})
		return
	}

	slog.Info("telemetry received", "trace_id", traceID, "file", req.File, "line", req.Line)

	rootDir := req.RootDir
	if rootDir == "" {
		rootDir = req.RootPath
	}

	var repoID string
	var installationID int64
	var projectContext string

	if s.db != nil && req.APIKey != "" {
		if repoRecord, err := s.db.GetRepositoryByAPIKey(r.Context(), req.APIKey); err == nil && repoRecord != nil {
			repoID = repoRecord.ID
			projectContext = repoRecord.Context
			if req.Owner == "" {
				req.Owner = repoRecord.Owner
			}
			if req.Repo == "" {
				req.Repo = repoRecord.Repo
			}
			if rootDir == "" {
				rootDir = repoRecord.RootDir
			}
			if installationID == 0 {
				installationID = repoRecord.InstallationID
			}
		}
	}

	if strings.Contains(req.Repo, "/") && req.Owner == "" {
		parts := strings.Split(req.Repo, "/")
		req.Owner = parts[0]
		req.Repo = parts[1]
	}
	if req.Owner == "" && s.db != nil {
		if projects, err := s.db.GetProjects(r.Context()); err == nil && len(projects) > 0 {
			req.Owner = projects[0].Owner
			req.Repo = projects[0].Repo
			if projectContext == "" {
				projectContext = projects[0].Context
			}
			if rootDir == "" {
				rootDir = projects[0].RootDir
			}
			if installationID == 0 {
				installationID = projects[0].InstallationID
			}
		}
	}

	if projectContext == "" && s.db != nil && req.Owner != "" && req.Repo != "" {
		if proj, err := s.db.GetProjectByOwnerRepo(r.Context(), req.Owner, req.Repo, rootDir); err == nil && proj != nil {
			projectContext = proj.Context
		}
	}

	if (installationID == 0 || installationID == 1001) && req.Owner != "" && req.Repo != "" && s.db != nil {
		if instID, err := s.db.GetInstallationForRepo(r.Context(), req.Owner, req.Repo); err == nil && instID > 0 {
			installationID = instID
		}
	}
	if (installationID == 0 || installationID == 1001) && s.db != nil {
		if inst, err := s.db.GetInstallation(r.Context()); err == nil && inst != nil {
			installationID = inst.InstallationID
		}
	}
	if (installationID == 0 || installationID == 1001) && s.githubApp != nil {
		if appInstalls, err := s.githubApp.ListAppInstallations(r.Context()); err == nil && len(appInstalls) > 0 {
			installationID = appInstalls[0].ID
		}
	}

	astSnippet := req.ASTSnippet
	if astSnippet == "" {
		if snippet, err := s.ExtractASTContext(r.Context(), req.Owner, req.Repo, req.Commit, req.File, req.Line, rootDir); err == nil {
			astSnippet = snippet
		} else {
			slog.Warn("could not isolate function AST node", "error", err, "file", req.File, "line", req.Line)
		}
	}

	llmAPIKey, llmModelName := s.GetLLMConfig(r.Context())

	var analysis *llm.AnalysisResult
	if astSnippet != "" && llmAPIKey != "" && llmModelName != "" {
		analysisCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var err error
		delimitedAST := astSnippet
		if !strings.HasPrefix(delimitedAST, "```") {
			delimitedAST = fmt.Sprintf("```go\n%s\n```", delimitedAST)
		}

		analysis, err = llm.AnalyzeCrash(analysisCtx, req.StackTrace, delimitedAST, llmAPIKey, llmModelName, projectContext)
		if err != nil {
			slog.Warn("Gemini root cause analysis skipped", "error", err)
		}
	}

	var gitHubIssue *GitHubIssue
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	incidentID := traceID
	if incidentID == "" {
		randBytes := make([]byte, 4)
		_, _ = rand.Read(randBytes)
		incidentID = fmt.Sprintf("INC-%s", strings.ToUpper(hex.EncodeToString(randBytes)))
	}

	if s.githubApp != nil && installationID > 0 && req.Owner != "" && req.Repo != "" {
		panicSummary := req.StackTrace
		if lines := strings.Split(panicSummary, "\n"); len(lines) > 0 {
			panicSummary = strings.TrimSpace(lines[0])
		}
		issueTitle := fmt.Sprintf("🚨 Panic in %s:%d: %s", req.File, req.Line, panicSummary)

		var issueBody strings.Builder
		issueBody.WriteString("## Runtime Go Panic Detected\n\n")
		issueBody.WriteString(fmt.Sprintf("A runtime panic was intercepted by **Triage** in `%s:%d`.\n\n", req.File, req.Line))
		issueBody.WriteString("---\n\n### Panic Details\n")
		if traceID != "" {
			issueBody.WriteString(fmt.Sprintf("- **Trace ID**: `%s`\n", traceID))
		}
		issueBody.WriteString(fmt.Sprintf("- **File**: `%s:%d`\n", req.File, req.Line))
		issueBody.WriteString(fmt.Sprintf("- **Timestamp**: `%s UTC`\n\n", time.Now().UTC().Format("2006-01-02 15:04:05")))

		if analysis != nil && analysis.RootCause != "" {
			issueBody.WriteString("---\n\n### Root Cause Analysis (Gemini AI)\n")
			issueBody.WriteString(fmt.Sprintf("%s\n\n", analysis.RootCause))
		}
		appURL := s.ResolveAppURL(r.Context(), r)
		if analysis != nil && analysis.SuggestedFix != "" {
			issueBody.WriteString("---\n\n### Recommended Fix\n")
			issueBody.WriteString(fmt.Sprintf("%s\n\n", analysis.SuggestedFix))
			if appURL != "" {
				fixURL := fmt.Sprintf("%s/?incident=%s", appURL, incidentID)
				issueBody.WriteString(fmt.Sprintf("- [ ] [**Generate Fix (PR)**](%s)\n\n", fixURL))
			} else {
				issueBody.WriteString("- [ ] **Generate Fix (PR)**\n\n")
			}
		}
		if astSnippet != "" {
			issueBody.WriteString("---\n\n### AST Context\n```go\n")
			issueBody.WriteString(fmt.Sprintf("%s\n```\n\n", astSnippet))
		}
		if req.StackTrace != "" {
			issueBody.WriteString("---\n\n### Stack Trace\n```\n")
			issueBody.WriteString(fmt.Sprintf("%s\n```\n\n", req.StackTrace))
		}
		if appURL != "" {
			issueBody.WriteString(fmt.Sprintf("---\n*Automatically created by [triage](%s)*\n", appURL))
		} else {
			issueBody.WriteString("---\n*Automatically created by triage*\n")
		}

		labels := []string{"panic", "triage", "bug"}
		issueNum, issueURL, err := s.githubApp.CreateIssue(r.Context(), installationID, req.Owner, req.Repo, issueTitle, issueBody.String(), labels)
		if err == nil {
			gitHubIssue = &GitHubIssue{
				Number:  issueNum,
				HTMLURL: issueURL,
				State:   "open",
				Title:   issueTitle,
			}
			slog.Info("auto-created GitHub issue", "issue_number", issueNum, "url", issueURL)
		} else {
			slog.Warn("failed to auto-create GitHub issue", "error", err)
		}
	}

	if s.db != nil {

		title := req.StackTrace
		if lines := strings.Split(title, "\n"); len(lines) > 0 {
			title = strings.TrimSpace(lines[0])
		}
		if title == "" {
			title = fmt.Sprintf("Panic at %s:%d", req.File, req.Line)
		}

		rootCause := ""
		suggestedFix := ""
		if analysis != nil {
			rootCause = analysis.RootCause
			suggestedFix = analysis.SuggestedFix
		}

		issueURL := ""
		issueNum := 0
		if gitHubIssue != nil {
			issueURL = gitHubIssue.HTMLURL
			issueNum = gitHubIssue.Number
		}

		inc := &db.Incident{
			ID:                incidentID,
			RepositoryID:      repoID,
			Title:             title,
			Status:            "CRITICAL",
			File:              req.File,
			Line:              req.Line,
			PanicMessage:      title,
			StackTrace:        req.StackTrace,
			ASTSnippet:        astSnippet,
			RootCause:         rootCause,
			SuggestedFix:      suggestedFix,
			GitHubIssueURL:    issueURL,
			GitHubIssueNumber: issueNum,
			CreatedAt:         time.Now().UTC(),
		}

		if err := s.db.SaveIncident(r.Context(), inc); err != nil {
			slog.Warn("failed to persist incident to database", "error", err, "incident_id", incidentID)
		}
		if s.eventBroker != nil {
			s.eventBroker.Publish("incident_created", inc)
		}
	}

	writeJSON(w, http.StatusOK, TelemetryResponse{
		Status:      "success",
		TraceID:     traceID,
		AST:         astSnippet,
		Analysis:    analysis,
		GitHubIssue: gitHubIssue,
	})
}
