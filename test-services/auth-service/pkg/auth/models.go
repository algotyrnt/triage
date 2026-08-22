// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package auth

import "time"

// UserProfile holds personal and security profile information.
type UserProfile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// User represents an authenticated account in the system.
type User struct {
	ID        string       `json:"id"`
	Username  string       `json:"username"`
	Profile   *UserProfile `json:"profile"`
	Roles     []string     `json:"roles"`
	IsActive  bool         `json:"is_active"`
	CreatedAt time.Time    `json:"created_at"`
}

// JWTClaims represents parsed claims from an authorization token.
type JWTClaims struct {
	Subject   string    `json:"sub"`
	User      *User     `json:"user"`
	Issuer    string    `json:"iss"`
	ExpiresAt time.Time `json:"exp"`
}

// Session represents an active login session with token metadata.
type Session struct {
	SessionID string     `json:"session_id"`
	Claims    *JWTClaims `json:"claims"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
}
