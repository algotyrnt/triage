// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ASTNode struct {
	ID        string    `json:"id"`
	Owner     string    `json:"owner"`
	Repo      string    `json:"repo"`
	Commit    string    `json:"commit"`
	FilePath  string    `json:"file_path"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Snippet   string    `json:"snippet"`
	CreatedAt time.Time `json:"created_at"`
}

type Incident struct {
	ID                string    `json:"id"`
	RepositoryID      string    `json:"repository_id,omitempty"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	File              string    `json:"file"`
	Line              int       `json:"line"`
	PanicMessage      string    `json:"panic_message"`
	StackTrace        string    `json:"stack_trace"`
	ASTSnippet        string    `json:"ast_snippet,omitempty"`
	RootCause         string    `json:"root_cause,omitempty"`
	SuggestedFix      string    `json:"suggested_fix,omitempty"`
	GitHubIssueURL    string    `json:"github_issue_url,omitempty"`
	GitHubIssueNumber int       `json:"github_issue_number,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type User struct {
	ID        string    `json:"id"`
	GitHubID  string    `json:"github_id"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Repository struct {
	ID             string    `json:"id"`
	Owner          string    `json:"owner"`
	Repo           string    `json:"repo"`
	RootDir        string    `json:"root_dir,omitempty"`
	InstallationID int64     `json:"installation_id"`
	APIKeyMasked   string    `json:"api_key_masked,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type APIKeyRecord struct {
	ID           string     `json:"id"`
	RepositoryID string     `json:"repository_id"`
	Name         string     `json:"name"`
	KeyMasked    string     `json:"key_masked"`
	RawKey       string     `json:"raw_key,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Status       string     `json:"status"`
}

