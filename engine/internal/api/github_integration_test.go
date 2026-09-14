// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"triage/engine/internal/db"
	"triage/engine/internal/github"
)

func TestGitHubIntegrationFlows(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	ghApp := setupTestGitHubApp(t)
	env.Server.githubApp = ghApp

	// In-memory mock GitHub API router
	mockHandler := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		// 1. Access tokens
		if strings.Contains(path, "/access_tokens") {
			body := `{"token":"ghs_mock_installation_token","expires_at":"2099-01-01T00:00:00Z"}`
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 2. Installations list
		if path == "/app/installations" {
			body := `[{"id":777,"account":{"login":"testorg","id":100,"type":"Organization"}}]`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 3. Single installation
		if strings.HasPrefix(path, "/app/installations/777") {
			body := `{"id":777,"account":{"login":"testorg","id":100,"type":"Organization"}}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 4. Installation repositories
		if path == "/installation/repositories" {
			body := `{"total_count":1,"repositories":[{"id":555,"name":"payments","owner":{"login":"testorg"},"default_branch":"main","language":"Go","private":false}]}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 5. Create Issue
		if strings.HasSuffix(path, "/issues") && req.Method == http.MethodPost {
			body := `{"number":42,"html_url":"https://github.com/testorg/payments/issues/42","title":"Panic","state":"open"}`
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 6. Get Repository details
		if path == "/repos/testorg/payments" && req.Method == http.MethodGet {
			body := `{"default_branch":"main"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 7. Get Branch Ref
		if strings.Contains(path, "/git/ref/heads/main") {
			body := `{"ref":"refs/heads/main","object":{"sha":"commit_abc123"}}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 8. Create Branch Ref
		if strings.Contains(path, "/git/refs") && req.Method == http.MethodPost {
			body := `{"ref":"refs/heads/triage/fix-1","object":{"sha":"commit_abc123"}}`
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 9. File Contents
		if strings.Contains(path, "/contents/") {
			if req.Method == http.MethodGet {
				fileContent := base64.StdEncoding.EncodeToString([]byte("package main\n\nfunc main() {}\n"))
				body := fmt.Sprintf(`{"sha":"blob_123","content":%q,"encoding":"base64"}`, fileContent)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			}
			if req.Method == http.MethodPut {
				body := `{"content":{"sha":"new_blob_sha"},"commit":{"sha":"new_commit_sha"}}`
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			}
		}

		// 10. Create Pull Request
		if strings.HasSuffix(path, "/pulls") && req.Method == http.MethodPost {
			body := `{"number":88,"html_url":"https://github.com/testorg/payments/pull/88","title":"Fix panic"}`
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 11. Git Trees (Detect Modules)
		if strings.Contains(path, "/git/trees/") {
			body := `{"tree":[{"path":"go.mod","type":"blob"},{"path":"service/worker/go.mod","type":"blob"}]}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		// 12. Verify App
		if path == "/app" {
			body := `{"id":12345,"slug":"test-triage-app","name":"Triage Engine"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})

	mockHTTP := mockClient(mockHandler)
	restoreGH := github.SetHTTPClient(mockHTTP)
	defer restoreGH()

	origTransport := http.DefaultTransport
	http.DefaultTransport = mockHandler
	defer func() { http.DefaultTransport = origTransport }()

	// --- Test A: HandleSetupTest ---
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/test", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleSetupTest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HandleSetupTest, got %d: %s", rec.Code, rec.Body.String())
	}

	// --- Test B: HandleSetupRepos ---
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/repos", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleSetupRepos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HandleSetupRepos, got %d", rec.Code)
	}
	var reposRes struct {
		Repos []SetupRepoItem `json:"repos"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&reposRes)
	if len(reposRes.Repos) != 1 || reposRes.Repos[0].Name != "testorg/payments" {
		t.Errorf("expected testorg/payments in repos: %+v", reposRes)
	}

	// --- Test C: HandleInstalledRepos ---
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/installed-repos", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleInstalledRepos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HandleInstalledRepos, got %d", rec.Code)
	}

	// --- Test D: HandleDetectModules ---
	_ = env.DB.SaveInstallation(ctx, 777, "testorg", 100, "Organization")
	_ = env.DB.SaveInstallationRepo(ctx, 777, "testorg", "payments")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/repos/detect-modules?owner=testorg&repo=payments", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleDetectModules(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HandleDetectModules, got %d", rec.Code)
	}
	var moduleRes struct {
		Modules []DetectedModule `json:"modules"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&moduleRes)
	if len(moduleRes.Modules) < 2 {
		t.Errorf("expected at least 2 detected modules, got %+v", moduleRes.Modules)
	}

	// --- Test E: HandleCreateIncidentIssue ---
	_, repoID, _ := env.DB.CreateProject(ctx, "testorg", "payments", "", "owner_user")
	_ = env.DB.UpdateRepositoryInstallation(ctx, "testorg", "payments", 777)

	inc := &db.Incident{
		ID:           "inc_gh_issue",
		RepositoryID: repoID,
		Title:        "Panic in pay.go",
		Status:       "OPEN",
		File:         "pay.go",
		Line:         10,
		PanicMessage: "nil pointer dereference",
		StackTrace:   "goroutine 1 [running]:\npay.go:10",
		CreatedAt:    time.Now().UTC(),
	}
	_ = env.DB.SaveIncident(ctx, inc)

	issuePayload, _ := json.Marshal(map[string]string{"incident_id": "inc_gh_issue"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", bytes.NewReader(issuePayload))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentIssue(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for create issue, got %d: %s", rec.Code, rec.Body.String())
	}
	var issueRes struct {
		Success     bool `json:"success"`
		GitHubIssue struct {
			Number  int    `json:"number"`
			HTMLURL string `json:"html_url"`
		} `json:"github_issue"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&issueRes)
	if !issueRes.Success || issueRes.GitHubIssue.Number != 42 {
		t.Errorf("expected issue 42 created: %+v", issueRes)
	}

	// --- Test F: HandleCreateIncidentPR ---
	prPayload, _ := json.Marshal(map[string]string{
		"incident_id": "inc_gh_issue",
		"patch_code":  "--- a/pay.go\n+++ b/pay.go\n@@ -10 +10 @@\n+ // fixed\n",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", bytes.NewReader(prPayload))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentPR(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for create PR, got %d: %s", rec.Code, rec.Body.String())
	}
	var prRes struct {
		Success     bool `json:"success"`
		PullRequest struct {
			Number  int    `json:"number"`
			HTMLURL string `json:"html_url"`
		} `json:"pull_request"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&prRes)
	if !prRes.Success || prRes.PullRequest.Number != 88 {
		t.Errorf("expected PR 88 created: %+v", prRes)
	}

	// --- Test G: HandleResolveIncident with linked GitHub Issue ---
	resolvePayload, _ := json.Marshal(map[string]string{"incident_id": "inc_gh_issue"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/resolve", bytes.NewReader(resolvePayload))
	rec = httptest.NewRecorder()
	env.Server.HandleResolveIncident(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for resolving incident with linked issue, got %d", rec.Code)
	}
}
