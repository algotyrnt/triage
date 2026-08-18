// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFuncAST_SingleFile(t *testing.T) {
	tempDir := t.TempDir()
	sampleFile := filepath.Join(tempDir, "sample.go")

	sampleCode := `package sample

type Config struct {
	Timeout int
}

func TargetFunc(cfg *Config) int {
	if cfg == nil {
		panic("nil pointer")
	}
	return cfg.Timeout
}
`

	err := os.WriteFile(sampleFile, []byte(sampleCode), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Line 8 is inside TargetFunc
	astStr, err := ExtractFuncAST(sampleFile, 8)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(astStr, "func TargetFunc(cfg *Config) int") {
		t.Errorf("expected astStr to contain TargetFunc, got:\n%s", astStr)
	}

	if !strings.Contains(astStr, "type Config struct") {
		t.Errorf("expected astStr to contain Config struct definition, got:\n%s", astStr)
	}
}

func TestExtractFuncAST_MultiFilePackage(t *testing.T) {
	tempDir := t.TempDir()

	// 1. types.go: Defines UserHandler and Notifier structs
	typesCode := `package handler

import "net/http"

type SlackNotifier struct {
	WebhookURL string
	Client     *http.Client
}

type UserHandler struct {
	Notifier *SlackNotifier
	MaxUsers int
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "types.go"), []byte(typesCode), 0644); err != nil {
		t.Fatalf("failed to write types.go: %v", err)
	}

	// 2. init.go: Defines constructor NewUserHandler
	initCode := `package handler

func NewUserHandler() *UserHandler {
	return &UserHandler{
		Notifier: nil, // Intentionally nil
		MaxUsers: 100,
	}
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "init.go"), []byte(initCode), 0644); err != nil {
		t.Fatalf("failed to write init.go: %v", err)
	}

	// 3. helpers.go: Defines package helper and package var
	helpersCode := `package handler

var DefaultRole = "admin"

func validateRequest(action string) bool {
	return len(action) > 0
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "helpers.go"), []byte(helpersCode), 0644); err != nil {
		t.Fatalf("failed to write helpers.go: %v", err)
	}

	// 4. user.go: Contains crashing method UpdateUser
	userCode := `package handler

import "net/http"

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if validateRequest("update") {
		role := DefaultRole
		_ = role
		h.Notifier.WebhookURL = "https://hooks.slack.com/services/..." // PANIC
	}
}
`
	userFile := filepath.Join(tempDir, "user.go")
	if err := os.WriteFile(userFile, []byte(userCode), 0644); err != nil {
		t.Fatalf("failed to write user.go: %v", err)
	}

	// Line 9 is inside UpdateUser (where panic occurs)
	astStr, err := ExtractFuncAST(userFile, 9)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Target Function
	if !strings.Contains(astStr, "func (h *UserHandler) UpdateUser") {
		t.Errorf("missing target function UpdateUser in AST context:\n%s", astStr)
	}

	// Verify Receiver Struct from types.go
	if !strings.Contains(astStr, "type UserHandler struct") {
		t.Errorf("missing receiver struct UserHandler from types.go in AST context:\n%s", astStr)
	}

	// Verify Transitive Nested Struct from types.go
	if !strings.Contains(astStr, "type SlackNotifier struct") {
		t.Errorf("missing transitive struct SlackNotifier from types.go in AST context:\n%s", astStr)
	}

	// Verify Constructor from init.go
	if !strings.Contains(astStr, "func NewUserHandler() *UserHandler") {
		t.Errorf("missing constructor NewUserHandler from init.go in AST context:\n%s", astStr)
	}

	// Verify Helper Function from helpers.go
	if !strings.Contains(astStr, "func validateRequest(action string) bool") {
		t.Errorf("missing helper function validateRequest from helpers.go in AST context:\n%s", astStr)
	}

	// Verify Package Variable from helpers.go
	if !strings.Contains(astStr, "DefaultRole") {
		t.Errorf("missing package variable DefaultRole from helpers.go in AST context:\n%s", astStr)
	}
}

func TestExtractPackageContextASTFromBytes(t *testing.T) {
	files := map[string][]byte{
		"models.go": []byte(`package service

type Account struct {
	ID   string
	Name string
}
`),
		"service.go": []byte(`package service

type Service struct {
	acc *Account
}

func (s *Service) GetName() string {
	return s.acc.Name
}
`),
	}

	// Line 8 in service.go is inside GetName
	astStr, err := ExtractPackageContextASTFromBytes(files, "service.go", 8)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(astStr, "func (s *Service) GetName()") {
		t.Errorf("missing GetName in AST:\n%s", astStr)
	}

	if !strings.Contains(astStr, "type Service struct") {
		t.Errorf("missing Service struct in AST:\n%s", astStr)
	}

	if !strings.Contains(astStr, "type Account struct") {
		t.Errorf("missing Account struct in AST:\n%s", astStr)
	}
}

func TestExtractPackageContextAST_SliceAndMapTypes(t *testing.T) {
	files := map[string][]byte{
		"types.go": []byte(`package team

type Member struct {
	Name string
}

type Team struct {
	Members []*Member
	Roles   map[string]*Member
}
`),
		"team.go": []byte(`package team

func (t *Team) FirstMember() string {
	return t.Members[0].Name
}
`),
	}

	// Line 4 in team.go is inside FirstMember
	astStr, err := ExtractPackageContextASTFromBytes(files, "team.go", 4)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(astStr, "type Team struct") {
		t.Errorf("missing Team struct in AST:\n%s", astStr)
	}

	if !strings.Contains(astStr, "type Member struct") {
		t.Errorf("missing transitive Member struct from slice field in AST:\n%s", astStr)
	}
}

func TestExtractFuncASTFromBytes(t *testing.T) {
	sampleCode := []byte(`package sample

type Worker struct {
	channel chan int
}

func CrashingFunc(w *Worker) {
	w.channel <- 42
}
`)

	// Line 8 is inside CrashingFunc
	astStr, err := ExtractFuncASTFromBytes(sampleCode, 8)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(astStr, "func CrashingFunc(w *Worker)") {
		t.Errorf("expected astStr to contain CrashingFunc, got %s", astStr)
	}

	if !strings.Contains(astStr, "type Worker struct") {
		t.Errorf("expected astStr to contain Worker struct, got %s", astStr)
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
		{
			name:     "developer absolute path with rootDir",
			file:     "/Users/punjitha/projects/triage/test-service/main.go",
			rootDir:  "test-service",
			expected: "test-service/main.go",
		},
		{
			name:     "developer absolute path with monorepo apps/engine",
			file:     "/Users/punjitha/projects/triage/apps/engine/main.go",
			rootDir:  "apps/engine",
			expected: "apps/engine/main.go",
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
