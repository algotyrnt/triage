// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
)

// schemaDDL embeds the canonical SQLite database schema definition from schema.sql.
//
//go:embed schema.sql
var schemaDDL string

// EnsureSchema idempotently provisions tables, columns, and indexes on startup.
func (db *DB) EnsureSchema(ctx context.Context) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database connection uninitialized")
	}

	tx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin schema transaction: %w", err)
	}
	defer tx.Rollback()

	statements := strings.Split(schemaDDL, ";")
	for _, rawStmt := range statements {
		stmt := strings.TrimSpace(rawStmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement (%s...): %w", truncate(stmt, 50), err)
		}
	}

	return tx.Commit()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
