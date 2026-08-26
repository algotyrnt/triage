// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// AnthropicProvider implements Provider for Anthropic's Claude models.
type AnthropicProvider struct {
	apiKey     string
	modelName  string
	baseURL    string
	httpClient *http.Client
}

// NewAnthropicProvider creates a provider targeted at Anthropic Claude models.
func NewAnthropicProvider(cfg Config) (*AnthropicProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	return &AnthropicProvider{
		apiKey:     apiKey,
		modelName:  model,
		baseURL:    baseURL,
		httpClient: client,
	}, nil
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) executeMessages(ctx context.Context, prompt string, maxTokens int) (string, error) {
	endpoint := fmt.Sprintf("%s/messages", p.baseURL)

	if maxTokens <= 0 {
		maxTokens = 4096
	}
	temp := 0.1

	reqPayload := anthropicRequest{
		Model:       p.modelName,
		MaxTokens:   maxTokens,
		Temperature: &temp,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to encode anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic messages request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errObj anthropicResponse
		if jsonErr := json.Unmarshal(respBytes, &errObj); jsonErr == nil && errObj.Error != nil && errObj.Error.Message != "" {
			return "", fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, errObj.Error.Message)
		}
		return "", fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var msgResp anthropicResponse
	if err := json.Unmarshal(respBytes, &msgResp); err != nil {
		return "", fmt.Errorf("failed to parse anthropic response: %w", err)
	}

	var sb strings.Builder
	for _, block := range msgResp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	return strings.TrimSpace(sb.String()), nil
}

func (p *AnthropicProvider) AnalyzeCrash(ctx context.Context, stackTrace, astSnippet string, projectContext ...string) (*AnalysisResult, error) {
	prompt := BuildAnalyzeCrashPrompt(stackTrace, astSnippet, projectContext...)
	content, err := p.executeMessages(ctx, prompt, 2048)
	if err != nil {
		return nil, err
	}

	rawText := CleanMarkdownCodeBlock(content, "json")

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(rawText), &analysis); err != nil {
		slog.Error("failed to unmarshal Anthropic response JSON", "error", err, "raw_response", rawText)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &analysis, nil
}

func (p *AnthropicProvider) GeneratePatch(ctx context.Context, file, panicMessage, astSnippet, stackTrace, rootCause string, projectContext ...string) (string, error) {
	prompt := BuildGeneratePatchPrompt(file, panicMessage, astSnippet, stackTrace, rootCause, projectContext...)
	content, err := p.executeMessages(ctx, prompt, 4096)
	if err != nil {
		return "", err
	}

	rawText := CleanMarkdownCodeBlock(content, "diff")
	return rawText, nil
}

func (p *AnthropicProvider) ApplyFixToFile(ctx context.Context, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch string, projectContext ...string) (string, error) {
	prompt := BuildApplyFixPrompt(file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch, projectContext...)
	content, err := p.executeMessages(ctx, prompt, 8192)
	if err != nil {
		return "", err
	}

	rawText := CleanMarkdownCodeBlock(content, "go")

	if formatted, err := format.Source([]byte(rawText)); err == nil {
		return string(formatted), nil
	}

	if !strings.HasSuffix(rawText, "\n") {
		rawText += "\n"
	}
	return rawText, nil
}

func (p *AnthropicProvider) TestConnection(ctx context.Context) error {
	_, err := p.executeMessages(ctx, "Respond with 'OK'", 10)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}
