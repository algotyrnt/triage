// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"crypto/sha256"
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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool *pgxpool.Pool
}

func NewManager(ctx context.Context, databaseURL string) (*Manager, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is missing or empty")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return &Manager{pool: pool}, nil
}

func (m *Manager) Close() {
	if m.pool != nil {
		m.pool.Close()
	}
}

func generateNodeID(owner, repo, commit, file string, line int) string {
	hash := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%s:%s:%d", owner, repo, commit, file, line))
	return "ast-" + hex.EncodeToString(hash[:12])
}

// GetASTNode queries pre-parsed AST node snippet directly from PostgreSQL.
func (m *Manager) GetASTNode(ctx context.Context, owner, repo, commit, file string, line int, rootDir ...string) (*ASTNode, error) {
	if m.pool == nil {
		return nil, fmt.Errorf("PostgreSQL database is uninitialized")
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
		err := m.pool.QueryRow(ctx, query, owner, repo, commit, cand, line).Scan(
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
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	return nil, pgx.ErrNoRows
}

// IndexRepositoryAST processes repository Go packages, resolves cross-file type definitions,
// receiver methods, and helpers, and saves rich multi-file AST context snippets directly to PostgreSQL.
func (m *Manager) IndexRepositoryAST(ctx context.Context, owner, repo, commit, workspacePath string, rootDir ...string) (int, error) {
	if workspacePath == "" {
		workspacePath = "."
	}

	if m.pool == nil {
		return 0, fmt.Errorf("PostgreSQL database is uninitialized")
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

			batch := &pgx.Batch{}
			query := `
				INSERT INTO ast_nodes (id, owner, repo, commit_sha, file_path, line_number, function_name, snippet)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (id) DO UPDATE SET
					snippet = EXCLUDED.snippet,
					commit_sha = EXCLUDED.commit_sha;
			`
			totalQueued := 0

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
				batch.Queue(query, nodeID, owner, repo, commit, relPath, fn.StartLine, fn.Name, richSnippet)
				totalQueued++

				if moduleRelPath != "" && moduleRelPath != relPath {
					altNodeID := generateNodeID(owner, repo, commit, moduleRelPath, fn.StartLine)
					batch.Queue(query, altNodeID, owner, repo, commit, moduleRelPath, fn.StartLine, fn.Name, richSnippet)
					totalQueued++
				}
			}

			if totalQueued > 0 {
				br := m.pool.SendBatch(ctx, batch)
				var firstErr error
				for i := 0; i < totalQueued; i++ {
					_, execErr := br.Exec()
					if execErr != nil && firstErr == nil {
						firstErr = execErr
					}
				}
				if closeErr := br.Close(); closeErr != nil && firstErr == nil {
					firstErr = closeErr
				}
				if firstErr != nil {
					slog.Error("AST batch execution error", "dir", dir, "error", firstErr)
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
	if m.pool == nil {
		return nil, fmt.Errorf("PostgreSQL database is uninitialized")
	}

	query := `
		SELECT file_path, COALESCE(function_name, ''), line_number, snippet
		FROM ast_nodes
		WHERE owner = $1 AND repo = $2
		ORDER BY file_path ASC, line_number ASC;
	`
	rows, err := m.pool.Query(ctx, query, owner, repo)
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
