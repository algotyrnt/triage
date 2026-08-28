// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (db *DB) SaveInstallation(ctx context.Context, installationID int64, orgLogin string, orgID int64, accountType string) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	id := fmt.Sprintf("inst_%d", installationID)
	_, err := db.SQL.ExecContext(ctx, `
		INSERT INTO github_installations (id, installation_id, org_login, org_id, account_type, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		ON CONFLICT (installation_id) DO UPDATE SET
			org_login = EXCLUDED.org_login,
			org_id = EXCLUDED.org_id,
			status = 'active';
	`, id, installationID, orgLogin, orgID, accountType)
	return err
}

func (db *DB) GetInstallation(ctx context.Context) (*GitHubInstallation, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	var inst GitHubInstallation
	err := db.SQL.QueryRowContext(ctx, `
		SELECT id, installation_id, org_login, org_id, account_type, status, created_at
		FROM github_installations
		WHERE status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&inst.ID, &inst.InstallationID, &inst.OrgLogin, &inst.OrgID, &inst.AccountType, &inst.Status, &inst.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &inst, nil
}

func (db *DB) GetAllInstallations(ctx context.Context) ([]GitHubInstallation, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	rows, err := db.SQL.QueryContext(ctx, `
		SELECT id, installation_id, org_login, org_id, account_type, status, created_at
		FROM github_installations
		WHERE status = 'active'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []GitHubInstallation
	for rows.Next() {
		var inst GitHubInstallation
		if err := rows.Scan(&inst.ID, &inst.InstallationID, &inst.OrgLogin, &inst.OrgID, &inst.AccountType, &inst.Status, &inst.CreatedAt); err != nil {
			return nil, err
		}
		installations = append(installations, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return installations, nil
}

func (db *DB) GetAllInstallationRepos(ctx context.Context) ([]InstallationRepo, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	rows, err := db.SQL.QueryContext(ctx, `
		SELECT DISTINCT ir.owner, ir.repo
		FROM installation_repos ir
		JOIN github_installations gi ON ir.installation_id = gi.installation_id
		WHERE gi.status = 'active'
		ORDER BY ir.owner, ir.repo
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []InstallationRepo
	for rows.Next() {
		var r InstallationRepo
		if err := rows.Scan(&r.Owner, &r.Repo); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return repos, nil
}

func (db *DB) SaveInstallationRepos(ctx context.Context, installationID int64, repos []InstallationRepo) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	tx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "DELETE FROM installation_repos WHERE installation_id = $1", installationID)
	if err != nil {
		return err
	}

	for i, repo := range repos {
		repoID := fmt.Sprintf("ir_%d_%s_%s_%d", installationID, repo.Owner, repo.Repo, i)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO installation_repos (id, installation_id, owner, repo)
			VALUES ($1, $2, $3, $4)
		`, repoID, installationID, repo.Owner, repo.Repo)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) SaveInstallationRepo(ctx context.Context, installationID int64, owner, repo string) error {
	if db == nil || db.SQL == nil {
		return fmt.Errorf("database uninitialized")
	}
	repoID := fmt.Sprintf("ir_%d_%s_%s", installationID, owner, repo)
	_, err := db.SQL.ExecContext(ctx, `
		INSERT INTO installation_repos (id, installation_id, owner, repo)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (installation_id, owner, repo) DO NOTHING
	`, repoID, installationID, owner, repo)
	return err
}

func (db *DB) GetInstallationRepos(ctx context.Context, installationID int64) ([]InstallationRepo, error) {
	if db == nil || db.SQL == nil {
		return nil, fmt.Errorf("database uninitialized")
	}
	rows, err := db.SQL.QueryContext(ctx, `
		SELECT owner, repo
		FROM installation_repos
		WHERE installation_id = $1
		ORDER BY repo
	`, installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []InstallationRepo
	for rows.Next() {
		var r InstallationRepo
		if err := rows.Scan(&r.Owner, &r.Repo); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return repos, nil
}

func (db *DB) GetInstallationForRepo(ctx context.Context, owner, repo string) (int64, error) {
	if db == nil || db.SQL == nil {
		return 0, fmt.Errorf("database uninitialized")
	}
	var installationID int64

	// 1. Check mapped installation repo
	err := db.SQL.QueryRowContext(ctx, `
		SELECT installation_id
		FROM installation_repos
		WHERE LOWER(owner) = LOWER($1) AND LOWER(repo) = LOWER($2) AND installation_id > 0
		LIMIT 1
	`, owner, repo).Scan(&installationID)
	if err == nil && installationID > 0 {
		return installationID, nil
	}

	// 2. Return active GitHub App installation
	err = db.SQL.QueryRowContext(ctx, `
		SELECT installation_id
		FROM github_installations
		WHERE status = 'active' AND installation_id > 0
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&installationID)
	if err == nil && installationID > 0 {
		return installationID, nil
	}

	return 0, fmt.Errorf("no active GitHub installation found for %s/%s", owner, repo)
}
