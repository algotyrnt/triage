// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/llm"
)

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

	if inc.RepositoryID != "" {
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
	}
	if owner == "" {
		if projects, err := s.db.GetProjects(r.Context()); err == nil && len(projects) > 0 {
			owner = projects[0].Owner
			repo = projects[0].Repo
			installationID = projects[0].InstallationID
		}
	}
	if (installationID == 0 || installationID == 1001) && owner != "" && repo != "" {
		if instID, err := s.db.GetInstallationForRepo(r.Context(), owner, repo); err == nil && instID > 0 {
			installationID = instID
		}
	}
	if installationID == 0 || installationID == 1001 {
		if inst, err := s.db.GetInstallation(r.Context()); err == nil && inst != nil {
			installationID = inst.InstallationID
		}
	}
	if installationID == 0 || installationID == 1001 {
		if appInstalls, err := s.githubApp.ListAppInstallations(r.Context()); err == nil && len(appInstalls) > 0 {
			installationID = appInstalls[0].ID
		}
	}

	if installationID == 0 || owner == "" || repo == "" {
		http.Error(w, "Unable to resolve GitHub repository or App installation for this incident", http.StatusBadRequest)
		return
	}

	panicMsg := inc.PanicMessage
	if panicMsg == "" {
		panicMsg = "Runtime panic"
	}
	issueTitle := fmt.Sprintf("🚨 Panic in %s:%d: %s", inc.File, inc.Line, panicMsg)
	var issueBody strings.Builder
	issueBody.WriteString("## Runtime Go Panic Detected\n\n")
	issueBody.WriteString(fmt.Sprintf("A runtime panic was intercepted by **Triage** in `%s:%d`.\n\n", inc.File, inc.Line))
	issueBody.WriteString("---\n\n### Panic Details\n")
	issueBody.WriteString(fmt.Sprintf("- **Incident ID**: `%s`\n", inc.ID))
	issueBody.WriteString(fmt.Sprintf("- **File**: `%s:%d`\n", inc.File, inc.Line))
	issueBody.WriteString(fmt.Sprintf("- **Timestamp**: `%s UTC`\n\n", inc.CreatedAt.UTC().Format("2006-01-02 15:04:05")))

	if inc.RootCause != "" {
		issueBody.WriteString("---\n\n### Root Cause Analysis (Gemini AI)\n")
		issueBody.WriteString(fmt.Sprintf("%s\n\n", inc.RootCause))
	}
	appURL := s.ResolveAppURL(r.Context(), r)
	if inc.SuggestedFix != "" {
		issueBody.WriteString("---\n\n### Recommended Fix\n")
		issueBody.WriteString(fmt.Sprintf("%s\n\n", inc.SuggestedFix))
		if appURL != "" {
			fixURL := fmt.Sprintf("%s/?incident=%s", appURL, inc.ID)
			issueBody.WriteString(fmt.Sprintf("- [ ] [**Generate Fix (PR)**](%s)\n\n", fixURL))
		} else {
			issueBody.WriteString("- [ ] **Generate Fix (PR)**\n\n")
		}
	}
	if inc.ASTSnippet != "" {
		issueBody.WriteString("---\n\n### AST Context\n```go\n")
		issueBody.WriteString(fmt.Sprintf("%s\n```\n\n", inc.ASTSnippet))
	}
	if inc.StackTrace != "" {
		issueBody.WriteString("---\n\n### Stack Trace\n```\n")
		issueBody.WriteString(fmt.Sprintf("%s\n```\n\n", inc.StackTrace))
	}
	if appURL != "" {
		issueBody.WriteString(fmt.Sprintf("---\n*Automatically created by [triage](%s)*\n", appURL))
	} else {
		issueBody.WriteString("---\n*Automatically created by triage*\n")
	}

	labels := []string{"panic", "triage", "bug"}
	issueNum, issueURL, err := s.githubApp.CreateIssue(r.Context(), installationID, owner, repo, issueTitle, issueBody.String(), labels)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create GitHub Issue: %v", err), http.StatusInternalServerError)
		return
	}

	_ = s.db.UpdateIncidentIssue(r.Context(), inc.ID, issueURL, issueNum)

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

	var owner, repo, rootDir string
	var installationID int64

	if inc.RepositoryID != "" {
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
	}
	if owner == "" {
		if projects, err := s.db.GetProjects(r.Context()); err == nil && len(projects) > 0 {
			owner = projects[0].Owner
			repo = projects[0].Repo
			rootDir = projects[0].RootDir
			installationID = projects[0].InstallationID
		}
	}
	if (installationID == 0 || installationID == 1001) && owner != "" && repo != "" {
		if instID, err := s.db.GetInstallationForRepo(r.Context(), owner, repo); err == nil && instID > 0 {
			installationID = instID
		}
	}
	if installationID == 0 || installationID == 1001 {
		if inst, err := s.db.GetInstallation(r.Context()); err == nil && inst != nil {
			installationID = inst.InstallationID
		}
	}
	if installationID == 0 || installationID == 1001 {
		if appInstalls, err := s.githubApp.ListAppInstallations(r.Context()); err == nil && len(appInstalls) > 0 {
			installationID = appInstalls[0].ID
		}
	}

	if installationID == 0 || owner == "" || repo == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Unable to resolve GitHub repository or App installation for this incident"})
		return
	}

	// 1. Fetch file content from GitHub repository
	normalizedFilePath := ast.NormalizeMonorepoPath(inc.File, rootDir)
	currentFileBytes, currentFileSHA, fetchErr := s.githubApp.FetchFileContent(r.Context(), installationID, owner, repo, "", normalizedFilePath)
	if fetchErr != nil {
		currentFileBytes, currentFileSHA, fetchErr = s.githubApp.FetchFileContent(r.Context(), installationID, owner, repo, "", inc.File)
	}

	// 2. Generate updated file content using Gemini
	llmAPIKey := ""
	llmModelName := ""
	if s.db != nil {
		llmAPIKey, _ = s.db.GetInstanceConfig(r.Context(), "gemini_api_key")
		llmModelName, _ = s.db.GetInstanceConfig(r.Context(), "gemini_model")
	}
	if llmAPIKey == "" {
		llmAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if llmModelName == "" {
		llmModelName = os.Getenv("GEMINI_MODEL_NAME")
	}
	if llmModelName == "" {
		llmModelName = "gemini-1.5-flash"
	}

	patchCode := req.PatchCode
	if patchCode == "" {
		patchCode = inc.SuggestedPatch
	}

	var updatedContent string
	if len(currentFileBytes) > 0 && llmAPIKey != "" {
		applyCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		var err error
		updatedContent, err = llm.ApplyFixToFile(
			applyCtx,
			inc.File,
			string(currentFileBytes),
			inc.PanicMessage,
			inc.ASTSnippet,
			inc.StackTrace,
			inc.RootCause,
			inc.SuggestedFix,
			patchCode,
			llmAPIKey,
			llmModelName,
		)
		if err != nil {
			slog.Warn("Gemini apply fix failed, falling back to patch text", "error", err)
		}
	}

	if updatedContent == "" {
		if len(currentFileBytes) > 0 {
			updatedContent = string(currentFileBytes)
		} else if patchCode != "" {
			updatedContent = patchCode
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Could not determine file content to commit for Pull Request"})
			return
		}
	}

	if !strings.HasSuffix(updatedContent, "\n") {
		updatedContent += "\n"
	}

	// 3. Get Default Branch & Base SHA
	defaultBranch, baseSHA, err := s.githubApp.GetDefaultBranch(r.Context(), installationID, owner, repo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to get default branch: %v", err)})
		return
	}

	// 4. Create new Branch
	cleanIncID := strings.ToLower(strings.ReplaceAll(inc.ID, "-", ""))
	branchName := fmt.Sprintf("triage/fix-%s-%d", cleanIncID, time.Now().Unix())
	if err := s.githubApp.CreateBranch(r.Context(), installationID, owner, repo, branchName, baseSHA); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to create git branch: %v", err)})
		return
	}

	// 5. Commit updated file to branch
	commitMsg := fmt.Sprintf("fix(triage): resolve panic in %s:%d", inc.File, inc.Line)
	commitFilePath := normalizedFilePath
	if commitFilePath == "" {
		commitFilePath = inc.File
	}
	if err := s.githubApp.UpdateFileContent(r.Context(), installationID, owner, repo, commitFilePath, commitMsg, updatedContent, branchName, currentFileSHA); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to commit file update to branch: %v", err)})
		return
	}

	// 6. Open Pull Request
	prTitle := commitMsg
	var prBody strings.Builder
	prBody.WriteString("## Triage Automated Bugfix Pull Request\n\n")
	prBody.WriteString(fmt.Sprintf("This Pull Request was generated automatically by **Triage** to fix a runtime panic in `%s:%d`.\n\n", inc.File, inc.Line))
	if inc.GitHubIssueNumber > 0 {
		prBody.WriteString(fmt.Sprintf("Closes #%d\n\n", inc.GitHubIssueNumber))
	}
	prBody.WriteString("---\n\n### Incident Details\n")
	prBody.WriteString(fmt.Sprintf("- **Incident ID**: `%s`\n", inc.ID))
	prBody.WriteString(fmt.Sprintf("- **Panic Message**: `%s`\n\n", inc.PanicMessage))
	if inc.RootCause != "" {
		prBody.WriteString("---\n\n### Gemini AI Root Cause Analysis\n")
		prBody.WriteString(fmt.Sprintf("%s\n\n", inc.RootCause))
	}
	if inc.SuggestedFix != "" {
		prBody.WriteString("---\n\n### Recommended Fix\n")
		prBody.WriteString(fmt.Sprintf("%s\n\n", inc.SuggestedFix))
	}
	if patchCode != "" {
		prBody.WriteString("---\n\n### Suggested Patch Diff\n```diff\n")
		prBody.WriteString(fmt.Sprintf("%s\n```\n\n", patchCode))
	}
	if appURL := s.ResolveAppURL(r.Context(), r); appURL != "" {
		prBody.WriteString(fmt.Sprintf("---\n*Automated PR generated by [triage](%s)*\n", appURL))
	} else {
		prBody.WriteString("---\n*Automated PR generated by triage*\n")
	}

	prNumber, prURL, err := s.githubApp.CreatePullRequest(r.Context(), installationID, owner, repo, prTitle, prBody.String(), branchName, defaultBranch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": fmt.Sprintf("Failed to create Pull Request on GitHub: %v", err)})
		return
	}

	// 7. Update Incident record in database
	_ = s.db.UpdateIncidentPR(r.Context(), inc.ID, prURL, prNumber, patchCode)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"pull_request": map[string]interface{}{
			"number":   prNumber,
			"html_url": prURL,
			"branch":   branchName,
		},
	})
}

