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
	"path/filepath"
	"strings"
	"time"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/llm"
)

// ValidatePatchTargetFile enforces repository safety policies on modified files.
func ValidatePatchTargetFile(filePath string) error {
	clean := filepath.Clean(strings.TrimSpace(filePath))
	if clean == "" || clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("invalid target file path: %s", filePath)
	}

	lower := strings.ToLower(filepath.ToSlash(clean))
	disallowedPrefixes := []string{
		".github/",
		".git/",
		".vscode/",
		".idea/",
	}
	for _, pref := range disallowedPrefixes {
		if strings.HasPrefix(lower, pref) {
			return fmt.Errorf("modifying files in %s is prohibited by security policy", pref)
		}
	}

	disallowedSubstrings := []string{
		"dockerfile",
		"docker-compose",
		".env",
		"credential",
		"secret",
		"id_rsa",
		"id_ed25519",
	}
	base := filepath.Base(lower)
	for _, sub := range disallowedSubstrings {
		if strings.Contains(base, sub) {
			return fmt.Errorf("modifying %s is prohibited by security policy", base)
		}
	}

	ext := filepath.Ext(lower)
	if ext == ".pem" || ext == ".key" || ext == ".pfx" || ext == ".p12" {
		return fmt.Errorf("modifying certificate or private key files is prohibited")
	}

	return nil
}

func (s *Server) HandleIncidents(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"incidents": []db.Incident{}})
		return
	}

	repoQuery := r.URL.Query().Get("repo")
	repoIDQuery := r.URL.Query().Get("repository_id")

	incidents, err := s.db.GetIncidents(r.Context(), 100)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch incidents: %v", err), http.StatusInternalServerError)
		return
	}

	if repoIDQuery != "" {
		filtered := make([]db.Incident, 0)
		for _, inc := range incidents {
			if inc.RepositoryID == repoIDQuery {
				filtered = append(filtered, inc)
			}
		}
		incidents = filtered
	} else if repoQuery != "" {
		var matchedRepoID string
		if projects, pErr := s.db.GetProjects(r.Context()); pErr == nil {
			for _, p := range projects {
				if fmt.Sprintf("%s/%s", p.Owner, p.Repo) == repoQuery || p.Repo == repoQuery {
					matchedRepoID = p.ID
					break
				}
			}
		}
		if matchedRepoID != "" {
			filtered := make([]db.Incident, 0)
			for _, inc := range incidents {
				if inc.RepositoryID == matchedRepoID {
					filtered = append(filtered, inc)
				}
			}
			incidents = filtered
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"incidents": incidents})
}

