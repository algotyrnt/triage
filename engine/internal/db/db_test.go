// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewDB_SQLiteDefault(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	ctx := context.Background()

	database, err := NewDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to initialize embedded SQLite: %v", err)
	}
	defer database.Close()

	if database.SQL == nil || database.Path != dbPath {
		t.Errorf("expected initialized SQLite database at %s, got %+v", dbPath, database)
	}

	// Verify schema was provisioned and operations work
	err = database.SaveInstanceConfig(ctx, "test_key", "test_val")
	if err != nil {
		t.Fatalf("failed to save instance config: %v", err)
	}

	val, err := database.GetInstanceConfig(ctx, "test_key")
	if err != nil || val != "test_val" {
		t.Errorf("expected test_val, got (%s, %v)", val, err)
	}

	// Test user creation and retrieval
	u, err := database.UpsertUser(ctx, "12345", "testuser", "https://avatar.com/u")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	if u.Username != "testuser" || u.Role != "Owner" {
		t.Errorf("unexpected user: %+v", u)
	}

	// Test project and API key creation
	rawKey, repoID, err := database.CreateProject(ctx, "owner", "repo", "", "testuser")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	if rawKey == "" || repoID == "" {
		t.Errorf("expected non-empty key and repoID, got (%s, %s)", rawKey, repoID)
	}

	if !database.VerifyAPIKey(ctx, rawKey) {
		t.Errorf("expected raw API key to verify successfully")
	}

	// Test incident recording
	inc := &Incident{
		ID:           "INC-TEST-001",
		RepositoryID: repoID,
		Title:        "nil pointer dereference",
		File:         "main.go",
		Line:         42,
		PanicMessage: "runtime error: invalid memory address",
		StackTrace:   "goroutine 1 [running]:",
	}
	if err := database.SaveIncident(ctx, inc); err != nil {
		t.Fatalf("failed to save incident: %v", err)
	}

	fetchedInc, err := database.GetIncidentByID(ctx, "INC-TEST-001")
	if err != nil || fetchedInc == nil {
		t.Fatalf("failed to get incident: %v", err)
	}
	if fetchedInc.Title != "nil pointer dereference" {
		t.Errorf("unexpected incident title: %s", fetchedInc.Title)
	}
}

func TestDB_NilPoolFallbacks(t *testing.T) {
	db := &DB{SQL: nil}
	ctx := context.Background()

	if db.VerifyAPIKey(ctx, "test_key") {
		t.Errorf("expected VerifyAPIKey to return false for nil db")
	}

	if err := db.EnsureSchema(ctx); err == nil {
		t.Errorf("expected EnsureSchema to fail for nil db")
	}

	inc, err := db.FindActiveIncidentByFingerprint(ctx, "repo_123", "fp_123")
	if err != nil || inc != nil {
		t.Errorf("expected (nil, nil) for FindActiveIncidentByFingerprint on nil db, got (%v, %v)", inc, err)
	}

	if err := db.IncrementIncidentOccurrence(ctx, "INC-123"); err == nil {
		t.Errorf("expected IncrementIncidentOccurrence to fail for nil db")
	}

	if err := db.SaveIncident(ctx, &Incident{ID: "INC-123", File: "main.go", Line: 10, PanicMessage: "crash"}); err == nil {
		t.Errorf("expected SaveIncident to fail for nil db")
	}

	incList, err := db.GetIncidents(ctx, 10)
	if err != nil || len(incList) != 0 {
		t.Errorf("expected empty incident list for nil db, got %v, err=%v", incList, err)
	}

	stats, err := db.GetStats(ctx)
	if err != nil || stats["database"] != "unconnected (in-memory mode)" {
		t.Errorf("unexpected stats for nil db: %v, err=%v", stats, err)
	}
}