func (s *Server) HandleGeminiAnalyzePanic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PanicMessage   string `json:"panicMessage"`
		RawStackTrace  string `json:"rawStackTrace"`
		TriggeringFile string `json:"triggeringFile"`
		ASTCode        string `json:"astCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	llmAPIKey := ""
	llmModelName := ""
	if s.db != nil {
		llmAPIKey, _ = s.db.GetInstanceConfig(r.Context(), "gemini_api_key")
		llmModelName, _ = s.db.GetInstanceConfig(r.Context(), "gemini_model")
	}
	if llmAPIKey == "" {
		llmAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if llmModelName == "" {
		llmModelName = os.Getenv("GEMINI_MODEL_NAME")
	}
	if llmModelName == "" {
		llmModelName = "gemini-1.5-flash"
	}

	if llmAPIKey == "" {
		http.Error(w, "Gemini API key is not configured. Please configure it in Project Settings or Setup Wizard.", http.StatusBadRequest)
		return
	}

	analysisCtx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	delimitedSnippet := req.ASTCode
	if !strings.HasPrefix(delimitedSnippet, "```") {
		delimitedSnippet = fmt.Sprintf("```go\n%s\n```", delimitedSnippet)
	}

	analysis, err := llm.AnalyzeCrash(analysisCtx, req.RawStackTrace, delimitedSnippet, llmAPIKey, llmModelName)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"rootCause":      analysis.RootCause,
		"explanation":    analysis.RootCause,
		"severity":       "CRITICAL",
		"recommendedFix": analysis.SuggestedFix,
	})
}

func (s *Server) HandleGeminiGeneratePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TriggeringFile string `json:"triggeringFile"`
		PanicMessage   string `json:"panicMessage"`
		ASTCode        string `json:"astCode"`
		RootCause      string `json:"rootCause,omitempty"`
		StackTrace     string `json:"stackTrace,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	llmAPIKey := ""
	llmModelName := ""
	if s.db != nil {
		llmAPIKey, _ = s.db.GetInstanceConfig(r.Context(), "gemini_api_key")
		llmModelName, _ = s.db.GetInstanceConfig(r.Context(), "gemini_model")
	}
	if llmAPIKey == "" {
		llmAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if llmModelName == "" {
		llmModelName = os.Getenv("GEMINI_MODEL_NAME")
	}
	if llmModelName == "" {
		llmModelName = "gemini-1.5-flash"
	}

	if llmAPIKey == "" {
		http.Error(w, "Gemini API key is not configured. Please configure it in Project Settings or Setup Wizard.", http.StatusBadRequest)
		return
	}

	patchCtx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	patch, err := llm.GeneratePatch(
		patchCtx,
		req.TriggeringFile,
		req.PanicMessage,
		req.ASTCode,
		req.StackTrace,
		req.RootCause,
		llmAPIKey,
		llmModelName,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Patch generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"patch":   patch,
	})
}
