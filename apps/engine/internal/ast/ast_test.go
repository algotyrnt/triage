// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFuncAST(t *testing.T) {
	tempDir := t.TempDir()
	sampleFile := filepath.Join(tempDir, "sample.go")

	sampleCode := `package sample

import "fmt"

func HelperFunc() string {
	return "hello"
}

func TargetFunc(val *int) int {
	if val == nil {
		panic("nil pointer")
	}
	return *val
}
`

	err := os.WriteFile(sampleFile, []byte(sampleCode), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Line 11 is inside TargetFunc
	astStr, err := ExtractFuncAST(sampleFile, 11)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(astStr, "func TargetFunc(val *int) int") {
		t.Errorf("expected astStr to contain TargetFunc, got:\n%s", astStr)
	}

	if strings.Contains(astStr, "HelperFunc") {
		t.Errorf("expected astStr NOT to contain HelperFunc, got:\n%s", astStr)
	}
}

func TestExtractFuncASTFromBytes(t *testing.T) {
	sampleCode := []byte(`package sample

func CrashingFunc() {
	var p *int
	_ = *p
}
`)

	astStr, err := ExtractFuncASTFromBytes(sampleCode, 4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(astStr, "func CrashingFunc()") {
		t.Errorf("expected astStr to contain CrashingFunc, got %s", astStr)
	}
}

func TestASTCache(t *testing.T) {
	cache := NewASTCache()

	owner := "algotyrnt"
	repo := "triage"
	commit := "abc1234"
	filePath := "main.go"
	line := 42
	snippet := "func Panic() { panic(1) }"

	if _, found := cache.Get(owner, repo, commit, filePath, line); found {
		t.Errorf("expected cache miss initially")
	}

	cache.Set(owner, repo, commit, filePath, line, snippet)

	got, found := cache.Get(owner, repo, commit, filePath, line)
	if !found {
		t.Fatalf("expected cache hit")
	}

	if got != snippet {
		t.Errorf("expected cached snippet %s, got %s", snippet, got)
	}
}

func TestNormalizeMonorepoPath(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		rootDir  string
		expected string
	}{
		{
			name:     "root service (empty rootDir)",
			file:     "pkg/handler/user.go",
			rootDir:  "",
			expected: "pkg/handler/user.go",
		},
		{
			name:     "monorepo backend subfolder",
			file:     "pkg/handler/user.go",
			rootDir:  "backend",
			expected: "backend/pkg/handler/user.go",
		},
		{
			name:     "monorepo rootDir with leading and trailing slashes",
			file:     "/api/routes.go",
			rootDir:  "/services/engine/",
			expected: "services/engine/api/routes.go",
		},
		{
			name:     "file already contains rootDir prefix",
			file:     "backend/pkg/handler/user.go",
			rootDir:  "backend",
			expected: "backend/pkg/handler/user.go",
		},
		{
			name:     "file equals rootDir",
			file:     "backend",
			rootDir:  "backend",
			expected: "backend",
		},
		{
			name:     "dot rootDir",
			file:     "main.go",
			rootDir:  ".",
			expected: "main.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := NormalizeMonorepoPath(tc.file, tc.rootDir)
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
