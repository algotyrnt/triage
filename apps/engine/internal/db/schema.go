// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	_ "embed"
	"fmt"
)

// schemaDDL embeds the canonical PostgreSQL schema definition from schema.sql.
//
//go:embed schema.sql
var schemaDDL string

// EnsureSchema idempotently provisions tables, columns, and indexes on startup.
// Uses PostgreSQL transaction-level advisory locking to safely serialize multi-replica boots.
func (db *DB) EnsureSchema(ctx context.Context) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL connection pool uninitialized")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin schema transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Acquire PostgreSQL advisory transaction lock so concurrent replicas serialize safely
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(7492026)"); err != nil {
		return fmt.Errorf("failed to acquire schema advisory lock: %w", err)
	}

	if _, err := tx.Exec(ctx, schemaDDL); err != nil {
		return fmt.Errorf("failed to execute schema DDL: %w", err)
	}

	return tx.Commit(ctx)
}
