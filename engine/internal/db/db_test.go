// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) (*DB, context.Context) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	ctx := context.Background()

	database, err := NewDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to initialize embedded SQLite: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})
	return database, ctx
}

func TestNewDB_SQLiteDefault(t *testing.T) {
	database, ctx := setupTestDB(t)

	if database.SQL == nil {
		t.Fatalf("expected initialized SQLite database")
	}

	// Verify schema was provisioned and operations work
	err := database.SaveInstanceConfig(ctx, "test_key", "test_val")
	if err != nil {
		t.Fatalf("failed to save instance config: %v", err)
	}

	val, err := database.GetInstanceConfig(ctx, "test_key")
	if err != nil || val != "test_val" {
		t.Errorf("expected test_val, got (%s, %v)", val, err)
	}

	stats, err := database.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get db stats: %v", err)
	}
	if stats["database"] != "connected (sqlite)" {
		t.Errorf("expected sqlite db type in stats, got %v", stats["database"])
	}
}

func TestDB_APIKeys_FullLifecycle(t *testing.T) {
	database, ctx := setupTestDB(t)

	// Create project first
	rawKey, repoID, err := database.CreateProject(ctx, "org", "api-repo", "backend", "testuser")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	if rawKey == "" || repoID == "" {
		t.Fatalf("expected non-empty key and repoID, got (%s, %s)", rawKey, repoID)
	}

	// Verify the key works
	if !database.VerifyAPIKey(ctx, rawKey) {
		t.Errorf("expected generated API key to verify successfully")
	}
	if database.VerifyAPIKey(ctx, "non_existent_key") {
		t.Errorf("expected non-existent key to fail verification")
	}
	if database.VerifyAPIKey(ctx, "") {
		t.Errorf("expected empty key to fail verification")
	}

	// Create a second key for the project
	createdKey, err := database.CreateAPIKey(ctx, "org", "api-repo", "backend", "Second Key")
	if err != nil {
		t.Fatalf("failed to create second API key: %v", err)
	}
	if createdKey.RawKey == "" || createdKey.ID == "" {
		t.Fatalf("invalid created key: %+v", createdKey)
	}

	// List keys
	keys, err := database.GetAPIKeys(ctx, "org", "api-repo", "backend")
	if err != nil {
		t.Fatalf("failed to get API keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	// Also test listing keys without owner/repo
	allKeys, err := database.GetAPIKeys(ctx, "", "", "")
	if err != nil {
		t.Fatalf("failed to get all API keys: %v", err)
	}
	if len(allKeys) < 2 {
		t.Errorf("expected at least 2 keys in all keys, got %d", len(allKeys))
	}

	// Verify repo lookup by API key
	repo, err := database.GetRepositoryByAPIKey(ctx, rawKey)
	if err != nil || repo == nil {
		t.Fatalf("failed to get repo by API key: %v", err)
	}
	if repo.Owner != "org" || repo.Repo != "api-repo" {
		t.Errorf("unexpected repo: %+v", repo)
	}

	// Lookup by non-existent key
	_, err = database.GetRepositoryByAPIKey(ctx, "bogus_key")
	if err == nil {
		t.Errorf("expected error looking up repository by bogus key")
	}

	// Revoke the second key
	if err := database.RevokeAPIKey(ctx, createdKey.ID); err != nil {
		t.Fatalf("failed to revoke key: %v", err)
	}

	// Verify the revoked key no longer verifies
	if database.VerifyAPIKey(ctx, createdKey.RawKey) {
		t.Errorf("expected revoked key to fail verification")
	}

	// List keys after revocation to test RevokedAt mapping
	keysAfterRevoke, err := database.GetAPIKeys(ctx, "org", "api-repo", "backend")
	if err != nil || len(keysAfterRevoke) != 2 {
		t.Errorf("expected 2 keys after revocation")
	}

	// Verify non-existent project returns empty keys
	emptyKeys, err := database.GetAPIKeys(ctx, "non", "existent", "")
	if err != nil || len(emptyKeys) != 0 {
		t.Errorf("expected empty keys list for non-existent repo, got %v, err=%v", emptyKeys, err)
	}

	// Create key on non-existent repo auto-provisions the repo
	autoKey, err := database.CreateAPIKey(ctx, "auto-org", "auto-repo", "", "Auto Key")
	if err != nil || autoKey == nil {
		t.Errorf("expected key to be created with auto-provisioned repo: %v", err)
	}
}

func TestDB_InstanceConfig_FullLifecycle(t *testing.T) {
	database, ctx := setupTestDB(t)

	// Initially not configured
	configured, err := database.IsInstanceConfigured(ctx)
	if err != nil {
		t.Fatalf("IsInstanceConfigured failed: %v", err)
	}
	if configured {
		t.Errorf("expected unconfigured instance initially")
	}

	// Save configs
	if err := database.SaveInstanceConfig(ctx, "app_url", "https://triage.dev"); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	if err := database.SaveInstanceConfig(ctx, "github_app_id", "12345"); err != nil {
		t.Fatalf("failed to save github_app_id: %v", err)
	}
	if err := database.SaveInstanceConfig(ctx, "github_oauth_client_id", "client_123"); err != nil {
		t.Fatalf("failed to save github_oauth_client_id: %v", err)
	}

	// Now configured
	configured, err = database.IsInstanceConfigured(ctx)
	if err != nil || !configured {
		t.Errorf("expected configured instance, got %v, err=%v", configured, err)
	}

	// Get specific config
	val, err := database.GetInstanceConfig(ctx, "app_url")
	if err != nil || val != "https://triage.dev" {
		t.Errorf("expected https://triage.dev, got %s, err=%v", val, err)
	}

	// Get non-existent config returns empty string, no error
	missing, err := database.GetInstanceConfig(ctx, "non_existent_setting")
	if err != nil || missing != "" {
		t.Errorf("expected empty string for missing config, got %s, err=%v", missing, err)
	}

	// Get all configs
	all, err := database.GetAllInstanceConfig(ctx)
	if err != nil {
		t.Fatalf("failed to get all configs: %v", err)
	}
	if all["app_url"] != "https://triage.dev" || all["github_app_id"] != "12345" || all["github_oauth_client_id"] != "client_123" {
		t.Errorf("unexpected all configs: %+v", all)
	}
}

func TestDB_GitHubInstallations_FullLifecycle(t *testing.T) {
	database, ctx := setupTestDB(t)

	// Save installation
	instID := int64(987654)
	err := database.SaveInstallation(ctx, instID, "octocat", 12345, "User")
	if err != nil {
		t.Fatalf("failed to save installation: %v", err)
	}

	// Get installation (latest active)
	inst, err := database.GetInstallation(ctx)
	if err != nil || inst == nil {
		t.Fatalf("failed to get installation: %v", err)
	}
	if inst.OrgLogin != "octocat" || inst.AccountType != "User" {
		t.Errorf("unexpected installation: %+v", inst)
	}

	// List all installations
	allInst, err := database.GetAllInstallations(ctx)
	if err != nil {
		t.Fatalf("failed to list installations: %v", err)
	}
	if len(allInst) != 1 {
		t.Errorf("expected 1 installation, got %d", len(allInst))
	}

	// Save installation repos (single and batch)
	if err := database.SaveInstallationRepo(ctx, instID, "octocat", "spoon-knife"); err != nil {
		t.Fatalf("failed to save single installation repo: %v", err)
	}

	batchRepos := []InstallationRepo{
		{
			Owner: "octocat",
			Repo:  "hello-world",
		},
	}
	if err := database.SaveInstallationRepos(ctx, instID, batchRepos); err != nil {
		t.Fatalf("failed to save batch installation repos: %v", err)
	}

	// Get installation repos
	repos, err := database.GetInstallationRepos(ctx, instID)
	if err != nil {
		t.Fatalf("failed to get installation repos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo after batch replace, got %d", len(repos))
	}

	// Save single again
	if err := database.SaveInstallationRepo(ctx, instID, "octocat", "spoon-knife"); err != nil {
		t.Fatalf("failed to save spoon-knife repo: %v", err)
	}

	// Get all installation repos
	allRepos, err := database.GetAllInstallationRepos(ctx)
	if err != nil {
		t.Fatalf("failed to get all installation repos: %v", err)
	}
	if len(allRepos) != 2 {
		t.Errorf("expected 2 repos across installations, got %d", len(allRepos))
	}

	// Get installation for specific repo
	lookedUpInstID, err := database.GetInstallationForRepo(ctx, "octocat", "spoon-knife")
	if err != nil || lookedUpInstID != instID {
		t.Fatalf("failed to get installation for repo: %v, got %d", err, lookedUpInstID)
	}
}

func TestDB_Incidents_FullLifecycle(t *testing.T) {
	database, ctx := setupTestDB(t)

	_, repoID, err := database.CreateProject(ctx, "org", "service-a", "", "testuser")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 1. Save incident
	inc := &Incident{
		ID:              "INC-001",
		RepositoryID:    repoID,
		Title:           "runtime error: invalid memory address",
		File:            "cmd/server/main.go",
		Line:            42,
		PanicMessage:    "nil pointer dereference",
		StackTrace:      "goroutine 1 [running]:\nmain.go:42",
		ASTSnippet:      "func handleRequest() { ... }",
		OccurrenceCount: 1,
		Fingerprint:     "fp_sha256_hash_123",
	}

	if err := database.SaveIncident(ctx, inc); err != nil {
		t.Fatalf("failed to save incident: %v", err)
	}

	// 2. Lookup by fingerprint
	foundInc, err := database.FindActiveIncidentByFingerprint(ctx, repoID, "fp_sha256_hash_123")
	if err != nil || foundInc == nil {
		t.Fatalf("failed to find active incident by fingerprint: %v", err)
	}
	if foundInc.ID != "INC-001" {
		t.Errorf("expected INC-001, got %s", foundInc.ID)
	}

	// Non-existent fingerprint
	missingInc, err := database.FindActiveIncidentByFingerprint(ctx, repoID, "missing_fp")
	if err != nil || missingInc != nil {
		t.Errorf("expected nil for missing fingerprint, got %v, err=%v", missingInc, err)
	}

	// 3. Increment occurrence
	if err := database.IncrementIncidentOccurrence(ctx, "INC-001"); err != nil {
		t.Fatalf("failed to increment occurrence: %v", err)
	}
	reloaded, err := database.GetIncidentByID(ctx, "INC-001")
	if err != nil || reloaded == nil {
		t.Fatalf("failed to get incident by ID: %v", err)
	}
	if reloaded.OccurrenceCount != 2 {
		t.Errorf("expected occurrence count 2, got %d", reloaded.OccurrenceCount)
	}

	// 4. Update incident issue
	if err := database.UpdateIncidentIssue(ctx, "INC-001", "https://github.com/org/service-a/issues/105", 105); err != nil {
		t.Fatalf("failed to update incident issue: %v", err)
	}

	// 5. Update incident patch
	if err := database.UpdateIncidentPatch(ctx, "INC-001", "diff --git ..."); err != nil {
		t.Fatalf("failed to update incident patch: %v", err)
	}

	// 6. Update incident PR
	if err := database.UpdateIncidentPR(ctx, "INC-001", "https://github.com/org/service-a/pull/42", 42, "new patch"); err != nil {
		t.Fatalf("failed to update incident PR: %v", err)
	}

	// Verify updates
	updated, _ := database.GetIncidentByID(ctx, "INC-001")
	if updated.GitHubIssueNumber != 105 {
		t.Errorf("expected issue 105, got %d", updated.GitHubIssueNumber)
	}
	if updated.GitHubPRNumber != 42 {
		t.Errorf("expected PR 42, got %d", updated.GitHubPRNumber)
	}
	if updated.SuggestedPatch != "new patch" {
		t.Errorf("unexpected patch: %+v", updated)
	}

	// 7. Get incidents list
	allIncidents, err := database.GetIncidents(ctx, 10)
	if err != nil || len(allIncidents) != 1 {
		t.Fatalf("expected 1 incident, got %d, err=%v", len(allIncidents), err)
	}

	// 8. Resolve incident
	if err := database.ResolveIncident(ctx, "INC-001"); err != nil {
		t.Fatalf("failed to resolve incident: %v", err)
	}
	resolved, _ := database.GetIncidentByID(ctx, "INC-001")
	if resolved.Status != "RESOLVED" {
		t.Errorf("expected status 'RESOLVED', got %s", resolved.Status)
	}

	// 9. After resolving, active fingerprint search returns nil
	resolvedFp, err := database.FindActiveIncidentByFingerprint(ctx, repoID, "fp_sha256_hash_123")
	if err != nil || resolvedFp != nil {
		t.Errorf("expected nil for resolved incident fingerprint search, got %v, err=%v", resolvedFp, err)
	}

	// 10. Non-existent incident by ID returns error
	missingByID, err := database.GetIncidentByID(ctx, "NON-EXISTENT")
	if err == nil || missingByID != nil {
		t.Errorf("expected error for non-existent incident ID, got %v, err=%v", missingByID, err)
	}

	// 11. Test SaveIncident with explicit ID
	inc2 := &Incident{
		ID:           "INC-002",
		RepositoryID: repoID,
		Title:        "crash without id",
		File:         "test.go",
		Line:         10,
		PanicMessage: "boom",
		StackTrace:   "stack",
	}
	if err := database.SaveIncident(ctx, inc2); err != nil {
		t.Fatalf("failed to save incident: %v", err)
	}
	fetchedInc2, err := database.GetIncidentByID(ctx, "INC-002")
	if err != nil || fetchedInc2 == nil {
		t.Fatalf("failed to retrieve INC-002: %v", err)
	}
}

func TestDB_RepositoriesAndProjects_FullLifecycle(t *testing.T) {
	database, ctx := setupTestDB(t)

	// Create projects
	_, id1, err := database.CreateProject(ctx, "org", "backend", "apps/backend", "alice")
	if err != nil {
		t.Fatalf("failed to create project 1: %v", err)
	}
	_, id2, err := database.CreateProject(ctx, "org", "frontend", "apps/frontend", "bob")
	if err != nil {
		t.Fatalf("failed to create project 2: %v", err)
	}

	// Update installation
	if err := database.UpdateRepositoryInstallation(ctx, "org", "backend", 555444); err != nil {
		t.Fatalf("failed to update repository installation: %v", err)
	}

	// Update project context
	if err := database.UpdateProjectContext(ctx, "org", "backend", "apps/backend", "Domain: Payment ledger microservice"); err != nil {
		t.Fatalf("failed to update project context: %v", err)
	}

	// List projects
	projects, err := database.GetProjects(ctx)
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// Get project by owner/repo
	p1, err := database.GetProjectByOwnerRepo(ctx, "org", "backend", "apps/backend")
	if err != nil || p1 == nil {
		t.Fatalf("failed to get project by owner/repo: %v", err)
	}
	if p1.ID != id1 || p1.Context != "Domain: Payment ledger microservice" {
		t.Errorf("unexpected project: %+v", p1)
	}
	if p1.InstallationID != 555444 {
		t.Errorf("expected installation 555444, got %v", p1.InstallationID)
	}

	// Non-existent project
	missing, err := database.GetProjectByOwnerRepo(ctx, "org", "unknown", "")
	if err == nil || missing != nil {
		t.Errorf("expected error for missing project, got %v, err=%v", missing, err)
	}

	_ = id2
}

func TestDB_UsersAndInvitations_FullLifecycle(t *testing.T) {
	database, ctx := setupTestDB(t)

	// Count initially 0
	count, err := database.CountUsers(ctx)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	// First user becomes Owner automatically via UpsertUser
	u1, err := database.UpsertUser(ctx, "1001", "alice", "https://avatar/alice.png")
	if err != nil {
		t.Fatalf("failed to upsert user 1: %v", err)
	}
	if u1.Role != "Owner" {
		t.Errorf("expected first user to be Owner, got %s", u1.Role)
	}

	// Second user with explicit role
	u2, err := database.UpsertUserWithRole(ctx, "1002", "bob", "bob@example.com", "https://avatar/bob.png", "Developer")
	if err != nil {
		t.Fatalf("failed to upsert user 2: %v", err)
	}
	if u2.Role != "Developer" {
		t.Errorf("expected Developer role, got %s", u2.Role)
	}

	// Update existing user via UpsertUserWithRole
	u2Updated, err := database.UpsertUserWithRole(ctx, "1002", "bob-updated", "bob@example.com", "https://avatar/bob2.png", "Developer")
	if err != nil {
		t.Fatalf("failed to update existing user: %v", err)
	}
	if u2Updated.Username != "bob-updated" {
		t.Errorf("expected bob-updated, got %s", u2Updated.Username)
	}

	// Get user by ID
	fetchedU1, err := database.GetUserByID(ctx, u1.ID)
	if err != nil || fetchedU1 == nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}
	if fetchedU1.Username != "alice" {
		t.Errorf("expected alice, got %s", fetchedU1.Username)
	}

	// List users
	users, err := database.ListUsers(ctx)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// Update user role
	if err := database.UpdateUserRole(ctx, u2.ID, "Admin"); err != nil {
		t.Fatalf("failed to update user role: %v", err)
	}
	fetchedU2, _ := database.GetUserByID(ctx, u2.ID)
	if fetchedU2.Role != "Admin" {
		t.Errorf("expected updated role Admin, got %s", fetchedU2.Role)
	}

	// Prevent updating role if only Owner
	err = database.UpdateUserRole(ctx, u1.ID, "Developer")
	if err == nil {
		t.Errorf("expected error updating sole owner's role")
	}

	// Prevent deleting sole owner
	err = database.DeleteUser(ctx, u1.ID)
	if err == nil {
		t.Errorf("expected error deleting sole owner")
	}

	// Delete regular user
	if err := database.DeleteUser(ctx, u2.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}
	usersAfterDelete, _ := database.ListUsers(ctx)
	if len(usersAfterDelete) != 1 {
		t.Errorf("expected 1 user after deletion, got %d", len(usersAfterDelete))
	}

	// Invitations
	inv, err := database.CreateInvitation(ctx, "charlie", "Developer", u1.ID)
	if err != nil {
		t.Fatalf("failed to create invitation: %v", err)
	}
	if inv.ID == "" || inv.GitHubUsername != "charlie" {
		t.Fatalf("unexpected invitation: %+v", inv)
	}

	// List invitations
	invList, err := database.ListInvitations(ctx)
	if err != nil {
		t.Fatalf("failed to list invitations: %v", err)
	}
	if len(invList) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(invList))
	}

	// Delete invitation
	if err := database.DeleteInvitation(ctx, inv.ID); err != nil {
		t.Fatalf("failed to delete invitation: %v", err)
	}
	invListAfter, _ := database.ListInvitations(ctx)
	if len(invListAfter) != 0 {
		t.Errorf("expected 0 invitations after delete, got %d", len(invListAfter))
	}

	// Non-existent user lookup
	missingUser, err := database.GetUserByID(ctx, "non-existent-user-id")
	if err == nil || missingUser != nil {
		t.Errorf("expected error for missing user, got %v, err=%v", missingUser, err)
	}
}

