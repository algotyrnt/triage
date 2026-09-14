// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"triage/engine/internal/db"
)

func TestHandleTeamMembers(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Database not connected
	noDB := NewServer(Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/members", nil)
	rec := httptest.NewRecorder()
	noDB.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for no DB, got %d", rec.Code)
	}

	// 2. GET members
	req = httptest.NewRequest(http.MethodGet, "/api/v1/team/members", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET members, got %d", rec.Code)
	}
	var listRes struct {
		Members []db.User `json:"members"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&listRes)
	if len(listRes.Members) < 3 {
		t.Errorf("expected at least 3 members (owner, dev, viewer), got %d", len(listRes.Members))
	}

	// 3. DELETE member - Non-owner forbidden
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/members?id=usr_viewer_1", nil)
	req.Header.Set("Authorization", "Bearer "+env.DevToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner DELETE, got %d", rec.Code)
	}

	// 4. DELETE member - Missing member ID
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/members", nil)
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing id, got %d", rec.Code)
	}

	// 5. DELETE member - Cannot remove yourself
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/members?id="+env.OwnerUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for deleting self, got %d", rec.Code)
	}

	// 6. DELETE member - Success
	// Create another member to delete
	userToDelete, _ := env.DB.UpsertUserWithRole(ctx, "9999", "temp_user", "temp@example.com", "", "Viewer")
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/members?id="+userToDelete.ID, nil)
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for delete member, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Method not allowed (POST)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/team/members", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMembers(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST team members, got %d", rec.Code)
	}
}

func TestHandleTeamMemberRole(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Method not allowed (GET)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/members/role", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleTeamMemberRole(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", rec.Code)
	}

	// 2. Database not connected
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", nil)
	rec = httptest.NewRecorder()
	noDB.HandleTeamMemberRole(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for no DB, got %d", rec.Code)
	}

	// 3. Forbidden: Viewer cannot update role
	req = httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", nil)
	req.Header.Set("Authorization", "Bearer "+env.ViewerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMemberRole(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for Viewer, got %d", rec.Code)
	}

	// 4. Invalid request body
	req = httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", strings.NewReader(`bad json`))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMemberRole(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad JSON, got %d", rec.Code)
	}

	// 5. Admin role user cannot promote anyone to Owner
	adminUser, _ := env.DB.UpsertUserWithRole(ctx, "5555", "admin_user", "admin@example.com", "", "Admin")
	adminToken, _ := GenerateUserJWT(adminUser, env.Secret)

	promotePayload, _ := json.Marshal(map[string]string{
		"id":   env.DevUser.ID,
		"role": "Owner",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", bytes.NewReader(promotePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMemberRole(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when non-owner tries to promote to Owner, got %d", rec.Code)
	}

	// 6. Success updating role (Owner promotes Dev to Admin)
	validPayload, _ := json.Marshal(map[string]string{
		"id":   env.DevUser.ID,
		"role": "Admin",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/team/members/role", bytes.NewReader(validPayload))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamMemberRole(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for role update, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTeamInvites(t *testing.T) {
	env := setupAPITestEnv(t)

	// 1. Database not connected
	noDB := NewServer(Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/invites", nil)
	rec := httptest.NewRecorder()
	noDB.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for no DB, got %d", rec.Code)
	}

	// 2. GET /api/v1/team/invites
	req = httptest.NewRequest(http.MethodGet, "/api/v1/team/invites", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET invites, got %d", rec.Code)
	}
	var getRes struct {
		Invitations []db.Invitation `json:"invitations"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&getRes)
	if len(getRes.Invitations) != 0 {
		t.Errorf("expected 0 initial invites, got %d", len(getRes.Invitations))
	}

	// 3. POST /api/v1/team/invites - Forbidden for Viewer
	req = httptest.NewRequest(http.MethodPost, "/api/v1/team/invites", strings.NewReader(`{"github_username":"testguy"}`))
	req.Header.Set("Authorization", "Bearer "+env.ViewerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for Viewer invite, got %d", rec.Code)
	}

	// 4. POST /api/v1/team/invites - Missing/invalid username
	req = httptest.NewRequest(http.MethodPost, "/api/v1/team/invites", strings.NewReader(`{"role":"Developer"}`))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty username, got %d", rec.Code)
	}

	// 5. POST /api/v1/team/invites - Success
	invitePayload, _ := json.Marshal(map[string]string{
		"github_username": "@newdeveloper",
		"role":            "Developer",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/team/invites", bytes.NewReader(invitePayload))
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for invite, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdRes struct {
		Status     string        `json:"status"`
		Invitation db.Invitation `json:"invitation"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&createdRes)
	invID := createdRes.Invitation.ID
	if invID == "" {
		t.Fatalf("expected non-empty invitation ID")
	}

	// 6. DELETE /api/v1/team/invites - Forbidden for Viewer
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/invites?id="+invID, nil)
	req.Header.Set("Authorization", "Bearer "+env.ViewerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for Viewer delete invite, got %d", rec.Code)
	}

	// 7. DELETE /api/v1/team/invites - Missing ID
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/invites", nil)
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing invite ID, got %d", rec.Code)
	}

	// 8. DELETE /api/v1/team/invites - Success
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/team/invites?id="+invID, nil)
	req.Header.Set("Authorization", "Bearer "+env.OwnerToken)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for delete invite, got %d: %s", rec.Code, rec.Body.String())
	}

	// 9. Method not allowed (PATCH)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/team/invites", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleTeamInvites(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for PATCH, got %d", rec.Code)
	}
}
