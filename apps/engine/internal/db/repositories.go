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
)

func (db *DB) CreateProject(ctx context.Context, owner, repo, rootDir, ownerUsername string, projectContext ...string) (string, string, error) {
	if db.Pool == nil {
		return "", "", fmt.Errorf("PostgreSQL pool uninitialized")
	}

	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	tupleStr := fmt.Sprintf("%s/%s:%s", owner, repo, cleanRootDir)
	tupleHash := sha256.Sum256([]byte(tupleStr))
	repoID := fmt.Sprintf("repo_%s", hex.EncodeToString(tupleHash[:16]))

	contextStr := ""
	if len(projectContext) > 0 {
		contextStr = projectContext[0]
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO repositories (id, owner, repo, root_dir, installation_id, context)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner, repo, root_dir) DO UPDATE SET
			root_dir = EXCLUDED.root_dir,
			context = CASE WHEN EXCLUDED.context != '' THEN EXCLUDED.context ELSE repositories.context END
	`, repoID, owner, repo, cleanRootDir, 1001, contextStr)
	if err != nil {
		return "", "", err
	}

	keySuffix := repo
	if cleanRootDir != "" {
		keySuffix = fmt.Sprintf("%s_%s", repo, strings.ReplaceAll(cleanRootDir, "/", "_"))
	}
	rawKey := fmt.Sprintf("tr_live_%s_%d", keySuffix, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	keyMasked := fmt.Sprintf("tr_live_...%s", rawKey[len(rawKey)-4:])
	keyID := fmt.Sprintf("key_%d", time.Now().UnixNano())

	keyName := fmt.Sprintf("Key for %s/%s", owner, repo)
	if cleanRootDir != "" {
		keyName = fmt.Sprintf("Key for %s/%s (%s)", owner, repo, cleanRootDir)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO api_keys (id, repository_id, name, key_hash, key_masked)
		VALUES ($1, $2, $3, $4, $5)
	`, keyID, repoID, keyName, keyHash, keyMasked)
	if err != nil {
		return "", "", err
	}

	return rawKey, repoID, nil
}

func (db *DB) UpdateProjectContext(ctx context.Context, owner, repo, rootDir, projectContext string) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL pool uninitialized")
	}
	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	_, err := db.Pool.Exec(ctx, `
		UPDATE repositories
		SET context = $4
		WHERE owner = $1 AND repo = $2 AND COALESCE(root_dir, '') = $3
	`, owner, repo, cleanRootDir, projectContext)
	return err
}

func (db *DB) GetProjects(ctx context.Context) ([]Repository, error) {
	if db.Pool == nil {
		return []Repository{}, nil
	}
	query := `
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id, COALESCE(r.context, ''),
		       COALESCE((
		           SELECT k.key_masked
		           FROM api_keys k
		           WHERE k.repository_id = r.id
		             AND (k.revoked_at IS NULL OR k.revoked_at > NOW())
		           ORDER BY k.created_at DESC
		           LIMIT 1
		       ), ''),
		       r.created_at
		FROM repositories r
		ORDER BY r.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query)
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
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	var r Repository
	err := db.Pool.QueryRow(ctx, `
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id, COALESCE(r.context, ''),
		       COALESCE((
		           SELECT k.key_masked
		           FROM api_keys k
		           WHERE k.repository_id = r.id
		             AND (k.revoked_at IS NULL OR k.revoked_at > NOW())
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