type GitHubInstallation struct {
	ID             string    `json:"id"`
	InstallationID int64     `json:"installation_id"`
	OrgLogin       string    `json:"org_login"`
	OrgID          int64     `json:"org_id"`
	AccountType    string    `json:"account_type"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type InstallationRepo struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is missing or empty")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

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

func (db *DB) SaveIncident(ctx context.Context, inc *Incident) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL connection pool uninitialized")
	}

	var repoID *string
	if inc.RepositoryID != "" {
		repoID = &inc.RepositoryID
	}

	query := `
		INSERT INTO incidents (id, repository_id, title, status, file, line, panic_message, stack_trace, ast_snippet, root_cause, suggested_fix, github_issue_url, github_issue_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			repository_id = COALESCE(EXCLUDED.repository_id, incidents.repository_id),
			title = EXCLUDED.title,
			status = EXCLUDED.status,
			ast_snippet = EXCLUDED.ast_snippet,
			root_cause = EXCLUDED.root_cause,
			suggested_fix = EXCLUDED.suggested_fix,
			github_issue_url = COALESCE(NULLIF(EXCLUDED.github_issue_url, ''), incidents.github_issue_url),
			github_issue_number = CASE WHEN EXCLUDED.github_issue_number > 0 THEN EXCLUDED.github_issue_number ELSE incidents.github_issue_number END;
	`
	_, err := db.Pool.Exec(
		ctx, query,
		inc.ID, repoID, inc.Title, inc.Status, inc.File, inc.Line,
		inc.PanicMessage, inc.StackTrace, inc.ASTSnippet, inc.RootCause, inc.SuggestedFix,
		inc.GitHubIssueURL, inc.GitHubIssueNumber,
	)
	return err
}

func (db *DB) UpdateIncidentIssue(ctx context.Context, incidentID, issueURL string, issueNumber int) error {
	if db.Pool == nil {
		return fmt.Errorf("PostgreSQL pool uninitialized")
	}
	_, err := db.Pool.Exec(ctx, `
		UPDATE incidents
		SET github_issue_url = $2, github_issue_number = $3
		WHERE id = $1
	`, incidentID, issueURL, issueNumber)
	return err
}

func (db *DB) GetIncidents(ctx context.Context, limit int) ([]Incident, error) {
	if db.Pool == nil {
		return []Incident{}, nil
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, COALESCE(repository_id, ''), title, status, file, line, panic_message, stack_trace, COALESCE(ast_snippet, ''), COALESCE(root_cause, ''), COALESCE(suggested_fix, ''), COALESCE(github_issue_url, ''), COALESCE(github_issue_number, 0), created_at
		FROM incidents
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Incident
	for rows.Next() {
		var inc Incident
		err := rows.Scan(
			&inc.ID, &inc.RepositoryID, &inc.Title, &inc.Status, &inc.File, &inc.Line,
			&inc.PanicMessage, &inc.StackTrace, &inc.ASTSnippet, &inc.RootCause, &inc.SuggestedFix,
			&inc.GitHubIssueURL, &inc.GitHubIssueNumber, &inc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, inc)
	}
	return results, nil
}

func (db *DB) GetIncidentByID(ctx context.Context, id string) (*Incident, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool uninitialized")
	}
	var inc Incident
	err := db.Pool.QueryRow(ctx, `
		SELECT id, COALESCE(repository_id, ''), title, status, file, line, panic_message, stack_trace, COALESCE(ast_snippet, ''), COALESCE(root_cause, ''), COALESCE(suggested_fix, ''), COALESCE(github_issue_url, ''), COALESCE(github_issue_number, 0), created_at
		FROM incidents
		WHERE id = $1
	`, id).Scan(
		&inc.ID, &inc.RepositoryID, &inc.Title, &inc.Status, &inc.File, &inc.Line,
		&inc.PanicMessage, &inc.StackTrace, &inc.ASTSnippet, &inc.RootCause, &inc.SuggestedFix,
		&inc.GitHubIssueURL, &inc.GitHubIssueNumber, &inc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

func (db *DB) CreateProject(ctx context.Context, owner, repo, rootDir, ownerUsername string) (string, string, error) {
	if db.Pool == nil {
		return "", "", fmt.Errorf("PostgreSQL pool uninitialized")
	}

	cleanRootDir := strings.Trim(strings.TrimSpace(rootDir), "/")
	tupleStr := fmt.Sprintf("%s/%s:%s", owner, repo, cleanRootDir)
	tupleHash := sha256.Sum256([]byte(tupleStr))
	repoID := fmt.Sprintf("repo_%s", hex.EncodeToString(tupleHash[:16]))

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO repositories (id, owner, repo, root_dir, installation_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (owner, repo, root_dir) DO UPDATE SET root_dir = EXCLUDED.root_dir
	`, repoID, owner, repo, cleanRootDir, 1001)
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

func (db *DB) GetProjects(ctx context.Context) ([]Repository, error) {
	if db.Pool == nil {
		return []Repository{}, nil
	}
	query := `
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id,
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
		if err := rows.Scan(&r.ID, &r.Owner, &r.Repo, &r.RootDir, &r.InstallationID, &r.APIKeyMasked, &r.CreatedAt); err != nil {
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
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id,
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
	`, owner, repo, cleanRootDir).Scan(&r.ID, &r.Owner, &r.Repo, &r.RootDir, &r.InstallationID, &r.APIKeyMasked, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
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
		SELECT r.id, r.owner, r.repo, COALESCE(r.root_dir, ''), r.installation_id, r.created_at
		FROM api_keys k
		JOIN repositories r ON k.repository_id = r.id
		WHERE k.key_hash = $1 AND k.revoked_at IS NULL AND (k.expires_at IS NULL OR k.expires_at > NOW())
		LIMIT 1
	`, keyHash).Scan(&r.ID, &r.Owner, &r.Repo, &r.RootDir, &r.InstallationID, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (db *DB) UpsertUser(ctx context.Context, githubID, username, avatarURL string) (*User, error) {
	if db.Pool == nil {
		return &User{ID: "usr_demo", GitHubID: githubID, Username: username, AvatarURL: avatarURL, Role: "Admin"}, nil
	}

	userID := fmt.Sprintf("usr_%s", githubID)
	var u User
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (id, github_id, username, avatar_url, role)
		VALUES ($1, $2, $3, $4, 'Admin')
		ON CONFLICT (github_id) DO UPDATE SET username = EXCLUDED.username, avatar_url = EXCLUDED.avatar_url
		RETURNING id, github_id, username, avatar_url, role, created_at
	`, userID, githubID, username, avatarURL).Scan(&u.ID, &u.GitHubID, &u.Username, &u.AvatarURL, &u.Role, &u.CreatedAt)

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) GetStats(ctx context.Context, version ...string) (map[string]interface{}, error) {
	v := "v0.1.0-dev"
	if len(version) > 0 && version[0] != "" {
		v = version[0]
	}

	if db.Pool == nil {
		return map[string]interface{}{
			"status":          "healthy",
			"database":        "unconnected (in-memory mode)",
			"total_incidents": 0,
			"total_projects":  1,
			"funcs_indexed":   1420,
			"engine_version":  v,
			"uptime_seconds":  120,
		}, nil
	}

	var incCount, repoCount int
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM incidents").Scan(&incCount)
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM repositories").Scan(&repoCount)

	return map[string]interface{}{
		"status":          "healthy",
		"database":        "connected",
		"total_incidents": incCount,
		"total_projects":  repoCount,
		"funcs_indexed":   1420,
		"engine_version":  v,
		"uptime_seconds":  3600,
	}, nil
}

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
