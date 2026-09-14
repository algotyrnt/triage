// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setupTestAppConfig(t *testing.T) (*AppConfig, []byte) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	der := x509.MarshalPKCS1PrivateKey(privKey)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})

	cfg, err := LoadAppConfig(12345, pemBytes, "whsec_test", "client_123", "sec_123")
	if err != nil {
		t.Fatalf("failed to load app config: %v", err)
	}
	return cfg, pemBytes
}

func jsonResponse(status int, body interface{}) *http.Response {
	var bodyReader io.ReadCloser
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = io.NopCloser(bytes.NewReader(b))
	} else {
		bodyReader = io.NopCloser(bytes.NewReader([]byte{}))
	}
	return &http.Response{
		StatusCode: status,
		Body:       bodyReader,
		Header:     make(http.Header),
	}
}

func TestSetDefaultHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com", nil)
	SetDefaultHeaders(req)

	if req.Header.Get("User-Agent") != "Triage-Engine" {
		t.Errorf("expected User-Agent Triage-Engine, got %s", req.Header.Get("User-Agent"))
	}
	if req.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
		t.Errorf("expected API version 2022-11-28, got %s", req.Header.Get("X-GitHub-Api-Version"))
	}

	// nil request should not panic
	SetDefaultHeaders(nil)
}

func TestLoadAppConfig_Formats(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// 1. PKCS1
	pkcs1DER := x509.MarshalPKCS1PrivateKey(privKey)
	pkcs1PEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcs1DER})
	cfg1, err := LoadAppConfig(1, pkcs1PEM, "s", "c", "cs")
	if err != nil || cfg1 == nil {
		t.Fatalf("PKCS1 loading failed: %v", err)
	}

	// 2. PKCS8
	pkcs8DER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS8: %v", err)
	}
	pkcs8PEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8DER})
	cfg2, err := LoadAppConfig(2, pkcs8PEM, "s", "c", "cs")
	if err != nil || cfg2 == nil {
		t.Fatalf("PKCS8 loading failed: %v", err)
	}

	// 3. Invalid PEM
	_, err = LoadAppConfig(3, []byte("invalid-pem-data"), "s", "c", "cs")
	if err == nil {
		t.Errorf("expected error for invalid PEM data")
	}

	// 4. Corrupt DER in PEM
	corruptPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("corrupt")})
	_, err = LoadAppConfig(4, corruptPEM, "s", "c", "cs")
	if err == nil {
		t.Errorf("expected error for corrupt DER in PEM")
	}
}

func TestSignAppJWT(t *testing.T) {
	cfg, _ := setupTestAppConfig(t)
	jwtToken, err := cfg.SignAppJWT()
	if err != nil {
		t.Fatalf("SignAppJWT failed: %v", err)
	}
	if len(strings.Split(jwtToken, ".")) != 3 {
		t.Errorf("expected 3 segment JWT, got %s", jwtToken)
	}
}

