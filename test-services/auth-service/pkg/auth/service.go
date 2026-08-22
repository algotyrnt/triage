// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"time"
)

// SessionManager coordinates session validation and user profile extraction.
type SessionManager struct {
	IssuerName   string
	TokenTimeout time.Duration
}

// NewSessionManager creates a SessionManager instance with defaults.
func NewSessionManager(issuer string) *SessionManager {
	return &SessionManager{
		IssuerName:   issuer,
		TokenTimeout: 24 * time.Hour,
	}
}

// GetUserEmail extracts the user's primary contact email from an active session.
// NOTE: Bug simulation: Accesses session.Claims.User.Profile.Email without checking if session.Claims, User, or Profile are nil.
func (m *SessionManager) GetUserEmail(session *Session) (string, error) {
	if session == nil {
		return "", fmt.Errorf("session cannot be nil")
	}

	// PANIC SITE: Nested nil pointer dereference if session.Claims or session.Claims.User or session.Claims.User.Profile is nil
	email := session.Claims.User.Profile.Email
	return email, nil
}

// BroadcastAuditLog sends security events to an audit channel.
// NOTE: Bug simulation: Sends on a closed channel.
func (m *SessionManager) BroadcastAuditLog(ch chan string, message string) {
	// PANIC SITE: send on closed channel
	ch <- message
}
