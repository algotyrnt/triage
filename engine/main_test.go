// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRun_InvalidArgs(t *testing.T) {
	ctx := context.Background()
	err := run(ctx, []string{"-nonexistent-flag"}, nil, false)
	if err == nil {
		t.Fatalf("expected error on invalid flag arguments")
	}
}

func TestRun_InvalidDatabasePath(t *testing.T) {
	ctx := context.Background()
	// An empty directory or non-existent parent path with trailing slash on SQLite file
	err := run(ctx, []string{"-db", "/dev/null/cannot_create_dir/triage.db"}, nil, false)
	if err == nil {
		t.Fatalf("expected error for invalid database file path")
	}
}

func TestRun_SuccessInit(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "main_test.db")

	args := []string{
		"-port", "8089",
		"-data-dir", tempDir,
		"-db", dbFile,
		"-log-level", "debug",
	}

	err := run(ctx, args, nil, false)
	if err != nil {
		t.Fatalf("expected successful initialization, got: %v", err)
	}

	// Verify database file was created
	if _, statErr := os.Stat(dbFile); statErr != nil {
		t.Errorf("expected database file to exist on disk: %v", statErr)
	}
}

func TestRun_WithVersionVariables(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "main_version_test.db")

	version = "v1.2.3"
	commit = "abc1234"
	date = "2026-09-14"
	defer func() {
		version = ""
		commit = ""
		date = ""
	}()

	args := []string{
		"-port", "8091",
		"-db", dbFile,
	}

	err := run(ctx, args, nil, false)
	if err != nil {
		t.Fatalf("expected successful run with version flags: %v", err)
	}
}

func TestRun_GracefulShutdown(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "main_shutdown_test.db")

	// Pick a high random port to avoid conflicts
	port := fmt.Sprintf("%d", 20000+time.Now().UnixNano()%10000)

	args := []string{
		"-port", port,
		"-db", dbFile,
	}

	stopChan := make(chan os.Signal, 1)

	// Send signal after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		stopChan <- syscall.SIGTERM
	}()

	err := run(ctx, args, stopChan, true)
	if err != nil {
		t.Fatalf("expected clean graceful shutdown, got: %v", err)
	}
}
