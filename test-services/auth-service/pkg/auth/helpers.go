// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package auth

import "strings"

var (
	// DefaultIssuer token authority string.
	DefaultIssuer = "auth.triage.local"
	// BearerPrefix standard Authorization header prefix.
	BearerPrefix = "Bearer "
)

// ExtractBearerToken strips the "Bearer " prefix from an authorization header.
// NOTE: Bug simulation: Takes header[7:] directly without checking if len(header) >= 7.
func ExtractBearerToken(authHeader string) string {
	// PANIC SITE: Slice bounds out of range if len(authHeader) < 7 (e.g. authHeader is "Basic" or "")
	token := authHeader[7:]
	return strings.TrimSpace(token)
}

// HasRole checks whether a role list contains the target role.
func HasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
