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

// SaveASTNode stores an extracted function AST snippet to the database for persistent caching.
func (m *Manager) SaveASTNode(ctx context.Context, owner, repo, commit, filePath, funcName string, startLine, endLine int, snippet string) error {
	if m.db == nil || m.db.SQL == nil {
		return fmt.Errorf("database is uninitialized")
	}
	if snippet == "" {
		return nil
	}

	nodeID := generateNodeID(owner, repo, commit, filePath, startLine)
	query := `
		INSERT INTO ast_nodes (id, owner, repo, commit_sha, file_path, line_number, function_name, snippet)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			snippet = EXCLUDED.snippet,
			commit_sha = EXCLUDED.commit_sha;
	`
	_, err := m.db.SQL.ExecContext(ctx, query, nodeID, owner, repo, commit, filePath, startLine, funcName, snippet)
	return err
}
