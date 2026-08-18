// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AppConfig struct {
	AppID              int64
	PrivateKey         *rsa.PrivateKey
	WebhookSecret      string
	ClientID           string
	ClientSecret       string
	installationTokens sync.Map
}

type InstallationToken struct {
	Token     string
	ExpiresAt time.Time
}

func LoadAppConfig(appID int64, pemKeyData []byte, webhookSecret, clientID, clientSecret string) (*AppConfig, error) {
	block, _ := pem.Decode(pemKeyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try parsing as PKCS8
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: pkcs1 err: %v, pkcs8 err: %v", err, err2)
		}
		var ok bool
		privKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
	}

	return &AppConfig{
		AppID:         appID,
		PrivateKey:    privKey,
		WebhookSecret: webhookSecret,
		ClientID:      clientID,
		ClientSecret:  clientSecret,
	}, nil
}

func (c *AppConfig) SignAppJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    strconv.FormatInt(c.AppID, 10),
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(c.PrivateKey)
}

func (c *AppConfig) GetInstallationToken(ctx context.Context, installationID int64) (string, error) {
	if val, ok := c.installationTokens.Load(installationID); ok {
		tok := val.(InstallationToken)
		if time.Until(tok.ExpiresAt) > 5*time.Minute {
			return tok.Token, nil
		}
	}

	appJWT, err := c.SignAppJWT()
	if err != nil {
		return "", fmt.Errorf("failed to sign app jwt: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create token, status: %d, body: %s", resp.StatusCode, body)
	}

	var payload struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	c.installationTokens.Store(installationID, InstallationToken{
		Token:     payload.Token,
		ExpiresAt: payload.ExpiresAt,
	})

	return payload.Token, nil
}

func (c *AppConfig) FetchFileContent(ctx context.Context, installationID int64, owner, repo, commitSHA, filePath string) ([]byte, string, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, filePath)
	if commitSHA != "" {
		url = fmt.Sprintf("%s?ref=%s", url, commitSHA)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("failed to fetch file content, status: %d, body: %s", resp.StatusCode, body)
	}

	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		SHA      string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	if payload.Encoding != "base64" && payload.Content != "" {
		return nil, "", fmt.Errorf("unsupported encoding: %s", payload.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(payload.Content)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return decoded, payload.SHA, nil
}

func (c *AppConfig) CreateIssue(ctx context.Context, installationID int64, owner, repo, title, body string, labels []string) (int, string, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)

	payloadReq := struct {
		Title  string   `json:"title"`
		Body   string   `json:"body"`
		Labels []string `json:"labels"`
	}{
		Title:  title,
		Body:   body,
		Labels: labels,
	}

	payloadBytes, err := json.Marshal(payloadReq)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		// If labels are restricted or failed, retry without labels once
		if len(labels) > 0 {
			payloadReq.Labels = nil
			retryBytes, _ := json.Marshal(payloadReq)
			retryReq, rErr := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(retryBytes))
			if rErr == nil {
				retryReq.Header.Set("Authorization", "Bearer "+token)
				retryReq.Header.Set("Accept", "application/vnd.github+json")
				retryReq.Header.Set("Content-Type", "application/json")
				retryReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

				retryResp, doErr := http.DefaultClient.Do(retryReq)
				if doErr == nil {
					defer retryResp.Body.Close()
					if retryResp.StatusCode == http.StatusCreated {
						var retryPayload struct {
							Number  int    `json:"number"`
							HTMLURL string `json:"html_url"`
						}
						if err := json.NewDecoder(retryResp.Body).Decode(&retryPayload); err == nil {
							return retryPayload.Number, retryPayload.HTMLURL, nil
						}
					}
				}
			}
		}
		return 0, "", fmt.Errorf("failed to create issue, status: %d, body: %s", resp.StatusCode, respBody)
	}

	var payload struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return payload.Number, payload.HTMLURL, nil
}

