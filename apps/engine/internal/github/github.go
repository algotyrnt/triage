// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"triage/engine/internal/llm"
)

type Client struct {
	AppID      string
	PrivateKey []byte
	HTTPClient *http.Client
}

type IssueRequest struct {
	File       string
	Line       int
	StackTrace string
	ASTSnippet string
	Analysis   *llm.AnalysisResult
}

type IssueResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
	Title   string `json:"title"`
}

type installationTokenResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewClient(appID string, privateKeyPEM []byte) *Client {
	return &Client{
		AppID:      appID,
		PrivateKey: privateKeyPEM,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) CreateIssue(ctx context.Context, owner string, repo string, installationID int64, req *IssueRequest) (*IssueResponse, error) {
	if c.AppID == "" || len(c.PrivateKey) == 0 {
		return nil, fmt.Errorf("github app ID or private key is missing")
	}

	jwtToken, err := c.GenerateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate GitHub App JWT: %w", err)
	}

	instToken, err := c.getInstallationAccessToken(ctx, jwtToken, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}

	fileName := req.File
	if idx := strings.LastIndex(fileName, "/"); idx != -1 {
		fileName = fileName[idx+1:]
	}

	title := fmt.Sprintf("[Triage] Panic in %s:%d", fileName, req.Line)
	body := formatIssueBody(req)

	payload := map[string]interface{}{
		"title":  title,
		"body":   body,
		"labels": []string{"triage-crash", "bug", "automated-triage"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue payload: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create issue HTTP request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+instToken)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("github API POST issue failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var issueResp IssueResponse
	if err := json.Unmarshal(respBytes, &issueResp); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub issue response: %w", err)
	}

	return &issueResp, nil
}

func (c *Client) getInstallationAccessToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to obtain installation access token (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp installationTokenResp
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.Token, nil
}

func (c *Client) GenerateJWT() (string, error) {
	block, _ := pem.Decode(c.PrivateKey)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsedKey, err8 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err8 != nil {
			return "", fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		privKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("private key is not RSA")
		}
	}

	now := time.Now().Unix()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	payload := map[string]interface{}{
		"iat": now - 60,
		"exp": now + 600,
		"iss": c.AppID,
	}

	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	b64Header := base64.RawURLEncoding.EncodeToString(headerJSON)
	b64Payload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	unsignedToken := b64Header + "." + b64Payload

	h := sha256.New()
	h.Write([]byte(unsignedToken))
	hashed := h.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, hashed)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	b64Sig := base64.RawURLEncoding.EncodeToString(sig)
	return unsignedToken + "." + b64Sig, nil
}

func formatIssueBody(req *IssueRequest) string {
	var sb strings.Builder

	sb.WriteString("## Triage Automated Crash Report\n\n")
	sb.WriteString(fmt.Sprintf("**Location:** `%s:%d`  \n", req.File, req.Line))
	sb.WriteString(fmt.Sprintf("**Timestamp:** `%s`  \n\n", time.Now().Format(time.RFC3339)))

	if req.Analysis != nil {
		sb.WriteString("### Root Cause Analysis (Gemini 2.5)\n\n")
		sb.WriteString(req.Analysis.RootCause)
		sb.WriteString("\n\n")

		sb.WriteString("### Suggested Fix\n\n```go\n")
		sb.WriteString(req.Analysis.SuggestedFix)
		sb.WriteString("\n```\n\n")
	}

	if req.ASTSnippet != "" {
		sb.WriteString("### Isolated Function AST Node\n\n```go\n")
		sb.WriteString(req.ASTSnippet)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("<details>\n<summary>Raw Stack Trace</summary>\n\n```text\n")
	sb.WriteString(req.StackTrace)
	sb.WriteString("\n```\n\n</details>\n\n")
	sb.WriteString("---\n*Reported automatically by Triage Go Telemetry Engine*")

	return sb.String()
}
