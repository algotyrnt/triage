// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidAPIKey(t *testing.T) {
	ctx := context.Background()

	// 1. Unset TRIAGE_API_KEY should fail closed
	_ = os.Unsetenv("TRIAGE_API_KEY")
	if isValidAPIKey(ctx, "any_key") {
		t.Errorf("expected isValidAPIKey to fail closed when TRIAGE_API_KEY is unset")
	}

	// 2. Empty input key should return false
	_ = os.Setenv("TRIAGE_API_KEY", "tr_valid_key")
	defer os.Unsetenv("TRIAGE_API_KEY")

	if isValidAPIKey(ctx, "") {
		t.Errorf("expected isValidAPIKey to return false for empty key")
	}

	// 3. Valid matching key should return true
	if !isValidAPIKey(ctx, "tr_valid_key") {
		t.Errorf("expected isValidAPIKey to return true for matching key")
	}

	// 4. Mismatched key should return false
	if isValidAPIKey(ctx, "tr_wrong_key") {
		t.Errorf("expected isValidAPIKey to return false for wrong key")
	}
}

func TestValidateAndResolveFilePath_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	externalDir := filepath.Join(tmpDir, "external")
	if err := os.MkdirAll(externalDir, 0755); err != nil {
		t.Fatalf("failed to create external dir: %v", err)
	}

	externalFile := filepath.Join(externalDir, "secret.go")
	if err := os.WriteFile(externalFile, []byte("package external\n"), 0644); err != nil {
		t.Fatalf("failed to create external file: %v", err)
	}

	// Create symlink inside workspace pointing to external file outside workspace
	symlinkPath := filepath.Join(workspaceDir, "symlink.go")
	if err := os.Symlink(externalFile, symlinkPath); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	_ = os.Setenv("AST_WORKSPACE_ROOT", workspaceDir)
	defer os.Unsetenv("AST_WORKSPACE_ROOT")

	// Expect validation to reject target because its resolved symlink path lies outside workspace root
	_, err := validateAndResolveFilePath("symlink.go")
	if err == nil {
		t.Errorf("expected error for symlink pointing outside workspace root, got nil")
	}
}
