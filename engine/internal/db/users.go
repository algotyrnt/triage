// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"fmt"
	"time"
)

func (db *DB) CountUsers(ctx context.Context) (int, error) {
	if db == nil || db.SQL == nil {
		return 0, nil
	}
	var count int
	err := db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&count)
	return count, err
}

func (db *DB) UpsertUserWithRole(ctx context.Context, githubID, username, email, avatarURL, defaultRole string) (*User, error) {
	if db == nil || db.SQL == nil {
		role := defaultRole
		if role == "" {
			role = "Owner"
		}
		return &User{
			ID:        "usr_" + githubID,
			GitHubID:  githubID,
			Username:  username,
			Email:     email,
			AvatarURL: avatarURL,
			Role:      role,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}

	// 1. Check if user already exists
	var existing User
	var existingEmail, existingAvatar *string
	err := db.SQL.QueryRowContext(ctx, `
		SELECT id, github_id, username, email, avatar_url, role, created_at, updated_at
		FROM users
		WHERE github_id = $1
	`, githubID).Scan(&existing.ID, &existing.GitHubID, &existing.Username, &existingEmail, &existingAvatar, &existing.Role, &existing.CreatedAt, &existing.UpdatedAt)

	if err == nil {
		if existingEmail != nil {
			existing.Email = *existingEmail
		}
		if existingAvatar != nil {
			existing.AvatarURL = *existingAvatar
		}

		var u User
		var retEmail, retAvatar *string
		updErr := db.SQL.QueryRowContext(ctx, `
			UPDATE users
			SET username = $2, email = $3, avatar_url = $4, updated_at = CURRENT_TIMESTAMP
			WHERE github_id = $1
			RETURNING id, github_id, username, email, avatar_url, role, created_at, updated_at
		`, githubID, username, email, avatarURL).Scan(&u.ID, &u.GitHubID, &u.Username, &retEmail, &retAvatar, &u.Role, &u.CreatedAt, &u.UpdatedAt)
		if updErr != nil {
			return nil, updErr
		}
		if retEmail != nil {
			u.Email = *retEmail
		}
		if retAvatar != nil {
			u.AvatarURL = *retAvatar
		}
		return &u, nil
	}

	// 2. New user registration
	assignedRole := defaultRole
	if assignedRole == "" {
		assignedRole = "Developer"
	}

	userCount, _ := db.CountUsers(ctx)
	if userCount == 0 {
		assignedRole = "Owner"
	} else {
		var invRole string
		invErr := db.SQL.QueryRowContext(ctx, `
			SELECT role FROM invitations WHERE LOWER(github_username) = LOWER($1) LIMIT 1
		`, username).Scan(&invRole)
		if invErr == nil && invRole != "" {
			assignedRole = invRole
			_, _ = db.SQL.ExecContext(ctx, `DELETE FROM invitations WHERE LOWER(github_username) = LOWER($1)`, username)
		}
	}

	userID := fmt.Sprintf("usr_%s", githubID)
	var u User
	var retEmail, retAvatar *string
	err = db.SQL.QueryRowContext(ctx, `
		INSERT INTO users (id, github_id, username, email, avatar_url, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, github_id, username, email, avatar_url, role, created_at, updated_at
	`, userID, githubID, username, email, avatarURL, assignedRole).Scan(
		&u.ID, &u.GitHubID, &u.Username, &retEmail, &retAvatar, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if retEmail != nil {
		u.Email = *retEmail
	}
	if retAvatar != nil {
		u.AvatarURL = *retAvatar
	}
	return &u, nil
}

func (db *DB) UpsertUser(ctx context.Context, githubID, username, avatarURL string) (*User, error) {
	return db.UpsertUserWithRole(ctx, githubID, username, "", avatarURL, "Developer")
}

func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	if db == nil || db.SQL == nil {
		return &User{ID: id, Username: "demo_user", Role: "Owner"}, nil
	}
	var u User
	var email, avatar *string
	err := db.SQL.QueryRowContext(ctx, `
		SELECT id, github_id, username, email, avatar_url, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.GitHubID, &u.Username, &email, &avatar, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if email != nil {
		u.Email = *email
	}
	if avatar != nil {
		u.AvatarURL = *avatar
	}
	return &u, nil
}

func (db *DB) ListUsers(ctx context.Context) ([]User, error) {
	if db == nil || db.SQL == nil {
		return []User{
			{ID: "usr_demo", GitHubID: "12345", Username: "algotyrnt", Role: "Owner", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}, nil
	}
	rows, err := db.SQL.QueryContext(ctx, `
		SELECT id, github_id, username, email, avatar_url, role, created_at, updated_at
		FROM users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var email, avatar *string
		if scanErr := rows.Scan(&u.ID, &u.GitHubID, &u.Username, &email, &avatar, &u.Role, &u.CreatedAt, &u.UpdatedAt); scanErr == nil {
			if email != nil {
				u.Email = *email
			}
			if avatar != nil {
				u.AvatarURL = *avatar
			}
			users = append(users, u)
		}
	}
	return users, nil
}

func (db *DB) UpdateUserRole(ctx context.Context, id, newRole string) error {
	if db == nil || db.SQL == nil {
		return nil
	}

	validRoles := map[string]bool{"Owner": true, "Admin": true, "Developer": true, "Viewer": true}
	if !validRoles[newRole] {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	var currentRole string
	err := db.SQL.QueryRowContext(ctx, "SELECT role FROM users WHERE id = $1", id).Scan(&currentRole)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if currentRole == "Owner" && newRole != "Owner" {
		var ownerCount int
		_ = db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE role = 'Owner'").Scan(&ownerCount)
		if ownerCount <= 1 {
			return fmt.Errorf("cannot demote the only instance Owner")
		}
	}

	_, err = db.SQL.ExecContext(ctx, `
		UPDATE users SET role = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, id, newRole)
	return err
}

func (db *DB) DeleteUser(ctx context.Context, id string) error {
	if db == nil || db.SQL == nil {
		return nil
	}

	var role string
	err := db.SQL.QueryRowContext(ctx, "SELECT role FROM users WHERE id = $1", id).Scan(&role)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if role == "Owner" {
		var ownerCount int
		_ = db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE role = 'Owner'").Scan(&ownerCount)
		if ownerCount <= 1 {
			return fmt.Errorf("cannot delete the only instance Owner")
		}
	}

	_, err = db.SQL.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

func (db *DB) CreateInvitation(ctx context.Context, githubUsername, role, invitedBy string) (*Invitation, error) {
	validRoles := map[string]bool{"Admin": true, "Developer": true, "Viewer": true}
	if !validRoles[role] {
		role = "Developer"
	}

	invID := fmt.Sprintf("inv_%d", time.Now().UnixNano())

	if db == nil || db.SQL == nil {
		return &Invitation{
			ID:             invID,
			GitHubUsername: githubUsername,
			Role:           role,
			InvitedBy:      invitedBy,
			CreatedAt:      time.Now(),
		}, nil
	}

	var inv Invitation
	var invBy *string
	err := db.SQL.QueryRowContext(ctx, `
		INSERT INTO invitations (id, github_username, role, invited_by, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (github_username) DO UPDATE SET role = EXCLUDED.role, created_at = CURRENT_TIMESTAMP
		RETURNING id, github_username, role, invited_by, created_at
	`, invID, githubUsername, role, invitedBy).Scan(&inv.ID, &inv.GitHubUsername, &inv.Role, &invBy, &inv.CreatedAt)

	if err != nil {
		return nil, err
	}
	if invBy != nil {
		inv.InvitedBy = *invBy
	}
	return &inv, nil
}

func (db *DB) ListInvitations(ctx context.Context) ([]Invitation, error) {
	if db == nil || db.SQL == nil {
		return []Invitation{}, nil
	}
	rows, err := db.SQL.QueryContext(ctx, `
		SELECT id, github_username, role, invited_by, created_at
		FROM invitations
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Invitation
	for rows.Next() {
		var inv Invitation
		var invBy *string
		if scanErr := rows.Scan(&inv.ID, &inv.GitHubUsername, &inv.Role, &invBy, &inv.CreatedAt); scanErr == nil {
			if invBy != nil {
				inv.InvitedBy = *invBy
			}
			list = append(list, inv)
		}
	}
	return list, nil
}

func (db *DB) DeleteInvitation(ctx context.Context, id string) error {
	if db == nil || db.SQL == nil {
		return nil
	}
	_, err := db.SQL.ExecContext(ctx, "DELETE FROM invitations WHERE id = $1 OR github_username = $1", id)
	return err
}
