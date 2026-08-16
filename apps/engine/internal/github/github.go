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
