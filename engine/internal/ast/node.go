// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"triage/engine/internal/db"
)

type ASTNode struct {
	ID        string    `json:"id"`
	Owner     string    `json:"owner"`
	Repo      string    `json:"repo"`
	Commit    string    `json:"commit"`
	FilePath  string    `json:"file_path"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Snippet   string    `json:"snippet"`
	CreatedAt time.Time `json:"created_at"`
}

type Manager struct {
	db *db.DB
}

func NewManager(ctx context.Context, databaseURL string) (*Manager, error) {
	database, err := db.NewDB(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Manager{db: database}, nil
}

func NewManagerWithDB(database *db.DB) *Manager {
	return &Manager{db: database}
}

func (m *Manager) Close() {
	// DB lifecycle is managed by caller or db.Close()
}

func generateNodeID(owner, repo, commit, file string, line int) string {
	hash := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%s:%s:%d", owner, repo, commit, file, line))
	return "ast-" + hex.EncodeToString(hash[:12])
}

// GetASTNode queries pre-parsed AST node snippet directly from the database.
func (m *Manager) GetASTNode(ctx context.Context, owner, repo, commit, file string, line int, rootDir ...string) (*ASTNode, error) {
	if m.db == nil || m.db.SQL == nil {
		return nil, fmt.Errorf("database is uninitialized")
	}

	query := `
		SELECT id, owner, repo, commit_sha, file_path, line_number, snippet, created_at
		FROM ast_nodes
		WHERE owner = $1 AND repo = $2 AND commit_sha = $3 AND file_path = $4 AND line_number = $5
		LIMIT 1;
	`

	rd := ""
	if len(rootDir) > 0 {
		rd = rootDir[0]
	}

	normPath := NormalizeMonorepoPath(file, rd)
	cleanOrig := strings.TrimPrefix(filepath.ToSlash(file), "/")

	candidates := []string{cleanOrig}
	if normPath != cleanOrig {
		candidates = append(candidates, normPath)
	}

	for _, cand := range candidates {
		var node ASTNode
		err := m.db.SQL.QueryRowContext(ctx, query, owner, repo, commit, cand, line).Scan(
			&node.ID,
			&node.Owner,
			&node.Repo,
			&node.Commit,
			&node.FilePath,
			&node.StartLine,
			&node.Snippet,
			&node.CreatedAt,
		)
		if err == nil {
			node.EndLine = node.StartLine
			return &node, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return nil, sql.ErrNoRows
}

// IndexRepositoryAST processes repository Go packages, resolves cross-file type definitions,
// receiver methods, and helpers, and saves rich multi-file AST context snippets directly to the database.
func (m *Manager) IndexRepositoryAST(ctx context.Context, owner, repo, commit, workspacePath string, rootDir ...string) (int, error) {
	if workspacePath == "" {
		workspacePath = "."
	}

	if m.db == nil || m.db.SQL == nil {
		return 0, fmt.Errorf("database is uninitialized")
	}

	cleanRootDir := ""
	if len(rootDir) > 0 {
		cleanRootDir = strings.Trim(strings.TrimSpace(filepath.ToSlash(rootDir[0])), "/")
	}

	walkPath := workspacePath
	if cleanRootDir != "" && cleanRootDir != "." {
		walkPath = filepath.Join(workspacePath, cleanRootDir)
	}

	statInfo, statErr := os.Stat(walkPath)
	if statErr != nil {
		return 0, statErr
	}
	if !statInfo.IsDir() {
		return 0, fmt.Errorf("walk path %s is not a directory", walkPath)
	}

	// Discover all package directories containing Go files
	pkgDirs := make(map[string]bool)
	err := filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("filepath walk error encountered", "path", path, "error", err)
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
			pkgDirs[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	count := 0
	for dir := range pkgDirs {
		fset := token.NewFileSet()
		filter := func(info os.FileInfo) bool {
			return strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
		}

		pkgs, parseErr := parser.ParseDir(fset, dir, filter, parser.ParseComments)
		if parseErr != nil || len(pkgs) == 0 {
			continue
		}

		for _, pkg := range pkgs {
			pkgCtx := NewPackageContext(fset, pkg.Name, pkg.Files)
			if len(pkgCtx.Functions) == 0 {
				continue
			}

			tx, txErr := m.db.SQL.BeginTx(ctx, nil)
			if txErr != nil {
				slog.Error("failed to start AST transaction", "dir", dir, "error", txErr)
				continue
			}

			query := `
				INSERT INTO ast_nodes (id, owner, repo, commit_sha, file_path, line_number, function_name, snippet)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (id) DO UPDATE SET
					snippet = EXCLUDED.snippet,
					commit_sha = EXCLUDED.commit_sha;
			`
			stmt, stmtErr := tx.PrepareContext(ctx, query)
			if stmtErr != nil {
				_ = tx.Rollback()
				slog.Error("failed to prepare AST statement", "dir", dir, "error", stmtErr)
				continue
			}

			var execErr error
			for i := range pkgCtx.Functions {
				fn := &pkgCtx.Functions[i]
				types, helpers, vars := pkgCtx.ResolveDependencies(fn)
				richSnippet := pkgCtx.FormatContext(fn, types, helpers, vars)

				relPath, relErr := filepath.Rel(workspacePath, fn.FilePath)
				if relErr != nil {
					relPath = fn.FilePath
				}
				relPath = filepath.ToSlash(relPath)

				moduleRelPath, _ := filepath.Rel(walkPath, fn.FilePath)
				moduleRelPath = filepath.ToSlash(moduleRelPath)

				nodeID := generateNodeID(owner, repo, commit, relPath, fn.StartLine)
				_, execErr = stmt.ExecContext(ctx, nodeID, owner, repo, commit, relPath, fn.StartLine, fn.Name, richSnippet)
				if execErr != nil {
					break
				}

				if moduleRelPath != "" && moduleRelPath != relPath {
					altNodeID := generateNodeID(owner, repo, commit, moduleRelPath, fn.StartLine)
					_, execErr = stmt.ExecContext(ctx, altNodeID, owner, repo, commit, moduleRelPath, fn.StartLine, fn.Name, richSnippet)
					if execErr != nil {
						break
					}
				}
			}

			_ = stmt.Close()

			if execErr != nil {
				_ = tx.Rollback()
				slog.Error("AST batch execution error", "dir", dir, "error", execErr)
			} else {
				if commitErr := tx.Commit(); commitErr != nil {
					slog.Error("failed to commit AST transaction", "dir", dir, "error", commitErr)
				} else {
					count += len(pkgCtx.Functions)
				}
			}
		}
	}

	return count, nil
}

type ASTSymbolItem struct {
	Name    string `json:"name"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet,omitempty"`
}

