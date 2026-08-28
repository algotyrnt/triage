// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/llm"
)

type TelemetryRequest struct {
	APIKey       string `json:"api_key"`
	Owner        string `json:"owner,omitempty"`
	Repo         string `json:"repo,omitempty"`
	Commit       string `json:"commit,omitempty"`
	RootDir      string `json:"root_dir,omitempty"`
	RootPath     string `json:"root_path,omitempty"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	PanicMessage string `json:"panic_message,omitempty"`
	StackTrace   string `json:"stack_trace"`
	ASTSnippet   string `json:"ast_snippet,omitempty"`
	TraceID      string `json:"trace_id,omitempty"`
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

	// 2. Pre-indexed database check
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

	// 3. On-demand fetch file source from GitHub App API
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

	return "", fmt.Errorf("could not retrieve AST context for %s:%d from GitHub", reqFile, line)
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
	s.LoadGitHubAppConfig(r.Context())

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

	if projectContext == "" && s.db != nil && req.Owner != "" && req.Repo != "" {
		if proj, err := s.db.GetProjectByOwnerRepo(r.Context(), req.Owner, req.Repo, rootDir); err == nil && proj != nil {
			projectContext = proj.Context
			if repoID == "" {
				repoID = proj.ID
			}
			if installationID == 0 {
				installationID = proj.InstallationID
			}
		}
	}

	if req.Owner != "" && req.Repo != "" {
		if liveID, err := s.ResolveInstallationID(r.Context(), req.Owner, req.Repo); err == nil && liveID > 0 {
			installationID = liveID
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

	llmCfg := s.GetLLMConfig(r.Context())

	var analysis *llm.AnalysisResult
	if astSnippet != "" && (llmCfg.APIKey != "" || llmCfg.Provider == "ollama" || llmCfg.Provider == "custom") {
		analysisCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var err error
		delimitedAST := astSnippet
		if !strings.HasPrefix(delimitedAST, "```") {
			delimitedAST = fmt.Sprintf("```go\n%s\n```", delimitedAST)
		}

		if provider, pErr := llm.NewProvider(llmCfg); pErr == nil {
			analysis, err = provider.AnalyzeCrash(analysisCtx, req.StackTrace, delimitedAST, projectContext)
			if err != nil {
				slog.Warn("LLM root cause analysis skipped", "error", err, "provider", llmCfg.Provider)
			}
		}
	}

	panicSummary := strings.TrimSpace(req.PanicMessage)
	if panicSummary == "" {
		// Parse "panic: ..." from stack trace if present
		for _, line := range strings.Split(req.StackTrace, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "panic:") {
				panicSummary = strings.TrimSpace(strings.TrimPrefix(trimmed, "panic:"))
				break
			}
		}
	}
	if panicSummary == "" {
		// Extract function name if available from top frames
		for _, line := range strings.Split(req.StackTrace, "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "goroutine ") && strings.Contains(trimmed, "(") && !strings.Contains(trimmed, "runtime/") && !strings.Contains(trimmed, "sdk/go") {
				if paren := strings.Index(trimmed, "("); paren != -1 {
					funcName := strings.TrimSpace(trimmed[:paren])
					if funcName != "" {
						panicSummary = fmt.Sprintf("Panic in %s()", funcName)
						break
					}
				}
			}
		}
	}
	if panicSummary == "" {
		panicSummary = fmt.Sprintf("Panic at %s:%d", req.File, req.Line)
	}

	title := panicSummary
	if analysis != nil && analysis.RootCause != "" && strings.HasPrefix(title, "Panic at ") {
		// Use concise AI root cause summary if location was the only title info
		rcFirstLine := strings.Split(strings.TrimSpace(analysis.RootCause), "\n")[0]
		if len(rcFirstLine) > 0 && len(rcFirstLine) <= 120 {
			title = rcFirstLine
		}
	}

	rawFingerprint := fmt.Sprintf("%s:%d:%s", req.File, req.Line, panicSummary)
	fpHash := sha256.Sum256([]byte(rawFingerprint))
	fingerprint := hex.EncodeToString(fpHash[:16])

	var existingIncident *db.Incident
	if s.db != nil {
		existingIncident, _ = s.db.FindActiveIncidentByFingerprint(r.Context(), repoID, fingerprint)
	}

	var gitHubIssue *GitHubIssue
	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}

	incidentID := traceID
	if incidentID == "" {
		if existingIncident != nil {
			incidentID = existingIncident.ID
		} else {
			randBytes := make([]byte, 4)
			_, _ = rand.Read(randBytes)
			incidentID = fmt.Sprintf("INC-%s", strings.ToUpper(hex.EncodeToString(randBytes)))
		}
	}

	// Only file new GitHub Issue if incident doesn't already exist
	if existingIncident == nil && s.githubApp != nil && installationID > 0 && req.Owner != "" && req.Repo != "" {
		panicSummary := req.StackTrace
		if lines := strings.Split(panicSummary, "\n"); len(lines) > 0 {
			panicSummary = strings.TrimSpace(lines[0])
		}
		rootCause := ""
		suggestedFix := ""
		severity := ""
		if analysis != nil {
			rootCause = analysis.RootCause
			suggestedFix = analysis.SuggestedFix
			if analysis.Severity != "" {
				severity = strings.ToUpper(strings.TrimSpace(analysis.Severity))
			}
		}

		issueTitle := fmt.Sprintf("🚨 Panic in %s:%d: %s", req.File, req.Line, panicSummary)

		issueBody := BuildGitHubIssueBody(GitHubIssueMarkdownParams{
			IncidentID:   incidentID,
			Owner:        req.Owner,
			Repo:         req.Repo,
			File:         req.File,
			Line:         req.Line,
			PanicMessage: panicSummary,
			Severity:     severity,
			Status:       "OPEN",
			CreatedAt:    time.Now().UTC(),
			TraceID:      traceID,
			RootCause:    rootCause,
			SuggestedFix: suggestedFix,
			ASTSnippet:   astSnippet,
			StackTrace:   req.StackTrace,
			AppURL:       s.ResolveAppURL(r.Context(), r),
		})

		labels := []string{"panic", "triage", "bug"}
		issueNum, issueURL, err := s.githubApp.CreateIssue(r.Context(), installationID, req.Owner, req.Repo, issueTitle, issueBody, labels)
		if err != nil && strings.Contains(err.Error(), "404") {
			if liveID, rErr := s.ResolveInstallationID(r.Context(), req.Owner, req.Repo); rErr == nil && liveID > 0 && liveID != installationID {
				installationID = liveID
				issueNum, issueURL, err = s.githubApp.CreateIssue(r.Context(), installationID, req.Owner, req.Repo, issueTitle, issueBody, labels)
			}
		}
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
		if existingIncident != nil {
			_ = s.db.IncrementIncidentOccurrence(r.Context(), existingIncident.ID)
			existingIncident.OccurrenceCount++
			existingIncident.LastSeenAt = time.Now().UTC()
			if s.eventBroker != nil {
				s.eventBroker.Publish("incident_updated", existingIncident)
			}
		} else {
			rootCause := ""
			suggestedFix := ""
			severity := ""
			if analysis != nil {
				rootCause = analysis.RootCause
				suggestedFix = analysis.SuggestedFix
				if analysis.Severity != "" {
					severity = strings.ToUpper(strings.TrimSpace(analysis.Severity))
				}
			}

			issueURL := ""
			issueNum := 0
			if gitHubIssue != nil {
				issueURL = gitHubIssue.HTMLURL
				issueNum = gitHubIssue.Number
			}

			now := time.Now().UTC()
			inc := &db.Incident{
				ID:                incidentID,
				RepositoryID:      repoID,
				Fingerprint:       fingerprint,
				OccurrenceCount:   1,
				Title:             title,
				Status:            "OPEN",
				Severity:          severity,
				AIProvider:        llmCfg.Provider,
				AIModel:           llmCfg.Model,
				File:              req.File,
				Line:              req.Line,
				PanicMessage:      title,
				StackTrace:        req.StackTrace,
				ASTSnippet:        astSnippet,
				RootCause:         rootCause,
				SuggestedFix:      suggestedFix,
				GitHubIssueURL:    issueURL,
				GitHubIssueNumber: issueNum,
				CreatedAt:         now,
				LastSeenAt:        now,
			}

			if err := s.db.SaveIncident(r.Context(), inc); err != nil {
				slog.Warn("failed to persist incident to database", "error", err, "incident_id", incidentID)
			}
			if s.eventBroker != nil {
				s.eventBroker.Publish("incident_created", inc)
			}
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
