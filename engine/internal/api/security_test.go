// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triage/engine/internal/config"
	"triage/engine/internal/db"
)

// setupSecurityTestServer creates a test server with routes registered on a ServeMux.
func setupSecurityTestServer(t *testing.T) (*Server, *http.ServeMux, string) {
	t.Helper()
	s := newTestAPIServer()
	secret, err := s.getSessionSecret(context.Background())
	if err != nil {
		t.Fatalf("failed to obtain session secret: %v", err)
	}

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	return s, mux, secret
}

// helper to create a valid JWT token for a specific role
func createTestToken(t *testing.T, secret, role string) string {
	t.Helper()
	user := &db.User{
		ID:        "usr_test_" + strings.ToLower(role),
		GitHubID:  "10001",
		Username:  "tester_" + strings.ToLower(role),
		Email:     "tester@example.com",
		AvatarURL: "https://example.com/avatar.png",
		Role:      role,
	}
	token, err := GenerateUserJWT(user, secret)
	if err != nil {
		t.Fatalf("failed to generate test token for role %s: %v", role, err)
	}
	return token
}

// TestSecurity_SEC001_UnauthenticatedAccessVerifications tests that protected routes enforce authentication.
func TestSecurity_SEC001_UnauthenticatedAccessVerifications(t *testing.T) {
	_, mux, _ := setupSecurityTestServer(t)

	protectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/incidents"},
		{http.MethodGet, "/api/v1/projects"},
		{http.MethodGet, "/api/v1/stats"},
		{http.MethodGet, "/api/v1/repos/detect-modules"},
		{http.MethodGet, "/api/v1/team/members"},
		{http.MethodPost, "/api/v1/incidents/create-issue"},
		{http.MethodPost, "/api/v1/incidents/create-pr"},
		{http.MethodPost, "/api/v1/projects/context"},
		{http.MethodPost, "/api/v1/llm/analyze-panic"},
		{http.MethodPost, "/api/v1/llm/generate-patch"},
		{http.MethodPut, "/api/v1/team/members/role"},
		{http.MethodGet, "/api/v1/team/invites"},
		{http.MethodPost, "/api/v1/team/invites"},
		{http.MethodGet, "/api/v1/settings/llm"},
		{http.MethodPost, "/api/v1/settings/llm/test"},
		{http.MethodGet, "/api/v1/projects/keys"},
		{http.MethodPost, "/api/v1/projects/keys/revoke"},
	}

	for _, route := range protectedRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 Unauthorized for unauthenticated %s %s, got %d", route.method, route.path, rec.Code)
			}
		})
	}
}

// TestSecurity_SEC002_InsecureSecretRejection tests that hardcoded/insecure secrets are rejected.
func TestSecurity_SEC002_InsecureSecretRejection(t *testing.T) {
	user := &db.User{
		ID:       "usr_sec2",
		Username: "sec2user",
		Role:     "Owner",
	}

	// 1. Generation with empty secret must fail
	if _, err := GenerateUserJWT(user, ""); err == nil {
		t.Errorf("expected error when generating JWT with empty secret")
	}

	// 2. Generation with insecure default secret must fail
	if _, err := GenerateUserJWT(user, config.InsecureDefaultSecret); err == nil {
		t.Errorf("expected error when generating JWT with insecure default secret")
	}

	// 3. Generation with short secret (<32 bytes) must fail
	if _, err := GenerateUserJWT(user, "too-short-secret-123"); err == nil {
		t.Errorf("expected error when generating JWT with short secret")
	}

	// 4. Verification with insecure default secret must fail
	if _, err := ParseAndVerifyUserJWT("some.fake.jwt", config.InsecureDefaultSecret); err == nil {
		t.Errorf("expected error when parsing JWT with insecure default secret")
	}
}

// TestSecurity_SEC003_SessionInQueryRejected tests that session tokens in URL query strings are rejected.
func TestSecurity_SEC003_SessionInQueryRejected(t *testing.T) {
	_, mux, secret := setupSecurityTestServer(t)
	token := createTestToken(t, secret, "Developer")

	// Passing ?token= in URL query must NOT grant access
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me?token="+token, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized when token is in query parameter, got %d", rec.Code)
	}

	// Passing Authorization: Bearer header must grant access
	authHeaderReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	authHeaderReq.Header.Set("Authorization", "Bearer "+token)
	authHeaderRec := httptest.NewRecorder()
	mux.ServeHTTP(authHeaderRec, authHeaderReq)

	if authHeaderRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when token is in Authorization header, got %d", authHeaderRec.Code)
	}

	// Passing triage_session cookie must grant access
	cookieReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	cookieReq.AddCookie(&http.Cookie{
		Name:  "triage_session",
		Value: token,
	})
	cookieRec := httptest.NewRecorder()
	mux.ServeHTTP(cookieRec, cookieReq)

	if cookieRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when token is in triage_session cookie, got %d", cookieRec.Code)
	}
}

// TestSecurity_SEC003_AuthLogoutClearsCookie tests that logging out invalidates session cookie.
func TestSecurity_SEC003_AuthLogoutClearsCookie(t *testing.T) {
	_, mux, _ := setupSecurityTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for logout, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "triage_session" {
			found = true
			if c.MaxAge != -1 || c.Value != "" {
				t.Errorf("expected session cookie to be cleared (MaxAge -1, empty value), got MaxAge=%d, Value=%q", c.MaxAge, c.Value)
			}
		}
	}
	if !found {
		t.Errorf("expected Set-Cookie header clearing triage_session")
	}
}