func TestSchema_Truncate(t *testing.T) {
	short := truncate("hello", 10)
	if short != "hello" {
		t.Errorf("expected 'hello', got %q", short)
	}
	long := truncate("hello world from triage", 5)
	if long != "hello..." {
		t.Errorf("expected 'hello...', got %q", long)
	}
}

func TestDB_NilPoolFallbacks(t *testing.T) {
	var nilDB *DB
	ctx := context.Background()

	// API Keys
	if nilDB.VerifyAPIKey(ctx, "test_key") {
		t.Errorf("expected VerifyAPIKey to return false for nil db")
	}
	keys, err := nilDB.GetAPIKeys(ctx, "o", "r", "")
	if err != nil || len(keys) != 0 {
		t.Errorf("expected empty list without error from GetAPIKeys on nil db")
	}
	if _, err := nilDB.CreateAPIKey(ctx, "o", "r", "k", "u"); err == nil {
		t.Errorf("expected error from CreateAPIKey on nil db")
	}
	if err := nilDB.RevokeAPIKey(ctx, "k1"); err == nil {
		t.Errorf("expected error from RevokeAPIKey on nil db")
	}
	if _, err := nilDB.GetRepositoryByAPIKey(ctx, "k"); err == nil {
		t.Errorf("expected error from GetRepositoryByAPIKey on nil db")
	}

	// Config
	if err := nilDB.SaveInstanceConfig(ctx, "k", "v"); err == nil {
		t.Errorf("expected error from SaveInstanceConfig on nil db")
	}
	if _, err := nilDB.GetInstanceConfig(ctx, "k"); err == nil {
		t.Errorf("expected error from GetInstanceConfig on nil db")
	}
	if _, err := nilDB.GetAllInstanceConfig(ctx); err == nil {
		t.Errorf("expected error from GetAllInstanceConfig on nil db")
	}
	if _, err := nilDB.IsInstanceConfigured(ctx); err == nil {
		t.Errorf("expected error from IsInstanceConfigured on nil db")
	}

	// Schema
	if err := nilDB.EnsureSchema(ctx); err == nil {
		t.Errorf("expected EnsureSchema to fail for nil db")
	}

	// Installations
	if err := nilDB.SaveInstallation(ctx, 1, "a", 1, "t"); err == nil {
		t.Errorf("expected error from SaveInstallation on nil db")
	}
	if _, err := nilDB.GetInstallation(ctx); err == nil {
		t.Errorf("expected error from GetInstallation on nil db")
	}
	if _, err := nilDB.GetAllInstallations(ctx); err == nil {
		t.Errorf("expected error from GetAllInstallations on nil db")
	}
	if _, err := nilDB.GetAllInstallationRepos(ctx); err == nil {
		t.Errorf("expected error from GetAllInstallationRepos on nil db")
	}
	if err := nilDB.SaveInstallationRepos(ctx, 1, nil); err == nil {
		t.Errorf("expected error from SaveInstallationRepos on nil db")
	}
	if err := nilDB.SaveInstallationRepo(ctx, 1, "o", "r"); err == nil {
		t.Errorf("expected error from SaveInstallationRepo on nil db")
	}
	if _, err := nilDB.GetInstallationRepos(ctx, 1); err == nil {
		t.Errorf("expected error from GetInstallationRepos on nil db")
	}
	if _, err := nilDB.GetInstallationForRepo(ctx, "o", "r"); err == nil {
		t.Errorf("expected error from GetInstallationForRepo on nil db")
	}

	// Incidents
	inc, err := nilDB.FindActiveIncidentByFingerprint(ctx, "repo_123", "fp_123")
	if err != nil || inc != nil {
		t.Errorf("expected (nil, nil) for FindActiveIncidentByFingerprint on nil db, got (%v, %v)", inc, err)
	}
	if err := nilDB.IncrementIncidentOccurrence(ctx, "INC-123"); err == nil {
		t.Errorf("expected IncrementIncidentOccurrence to fail for nil db")
	}
	if err := nilDB.SaveIncident(ctx, &Incident{ID: "INC-123", File: "main.go", Line: 10, PanicMessage: "crash"}); err == nil {
		t.Errorf("expected SaveIncident to fail for nil db")
	}
	if err := nilDB.UpdateIncidentIssue(ctx, "INC-123", "http", 1); err == nil {
		t.Errorf("expected error from UpdateIncidentIssue on nil db")
	}
	if err := nilDB.UpdateIncidentPatch(ctx, "INC-123", "p"); err == nil {
		t.Errorf("expected error from UpdateIncidentPatch on nil db")
	}
	if err := nilDB.ResolveIncident(ctx, "INC-123"); err == nil {
		t.Errorf("expected error from ResolveIncident on nil db")
	}
	if err := nilDB.UpdateIncidentPR(ctx, "INC-123", "http", 1, "p"); err == nil {
		t.Errorf("expected error from UpdateIncidentPR on nil db")
	}
	incList, err := nilDB.GetIncidents(ctx, 10)
	if err != nil || len(incList) != 0 {
		t.Errorf("expected empty incident list for nil db, got %v, err=%v", incList, err)
	}
	if _, err := nilDB.GetIncidentByID(ctx, "INC-123"); err == nil {
		t.Errorf("expected error from GetIncidentByID on nil db")
	}

	// Repositories / Projects
	if _, _, err := nilDB.CreateProject(ctx, "o", "r", "d", "u"); err == nil {
		t.Errorf("expected error from CreateProject on nil db")
	}
	if err := nilDB.UpdateRepositoryInstallation(ctx, "o", "r", 1); err == nil {
		t.Errorf("expected error from UpdateRepositoryInstallation on nil db")
	}
	if err := nilDB.UpdateProjectContext(ctx, "o", "r", "d", "ctx"); err == nil {
		t.Errorf("expected error from UpdateProjectContext on nil db")
	}
	if _, err := nilDB.GetProjects(ctx); err != nil {
		t.Errorf("unexpected error from GetProjects on nil db: %v", err)
	}
	if _, err := nilDB.GetProjectByOwnerRepo(ctx, "o", "r", "d"); err == nil {
		t.Errorf("expected error from GetProjectByOwnerRepo on nil db")
	}

	// Users & Invitations
	if _, err := nilDB.CountUsers(ctx); err != nil {
		t.Errorf("unexpected error from CountUsers on nil db: %v", err)
	}
	if _, err := nilDB.UpsertUserWithRole(ctx, "1", "u", "e", "a", "r"); err != nil {
		t.Errorf("unexpected error from UpsertUserWithRole on nil db: %v", err)
	}
	if _, err := nilDB.GetUserByID(ctx, "1"); err != nil {
		t.Errorf("unexpected error from GetUserByID on nil db: %v", err)
	}
	if _, err := nilDB.ListUsers(ctx); err != nil {
		t.Errorf("unexpected error from ListUsers on nil db: %v", err)
	}
	if err := nilDB.UpdateUserRole(ctx, "1", "Admin"); err != nil {
		t.Errorf("unexpected error from UpdateUserRole on nil db: %v", err)
	}
	if err := nilDB.DeleteUser(ctx, "1"); err != nil {
		t.Errorf("unexpected error from DeleteUser on nil db: %v", err)
	}
	if _, err := nilDB.CreateInvitation(ctx, "e", "r", "i"); err != nil {
		t.Errorf("unexpected error from CreateInvitation on nil db: %v", err)
	}
	if _, err := nilDB.ListInvitations(ctx); err != nil {
		t.Errorf("unexpected error from ListInvitations on nil db: %v", err)
	}
	if err := nilDB.DeleteInvitation(ctx, "t"); err != nil {
		t.Errorf("unexpected error from DeleteInvitation on nil db: %v", err)
	}

	// Stats
	stats, err := nilDB.GetStats(ctx)
	if err != nil || stats["database"] != "unconnected (in-memory mode)" {
		t.Errorf("unexpected stats for nil db: %v, err=%v", stats, err)
	}
}

