// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

func (db *DB) FindActiveIncidentByFingerprint(ctx context.Context, repositoryID, fingerprint string) (*Incident, error) {
	if db == nil || db.SQL == nil || fingerprint == "" {
		return nil, nil
	}
	var inc Incident
	err := db.SQL.QueryRowContext(ctx, `
		SELECT id, COALESCE(repository_id, ''), COALESCE(fingerprint, ''), occurrence_count, title, status, COALESCE(severity, 'CRITICAL'), COALESCE(ai_provider, ''), COALESCE(ai_model, ''), file, line, panic_message, stack_trace, COALESCE(ast_snippet, ''), COALESCE(root_cause, ''), COALESCE(suggested_fix, ''), COALESCE(suggested_patch, ''), COALESCE(github_issue_url, ''), COALESCE(github_issue_number, 0), COALESCE(github_pr_url, ''), COALESCE(github_pr_number, 0), created_at, last_seen_at
		FROM incidents
		WHERE (repository_id = $1 OR ($1 = '' AND repository_id IS NULL))
		  AND fingerprint = $2
		  AND status != 'RESOLVED'
		ORDER BY last_seen_at DESC
		LIMIT 1
	`, repositoryID, fingerprint).Scan(
		&inc.ID, &inc.RepositoryID, &inc.Fingerprint, &inc.OccurrenceCount, &inc.Title, &inc.Status, &inc.Severity, &inc.AIProvider, &inc.AIModel, &inc.File, &inc.Line,
		&inc.PanicMessage, &inc.StackTrace, &inc.ASTSnippet, &inc.RootCause, &inc.SuggestedFix, &inc.SuggestedPatch,
		&inc.GitHubIssueURL, &inc.GitHubIssueNumber, &inc.GitHubPRURL, &inc.GitHubPRNumber, &inc.CreatedAt, &inc.LastSeenAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &inc, nil
}

func (db *DB) IncrementIncidentOccurrence(ctx context.Context, incidentID string) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	_, err := db.SQL.ExecContext(ctx, `
		UPDATE incidents
		SET occurrence_count = occurrence_count + 1,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, incidentID)
	return err
}

func (db *DB) SaveIncident(ctx context.Context, inc *Incident) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database connection uninitialized")
	}

	var repoID *string
	if inc.RepositoryID != "" {
		repoID = &inc.RepositoryID
	}

	if inc.Fingerprint == "" {
		raw := fmt.Sprintf("%s:%d:%s", inc.File, inc.Line, inc.PanicMessage)
		hash := sha256.Sum256([]byte(raw))
		inc.Fingerprint = hex.EncodeToString(hash[:16])
	}
	if inc.OccurrenceCount <= 0 {
		inc.OccurrenceCount = 1
	}
	if inc.Severity == "" {
		inc.Severity = "CRITICAL"
	}

	query := `
		INSERT INTO incidents (
			id, repository_id, fingerprint, occurrence_count, title, status, severity, ai_provider, ai_model,
			file, line, panic_message, stack_trace, ast_snippet, root_cause, suggested_fix, suggested_patch,
			github_issue_url, github_issue_number, github_pr_url, github_pr_number, created_at, last_seen_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		ON CONFLICT (id) DO UPDATE SET
			repository_id = COALESCE(EXCLUDED.repository_id, incidents.repository_id),
			fingerprint = COALESCE(NULLIF(EXCLUDED.fingerprint, ''), incidents.fingerprint),
			occurrence_count = CASE WHEN EXCLUDED.occurrence_count > 1 THEN EXCLUDED.occurrence_count ELSE incidents.occurrence_count + 1 END,
			title = EXCLUDED.title,
			status = EXCLUDED.status,
			severity = COALESCE(NULLIF(EXCLUDED.severity, ''), incidents.severity),
			ai_provider = COALESCE(NULLIF(EXCLUDED.ai_provider, ''), incidents.ai_provider),
			ai_model = COALESCE(NULLIF(EXCLUDED.ai_model, ''), incidents.ai_model),
			ast_snippet = COALESCE(NULLIF(EXCLUDED.ast_snippet, ''), incidents.ast_snippet),
			root_cause = COALESCE(NULLIF(EXCLUDED.root_cause, ''), incidents.root_cause),
			suggested_fix = COALESCE(NULLIF(EXCLUDED.suggested_fix, ''), incidents.suggested_fix),
			suggested_patch = COALESCE(NULLIF(EXCLUDED.suggested_patch, ''), incidents.suggested_patch),
			github_issue_url = COALESCE(NULLIF(EXCLUDED.github_issue_url, ''), incidents.github_issue_url),
			github_issue_number = CASE WHEN EXCLUDED.github_issue_number > 0 THEN EXCLUDED.github_issue_number ELSE incidents.github_issue_number END,
			github_pr_url = COALESCE(NULLIF(EXCLUDED.github_pr_url, ''), incidents.github_pr_url),
			github_pr_number = CASE WHEN EXCLUDED.github_pr_number > 0 THEN EXCLUDED.github_pr_number ELSE incidents.github_pr_number END,
			last_seen_at = CURRENT_TIMESTAMP;
	`
	now := time.Now().UTC()
	createdAt := inc.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	lastSeenAt := inc.LastSeenAt
	if lastSeenAt.IsZero() {
		lastSeenAt = now
	}

	_, err := db.SQL.ExecContext(
		ctx, query,
		inc.ID, repoID, inc.Fingerprint, inc.OccurrenceCount, inc.Title, inc.Status, inc.Severity, inc.AIProvider, inc.AIModel,
		inc.File, inc.Line, inc.PanicMessage, inc.StackTrace, inc.ASTSnippet, inc.RootCause, inc.SuggestedFix, inc.SuggestedPatch,
		inc.GitHubIssueURL, inc.GitHubIssueNumber, inc.GitHubPRURL, inc.GitHubPRNumber, createdAt, lastSeenAt,
	)
	return err
}

func (db *DB) UpdateIncidentIssue(ctx context.Context, incidentID, issueURL string, issueNumber int) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	_, err := db.SQL.ExecContext(ctx, `
		UPDATE incidents
		SET github_issue_url = $2, github_issue_number = $3
		WHERE id = $1
	`, incidentID, issueURL, issueNumber)
	return err
}

func (db *DB) UpdateIncidentPR(ctx context.Context, incidentID, prURL string, prNumber int, patch string) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	_, err := db.SQL.ExecContext(ctx, `
		UPDATE incidents
		SET github_pr_url = $2,
		    github_pr_number = $3,
		    suggested_patch = CASE WHEN $4 != '' THEN $4 ELSE suggested_patch END
		WHERE id = $1
	`, incidentID, prURL, prNumber, patch)
	return err
}

func (db *DB) GetIncidents(ctx context.Context, limit int) ([]Incident, error) {
	if db == nil || db.SQL == nil {
		return []Incident{}, nil
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.SQL.QueryContext(ctx, `
		SELECT id, COALESCE(repository_id, ''), COALESCE(fingerprint, ''), occurrence_count, title, status, COALESCE(severity, 'CRITICAL'), COALESCE(ai_provider, ''), COALESCE(ai_model, ''), file, line, panic_message, stack_trace, COALESCE(ast_snippet, ''), COALESCE(root_cause, ''), COALESCE(suggested_fix, ''), COALESCE(suggested_patch, ''), COALESCE(github_issue_url, ''), COALESCE(github_issue_number, 0), COALESCE(github_pr_url, ''), COALESCE(github_pr_number, 0), created_at, last_seen_at
		FROM incidents
		ORDER BY last_seen_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Incident
	for rows.Next() {
		var inc Incident
		err := rows.Scan(
			&inc.ID, &inc.RepositoryID, &inc.Fingerprint, &inc.OccurrenceCount, &inc.Title, &inc.Status, &inc.Severity, &inc.AIProvider, &inc.AIModel, &inc.File, &inc.Line,
			&inc.PanicMessage, &inc.StackTrace, &inc.ASTSnippet, &inc.RootCause, &inc.SuggestedFix, &inc.SuggestedPatch,
			&inc.GitHubIssueURL, &inc.GitHubIssueNumber, &inc.GitHubPRURL, &inc.GitHubPRNumber, &inc.CreatedAt, &inc.LastSeenAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, inc)
	}
	return results, nil
}

func (db *DB) GetIncidentByID(ctx context.Context, id string) (*Incident, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	var inc Incident
	err := db.SQL.QueryRowContext(ctx, `
		SELECT id, COALESCE(repository_id, ''), COALESCE(fingerprint, ''), occurrence_count, title, status, COALESCE(severity, 'CRITICAL'), COALESCE(ai_provider, ''), COALESCE(ai_model, ''), file, line, panic_message, stack_trace, COALESCE(ast_snippet, ''), COALESCE(root_cause, ''), COALESCE(suggested_fix, ''), COALESCE(suggested_patch, ''), COALESCE(github_issue_url, ''), COALESCE(github_issue_number, 0), COALESCE(github_pr_url, ''), COALESCE(github_pr_number, 0), created_at, last_seen_at
		FROM incidents
		WHERE id = $1
	`, id).Scan(
		&inc.ID, &inc.RepositoryID, &inc.Fingerprint, &inc.OccurrenceCount, &inc.Title, &inc.Status, &inc.Severity, &inc.AIProvider, &inc.AIModel, &inc.File, &inc.Line,
		&inc.PanicMessage, &inc.StackTrace, &inc.ASTSnippet, &inc.RootCause, &inc.SuggestedFix, &inc.SuggestedPatch,
		&inc.GitHubIssueURL, &inc.GitHubIssueNumber, &inc.GitHubPRURL, &inc.GitHubPRNumber, &inc.CreatedAt, &inc.LastSeenAt,
	)
	if err != nil {
		return nil, err
	}
	return &inc, nil
}
