// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"testing"
)

func TestSessionJWT_Lifecycle(t *testing.T) {
	secret := "super-secure-session-secret-12345"
	userID := "usr_12345"
	username := "algotyrnt"
	avatarURL := "https://avatars.githubusercontent.com/u/12345"
	githubID := "12345"
	githubToken := "gho_sampleTemporaryAccessToken"

	tokenString, err := MintSessionJWT(userID, username, avatarURL, githubID, githubToken, secret)
	if err != nil {
		t.Fatalf("failed to mint session JWT: %v", err)
	}

	claims, err := ValidateSessionJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("failed to validate session JWT: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("expected Username %s, got %s", username, claims.Username)
	}
	if claims.GitHubToken != githubToken {
		t.Errorf("expected GitHubToken %s, got %s", githubToken, claims.GitHubToken)
	}

	// Validation with wrong secret fails
	_, err = ValidateSessionJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Errorf("expected validation failure with wrong secret")
	}
}