// HandleResolveIncident resolves an incident manually by ID and closes any linked GitHub issue.
func (s *Server) HandleResolveIncident(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.db == nil {
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		IncidentID string `json:"incident_id"`
		ID         string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	targetID := req.IncidentID
	if targetID == "" {
		targetID = req.ID
	}
	if targetID == "" {
		http.Error(w, "Missing incident_id", http.StatusBadRequest)
		return
	}

	if err := s.db.ResolveIncident(r.Context(), targetID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve incident: %v", err), http.StatusInternalServerError)
		return
	}

	inc, err := s.db.GetIncidentByID(r.Context(), targetID)
	if err == nil && inc != nil {
		// Close linked GitHub issue if one exists
		if inc.GitHubIssueNumber > 0 && inc.RepositoryID != "" {
			if s.githubApp == nil {
				s.LoadGitHubAppConfig(r.Context())
			}
			if s.githubApp != nil && s.githubApp.AppID > 0 {
				var owner, repo string
				var instID int64
				if projects, pErr := s.db.GetProjects(r.Context()); pErr == nil {
					for _, p := range projects {
						if p.ID == inc.RepositoryID {
							owner = p.Owner
							repo = p.Repo
							instID = p.InstallationID
							break
						}
					}
				}
				if owner != "" && repo != "" {
					if liveID, rErr := s.ResolveInstallationID(r.Context(), owner, repo); rErr == nil && liveID > 0 {
						instID = liveID
					}
					if instID > 0 {
						if cErr := s.githubApp.CloseIssue(r.Context(), instID, owner, repo, inc.GitHubIssueNumber); cErr != nil {
							slog.Warn("failed to close linked GitHub issue", "error", cErr, "issue_number", inc.GitHubIssueNumber, "repo", fmt.Sprintf("%s/%s", owner, repo))
						} else {
							slog.Info("closed linked GitHub issue successfully", "issue_number", inc.GitHubIssueNumber, "repo", fmt.Sprintf("%s/%s", owner, repo))
						}
					}
				}
			}
		}

		if s.eventBroker != nil {
			s.eventBroker.Publish("incident_updated", inc)
			s.eventBroker.Publish("incident_resolved", inc)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"incident": inc,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) HandleCreateIncidentIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.db == nil {
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		IncidentID string `json:"incident_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IncidentID == "" {
		http.Error(w, "Missing incident_id", http.StatusBadRequest)
		return
	}

	inc, err := s.db.GetIncidentByID(r.Context(), req.IncidentID)
	if err != nil || inc == nil {
		http.Error(w, "Incident not found", http.StatusNotFound)
		return
	}

	if inc.GitHubIssueNumber > 0 && inc.GitHubIssueURL != "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"github_issue": map[string]interface{}{
				"number":   inc.GitHubIssueNumber,
				"html_url": inc.GitHubIssueURL,
			},
		})
		return
	}

	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}
	if s.githubApp == nil {
		http.Error(w, "GitHub App is not configured on this Triage instance", http.StatusBadRequest)
		return
	}

	var owner, repo string
	var installationID int64

	if inc.RepositoryID == "" {
		http.Error(w, "Incident is not associated with any tracked repository", http.StatusBadRequest)
		return
	}

	if projects, err := s.db.GetProjects(r.Context()); err == nil {
		for _, p := range projects {
			if p.ID == inc.RepositoryID {
				owner = p.Owner
				repo = p.Repo
				installationID = p.InstallationID
				break
			}
		}
	}

	if owner == "" || repo == "" {
		http.Error(w, "Target repository for this incident not found", http.StatusNotFound)
		return
	}

	if liveID, err := s.ResolveInstallationID(r.Context(), owner, repo); err == nil && liveID > 0 {
		installationID = liveID
	}

	if installationID == 0 {
		http.Error(w, "Unable to resolve GitHub App installation for this repository", http.StatusBadRequest)
		return
	}

	panicMsg := inc.PanicMessage
	if panicMsg == "" {
		panicMsg = "Runtime panic"
	}
	issueTitle := fmt.Sprintf("🚨 Panic in %s:%d: %s", inc.File, inc.Line, panicMsg)
	issueBody := BuildGitHubIssueBody(GitHubIssueMarkdownParams{
		IncidentID:   inc.ID,
		Owner:        owner,
		Repo:         repo,
		File:         inc.File,
		Line:         inc.Line,
		PanicMessage: panicMsg,
		Severity:     inc.Severity,
		Status:       inc.Status,
		CreatedAt:    inc.CreatedAt,
		RootCause:    inc.RootCause,
		SuggestedFix: inc.SuggestedFix,
		ASTSnippet:   inc.ASTSnippet,
		StackTrace:   inc.StackTrace,
		AppURL:       s.ResolveAppURL(r.Context(), r),
	})

	labels := []string{"panic", "triage", "bug"}
	issueNum, issueURL, err := s.githubApp.CreateIssue(r.Context(), installationID, owner, repo, issueTitle, issueBody, labels)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create GitHub Issue: %v", err), http.StatusInternalServerError)
		return
	}

	_ = s.db.UpdateIncidentIssue(r.Context(), inc.ID, issueURL, issueNum)
	inc.GitHubIssueURL = issueURL
	inc.GitHubIssueNumber = issueNum
	if s.eventBroker != nil {
		s.eventBroker.Publish("incident_updated", inc)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"github_issue": map[string]interface{}{
			"number":   issueNum,
			"html_url": issueURL,
		},
	})
}

func (s *Server) HandleCreateIncidentPR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "Method Not Allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"success": false, "error": "Database unavailable"})
		return
	}

	var req struct {
		IncidentID string `json:"incident_id"`
		PatchCode  string `json:"patch_code,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IncidentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Missing incident_id"})
		return
	}

	inc, err := s.db.GetIncidentByID(r.Context(), req.IncidentID)
	if err != nil || inc == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"success": false, "error": "Incident not found"})
		return
	}

	if inc.GitHubPRNumber > 0 && inc.GitHubPRURL != "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"pull_request": map[string]interface{}{
				"number":   inc.GitHubPRNumber,
				"html_url": inc.GitHubPRURL,
			},
		})
		return
	}

	if s.githubApp == nil {
		s.LoadGitHubAppConfig(r.Context())
	}
	if s.githubApp == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "GitHub App is not configured on this Triage instance"})
		return
	}

	if inc.RepositoryID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Incident is not associated with any tracked repository"})
		return
	}

	var owner, repo, rootDir string
	var installationID int64

	if projects, err := s.db.GetProjects(r.Context()); err == nil {
		for _, p := range projects {
			if p.ID == inc.RepositoryID {
				owner = p.Owner
				repo = p.Repo
				rootDir = p.RootDir
				installationID = p.InstallationID
				break
			}
		}
	}

	if owner == "" || repo == "" {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"success": false, "error": "Target repository for this incident not found"})
		return
	}

	if liveID, err := s.ResolveInstallationID(r.Context(), owner, repo); err == nil && liveID > 0 {
		installationID = liveID
	}

	if installationID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Unable to resolve GitHub App installation for this repository"})
		return
	}

	// 1. Fetch file content from GitHub repository
	normalizedFilePath := ast.NormalizeMonorepoPath(inc.File, rootDir)
	commitFilePath := normalizedFilePath
	if commitFilePath == "" {
		commitFilePath = inc.File
	}

	// Validate target file safety policy (SEC-005, SEC-010)
	if err := ValidatePatchTargetFile(commitFilePath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	currentFileBytes, currentFileSHA, fetchErr := s.githubApp.FetchFileContent(r.Context(), installationID, owner, repo, "", normalizedFilePath)
	if fetchErr != nil {
		currentFileBytes, currentFileSHA, fetchErr = s.githubApp.FetchFileContent(r.Context(), installationID, owner, repo, "", inc.File)
	}
	if fetchErr != nil || currentFileSHA == "" {
		writeJSON(w, http.StatusConflict, map[string]interface{}{"success": false, "error": "Could not fetch target file from GitHub repository or file is missing"})
		return
	}

	// 2. Generate updated file content using active LLM provider
	llmCfg := s.GetLLMConfig(r.Context())

	var projectContext string
	if s.db != nil && owner != "" && repo != "" {
		if proj, err := s.db.GetProjectByOwnerRepo(r.Context(), owner, repo, rootDir); err == nil && proj != nil {
			projectContext = proj.Context
		}
	}

	patchCode := req.PatchCode
	if patchCode == "" {
		patchCode = inc.SuggestedPatch
	}

	// If no patch exists yet and user opened PR straight, generate and persist patch
	if patchCode == "" && (llmCfg.APIKey != "" || llmCfg.Provider == "ollama" || llmCfg.Provider == "custom") {
		if provider, pErr := llm.NewProvider(llmCfg); pErr == nil {
			patchCtx, pCancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer pCancel()
			if generatedPatch, gErr := provider.GeneratePatch(
				patchCtx,
				inc.File,
				inc.PanicMessage,
				inc.ASTSnippet,
				inc.StackTrace,
				inc.RootCause,
				projectContext,
			); gErr == nil && generatedPatch != "" {
				patchCode = generatedPatch
				_ = s.db.UpdateIncidentPatch(r.Context(), inc.ID, patchCode)
			}
		}
	}

	var updatedContent string
	if len(currentFileBytes) > 0 && (llmCfg.APIKey != "" || llmCfg.Provider == "ollama" || llmCfg.Provider == "custom") {
		applyCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		if provider, pErr := llm.NewProvider(llmCfg); pErr == nil {
			var err error
			updatedContent, err = provider.ApplyFixToFile(
				applyCtx,
				inc.File,
				string(currentFileBytes),
				inc.PanicMessage,
				inc.ASTSnippet,
				inc.StackTrace,
				inc.RootCause,
				inc.SuggestedFix,
				patchCode,
				projectContext,
			)
			if err != nil {
				slog.Warn("LLM apply fix failed, falling back to patch text", "error", err, "provider", llmCfg.Provider)
			}
		}
	}

	if updatedContent == "" {
		if patchCode != "" {
			updatedContent = patchCode
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Could not determine modified file content to commit for Pull Request"})
			return
		}
	}

	if !strings.HasSuffix(updatedContent, "\n") {
		updatedContent += "\n"
	}

	// Check unchanged file guard (SEC-005)
	if len(currentFileBytes) > 0 && string(currentFileBytes) == updatedContent {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "No code changes were generated; refusing to commit unchanged file"})
		return
	}

	// 3. Get Default Branch & Base SHA
	defaultBranch, baseSHA, err := s.githubApp.GetDefaultBranch(r.Context(), installationID, owner, repo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to get default branch: %v", err)})
		return
	}

	// 4. Create new Branch with cryptographically unique name (SEC-016)
	cleanIncID := strings.ToLower(strings.ReplaceAll(inc.ID, "-", ""))
	randSuffix := make([]byte, 4)
	_, _ = rand.Read(randSuffix)
	branchName := fmt.Sprintf("triage/fix-%s-%s", cleanIncID, hex.EncodeToString(randSuffix))
	if err := s.githubApp.CreateBranch(r.Context(), installationID, owner, repo, branchName, baseSHA); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to create git branch: %v", err)})
		return
	}

	// 5. Commit updated file to branch
	commitMsg := fmt.Sprintf("fix(triage): resolve panic in %s:%d", inc.File, inc.Line)
	if err := s.githubApp.UpdateFileContent(r.Context(), installationID, owner, repo, commitFilePath, commitMsg, updatedContent, branchName, currentFileSHA); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to commit file update to branch: %v", err)})
		return
	}

	// 6. Open Pull Request
	prTitle := commitMsg
	prBody := BuildGitHubPRBody(GitHubPRMarkdownParams{
		IncidentID:    inc.ID,
		Owner:         owner,
		Repo:          repo,
		File:          inc.File,
		Line:          inc.Line,
		PanicMessage:  inc.PanicMessage,
		IssueNumber:   inc.GitHubIssueNumber,
		RootCause:     inc.RootCause,
		SuggestedFix:  inc.SuggestedFix,
		PatchCode:     patchCode,
		AppURL:        s.ResolveAppURL(r.Context(), r),
		DefaultBranch: defaultBranch,
	})

	prNumber, prURL, err := s.githubApp.CreatePullRequest(r.Context(), installationID, owner, repo, prTitle, prBody, branchName, defaultBranch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to create Pull Request on GitHub: %v", err)})
		return
	}

	// 7. Update Incident record in database
	_ = s.db.UpdateIncidentPR(r.Context(), inc.ID, prURL, prNumber, patchCode)
	inc.GitHubPRURL = prURL
	inc.GitHubPRNumber = prNumber
	inc.SuggestedPatch = patchCode
	if s.eventBroker != nil {
		s.eventBroker.Publish("incident_updated", inc)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"pull_request": map[string]interface{}{
			"number":   prNumber,
			"html_url": prURL,
			"branch":   branchName,
		},
	})
}