// TestSecurity_SEC004_RoleHierarchyEnforcement tests RBAC role restrictions across endpoints.
func TestSecurity_SEC004_RoleHierarchyEnforcement(t *testing.T) {
	_, mux, secret := setupSecurityTestServer(t)

	viewerToken := createTestToken(t, secret, "Viewer")
	devToken := createTestToken(t, secret, "Developer")
	adminToken := createTestToken(t, secret, "Admin")
	ownerToken := createTestToken(t, secret, "Owner")

	// 1. Viewer cannot create issues or PRs (403 Forbidden)
	viewerPRReq := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", bytes.NewReader([]byte(`{"incident_id":"inc_1"}`)))
	viewerPRReq.Header.Set("Authorization", "Bearer "+viewerToken)
	viewerPRRec := httptest.NewRecorder()
	mux.ServeHTTP(viewerPRRec, viewerPRReq)

	if viewerPRRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for Viewer create-pr, got %d", viewerPRRec.Code)
	}

	// 2. Viewer cannot update project context (403 Forbidden)
	viewerCtxReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/context", bytes.NewReader([]byte(`{"repo":"test","context":"new context"}`)))
	viewerCtxReq.Header.Set("Authorization", "Bearer "+viewerToken)
	viewerCtxRec := httptest.NewRecorder()
	mux.ServeHTTP(viewerCtxRec, viewerCtxReq)

	if viewerCtxRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for Viewer project context update, got %d", viewerCtxRec.Code)
	}

	// 3. Developer cannot administer team roles (403 Forbidden)
	devRoleReq := httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", bytes.NewReader([]byte(`{"id":"usr_1","role":"Admin"}`)))
	devRoleReq.Header.Set("Authorization", "Bearer "+devToken)
	devRoleRec := httptest.NewRecorder()
	mux.ServeHTTP(devRoleRec, devRoleReq)

	if devRoleRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for Developer team role update, got %d", devRoleRec.Code)
	}

	// 4. Developer cannot invite team members (403 Forbidden)
	devInviteReq := httptest.NewRequest(http.MethodPost, "/api/v1/team/invites", bytes.NewReader([]byte(`{"github_username":"newdev","role":"Developer"}`)))
	devInviteReq.Header.Set("Authorization", "Bearer "+devToken)
	devInviteRec := httptest.NewRecorder()
	mux.ServeHTTP(devInviteRec, devInviteReq)

	if devInviteRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for Developer sending team invites, got %d", devInviteRec.Code)
	}

	// 5. Admin cannot assign Owner role (tested in HandleTeamMemberRole)
	adminRoleReq := httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", bytes.NewReader([]byte(`{"id":"usr_1","role":"Owner"}`)))
	adminRoleReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminRoleRec := httptest.NewRecorder()
	mux.ServeHTTP(adminRoleRec, adminRoleReq)

	if adminRoleRec.Code != http.StatusForbidden && adminRoleRec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 403 Forbidden when Admin tries to assign Owner role, got %d", adminRoleRec.Code)
	}

	_ = ownerToken
}

// TestSecurity_SEC005_PRTargetFileSafetyPolicy tests validation of target commit paths in PRs.
func TestSecurity_SEC005_PRTargetFileSafetyPolicy(t *testing.T) {
	disallowedPaths := []string{
		".github/workflows/deploy.yml",
		".github/workflows/ci.yml",
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.prod.yaml",
		".env",
		".env.production",
		"secrets.json",
		"id_rsa",
		"id_ed25519",
		"server.key",
		"cert.pem",
		"../outside/file.go",
		"/etc/passwd",
	}

	for _, path := range disallowedPaths {
		t.Run("Disallowed_"+path, func(t *testing.T) {
			if err := ValidatePatchTargetFile(path); err == nil {
				t.Errorf("expected ValidatePatchTargetFile to reject %q, but it passed", path)
			}
		})
	}

	allowedPaths := []string{
		"main.go",
		"internal/api/handlers.go",
		"pkg/util/strings.go",
		"cmd/server/main.go",
	}

	for _, path := range allowedPaths {
		t.Run("Allowed_"+path, func(t *testing.T) {
			if err := ValidatePatchTargetFile(path); err != nil {
				t.Errorf("expected ValidatePatchTargetFile to accept %q, but got error: %v", path, err)
			}
		})
	}
}

// TestSecurity_SEC005_PRSafetyGatesFailClosed tests that create-pr fails closed on missing repo.
func TestSecurity_SEC005_PRSafetyGatesFailClosed(t *testing.T) {
	_, mux, secret := setupSecurityTestServer(t)
	devToken := createTestToken(t, secret, "Developer")

	// Missing incident -> 404
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", bytes.NewReader([]byte(`{"incident_id":"nonexistent_inc_999"}`)))
	req.Header.Set("Authorization", "Bearer "+devToken)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound && rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 404 Not Found for nonexistent incident, got %d", rec.Code)
	}
}
