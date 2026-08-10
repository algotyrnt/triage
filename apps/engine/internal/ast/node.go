// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		databaseURL = "postgresql://postgres:postgres@localhost:5432/triage_db"
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

func generateNodeID(file string, line int) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", file, line)))
	return "ast-" + hex.EncodeToString(hash[:12])
}

// GetASTNode queries pre-parsed AST node snippet directly from PostgreSQL.
func (m *Manager) GetASTNode(ctx context.Context, file string, line int) (*ASTNode, error) {
	if m.pool == nil {
		return nil, fmt.Errorf("PostgreSQL database is uninitialized")
	}

	query := `
		SELECT id, owner, repo, commit_sha, file_path, line_number, snippet, created_at
		FROM ast_nodes
		WHERE file_path = $1 AND line_number = $2
		LIMIT 1;
	`

	var node ASTNode
	err := m.pool.QueryRow(ctx, query, file, line).Scan(
		&node.ID,
		&node.Owner,
		&node.Repo,
		&node.Commit,
		&node.FilePath,
		&node.StartLine,
		&node.Snippet,
		&node.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	node.EndLine = node.StartLine
	return &node, nil
}

// IndexRepositoryAST processes repository Go files once, extracts FuncDecl AST nodes,
// and saves them directly to PostgreSQL.
func (m *Manager) IndexRepositoryAST(ctx context.Context, owner, repo, commit, workspacePath string) (int, error) {
	if workspacePath == "" {
		workspacePath = "."
	}

	if m.pool == nil {
		return 0, fmt.Errorf("PostgreSQL database is uninitialized")
	}

	count := 0
	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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

		relPath, _ := filepath.Rel(workspacePath, path)

		// Parse Go file using Go standard library parser
		fset, node, parseErr := parseGoFile(path)
		if parseErr != nil {
			return nil
		}

		for _, fn := range extractFunctions(fset, node) {
			nodeID := generateNodeID(relPath, fn.StartLine)
			query := `
				INSERT INTO ast_nodes (id, owner, repo, commit_sha, file_path, line_number, function_name, snippet)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (id) DO UPDATE SET
					snippet = EXCLUDED.snippet,
					commit_sha = EXCLUDED.commit_sha;
			`
			_, execErr := m.pool.Exec(ctx, query, nodeID, owner, repo, commit, relPath, fn.StartLine, fn.Name, fn.Snippet)
			if execErr == nil {
				count++
			}
		}

		return nil
	})

	return count, err
}
