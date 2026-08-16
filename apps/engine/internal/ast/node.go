// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
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
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%s:%d", owner, repo, commit, file, line)))
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
	}

	return nil, pgx.ErrNoRows
}

// IndexRepositoryAST processes repository Go files once, extracts FuncDecl AST nodes,
// and saves them directly to PostgreSQL.
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

	count := 0
	err := filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("[AST INDEX] Walk error for path %s: %v", path, err)
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		relPath, relErr := filepath.Rel(workspacePath, path)
		if relErr != nil {
			log.Printf("[AST INDEX] Failed to compute relative path for %s: %v", path, relErr)
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Module-relative path (relative to the scoped service directory)
		moduleRelPath, _ := filepath.Rel(walkPath, path)
		moduleRelPath = filepath.ToSlash(moduleRelPath)

		// Parse Go file using Go standard library parser
		fset, node, parseErr := parseGoFile(path)
		if parseErr != nil {
			return nil
		}

		funcs := extractFunctions(fset, node)
		if len(funcs) == 0 {
			return nil
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
		for _, fn := range funcs {
			nodeID := generateNodeID(owner, repo, commit, relPath, fn.StartLine)
			batch.Queue(query, nodeID, owner, repo, commit, relPath, fn.StartLine, fn.Name, fn.Snippet)
			totalQueued++

			// Also index by moduleRelPath if it differs from relPath
			if moduleRelPath != "" && moduleRelPath != relPath {
				altNodeID := generateNodeID(owner, repo, commit, moduleRelPath, fn.StartLine)
				batch.Queue(query, altNodeID, owner, repo, commit, moduleRelPath, fn.StartLine, fn.Name, fn.Snippet)
				totalQueued++
			}
		}

		br := m.pool.SendBatch(ctx, batch)

		var firstErr error
		for i := 0; i < totalQueued; i++ {
			_, execErr := br.Exec()
			if execErr != nil {
				if firstErr == nil {
					firstErr = execErr
				}
			}
		}
		if closeErr := br.Close(); closeErr != nil && firstErr == nil {
			firstErr = closeErr
		}
		if firstErr != nil {
			return firstErr
		}

		count += len(funcs)
		return nil
	})

	return count, err
}
