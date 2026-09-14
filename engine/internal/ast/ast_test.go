// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "triage/engine/internal/github"
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

	// Test Function-wise range caching
	funcSnippet := "func ProcessOrder(id string) error {\n\t// line 50\n\t// line 55\n\treturn nil\n}"
	cache.SetFunction(owner, repo, commit, "pkg/order/order.go", "ProcessOrder", 45, 60, funcSnippet)

	// Any line in [45, 60] should hit the cache
	for _, targetLine := range []int{45, 48, 52, 58, 60} {
		fnGot, fnFound := cache.Get(owner, repo, commit, "pkg/order/order.go", targetLine)
		if !fnFound {
			t.Fatalf("expected function-wise cache hit for line %d", targetLine)
		}
		if fnGot != funcSnippet {
			t.Errorf("expected %s for line %d, got %s", funcSnippet, targetLine, fnGot)
		}
	}

	// Line outside range should miss
	if _, outFound := cache.Get(owner, repo, commit, "pkg/order/order.go", 61); outFound {
		t.Errorf("expected cache miss for line 61 outside function range")
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
		{
			name:     "nested monorepo with service base name prefix in stack file",
			file:     "order-service/pkg/orders/service.go",
			rootDir:  "test-services/order-service",
			expected: "test-services/order-service/pkg/orders/service.go",
		},
		{
			name:     "nested monorepo main.go with service base name prefix",
			file:     "order-service/main.go",
			rootDir:  "test-services/order-service",
			expected: "test-services/order-service/main.go",
		},
		{
			name:     "nested monorepo with clean subfolder path",
			file:     "pkg/orders/service.go",
			rootDir:  "test-services/order-service",
			expected: "test-services/order-service/pkg/orders/service.go",
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

func TestManagerAndNode(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "ast_test.db")

	mgr, err := NewManager(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to create AST Manager: %v", err)
	}
	defer mgr.Close()

	// 1. Save AST Node
	snippet := "func HandleCrash() { panic(\"boom\") }"
	err = mgr.SaveASTNode(ctx, "owner", "repo", "commit123", "pkg/handler.go", "HandleCrash", 15, 25, snippet)
	if err != nil {
		t.Fatalf("failed to save AST node: %v", err)
	}

	// 2. Empty snippet does not error
	if err := mgr.SaveASTNode(ctx, "owner", "repo", "commit123", "pkg/handler.go", "HandleCrash", 15, 25, ""); err != nil {
		t.Errorf("expected nil error for empty snippet: %v", err)
	}

	// 3. Get AST Node exact match
	node, err := mgr.GetASTNode(ctx, "owner", "repo", "commit123", "pkg/handler.go", 15)
	if err != nil || node == nil {
		t.Fatalf("failed to get AST node: %v", err)
	}
	if node.Snippet != snippet || node.StartLine != 15 {
		t.Errorf("unexpected node: %+v", node)
	}

	// 4. Get AST Node via Monorepo path candidate
	nodeCandidate, err := mgr.GetASTNode(ctx, "owner", "repo", "commit123", "handler.go", 15, "pkg")
	if err != nil || nodeCandidate == nil {
		t.Fatalf("failed to get AST node via monorepo candidate: %v", err)
	}
	if nodeCandidate.Snippet != snippet {
		t.Errorf("unexpected candidate snippet: %s", nodeCandidate.Snippet)
	}

	// 5. Not found returns sql.ErrNoRows
	_, err = mgr.GetASTNode(ctx, "owner", "repo", "commit123", "missing.go", 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows for missing node, got %v", err)
	}

	// 6. Nil DB checks
	nilMgr := NewManagerWithDB(nil)
	nilMgr.Close()
	if _, err := nilMgr.GetASTNode(ctx, "o", "r", "c", "f", 1); err == nil {
		t.Errorf("expected error from GetASTNode on nil DB")
	}
	if err := nilMgr.SaveASTNode(ctx, "o", "r", "c", "f", "fn", 1, 2, "code"); err == nil {
		t.Errorf("expected error from SaveASTNode on nil DB")
	}

	// 7. Node ID generation
	id1 := generateNodeID("owner", "repo", "commit", "file.go", 10)
	id2 := generateNodeID("owner", "repo", "commit", "file.go", 10)
	id3 := generateNodeID("owner", "repo", "commit", "file.go", 11)
	if !strings.HasPrefix(id1, "ast-") {
		t.Errorf("expected 'ast-' prefix on node ID, got %s", id1)
	}
	if id1 != id2 {
		t.Errorf("expected deterministic node ID, got %s vs %s", id1, id2)
	}
	if id1 == id3 {
		t.Errorf("expected different IDs for different lines, got same: %s", id1)
	}

	// 8. NewManager invalid path
	_, err = NewManager(ctx, "/dev/null/impossible/ast.db")
	if err == nil {
		t.Errorf("expected error from NewManager with impossible path")
	}
}

func TestOnDemandFetcher(t *testing.T) {
	ctx := context.Background()

	// 1. NewOnDemandFetcher without GitHubApp fails
	fetcher := NewOnDemandFetcher()
	_, err := fetcher.FetchFile(ctx, "owner", "repo", "commit", "main.go")
	if err == nil {
		t.Errorf("expected error fetching file without configured GitHubApp")
	}

	// 2. Fetcher with installation error
	fetcher.GetInstallationID = func(ctx context.Context, owner, repo string) (int64, error) {
		return 0, fmt.Errorf("no installation for %s/%s", owner, repo)
	}
	fetcher.GitHubApp = &gh.AppConfig{}

	_, err = fetcher.FetchFile(ctx, "owner", "repo", "commit", "main.go")
	if err == nil {
		t.Errorf("expected error when GetInstallationID fails")
	}
}
