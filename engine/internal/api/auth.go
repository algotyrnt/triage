// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"triage/engine/internal/db"
	"triage/engine/internal/github"

	"github.com/golang-jwt/jwt/v5"
)

// UserClaims defines the structured payload inside signed Triage session JWTs.
type UserClaims struct {
	UserID    string `json:"user_id"`
	GitHubID  string `json:"github_id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateUserJWT issues a standard signed JWT for an authenticated user.
func GenerateUserJWT(user *db.User, secret string) (string, error) {
	if secret == "" {
		secret = "dev-secret-change-me-in-production"
	}

	now := time.Now()
	claims := UserClaims{
		UserID:    user.ID,
		GitHubID:  user.GitHubID,
		Username:  user.Username,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "triage",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)), // 30-day session
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAndVerifyUserJWT validates the token signature and returns the user claims.
func ParseAndVerifyUserJWT(tokenString, secret string) (*UserClaims, error) {
	if secret == "" {
		secret = "dev-secret-change-me-in-production"
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func (s *Server) getSessionSecret(ctx context.Context) string {
	if s.db != nil && ctx != nil {
		if secret, _ := s.db.GetInstanceConfig(ctx, "session_secret"); secret != "" {
			return secret
		}
	}
	return "dev-secret-change-me-in-production"
}

// HandleAuthGitHub starts the secure GitHub OAuth login flow.
func (s *Server) HandleAuthGitHub(w http.ResponseWriter, r *http.Request) {
	targetAppURL := s.ResolveAppURL(r.Context(), r)
	engineURL := s.ResolveEngineURL(r)

	clientID := ""
	if s.db != nil {
		clientID, _ = s.db.GetInstanceConfig(r.Context(), "github_oauth_client_id")
		if clientID == "" {
			clientID, _ = s.db.GetInstanceConfig(r.Context(), "github_app_client_id")
		}
		if clientID == "" {
			clientID, _ = s.db.GetInstanceConfig(r.Context(), "github_client_id")
		}
	}

	if clientID == "" {
		slog.Warn("HandleAuthGitHub: GitHub OAuth client ID not configured")
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=oauth_not_configured", http.StatusFound)
		return
	}

	// Generate cryptographic CSRF state
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Set short-lived secure CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "triage_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	callbackURL := engineURL + "/api/v1/auth/github/callback"
	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape("user:email,read:user"),
		url.QueryEscape(state),
	)

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// HandleAuthGitHubCallback receives the authorization code, verifies identity, and issues a role-based JWT.
func (s *Server) HandleAuthGitHubCallback(w http.ResponseWriter, r *http.Request) {
	targetAppURL := s.ResolveAppURL(r.Context(), r)

	// Validate CSRF state
	stateCookie, _ := r.Cookie("triage_oauth_state")
	queryState := r.URL.Query().Get("state")
	if stateCookie == nil || stateCookie.Value == "" || stateCookie.Value != queryState {
		slog.Warn("HandleAuthGitHubCallback: invalid OAuth CSRF state")
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=invalid_state", http.StatusFound)
		return
	}

	// Clear CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "triage_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("HandleAuthGitHubCallback: missing authorization code")
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=missing_code", http.StatusFound)
		return
	}

	clientID := ""
	clientSecret := ""
	if s.db != nil {
		clientID, _ = s.db.GetInstanceConfig(r.Context(), "github_oauth_client_id")
		if clientID == "" {
			clientID, _ = s.db.GetInstanceConfig(r.Context(), "github_app_client_id")
		}
		if clientID == "" {
			clientID, _ = s.db.GetInstanceConfig(r.Context(), "github_client_id")
		}

		clientSecret, _ = s.db.GetInstanceConfig(r.Context(), "github_oauth_client_secret")
		if clientSecret == "" {
			clientSecret, _ = s.db.GetInstanceConfig(r.Context(), "github_app_client_secret")
		}
		if clientSecret == "" {
			clientSecret, _ = s.db.GetInstanceConfig(r.Context(), "github_client_secret")
		}
	}

	if clientID == "" || clientSecret == "" {
		slog.Warn("HandleAuthGitHubCallback: OAuth credentials missing in database")
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=oauth_not_configured", http.StatusFound)
		return
	}

	// Exchange authorization code for GitHub access token
	tokenReqBody, _ := json.Marshal(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
	})

	tokenReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewReader(tokenReqBody))
	if err != nil {
		slog.Error("failed to construct token exchange request", "error", err)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=token_exchange_failed", http.StatusFound)
		return
	}
	tokenReq.Header.Set("Content-Type", "application/json")
	tokenReq.Header.Set("Accept", "application/json")
	github.SetDefaultHeaders(tokenReq)

	client := &http.Client{Timeout: 15 * time.Second}
	tokenResp, err := client.Do(tokenReq)
	if err != nil || tokenResp.StatusCode != http.StatusOK {
		slog.Error("failed token exchange with GitHub", "error", err)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=token_exchange_failed", http.StatusFound)
		return
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil || tokenData.AccessToken == "" {
		slog.Error("invalid token response from GitHub", "error", err, "gh_error", tokenData.Error)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=token_exchange_failed", http.StatusFound)
		return
	}

	// Fetch user profile from GitHub API
	userReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		slog.Error("failed to construct user profile request", "error", err)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=user_fetch_failed", http.StatusFound)
		return
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")
	github.SetDefaultHeaders(userReq)

	userResp, err := client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		slog.Error("failed to fetch user profile from GitHub", "error", err)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=user_fetch_failed", http.StatusFound)
		return
	}
	defer userResp.Body.Close()

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil || ghUser.Login == "" {
		slog.Error("failed to decode GitHub user response", "error", err)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=user_fetch_failed", http.StatusFound)
		return
	}

	// Upsert user into database with RBAC role assignment
	githubIDStr := strconv.FormatInt(ghUser.ID, 10)
	var userRecord *db.User
	if s.db != nil {
		userRecord, err = s.db.UpsertUserWithRole(r.Context(), githubIDStr, ghUser.Login, ghUser.Email, ghUser.AvatarURL, "Developer")
		if err != nil {
			slog.Error("failed to upsert user in database", "error", err)
			http.Redirect(w, r, targetAppURL+"?auth=error&reason=db_error", http.StatusFound)
			return
		}
	} else {
		userRecord = &db.User{
			ID:        "usr_" + githubIDStr,
			GitHubID:  githubIDStr,
			Username:  ghUser.Login,
			Email:     ghUser.Email,
			AvatarURL: ghUser.AvatarURL,
			Role:      "Owner",
		}
	}

	// Issue signed session JWT
	jwtSecret := s.getSessionSecret(r.Context())
	sessionToken, err := GenerateUserJWT(userRecord, jwtSecret)
	if err != nil {
		slog.Error("failed to generate session JWT", "error", err)
		http.Redirect(w, r, targetAppURL+"?auth=error&reason=jwt_error", http.StatusFound)
		return
	}

	slog.Info("user authenticated successfully", "username", userRecord.Username, "role", userRecord.Role)

	// Redirect back to Dashboard with token
	http.Redirect(w, r, targetAppURL+"?token="+url.QueryEscape(sessionToken)+"&auth=success", http.StatusFound)
}

// HandleAuthMe returns the profile and active role of the authenticated caller.
func (s *Server) HandleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jwtSecret := s.getSessionSecret(r.Context())
	claims, err := ParseAndVerifyUserJWT(tokenStr, jwtSecret)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// If DB is connected, fetch fresh user data to reflect immediate role updates
	var user = map[string]interface{}{
		"id":         claims.UserID,
		"github_id":  claims.GitHubID,
		"username":   claims.Username,
		"email":      claims.Email,
		"avatar_url": claims.AvatarURL,
		"role":       claims.Role,
	}

	if s.db != nil {
		if dbUser, err := s.db.GetUserByID(r.Context(), claims.UserID); err == nil && dbUser != nil {
			user["username"] = dbUser.Username
			user["email"] = dbUser.Email
			user["avatar_url"] = dbUser.AvatarURL
			user["role"] = dbUser.Role
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}
