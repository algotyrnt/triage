// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"triage/engine/internal/session"
)

func (s *Server) getGitHubOAuthCredentials(ctx context.Context) (clientID, clientSecret string) {
	if s.db != nil {
		clientID, _ = s.db.GetInstanceConfig(ctx, "github_oauth_client_id")
		clientSecret, _ = s.db.GetInstanceConfig(ctx, "github_oauth_client_secret")
	}
	if v := os.Getenv("GITHUB_OAUTH_CLIENT_ID"); v != "" {
		clientID = v
	} else if clientID == "" {
		clientID = os.Getenv("GITHUB_CLIENT_ID")
	}
	if v := os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"); v != "" {
		clientSecret = v
	} else if clientSecret == "" {
		clientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	}
	return clientID, clientSecret
}

func (s *Server) HandleGitHubAuth(w http.ResponseWriter, r *http.Request) {
	clientID, _ := s.getGitHubOAuthCredentials(r.Context())
	if clientID == "" {
		http.Redirect(w, r, s.appURL+"?user=algotyrnt&auth=dev", http.StatusFound)
		return
	}

	callbackURL := fmt.Sprintf("%s/api/v1/auth/github/callback", s.engineURL)
	redirectURI := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=repo,read:org,user:email,read:user", clientID, callbackURL)
	http.Redirect(w, r, redirectURI, http.StatusFound)
}

func (s *Server) HandleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, s.appURL+"?auth=error&reason=missing_code", http.StatusFound)
		return
	}

	clientID, clientSecret := s.getGitHubOAuthCredentials(r.Context())
	if clientID == "" || clientSecret == "" {
		http.Redirect(w, r, s.appURL+"?auth=error&reason=oauth_not_configured", http.StatusFound)
		return
	}

	tokenURL := "https://github.com/login/oauth/access_token"
	tokenBody := fmt.Sprintf(`{"client_id":"%s","client_secret":"%s","code":"%s"}`, clientID, clientSecret, code)
	tokenReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenURL, strings.NewReader(tokenBody))
	tokenReq.Header.Set("Accept", "application/json")
	tokenReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		slog.Error("OAuth token exchange failed", "error", err)
		http.Redirect(w, r, s.appURL+"?auth=error&reason=token_exchange_failed", http.StatusFound)
		return
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	_ = json.NewDecoder(tokenResp.Body).Decode(&tokenData)
	if tokenData.AccessToken == "" {
		slog.Warn("no access token returned from GitHub", "github_error", tokenData.Error)
		http.Redirect(w, r, s.appURL+"?auth=error&reason=no_access_token", http.StatusFound)
		return
	}

	userReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := client.Do(userReq)
	if err != nil {
		slog.Error("failed to fetch authenticated GitHub user", "error", err)
		http.Redirect(w, r, s.appURL+"?auth=error&reason=user_fetch_failed", http.StatusFound)
		return
	}
	defer userResp.Body.Close()

	var userData struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	_ = json.NewDecoder(userResp.Body).Decode(&userData)
	if userData.Login == "" {
		http.Redirect(w, r, s.appURL+"?auth=error&reason=invalid_user", http.StatusFound)
		return
	}

	githubIDStr := strconv.FormatInt(userData.ID, 10)
	if s.db != nil {
		s.db.UpsertUser(r.Context(), githubIDStr, userData.Login, userData.AvatarURL)
	}

	userID := fmt.Sprintf("usr_%s", githubIDStr)
	token, err := session.MintSessionJWT(userID, userData.Login, userData.AvatarURL, githubIDStr, tokenData.AccessToken, s.sessionSecret)
	if err != nil {
		slog.Error("JWT session minting failed", "error", err, "username", userData.Login)
		http.Redirect(w, r, s.appURL+"?auth=error&reason=jwt_error", http.StatusFound)
		return
	}

	slog.Info("user authenticated successfully", "username", userData.Login, "github_id", userData.ID)
	http.Redirect(w, r, fmt.Sprintf("%s?token=%s&auth=success", s.appURL, token), http.StatusFound)
}

func (s *Server) HandleAuthMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid authorization header"})
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := s.ValidateSessionToken(tokenStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired session"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         claims.UserID,
		"username":   claims.Username,
		"avatar_url": claims.AvatarURL,
		"github_id":  claims.GitHubID,
	})
}
