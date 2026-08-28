// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package db

import "time"

// ASTNode represents an indexed AST function declaration.
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

// Incident represents a captured panic crash and its AI-analyzed diagnosis.
type Incident struct {
	ID                string    `json:"id"`
	RepositoryID      string    `json:"repository_id,omitempty"`
	RepositoryName    string    `json:"repository_name,omitempty"`
	Fingerprint       string    `json:"fingerprint,omitempty"`
	OccurrenceCount   int       `json:"occurrence_count"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	Severity          string    `json:"severity,omitempty"`
	AIProvider        string    `json:"ai_provider,omitempty"`
	AIModel           string    `json:"ai_model,omitempty"`
	File              string    `json:"file"`
	Line              int       `json:"line"`
	PanicMessage      string    `json:"panic_message"`
	StackTrace        string    `json:"stack_trace"`
	ASTSnippet        string    `json:"ast_snippet,omitempty"`
	RootCause         string    `json:"root_cause,omitempty"`
	SuggestedFix      string    `json:"suggested_fix,omitempty"`
	SuggestedPatch    string    `json:"suggested_patch,omitempty"`
	GitHubIssueURL    string    `json:"github_issue_url,omitempty"`
	GitHubIssueNumber int       `json:"github_issue_number,omitempty"`
	GitHubPRURL       string    `json:"github_pr_url,omitempty"`
	GitHubPRNumber    int       `json:"github_pr_number,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
}

// User represents an authenticated dashboard user with RBAC role.
type User struct {
	ID        string    `json:"id"`
	GitHubID  string    `json:"github_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Invitation represents a pending team invitation for a GitHub username.
type Invitation struct {
	ID             string    `json:"id"`
	GitHubUsername string    `json:"github_username"`
	Role           string    `json:"role"`
	InvitedBy      string    `json:"invited_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Repository represents a configured repository and monorepo root directory.
type Repository struct {
	ID             string    `json:"id"`
	Owner          string    `json:"owner"`
	Repo           string    `json:"repo"`
	RootDir        string    `json:"root_dir,omitempty"`
	InstallationID int64     `json:"installation_id"`
	Context        string    `json:"context,omitempty"`
	APIKeyMasked   string    `json:"api_key_masked,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// APIKeyRecord represents an ingestion API key.
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

// GitHubInstallation represents a connected GitHub App installation.
type GitHubInstallation struct {
	ID             string    `json:"id"`
	InstallationID int64     `json:"installation_id"`
	OrgLogin       string    `json:"org_login"`
	OrgID          int64     `json:"org_id"`
	AccountType    string    `json:"account_type"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// InstallationRepo maps an installation to an accessible repository.
type InstallationRepo struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}