func TestGitHubClient_MockedAPIs(t *testing.T) {
	cfg, _ := setupTestAppConfig(t)
	ctx := context.Background()

	oldTransport := githubHTTPClient.Transport
	defer func() { githubHTTPClient.Transport = oldTransport }()

	githubHTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		// 1. Installation Token
		if strings.HasPrefix(path, "/app/installations/") && strings.HasSuffix(path, "/access_tokens") {
			return jsonResponse(http.StatusCreated, map[string]interface{}{
				"token":      "ghs_mock_installation_token_123",
				"expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			}), nil
		}

		// 2. Fetch file content
		if strings.HasPrefix(path, "/repos/org/repo/contents/main.go") && req.Method == http.MethodGet {
			content := base64.StdEncoding.EncodeToString([]byte("package main\n\nfunc main() {}\n"))
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"content":  content,
				"encoding": "base64",
				"sha":      "blob_sha_123",
			}), nil
		}

		// 3. Create Issue
		if path == "/repos/org/repo/issues" && req.Method == http.MethodPost {
			var body map[string]interface{}
			_ = json.NewDecoder(req.Body).Decode(&body)
			if labels, ok := body["labels"].([]interface{}); ok && len(labels) > 0 && labels[0] == "fail-labels" {
				return jsonResponse(http.StatusForbidden, map[string]string{"message": "label forbidden"}), nil
			}
			return jsonResponse(http.StatusCreated, map[string]interface{}{
				"number":   101,
				"html_url": "https://github.com/org/repo/issues/101",
			}), nil
		}

		// 4. Close Issue
		if strings.HasPrefix(path, "/repos/org/repo/issues/101") && req.Method == http.MethodPatch {
			return jsonResponse(http.StatusOK, map[string]interface{}{"state": "closed"}), nil
		}

		// 5. Default Branch & Ref
		if path == "/repos/org/repo" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"default_branch": "main",
			}), nil
		}
		if path == "/repos/org/repo/git/ref/heads/main" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"object": map[string]string{"sha": "commit_sha_abc123"},
			}), nil
		}

		// 6. Create Branch
		if path == "/repos/org/repo/git/refs" && req.Method == http.MethodPost {
			return jsonResponse(http.StatusCreated, map[string]interface{}{"ref": "refs/heads/patch-1"}), nil
		}

		// 7. Update File Content
		if strings.HasPrefix(path, "/repos/org/repo/contents/") && req.Method == http.MethodPut {
			return jsonResponse(http.StatusOK, map[string]interface{}{"commit": map[string]string{"sha": "new_commit_sha"}}), nil
		}

		// 8. Create PR
		if path == "/repos/org/repo/pulls" && req.Method == http.MethodPost {
			return jsonResponse(http.StatusCreated, map[string]interface{}{
				"number":   55,
				"html_url": "https://github.com/org/repo/pull/55",
			}), nil
		}

		// 9. Verify App
		if path == "/app" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": 12345}), nil
		}

		// 10. Get Repo Installation
		if path == "/repos/org/repo/installation" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": 98765}), nil
		}

		// 11. List App Installations
		if path == "/app/installations" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, []map[string]interface{}{
				{
					"id": 98765,
					"account": map[string]interface{}{
						"login": "org",
						"id":    1001,
						"type":  "Organization",
					},
				},
			}), nil
		}

		// 12. List Installation Repositories
		if path == "/installation/repositories" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"total_count": 1,
				"repositories": []map[string]interface{}{
					{
						"name":           "repo",
						"full_name":      "org/repo",
						"private":        false,
						"default_branch": "main",
						"language":       "Go",
						"owner":          map[string]string{"login": "org"},
					},
				},
			}), nil
		}

		// 13. Fetch User Repositories
		if path == "/user/repos" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, []map[string]interface{}{
				{
					"name":           "repo",
					"full_name":      "org/repo",
					"private":        false,
					"default_branch": "main",
					"language":       "Go",
					"owner":          map[string]string{"login": "org"},
				},
			}), nil
		}

		return jsonResponse(http.StatusNotFound, map[string]string{"message": "Not Found"}), nil
	})

	// Run all methods through the mock transport
	tok, err := cfg.GetInstallationToken(ctx, 98765)
	if err != nil || tok != "ghs_mock_installation_token_123" {
		t.Fatalf("GetInstallationToken failed: %v, got %s", err, tok)
	}

	// Cached token check
	tok2, err := cfg.GetInstallationToken(ctx, 98765)
	if err != nil || tok2 != tok {
		t.Fatalf("expected cached token %s, got %s", tok, tok2)
	}

	// FetchFileContent
	fileContent, sha, err := cfg.FetchFileContent(ctx, 98765, "org", "repo", "commit1", "main.go")
	if err != nil || sha != "blob_sha_123" || !strings.Contains(string(fileContent), "package main") {
		t.Fatalf("FetchFileContent failed: %v, content=%s", err, fileContent)
	}

	// CreateIssue
	issueNum, issueURL, err := cfg.CreateIssue(ctx, 98765, "org", "repo", "Bug Title", "Bug Body", []string{"bug"})
	if err != nil || issueNum != 101 || issueURL != "https://github.com/org/repo/issues/101" {
		t.Fatalf("CreateIssue failed: %v, got (%d, %s)", err, issueNum, issueURL)
	}

	// CloseIssue
	if err := cfg.CloseIssue(ctx, 98765, "org", "repo", 101); err != nil {
		t.Fatalf("CloseIssue failed: %v", err)
	}

	// GetDefaultBranch
	defBranch, headSHA, err := cfg.GetDefaultBranch(ctx, 98765, "org", "repo")
	if err != nil || defBranch != "main" || headSHA != "commit_sha_abc123" {
		t.Fatalf("GetDefaultBranch failed: %v, got (%s, %s)", err, defBranch, headSHA)
	}

	// CreateBranch
	if err := cfg.CreateBranch(ctx, 98765, "org", "repo", "patch-1", headSHA); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// UpdateFileContent
	if err := cfg.UpdateFileContent(ctx, 98765, "org", "repo", "main.go", "fix panic", "package main\n\nfunc main() {}\n", "patch-1", sha); err != nil {
		t.Fatalf("UpdateFileContent failed: %v", err)
	}

	// CreatePullRequest
	prNum, prURL, err := cfg.CreatePullRequest(ctx, 98765, "org", "repo", "Fix panic", "Fix body", "patch-1", "main")
	if err != nil || prNum != 55 || prURL != "https://github.com/org/repo/pull/55" {
		t.Fatalf("CreatePullRequest failed: %v, got (%d, %s)", err, prNum, prURL)
	}

	// VerifyApp
	if err := cfg.VerifyApp(ctx); err != nil {
		t.Fatalf("VerifyApp failed: %v", err)
	}

	// GetRepoInstallation
	instID, err := cfg.GetRepoInstallation(ctx, "org", "repo")
	if err != nil || instID != 98765 {
		t.Fatalf("GetRepoInstallation failed: %v, got %d", err, instID)
	}

	// ListAppInstallations
	installations, err := cfg.ListAppInstallations(ctx)
	if err != nil || len(installations) != 1 || installations[0].AccountLogin != "org" {
		t.Fatalf("ListAppInstallations failed: %v, got %+v", err, installations)
	}

	// ListInstallationRepositories
	repos, err := cfg.ListInstallationRepositories(ctx, 98765)
	if err != nil || len(repos) != 1 || repos[0].Repo != "repo" {
		t.Fatalf("ListInstallationRepositories failed: %v, got %+v", err, repos)
	}

	// FetchUserRepositories
	userRepos, err := FetchUserRepositories(ctx, "user", "user_mock_token")
	if err != nil || len(userRepos) != 1 {
		t.Fatalf("FetchUserRepositories failed: %v, got %+v", err, userRepos)
	}

	// FetchUserRepositories without token
	_ = os.Unsetenv("GITHUB_TOKEN")
	_, err = FetchUserRepositories(ctx, "user")
	if err == nil {
		t.Errorf("expected error when no github token provided for FetchUserRepositories")
	}
}

