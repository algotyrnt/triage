// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (db *DB) SaveInstallation(ctx context.Context, installationID int64, orgLogin string, orgID int64, accountType string) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL pool uninitialized")
	}
	id := fmt.Sprintf("inst_%d", installationID)
	_, err := db.Pool.Exec(ctx, `
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
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	var inst GitHubInstallation
	err := db.Pool.QueryRow(ctx, `
		SELECT id, installation_id, org_login, org_id, account_type, status, created_at
		FROM github_installations
		WHERE status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&inst.ID, &inst.InstallationID, &inst.OrgLogin, &inst.OrgID, &inst.AccountType, &inst.Status, &inst.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &inst, nil
}

func (db *DB) GetAllInstallations(ctx context.Context) ([]GitHubInstallation, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	rows, err := db.Pool.Query(ctx, `
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
	return installations, nil
}

func (db *DB) GetAllInstallationRepos(ctx context.Context) ([]InstallationRepo, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	rows, err := db.Pool.Query(ctx, `
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
	return repos, nil
}

func (db *DB) SaveInstallationRepos(ctx context.Context, installationID int64, repos []InstallationRepo) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL pool uninitialized")
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM installation_repos WHERE installation_id = $1", installationID)
	if err != nil {
		return err
	}

	for i, repo := range repos {
		repoID := fmt.Sprintf("ir_%d_%s_%s_%d", installationID, repo.Owner, repo.Repo, i)
		_, err = tx.Exec(ctx, `
			INSERT INTO installation_repos (id, installation_id, owner, repo)
			VALUES ($1, $2, $3, $4)
		`, repoID, installationID, repo.Owner, repo.Repo)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (db *DB) GetInstallationRepos(ctx context.Context, installationID int64) ([]InstallationRepo, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	rows, err := db.Pool.Query(ctx, `
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
	return repos, nil
}

func (db *DB) GetInstallationForRepo(ctx context.Context, owner, repo string) (int64, error) {
	if db.Pool == nil {
		return 0, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	var installationID int64
	err := db.Pool.QueryRow(ctx, `
		SELECT installation_id
		FROM installation_repos
		WHERE owner = $1 AND repo = $2
	`, owner, repo).Scan(&installationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("installation not found for repo %s/%s", owner, repo)
		}
		return 0, err
	}
	return installationID, nil
}
