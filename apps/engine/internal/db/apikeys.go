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

	"github.com/jackc/pgx/v5"
)

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

func (db *DB) GetAPIKeys(ctx context.Context, owner, repo, rootDir string) ([]APIKeyRecord, error) {
	if db.Pool == nil {
		return []APIKeyRecord{}, nil
	}
	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")

	var rows pgx.Rows
	var err error

	if owner != "" && repo != "" {
		rows, err = db.Pool.Query(ctx, `
			SELECT k.id, k.repository_id, k.name, k.key_masked, k.created_at, k.revoked_at, k.expires_at,
			       CASE
			           WHEN k.revoked_at IS NOT NULL AND k.revoked_at <= NOW() THEN 'REVOKED'
			           WHEN k.expires_at IS NOT NULL AND k.expires_at <= NOW() THEN 'EXPIRED'
			           ELSE 'ACTIVE'
			       END as status
			FROM api_keys k
			JOIN repositories r ON k.repository_id = r.id
			WHERE r.owner = $1 AND r.repo = $2 AND COALESCE(r.root_dir, '') = $3
			ORDER BY k.created_at DESC
		`, owner, repo, cleanRootDir)
	} else {
		rows, err = db.Pool.Query(ctx, `
			SELECT k.id, k.repository_id, k.name, k.key_masked, k.created_at, k.revoked_at, k.expires_at,
			       CASE
			           WHEN k.revoked_at IS NOT NULL AND k.revoked_at <= NOW() THEN 'REVOKED'
			           WHEN k.expires_at IS NOT NULL AND k.expires_at <= NOW() THEN 'EXPIRED'
			           ELSE 'ACTIVE'
			       END as status
			FROM api_keys k
			ORDER BY k.created_at DESC
		`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKeyRecord
	for rows.Next() {
		var k APIKeyRecord
		if err := rows.Scan(&k.ID, &k.RepositoryID, &k.Name, &k.KeyMasked, &k.CreatedAt, &k.RevokedAt, &k.ExpiresAt, &k.Status); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (db *DB) CreateAPIKey(ctx context.Context, owner, repo, rootDir, keyName string) (*APIKeyRecord, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}

	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	tupleStr := fmt.Sprintf("%s/%s:%s", owner, repo, cleanRootDir)
	tupleHash := sha256.Sum256([]byte(tupleStr))
	repoID := fmt.Sprintf("repo_%s", hex.EncodeToString(tupleHash[:16]))

	// Ensure repository exists
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO repositories (id, owner, repo, root_dir, installation_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (owner, repo, root_dir) DO NOTHING
	`, repoID, owner, repo, cleanRootDir, 1001)
	if err != nil {
		return nil, err
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

	if keyName == "" {
		keyName = fmt.Sprintf("Key for %s/%s", owner, repo)
		if cleanRootDir != "" {
			keyName = fmt.Sprintf("Key for %s/%s (%s)", owner, repo, cleanRootDir)
		}
	}

	now := time.Now().UTC()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO api_keys (id, repository_id, name, key_hash, key_masked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, keyID, repoID, keyName, keyHash, keyMasked, now)
	if err != nil {
		return nil, err
	}

	return &APIKeyRecord{
		ID:           keyID,
		RepositoryID: repoID,
		Name:         keyName,
		KeyMasked:    keyMasked,
		RawKey:       rawKey,
		CreatedAt:    now,
		Status:       "ACTIVE",
	}, nil
}

func (db *DB) RevokeAPIKey(ctx context.Context, keyID string) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL pool uninitialized")
	}
	res, err := db.Pool.Exec(ctx, `
		UPDATE api_keys
		SET revoked_at = NOW()
		WHERE id = $1 AND (revoked_at IS NULL OR revoked_at > NOW())
	`, keyID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("key not found or already revoked")
	}
	return nil
}

func (db *DB) GetRepositoryByAPIKey(ctx context.Context, key string) (*Repository, error) {
	if db.Pool == nil || key == "" {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized or key is empty")
	}
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	var r Repository
	err := db.Pool.QueryRow(ctx, `
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id, COALESCE(r.context, ''), r.created_at
		FROM api_keys k
		JOIN repositories r ON k.repository_id = r.id
		WHERE k.key_hash = $1 AND k.revoked_at IS NULL AND (k.expires_at IS NULL OR k.expires_at > NOW())
		LIMIT 1
	`, keyHash).Scan(&r.ID, &r.Owner, &r.Repo, &r.RootDir, &r.InstallationID, &r.Context, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
