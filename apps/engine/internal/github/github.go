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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SetDefaultHeaders attaches standard GitHub API headers including User-Agent and API version.
func SetDefaultHeaders(req *http.Request) {
	if req != nil {
		req.Header.Set("User-Agent", "Triage-Engine")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
}

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
	SetDefaultHeaders(req)

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
	SetDefaultHeaders(req)

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
	SetDefaultHeaders(req)

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
				SetDefaultHeaders(retryReq)

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

func (c *AppConfig) GetDefaultBranch(ctx context.Context, installationID int64, owner, repo string) (string, string, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get token: %w", err)
	}

	repoURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", repoURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	SetDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch repo info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get repo info: status %d, body: %s", resp.StatusCode, b)
	}

	var repoPayload struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoPayload); err != nil {
		return "", "", fmt.Errorf("failed to decode repo payload: %w", err)
	}

	defaultBranch := repoPayload.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	refURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/heads/%s", owner, repo, defaultBranch)
	refReq, err := http.NewRequestWithContext(ctx, "GET", refURL, nil)
	if err != nil {
		return defaultBranch, "", fmt.Errorf("failed to create ref request: %w", err)
	}
	refReq.Header.Set("Authorization", "Bearer "+token)
	refReq.Header.Set("Accept", "application/vnd.github+json")
	SetDefaultHeaders(refReq)

	refResp, err := http.DefaultClient.Do(refReq)
	if err != nil {
		return defaultBranch, "", fmt.Errorf("failed to get ref: %w", err)
	}
	defer refResp.Body.Close()

	if refResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(refResp.Body)
		return defaultBranch, "", fmt.Errorf("failed to get branch ref: status %d, body: %s", refResp.StatusCode, b)
	}

	var refPayload struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(refResp.Body).Decode(&refPayload); err != nil {
		return defaultBranch, "", fmt.Errorf("failed to decode ref payload: %w", err)
	}

	return defaultBranch, refPayload.Object.SHA, nil
}

func (c *AppConfig) CreateBranch(ctx context.Context, installationID int64, owner, repo, branchName, baseSHA string) error {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs", owner, repo)
	payload := map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": baseSHA,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create branch request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	SetDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusUnprocessableEntity {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create branch: status %d, body: %s", resp.StatusCode, b)
	}

	return nil
}

func (c *AppConfig) UpdateFileContent(ctx context.Context, installationID int64, owner, repo, filePath, message, content, branch, fileSHA string) error {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, filePath)
	payload := map[string]interface{}{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  branch,
	}
	if fileSHA != "" {
		payload["sha"] = fileSHA
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create file update request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	SetDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to commit file: status %d, body: %s", resp.StatusCode, b)
	}

	return nil
}

func (c *AppConfig) CreatePullRequest(ctx context.Context, installationID int64, owner, repo, title, body, headBranch, baseBranch string) (int, string, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  headBranch,
		"base":  baseBranch,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create PR request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	SetDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create PR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("failed to create pull request: status %d, body: %s", resp.StatusCode, b)
	}

	var prPayload struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prPayload); err != nil {
		return 0, "", fmt.Errorf("failed to decode PR response: %w", err)
	}

	return prPayload.Number, prPayload.HTMLURL, nil
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
	SetDefaultHeaders(req)

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
		SetDefaultHeaders(req)

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
		SetDefaultHeaders(req)

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

// FetchUserRepositories fetches all repositories accessible to the given user token.
// NOTE: This function is kept for potential tooling/test use.
// The dashboard fetches user repos directly from the browser; the engine uses GitHub App tokens.
func FetchUserRepositories(ctx context.Context, username string, accessToken ...string) ([]RepositoryInfo, error) {
	token := ""
	if len(accessToken) > 0 && accessToken[0] != "" {
		token = accessToken[0]
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GITHUB_ACCESS_TOKEN")
	}

	if token == "" {
		return nil, fmt.Errorf("GitHub authentication token is required to fetch repositories (public fallback disabled)")
	}

	var allRepos []RepositoryInfo
	seen := make(map[string]bool)
	client := &http.Client{Timeout: 15 * time.Second}

	// 1. Fetch user & affiliated repositories (both public and private)
	page := 1
	for page <= 10 {
		url := fmt.Sprintf("https://api.github.com/user/repos?visibility=all&per_page=100&page=%d&sort=updated&affiliation=owner,collaborator,organization_member", page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		SetDefaultHeaders(req)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user repos: %w", err)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, fmt.Errorf("github token expired or invalid (HTTP 401)")
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
			key := strings.ToLower(fmt.Sprintf("%s/%s", r.Owner.Login, r.Name))
			if !seen[key] {
				seen[key] = true
				allRepos = append(allRepos, RepositoryInfo{
					Owner:         r.Owner.Login,
					Repo:          r.Name,
					DefaultBranch: r.DefaultBranch,
					Language:      r.Language,
					Private:       r.Private,
				})
			}
		}

		if len(items) < 100 {
			break
		}
		page++
	}

	// 2. Discover organization memberships and fetch all their repos
	orgsReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/orgs?per_page=100", nil)
	if err == nil {
		orgsReq.Header.Set("Authorization", "Bearer "+token)
		orgsReq.Header.Set("Accept", "application/vnd.github+json")
		SetDefaultHeaders(orgsReq)

		if orgsResp, oErr := client.Do(orgsReq); oErr == nil && orgsResp.StatusCode == http.StatusOK {
			var orgs []struct {
				Login string `json:"login"`
			}
			_ = json.NewDecoder(orgsResp.Body).Decode(&orgs)
			orgsResp.Body.Close()

			for _, org := range orgs {
				orgPage := 1
				for orgPage <= 5 {
					orgUrl := fmt.Sprintf("https://api.github.com/orgs/%s/repos?type=all&per_page=100&page=%d", org.Login, orgPage)
					oReq, rErr := http.NewRequestWithContext(ctx, "GET", orgUrl, nil)
					if rErr != nil {
						break
					}
					oReq.Header.Set("Authorization", "Bearer "+token)
					oReq.Header.Set("Accept", "application/vnd.github+json")
					SetDefaultHeaders(oReq)

					oResp, dErr := client.Do(oReq)
					if dErr != nil || oResp.StatusCode != http.StatusOK {
						if oResp != nil {
							oResp.Body.Close()
						}
						break
					}

					var orgItems []struct {
						Name          string `json:"name"`
						Private       bool   `json:"private"`
						DefaultBranch string `json:"default_branch"`
						Language      string `json:"language"`
						Owner         struct {
							Login string `json:"login"`
						} `json:"owner"`
					}
					_ = json.NewDecoder(oResp.Body).Decode(&orgItems)
					oResp.Body.Close()

					if len(orgItems) == 0 {
						break
					}

					for _, r := range orgItems {
						key := strings.ToLower(fmt.Sprintf("%s/%s", r.Owner.Login, r.Name))
						if !seen[key] {
							seen[key] = true
							allRepos = append(allRepos, RepositoryInfo{
								Owner:         r.Owner.Login,
								Repo:          r.Name,
								DefaultBranch: r.DefaultBranch,
								Language:      r.Language,
								Private:       r.Private,
							})
						}
					}

					if len(orgItems) < 100 {
						break
					}
					orgPage++
				}
			}
		}
	}

	return allRepos, nil
}
