// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"triage/engine/internal/crypto"
)

func (db *DB) CreateProject(ctx context.Context, owner, repo, rootDir, ownerUsername string, projectContext ...string) (string, string, error) {
	if db == nil || db.SQL == nil {
		return "", "", fmt.Errorf("database uninitialized")
	}

	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	tupleStr := fmt.Sprintf("%s/%s:%s", owner, repo, cleanRootDir)
	tupleHash := sha256.Sum256([]byte(tupleStr))
	repoID := fmt.Sprintf("repo_%s", hex.EncodeToString(tupleHash[:16]))

	contextStr := ""
	if len(projectContext) > 0 {
		contextStr = projectContext[0]
	}

	// Insert or update repository record
	_, err := db.SQL.ExecContext(ctx, `
		INSERT INTO repositories (id, owner, repo, root_dir, installation_id, context)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner, repo, root_dir) DO UPDATE SET
			root_dir = EXCLUDED.root_dir,
			context = CASE WHEN EXCLUDED.context != '' THEN EXCLUDED.context ELSE repositories.context END
	`, repoID, owner, repo, cleanRootDir, 1001, contextStr)
	if err != nil {
		return "", "", err
	}
	var existingActiveCount int
	_ = db.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE repository_id = $1 AND (revoked_at IS NULL OR revoked_at > CURRENT_TIMESTAMP)`, repoID).Scan(&existingActiveCount)
	if existingActiveCount > 0 {
		return "", repoID, nil
	}

	rawKey := crypto.GenerateSecureAPIKey()
	keyHash := crypto.HashKey(rawKey)
	keyMasked := crypto.MaskAPIKey(rawKey)
	keyID := fmt.Sprintf("key_%d", time.Now().UnixNano())

	keyName := fmt.Sprintf("Key for %s/%s", owner, repo)
	if cleanRootDir != "" {
		keyName = fmt.Sprintf("Key for %s/%s (%s)", owner, repo, cleanRootDir)
	}

	_, err = db.SQL.ExecContext(ctx, `
		INSERT INTO api_keys (id, repository_id, name, key_hash, key_masked)
		VALUES ($1, $2, $3, $4, $5)
	`, keyID, repoID, keyName, keyHash, keyMasked)
	if err != nil {
		return "", "", err
	}

	return rawKey, repoID, nil
}

func (db *DB) UpdateProjectContext(ctx context.Context, owner, repo, rootDir, projectContext string) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	_, err := db.SQL.ExecContext(ctx, `
		UPDATE repositories
		SET context = $4
		WHERE owner = $1 AND repo = $2 AND COALESCE(root_dir, '') = $3
	`, owner, repo, cleanRootDir, projectContext)
	return err
}

func (db *DB) GetProjects(ctx context.Context) ([]Repository, error) {
	if db == nil || db.SQL == nil {
		return []Repository{}, nil
	}
	query := `
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id, COALESCE(r.context, ''),
		       COALESCE((
		           SELECT k.key_masked
		           FROM api_keys k
		           WHERE k.repository_id = r.id
		             AND (k.revoked_at IS NULL OR k.revoked_at > CURRENT_TIMESTAMP)
		           ORDER BY k.created_at DESC
		           LIMIT 1
		       ), ''),
		       r.created_at
		FROM repositories r
		ORDER BY r.created_at DESC
	`
	rows, err := db.SQL.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(&r.ID, &r.Owner, &r.Repo, &r.RootDir, &r.InstallationID, &r.Context, &r.APIKeyMasked, &r.CreatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, nil
}

func (db *DB) GetProjectByOwnerRepo(ctx context.Context, owner, repo, rootDir string) (*Repository, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	var r Repository
	err := db.SQL.QueryRowContext(ctx, `
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id, COALESCE(r.context, ''),
		       COALESCE((
		           SELECT k.key_masked
		           FROM api_keys k
		           WHERE k.repository_id = r.id
		             AND (k.revoked_at IS NULL OR k.revoked_at > CURRENT_TIMESTAMP)
		           ORDER BY k.created_at DESC
		           LIMIT 1
		       ), ''),
		       r.created_at
		FROM repositories r
		WHERE r.owner = $1 AND r.repo = $2 AND COALESCE(r.root_dir, '') = $3
		LIMIT 1
	`, owner, repo, cleanRootDir).Scan(&r.ID, &r.Owner, &r.Repo, &r.RootDir, &r.InstallationID, &r.Context, &r.APIKeyMasked, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