func (c *AppConfig) VerifyApp(ctx context.Context) error {
	appJWT, err := c.SignAppJWT()
	if err != nil {
		return fmt.Errorf("failed to sign app jwt: %w", err)
	}

	url := "https://api.github.com/app"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to verify app, status: %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

type AppInstallationInfo struct {
	ID           int64  `json:"id"`
	AccountType  string `json:"account_type"`
	AccountLogin string `json:"account_login"`
	AccountID    int64  `json:"account_id"`
}

type RepositoryInfo struct {
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	DefaultBranch string `json:"default_branch"`
	Language      string `json:"language"`
	Private       bool   `json:"private"`
}

func (c *AppConfig) ListAppInstallations(ctx context.Context) ([]AppInstallationInfo, error) {
	appJWT, err := c.SignAppJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to sign app jwt: %w", err)
	}

	var allInstallations []AppInstallationInfo
	page := 1
	client := &http.Client{Timeout: 15 * time.Second}

	for {
		url := fmt.Sprintf("https://api.github.com/app/installations?per_page=100&page=%d", page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+appJWT)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to list installations: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("failed to list installations, status: %d, body: %s", resp.StatusCode, body)
		}

		var items []struct {
			ID      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
				ID    int64  `json:"id"`
				Type  string `json:"type"`
			} `json:"account"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode installations: %w", err)
		}
		resp.Body.Close()

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			allInstallations = append(allInstallations, AppInstallationInfo{
				ID:           item.ID,
				AccountLogin: item.Account.Login,
				AccountID:    item.Account.ID,
				AccountType:  item.Account.Type,
			})
		}

		if len(items) < 100 {
			break
		}
		page++
	}

	return allInstallations, nil
}

func (c *AppConfig) ListInstallationRepositories(ctx context.Context, installationID int64) ([]RepositoryInfo, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}

	var allRepos []RepositoryInfo
	page := 1
	client := &http.Client{Timeout: 15 * time.Second}

	for {
		url := fmt.Sprintf("https://api.github.com/installation/repositories?per_page=100&page=%d", page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch installation repositories: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch repositories, status: %d, body: %s", resp.StatusCode, body)
		}

		var payload struct {
			TotalCount   int `json:"total_count"`
			Repositories []struct {
				Name          string `json:"name"`
				FullName      string `json:"full_name"`
				Private       bool   `json:"private"`
				DefaultBranch string `json:"default_branch"`
				Language      string `json:"language"`
				Owner         struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repositories"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode repositories response: %w", err)
		}
		resp.Body.Close()

		if len(payload.Repositories) == 0 {
			break
		}

		for _, r := range payload.Repositories {
			allRepos = append(allRepos, RepositoryInfo{
				Owner:         r.Owner.Login,
				Repo:          r.Name,
				DefaultBranch: r.DefaultBranch,
				Language:      r.Language,
				Private:       r.Private,
			})
		}

		if len(allRepos) >= payload.TotalCount || len(payload.Repositories) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}

func FetchUserRepositories(ctx context.Context, username string) ([]RepositoryInfo, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	var allRepos []RepositoryInfo
	page := 1
	client := &http.Client{Timeout: 15 * time.Second}

	for page <= 3 {
		url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&page=%d&sort=updated", username, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user repos: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			break
		}

		var items []struct {
			Name          string `json:"name"`
			Private       bool   `json:"private"`
			DefaultBranch string `json:"default_branch"`
			Language      string `json:"language"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			resp.Body.Close()
			break
		}
		resp.Body.Close()

		if len(items) == 0 {
			break
		}

		for _, r := range items {
			allRepos = append(allRepos, RepositoryInfo{
				Owner:         r.Owner.Login,
				Repo:          r.Name,
				DefaultBranch: r.DefaultBranch,
				Language:      r.Language,
				Private:       r.Private,
			})
		}

		if len(items) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}
