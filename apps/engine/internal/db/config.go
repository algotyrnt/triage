// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (db *DB) SaveInstanceConfig(ctx context.Context, key, value string) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL pool uninitialized")
	}
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO instance_config (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = NOW();
	`, key, value)
	return err
}

func (db *DB) GetInstanceConfig(ctx context.Context, key string) (string, error) {
	if db.Pool == nil {
		return "", fmt.Errorf("PostgreSQL pool uninitialized")
	}
	var value string
	err := db.Pool.QueryRow(ctx, "SELECT value FROM instance_config WHERE key = $1", key).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (db *DB) GetAllInstanceConfig(ctx context.Context) (map[string]string, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	rows, err := db.Pool.Query(ctx, "SELECT key, value FROM instance_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, nil
}

func (db *DB) IsInstanceConfigured(ctx context.Context) (bool, error) {
	if db.Pool == nil {
		return false, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM instance_config WHERE key IN ('github_app_id', 'github_oauth_client_id')").Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 2, nil
}
