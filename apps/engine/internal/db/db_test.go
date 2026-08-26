// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"testing"
)

func TestNewDB_MissingURL(t *testing.T) {
	_, err := NewDB(context.Background(), "")
	if err == nil {
		t.Errorf("expected error when databaseURL is empty, got nil")
	}
}

func TestDB_NilPoolFallbacks(t *testing.T) {
	db := &DB{Pool: nil}
	ctx := context.Background()

	if db.VerifyAPIKey(ctx, "test_key") {
		t.Errorf("expected VerifyAPIKey to return false for nil pool")
	}

	if err := db.EnsureSchema(ctx); err == nil {
		t.Errorf("expected EnsureSchema to fail for nil pool")
	}

	inc, err := db.FindActiveIncidentByFingerprint(ctx, "repo_123", "fp_123")
	if err != nil || inc != nil {
		t.Errorf("expected (nil, nil) for FindActiveIncidentByFingerprint on nil pool, got (%v, %v)", inc, err)
	}

	if err := db.IncrementIncidentOccurrence(ctx, "INC-123"); err == nil {
		t.Errorf("expected IncrementIncidentOccurrence to fail for nil pool")
	}

	if err := db.SaveIncident(ctx, &Incident{ID: "INC-123", File: "main.go", Line: 10, PanicMessage: "crash"}); err == nil {
		t.Errorf("expected SaveIncident to fail for nil pool")
	}

	incList, err := db.GetIncidents(ctx, 10)
	if err != nil || len(incList) != 0 {
		t.Errorf("expected empty incident list for nil pool, got %v, err=%v", incList, err)
	}

	stats, err := db.GetStats(ctx)
	if err != nil || stats["database"] != "unconnected (in-memory mode)" {
		t.Errorf("unexpected stats for nil pool: %v, err=%v", stats, err)
	}
}