func (s *Server) HandleLLMAnalyzePanic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PanicMessage   string `json:"panicMessage"`
		RawStackTrace  string `json:"rawStackTrace"`
		TriggeringFile string `json:"triggeringFile"`
		ASTCode        string `json:"astCode"`
		ProjectContext string `json:"projectContext,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	llmCfg := s.GetLLMConfig(r.Context())
	if llmCfg.APIKey == "" && llmCfg.Provider != "ollama" && llmCfg.Provider != "custom" {
		http.Error(w, "AI model is not configured. Please configure your LLM provider in Settings or Setup Wizard.", http.StatusBadRequest)
		return
	}

	provider, err := llm.NewProvider(llmCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize LLM provider (%s): %v", llmCfg.Provider, err), http.StatusInternalServerError)
		return
	}

	analysisCtx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	delimitedSnippet := req.ASTCode
	if !strings.HasPrefix(delimitedSnippet, "```") {
		delimitedSnippet = fmt.Sprintf("```go\n%s\n```", delimitedSnippet)
	}

	analysis, err := provider.AnalyzeCrash(analysisCtx, req.RawStackTrace, delimitedSnippet, req.ProjectContext)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	severity := ""
	if analysis != nil && analysis.Severity != "" {
		severity = strings.ToUpper(strings.TrimSpace(analysis.Severity))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"rootCause":      analysis.RootCause,
		"explanation":    analysis.RootCause,
		"severity":       severity,
		"recommendedFix": analysis.SuggestedFix,
	})
}

