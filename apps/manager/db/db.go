// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

type Incident struct {
	ID                string    `json:"id"`
	RepositoryID      string    `json:"repository_id,omitempty"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	File              string    `json:"file"`
	Line              int       `json:"line"`
	PanicMessage      string    `json:"panic_message"`
	StackTrace        string    `json:"stack_trace"`
	ASTSnippet        string    `json:"ast_snippet,omitempty"`
	RootCause         string    `json:"root_cause,omitempty"`
	SuggestedFix      string    `json:"suggested_fix,omitempty"`
	GitHubIssueURL    string    `json:"github_issue_url,omitempty"`
	GitHubIssueNumber int       `json:"github_issue_number,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type WebhookLog struct {
	ID           string    `json:"id"`
	DeliveryID   string    `json:"delivery_id"`
	EventType    string    `json:"event_type"`
	Status       string    `json:"status"`
	StatusCode   int       `json:"status_code"`
	RequestBody  string    `json:"request_body"`
	ResponseBody string    `json:"response_body"`
	CreatedAt    time.Time `json:"created_at"`
}

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
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
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) VerifyAPIKey(ctx context.Context, key string) bool {
	if db.Pool == nil || key == "" {
		return false
	}
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	var count int
	err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM api_keys WHERE key_hash = $1 AND (revoked_at IS NULL OR revoked_at > NOW()) AND (expires_at IS NULL OR expires_at > NOW())", keyHash).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func (db *DB) GetASTNode(ctx context.Context, owner, repo, file string, line int) (*ASTNode, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL connection pool uninitialized")
	}

	query := `
		SELECT id, owner, repo, commit_sha, file_path, line_number, snippet, created_at
		FROM ast_nodes
		WHERE owner = $1 AND repo = $2 AND file_path = $3 AND line_number = $4
		ORDER BY created_at DESC
		LIMIT 1;
	`

	var node ASTNode
	err := db.Pool.QueryRow(ctx, query, owner, repo, file, line).Scan(
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

func (db *DB) SaveASTNode(ctx context.Context, nodeID, owner, repo, commit, filePath string, line int, funcName, snippet string) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL connection pool uninitialized")
	}

	query := `
		INSERT INTO ast_nodes (id, owner, repo, commit_sha, file_path, line_number, function_name, snippet)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			snippet = EXCLUDED.snippet,
			commit_sha = EXCLUDED.commit_sha;
	`
	_, err := db.Pool.Exec(ctx, query, nodeID, owner, repo, commit, filePath, line, funcName, snippet)
	return err
}

func (db *DB) SaveIncident(ctx context.Context, inc *Incident) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL connection pool uninitialized")
	}

	query := `
		INSERT INTO incidents (id, title, status, file, line, panic_message, stack_trace, ast_snippet, root_cause, suggested_fix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
	`
	_, err := db.Pool.Exec(ctx, query, inc.ID, inc.Title, inc.Status, inc.File, inc.Line, inc.PanicMessage, inc.StackTrace, inc.ASTSnippet, inc.RootCause, inc.SuggestedFix)
	return err
}

func (db *DB) IsWebhookDuplicate(ctx context.Context, deliveryID string) (bool, error) {
	if db.Pool == nil {
		return false, fmt.Errorf("PostgreSQL connection pool uninitialized")
	}
	if deliveryID == "" {
		return false, nil
	}
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM webhook_logs WHERE delivery_id = $1", deliveryID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) SaveWebhookLog(ctx context.Context, log *WebhookLog) error {
	if db.Pool == nil || log.DeliveryID == "" {
		return nil
	}

	query := `
		INSERT INTO webhook_logs (id, delivery_id, event_type, status, status_code, request_body, response_body)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (delivery_id) DO UPDATE SET
			status = EXCLUDED.status,
			status_code = EXCLUDED.status_code,
			response_body = EXCLUDED.response_body;
	`
	_, err := db.Pool.Exec(ctx, query, log.ID, log.DeliveryID, log.EventType, log.Status, log.StatusCode, log.RequestBody, log.ResponseBody)
	return err
}
