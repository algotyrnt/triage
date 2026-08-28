// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (db *DB) SaveInstanceConfig(ctx context.Context, key, value string) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	_, err := db.SQL.ExecContext(ctx, `
		INSERT INTO instance_config (key, value, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP;
	`, key, value)
	return err
}

func (db *DB) GetInstanceConfig(ctx context.Context, key string) (string, error) {
	if db == nil || db.SQL == nil {
		return "", fmt.Errorf("database uninitialized")
	}
	var value string
	err := db.SQL.QueryRowContext(ctx, "SELECT value FROM instance_config WHERE key = $1", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (db *DB) GetAllInstanceConfig(ctx context.Context) (map[string]string, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	rows, err := db.SQL.QueryContext(ctx, "SELECT key, value FROM instance_config")
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (db *DB) IsInstanceConfigured(ctx context.Context) (bool, error) {
	if db == nil || db.SQL == nil {
		return false, fmt.Errorf("database uninitialized")
	}
	var count int
	err := db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM instance_config WHERE key IN ('github_app_id', 'github_oauth_client_id')").Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 2, nil
}