func (s *Server) HandleLLMGeneratePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IncidentID     string `json:"incidentId,omitempty"`
		TriggeringFile string `json:"triggeringFile"`
		PanicMessage   string `json:"panicMessage"`
		ASTCode        string `json:"astCode"`
		RootCause      string `json:"rootCause,omitempty"`
		StackTrace     string `json:"stackTrace,omitempty"`
		ProjectContext string `json:"projectContext,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 1. If incident already has a persisted patch in DB, return that persisted fix (it should not be changed)
	if req.IncidentID != "" && s.db != nil {
		if inc, err := s.db.GetIncidentByID(r.Context(), req.IncidentID); err == nil && inc != nil && inc.SuggestedPatch != "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"patch":   inc.SuggestedPatch,
			})
			return
		}
	}

	llmCfg := s.GetLLMConfig(r.Context())
	if llmCfg.APIKey == "" && llmCfg.Provider != "ollama" && llmCfg.Provider != "custom" {
		http.Error(w, "AI model is not configured. Please configure your LLM provider in Settings or Setup Wizard.", http.StatusBadRequest)
		return
	}

	provider, err := llm.NewProvider(llmCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize LLM provider (%s): %v", llmCfg.Provider, err), http.StatusInternalServerError)
		return
	}

	patchCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	patch, err := provider.GeneratePatch(
		patchCtx,
		req.TriggeringFile,
		req.PanicMessage,
		req.ASTCode,
		req.StackTrace,
		req.RootCause,
		req.ProjectContext,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Patch generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 2. Persist generated patch to DB
	if req.IncidentID != "" && s.db != nil && patch != "" {
		_ = s.db.UpdateIncidentPatch(r.Context(), req.IncidentID, patch)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"patch":   patch,
	})
}