func TestDB_EdgeCasesAndBranches(t *testing.T) {
	database, ctx := setupTestDB(t)

	// 1. Fresh DB GetInstallation (ErrNoRows branch)
	inst, err := database.GetInstallation(ctx)
	if err != nil || inst != nil {
		t.Errorf("expected (nil, nil) for fresh DB installation query, got %v, err=%v", inst, err)
	}

	// 2. NewDB invalid path error branch
	_, err = NewDB(ctx, "/dev/null/impossible_dir/triage.db")
	if err == nil {
		t.Errorf("expected error initializing DB with impossible path")
	}

	// 3. CreateProject duplicate ON CONFLICT
	_, _, err = database.CreateProject(ctx, "acme", "core", "api", "alice")
	if err != nil {
		t.Fatalf("first CreateProject failed: %v", err)
	}
	_, _, err = database.CreateProject(ctx, "acme", "core", "api", "bob")
	if err != nil {
		t.Errorf("duplicate CreateProject should succeed idempotently: %v", err)
	}

	// 4. CreateAPIKey with empty name
	keyWithDefaultName, err := database.CreateAPIKey(ctx, "acme", "core", "api", "")
	if err != nil || keyWithDefaultName == nil {
		t.Fatalf("failed to create key with default name: %v", err)
	}

	// 5. RevokeAPIKey non-existent key returns error
	err = database.RevokeAPIKey(ctx, "non_existent_key_id")
	if err == nil {
		t.Errorf("expected error when revoking non-existent key")
	}

	// 6. SaveIncident with CRITICAL severity and update ON CONFLICT
	inc := &Incident{
		ID:           "INC-EDGE-01",
		Title:        "Critical Crash",
		File:         "server.go",
		Line:         55,
		PanicMessage: "fatal error",
		StackTrace:   "stack",
		Severity:     "critical",
	}
	if err := database.SaveIncident(ctx, inc); err != nil {
		t.Fatalf("SaveIncident critical failed: %v", err)
	}
	// Update same incident with invalid severity (gets normalized)
	inc.Severity = "bogus_severity"
	inc.OccurrenceCount = 5
	if err := database.SaveIncident(ctx, inc); err != nil {
		t.Fatalf("SaveIncident update failed: %v", err)
	}

	// 7. GetIncidents with limit <= 0
	listDefaultLimit, err := database.GetIncidents(ctx, 0)
	if err != nil || len(listDefaultLimit) == 0 {
		t.Errorf("GetIncidents limit 0 should default to 50: %v", err)
	}

	// 8. UpdateUserRole with invalid role name
	err = database.UpdateUserRole(ctx, "some_id", "SuperHero")
	if err == nil {
		t.Errorf("expected error for invalid role name")
	}

	// 9. UpdateUserRole for non-existent user
	err = database.UpdateUserRole(ctx, "non_existent_usr_id", "Admin")
	if err == nil {
		t.Errorf("expected error for non-existent user update")
	}

	// 10. DeleteUser for non-existent user
	err = database.DeleteUser(ctx, "non_existent_usr_id")
	if err == nil {
		t.Errorf("expected error for non-existent user deletion")
	}

	// 11. CreateInvitation with invalid role fallback
	u, err := database.UpsertUser(ctx, "9999", "admin_user", "")
	if err != nil {
		t.Fatalf("failed to upsert user for invitation test: %v", err)
	}
	inv, err := database.CreateInvitation(ctx, "david", "UnauthorizedRole", u.ID)
	if err != nil || inv == nil || inv.Role != "Developer" {
		t.Errorf("expected fallback to Developer role, got %+v, err=%v", inv, err)
	}

	// 12. UpdateRepositoryInstallation invalid installation ID
	err = database.UpdateRepositoryInstallation(ctx, "acme", "core", 0)
	if err == nil {
		t.Errorf("expected error for invalid installation ID")
	}

	// 13. Empty GetInstallationRepos
	emptyRepos, err := database.GetInstallationRepos(ctx, 999999)
	if err != nil || len(emptyRepos) != 0 {
		t.Errorf("expected empty repos list, got %v, err=%v", emptyRepos, err)
	}

	// 14. Fresh DB empty query branches
	freshDB, freshCtx := setupTestDB(t)
	freshConfigs, err := freshDB.GetAllInstanceConfig(freshCtx)
	if err != nil || len(freshConfigs) != 0 {
		t.Errorf("expected 0 configs in fresh DB, got %v, err=%v", freshConfigs, err)
	}
	emptyProjects, err := freshDB.GetProjects(freshCtx)
	if err != nil || len(emptyProjects) != 0 {
		t.Errorf("expected 0 projects in fresh DB, got %v, err=%v", emptyProjects, err)
	}
	emptyIncidents, err := freshDB.GetIncidents(freshCtx, 10)
	if err != nil || len(emptyIncidents) != 0 {
		t.Errorf("expected 0 incidents in fresh DB, got %v, err=%v", emptyIncidents, err)
	}
	emptyAllInst, err := freshDB.GetAllInstallations(freshCtx)
	if err != nil || len(emptyAllInst) != 0 {
		t.Errorf("expected 0 installations in fresh DB, got %v, err=%v", emptyAllInst, err)
	}
	emptyAllRepos, err := freshDB.GetAllInstallationRepos(freshCtx)
	if err != nil || len(emptyAllRepos) != 0 {
		t.Errorf("expected 0 installation repos in fresh DB, got %v, err=%v", emptyAllRepos, err)
	}
	emptyUsersList, err := freshDB.ListUsers(freshCtx)
	if err != nil || len(emptyUsersList) != 0 {
		t.Errorf("expected 0 users in fresh DB, got %v, err=%v", emptyUsersList, err)
	}
	emptyInvsList, err := freshDB.ListInvitations(freshCtx)
	if err != nil || len(emptyInvsList) != 0 {
		t.Errorf("expected 0 invitations in fresh DB, got %v, err=%v", emptyInvsList, err)
	}
	emptyKeysList, err := freshDB.GetAPIKeys(freshCtx, "", "", "")
	if err != nil || len(emptyKeysList) != 0 {
		t.Errorf("expected 0 keys in fresh DB, got %v, err=%v", emptyKeysList, err)
	}

	// 15. Incident without repository ID
	incNoRepo := &Incident{
		ID:           "INC-NO-REPO",
		Title:        "No Repo Crash",
		File:         "test.go",
		Line:         1,
		PanicMessage: "crash",
		StackTrace:   "stack",
		Fingerprint:  "fp_no_repo",
	}
	if err := database.SaveIncident(ctx, incNoRepo); err != nil {
		t.Fatalf("SaveIncident no repo failed: %v", err)
	}
	foundNoRepo, err := database.FindActiveIncidentByFingerprint(ctx, "", "fp_no_repo")
	if err != nil || foundNoRepo == nil {
		t.Errorf("expected to find incident with empty repo ID")
	}

	// 16. Expired API key branch in GetAPIKeys
	_, err = database.SQL.ExecContext(ctx, `
		INSERT INTO api_keys (id, repository_id, name, key_hash, key_masked, expires_at)
		VALUES ('key_exp', 'repo_123', 'Expired Key', 'hash_exp', '...exp', datetime('now', '-1 day'))
	`)
	if err == nil {
		_, _ = database.GetAPIKeys(ctx, "", "", "")
	}

	// 17. UpsertUserWithRole updating existing user without email/avatar
	u1Updated, err := database.UpsertUserWithRole(ctx, "1001", "alice-updated", "alice@example.com", "https://avatar/alice2.png", "Owner")
	if err != nil || u1Updated.Username != "alice-updated" {
		t.Errorf("expected updated user 1001, got %+v, err=%v", u1Updated, err)
	}

	// 18. NewDB path prefixes (sqlite://, sqlite:, file:)
	tmpDir := t.TempDir()
	dbPrefixed, err := NewDB(ctx, "sqlite://"+filepath.Join(tmpDir, "p1.db"))
	if err != nil {
		t.Fatalf("failed NewDB with sqlite:// prefix: %v", err)
	}
	dbPrefixed.Close()

	dbPrefixed2, err := NewDB(ctx, "sqlite:"+filepath.Join(tmpDir, "p2.db"))
	if err != nil {
		t.Fatalf("failed NewDB with sqlite: prefix: %v", err)
	}
	dbPrefixed2.Close()

	dbPrefixed3, err := NewDB(ctx, "file:"+filepath.Join(tmpDir, "p3.db"))
	if err != nil {
		t.Fatalf("failed NewDB with file: prefix: %v", err)
	}
	dbPrefixed3.Close()

	// 19. CreateAPIKey for repo with existing installation
	_ = database.SaveInstallation(ctx, 987654, "installed-org", 111, "Organization")
	_ = database.SaveInstallationRepo(ctx, 987654, "installed-org", "installed-repo")
	keyWithInst, err := database.CreateAPIKey(ctx, "installed-org", "installed-repo", "", "Inst Key")
	if err != nil || keyWithInst == nil {
		t.Fatalf("failed to create API key with existing installation: %v", err)
	}

	// 20. NewDB with empty path defaults to data/triage.db
	emptyDB, err := NewDB(ctx, "")
	if err == nil {
		emptyDB.Close()
	}

	// 21. Invitation without inviter (NULL invited_by)
	_, _ = database.SQL.ExecContext(ctx, `
		INSERT INTO invitations (id, github_username, role, invited_by, created_at)
		VALUES ('inv_no_by', 'no_inviter_user', 'Developer', NULL, CURRENT_TIMESTAMP)
	`)
	_, _ = database.ListInvitations(ctx)

	// 22. FindActiveIncidentByFingerprint with empty fingerprint
	incEmptyFp, err := database.FindActiveIncidentByFingerprint(ctx, "repo_1", "")
	if err != nil || incEmptyFp != nil {
		t.Errorf("expected (nil, nil) for empty fingerprint")
	}

	// 23. SaveIncident with full metadata
	fullInc := &Incident{
		ID:              "INC-FULL",
		RepositoryID:    "",
		Title:           "Full Incident",
		Status:          "OPEN",
		Severity:        "HIGH",
		AIProvider:      "openai",
		AIModel:         "gpt-4o",
		File:            "cmd/app.go",
		Line:            100,
		PanicMessage:    "panic",
		StackTrace:      "stack",
		ASTSnippet:      "ast",
		RootCause:       "root cause",
		SuggestedFix:    "fix",
		SuggestedPatch:  "patch",
		OccurrenceCount: 1,
		Fingerprint:     "fp_full",
	}
	if err := database.SaveIncident(ctx, fullInc); err != nil {
		t.Fatalf("SaveIncident full failed: %v", err)
	}

	// 24. Canceled context branches in EnsureSchema and NewDB
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := database.EnsureSchema(canceledCtx); err == nil {
		t.Errorf("expected error from EnsureSchema with canceled context")
	}
	if _, err := NewDB(canceledCtx, filepath.Join(t.TempDir(), "canceled.db")); err == nil {
		t.Errorf("expected error from NewDB with canceled context")
	}
}