type ASTFileItem struct {
	Path      string          `json:"path"`
	Name      string          `json:"name"`
	Package   string          `json:"package"`
	Functions []ASTSymbolItem `json:"functions"`
}

// ListASTFiles queries grouped Go files and their indexed function symbols for a given repository.
func (m *Manager) ListASTFiles(ctx context.Context, owner, repo string, rootDir ...string) ([]ASTFileItem, error) {
	if m.db == nil || m.db.SQL == nil {
		return nil, fmt.Errorf("database is uninitialized")
	}

	query := `
		SELECT file_path, COALESCE(function_name, ''), line_number, snippet
		FROM ast_nodes
		WHERE owner = $1 AND repo = $2
		ORDER BY file_path ASC, line_number ASC;
	`
	rows, err := m.db.SQL.QueryContext(ctx, query, owner, repo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fileMap := make(map[string]*ASTFileItem)
	var orderedPaths []string

	for rows.Next() {
		var filePath, funcName, snippet string
		var lineNum int
		if err := rows.Scan(&filePath, &funcName, &lineNum, &snippet); err != nil {
			continue
		}

		item, exists := fileMap[filePath]
		if !exists {
			baseName := filepath.Base(filePath)
			dirName := filepath.Dir(filePath)
			pkgName := filepath.Base(dirName)
			if pkgName == "." || pkgName == "/" {
				pkgName = "main"
			}
			item = &ASTFileItem{
				Path:      filePath,
				Name:      baseName,
				Package:   pkgName,
				Functions: []ASTSymbolItem{},
			}
			fileMap[filePath] = item
			orderedPaths = append(orderedPaths, filePath)
		}

		if funcName != "" {
			item.Functions = append(item.Functions, ASTSymbolItem{
				Name:    funcName,
				Line:    lineNum,
				Snippet: snippet,
			})
		}
	}

	var results []ASTFileItem
	for _, p := range orderedPaths {
		results = append(results, *fileMap[p])
	}

	return results, nil
}

// ScanLocalASTFiles dynamically parses Go packages within a workspace directory and returns the live AST file tree.
func ScanLocalASTFiles(workspacePath, rootDir string) ([]ASTFileItem, error) {
	if workspacePath == "" {
		workspacePath = "."
	}

	cleanRootDir := strings.Trim(strings.TrimSpace(filepath.ToSlash(rootDir)), "/")
	walkPath := workspacePath
	if cleanRootDir != "" && cleanRootDir != "." {
		walkPath = filepath.Join(workspacePath, cleanRootDir)
	}

	statInfo, statErr := os.Stat(walkPath)
	if statErr != nil {
		return nil, statErr
	}
	if !statInfo.IsDir() {
		return nil, fmt.Errorf("walk path %s is not a directory", walkPath)
	}

	pkgDirs := make(map[string]bool)
	_ = filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
			pkgDirs[filepath.Dir(path)] = true
		}
		return nil
	})

	var results []ASTFileItem
	for dir := range pkgDirs {
		fset := token.NewFileSet()
		filter := func(info os.FileInfo) bool {
			return strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
		}

		pkgs, parseErr := parser.ParseDir(fset, dir, filter, parser.ParseComments)
		if parseErr != nil || len(pkgs) == 0 {
			continue
		}

		for _, pkg := range pkgs {
			pkgCtx := NewPackageContext(fset, pkg.Name, pkg.Files)
			fileMap := make(map[string][]ASTSymbolItem)

			for i := range pkgCtx.Functions {
				fn := &pkgCtx.Functions[i]
				types, helpers, vars := pkgCtx.ResolveDependencies(fn)
				richSnippet := pkgCtx.FormatContext(fn, types, helpers, vars)

				relPath, err := filepath.Rel(workspacePath, fn.FilePath)
				if err != nil {
					relPath = fn.FilePath
				}
				relPath = filepath.ToSlash(relPath)

				fileMap[relPath] = append(fileMap[relPath], ASTSymbolItem{
					Name:    fn.Name,
					Line:    fn.StartLine,
					Snippet: richSnippet,
				})
			}

			for filePath, symbols := range fileMap {
				results = append(results, ASTFileItem{
					Path:      filePath,
					Name:      filepath.Base(filePath),
					Package:   pkg.Name,
					Functions: symbols,
				})
			}
		}
	}

	return results, nil
}