func TestGitHubClient_ErrorAndFallbackBranches(t *testing.T) {
	cfg, _ := setupTestAppConfig(t)
	ctx := context.Background()

	oldTransport := githubHTTPClient.Transport
	defer func() { githubHTTPClient.Transport = oldTransport }()

	githubHTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		// Installation Token
		if strings.HasPrefix(path, "/app/installations/9999/access_tokens") {
			return jsonResponse(http.StatusUnauthorized, map[string]string{"message": "bad token"}), nil
		}
		if strings.HasPrefix(path, "/app/installations/") && strings.HasSuffix(path, "/access_tokens") {
			return jsonResponse(http.StatusCreated, map[string]interface{}{
				"token":      "ghs_token_retry",
				"expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			}), nil
		}

		// FetchFileContent with ref fails first, retries without ref and succeeds
		if path == "/repos/org/repo/contents/fallback.go" {
			if req.URL.RawQuery == "ref=stale_sha" {
				return jsonResponse(http.StatusNotFound, map[string]string{"message": "not found"}), nil
			}
			content := base64.StdEncoding.EncodeToString([]byte("package main\n"))
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"content":  content,
				"encoding": "base64",
				"sha":      "fallback_sha",
			}), nil
		}

		// CreateIssue fails with labels, retries without labels and succeeds
		if path == "/repos/org/repo/issues" && req.Method == http.MethodPost {
			var body map[string]interface{}
			_ = json.NewDecoder(req.Body).Decode(&body)
			if labels, ok := body["labels"].([]interface{}); ok && len(labels) > 0 {
				return jsonResponse(http.StatusForbidden, map[string]string{"message": "labels restricted"}), nil
			}
			return jsonResponse(http.StatusCreated, map[string]interface{}{
				"number":   202,
				"html_url": "https://github.com/org/repo/issues/202",
			}), nil
		}

		// VerifyApp failure
		if path == "/app" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusUnauthorized, map[string]string{"message": "bad jwt"}), nil
		}

		// GetRepoInstallation failure
		if path == "/repos/org/repo/installation" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusNotFound, map[string]string{"message": "not found"}), nil
		}

		// FetchUserRepositories orgs & org repos
		if path == "/user/repos" {
			return jsonResponse(http.StatusOK, []map[string]interface{}{
				{
					"name":           "personal-repo",
					"full_name":      "user/personal-repo",
					"private":        true,
					"default_branch": "main",
					"language":       "Go",
					"owner":          map[string]string{"login": "user"},
				},
			}), nil
		}
		if path == "/user/orgs" {
			return jsonResponse(http.StatusOK, []map[string]interface{}{
				{"login": "my-org"},
			}), nil
		}
		if path == "/orgs/my-org/repos" {
			return jsonResponse(http.StatusOK, []map[string]interface{}{
				{
					"name":           "org-repo",
					"full_name":      "my-org/org-repo",
					"private":        true,
					"default_branch": "main",
					"language":       "Go",
					"owner":          map[string]string{"login": "my-org"},
				},
			}), nil
		}

		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "mock error"}), nil
	})

	// 1. Installation token error
	_, err := cfg.GetInstallationToken(ctx, 9999)
	if err == nil {
		t.Errorf("expected error getting installation token for 9999")
	}

	// 2. FetchFileContent with fallback
	content, sha, err := cfg.FetchFileContent(ctx, 12345, "org", "repo", "stale_sha", "fallback.go")
	if err != nil || sha != "fallback_sha" || !strings.Contains(string(content), "package main") {
		t.Fatalf("expected fallback file content fetch: %v", err)
	}

	// 3. CreateIssue retry without labels
	num, url, err := cfg.CreateIssue(ctx, 12345, "org", "repo", "Bug Title", "Body", []string{"custom-label"})
	if err != nil || num != 202 || url != "https://github.com/org/repo/issues/202" {
		t.Fatalf("expected CreateIssue retry without labels to succeed: %v, got (%d, %s)", err, num, url)
	}

	// 4. VerifyApp error
	err = cfg.VerifyApp(ctx)
	if err == nil {
		t.Errorf("expected error from VerifyApp on 401")
	}

	// 5. GetRepoInstallation error
	_, err = cfg.GetRepoInstallation(ctx, "org", "repo")
	if err == nil {
		t.Errorf("expected error from GetRepoInstallation on 404")
	}

	// 6. FetchUserRepositories with orgs and env token
	_ = os.Setenv("GITHUB_TOKEN", "env_token_123")
	defer os.Unsetenv("GITHUB_TOKEN")

	repos, err := FetchUserRepositories(ctx, "user")
	if err != nil || len(repos) != 2 {
		t.Fatalf("expected 2 repos from user & orgs, got %d, err=%v", len(repos), err)
	}

	// 7. Test GITHUB_ACCESS_TOKEN env fallback
	_ = os.Unsetenv("GITHUB_TOKEN")
	_ = os.Setenv("GITHUB_ACCESS_TOKEN", "access_token_123")
	defer os.Unsetenv("GITHUB_ACCESS_TOKEN")

	repos2, err := FetchUserRepositories(ctx, "user")
	if err != nil || len(repos2) != 2 {
		t.Fatalf("expected 2 repos with GITHUB_ACCESS_TOKEN fallback, got %d, err=%v", len(repos2), err)
	}

	// 8. CreateBranch 422 branch already exists
	githubHTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/git/refs") {
			return jsonResponse(http.StatusUnprocessableEntity, map[string]string{"message": "Reference already exists"}), nil
		}
		if strings.Contains(req.URL.Path, "/pulls") {
			return jsonResponse(http.StatusUnprocessableEntity, map[string]string{"message": "PR already exists"}), nil
		}
		if strings.Contains(req.URL.Path, "/issues/101") {
			return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "server error"}), nil
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{}), nil
	})

	// 422 on CreateBranch is tolerated
	if err := cfg.CreateBranch(ctx, 12345, "org", "repo", "existing-branch", "sha123"); err != nil {
		t.Errorf("expected 422 to be tolerated on CreateBranch, got: %v", err)
	}

	// 422 on CreatePullRequest returns error
	if _, _, err := cfg.CreatePullRequest(ctx, 12345, "org", "repo", "Title", "Body", "h", "b"); err == nil {
		t.Errorf("expected error on CreatePullRequest 422")
	}

	// 500 on CloseIssue returns error
	if err := cfg.CloseIssue(ctx, 12345, "org", "repo", 101); err == nil {
		t.Errorf("expected error on CloseIssue 500")
	}
}

func TestSetHTTPClient(t *testing.T) {
	custom := &http.Client{}
	cleanup := SetHTTPClient(custom)
	if githubHTTPClient != custom {
		t.Errorf("expected custom client")
	}
	cleanup()
	if githubHTTPClient == custom {
		t.Errorf("expected client restored")
	}
}
