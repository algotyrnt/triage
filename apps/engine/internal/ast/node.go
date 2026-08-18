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
