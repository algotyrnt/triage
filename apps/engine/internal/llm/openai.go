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

// OpenAICompatibleProvider handles OpenAI, Ollama, vLLM, LM Studio, and LocalAI endpoints.
type OpenAICompatibleProvider struct {
	apiKey     string
	modelName  string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a provider targeted at the official OpenAI platform.
func NewOpenAIProvider(cfg Config) (*OpenAICompatibleProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("openai api key is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4o"
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	return &OpenAICompatibleProvider{
		apiKey:     apiKey,
		modelName:  model,
		baseURL:    baseURL,
		httpClient: client,
	}, nil
}

// NewOpenAICompatibleProvider creates a provider for Ollama, vLLM, or custom OpenAI-compatible endpoints.
func NewOpenAICompatibleProvider(cfg Config) (*OpenAICompatibleProvider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		if strings.ToLower(cfg.Provider) == "ollama" {
			baseURL = "http://localhost:11434/v1"
		} else {
			baseURL = "https://api.openai.com/v1"
		}
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		if strings.ToLower(cfg.Provider) == "ollama" {
			model = "deepseek-coder-v2"
		} else {
			model = "gpt-4o"
		}
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	return &OpenAICompatibleProvider{
		apiKey:     strings.TrimSpace(cfg.APIKey),
		modelName:  model,
		baseURL:    baseURL,
		httpClient: client,
	}, nil
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIChatRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIChatMessage   `json:"messages"`
	Temperature    *float64              `json:"temperature,omitempty"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func (p *OpenAICompatibleProvider) executeChat(ctx context.Context, prompt string, jsonMode bool) (string, error) {
	endpoint := fmt.Sprintf("%s/chat/completions", p.baseURL)

	reqPayload := openAIChatRequest{
		Model: p.modelName,
		Messages: []openAIChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	temp := 0.1
	reqPayload.Temperature = &temp
	if jsonMode {
		reqPayload.ResponseFormat = &openAIResponseFormat{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to encode chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat completion request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errObj openAIChatResponse
		if jsonErr := json.Unmarshal(respBytes, &errObj); jsonErr == nil && errObj.Error != nil && errObj.Error.Message != "" {
			return "", fmt.Errorf("chat completion error (%d): %s", resp.StatusCode, errObj.Error.Message)
		}
		return "", fmt.Errorf("chat completion returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse chat response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned by model")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

func (p *OpenAICompatibleProvider) AnalyzeCrash(ctx context.Context, stackTrace, astSnippet string, projectContext ...string) (*AnalysisResult, error) {
	prompt := BuildAnalyzeCrashPrompt(stackTrace, astSnippet, projectContext...)
	content, err := p.executeChat(ctx, prompt, true)
	if err != nil {
		// Fallback without strict json_object response_format if provider doesn't support it
		content, err = p.executeChat(ctx, prompt, false)
		if err != nil {
			return nil, err
		}
	}

	rawText := CleanMarkdownCodeBlock(content, "json")

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(rawText), &analysis); err != nil {
		slog.Error("failed to unmarshal OpenAI response JSON", "error", err, "raw_response", rawText)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &analysis, nil
}

func (p *OpenAICompatibleProvider) GeneratePatch(ctx context.Context, file, panicMessage, astSnippet, stackTrace, rootCause string, projectContext ...string) (string, error) {
	prompt := BuildGeneratePatchPrompt(file, panicMessage, astSnippet, stackTrace, rootCause, projectContext...)
	content, err := p.executeChat(ctx, prompt, false)
	if err != nil {
		return "", err
	}

	rawText := CleanMarkdownCodeBlock(content, "diff")
	return rawText, nil
}

func (p *OpenAICompatibleProvider) ApplyFixToFile(ctx context.Context, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch string, projectContext ...string) (string, error) {
	prompt := BuildApplyFixPrompt(file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch, projectContext...)
	content, err := p.executeChat(ctx, prompt, false)
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

func (p *OpenAICompatibleProvider) TestConnection(ctx context.Context) error {
	_, err := p.executeChat(ctx, "Respond with 'OK'", false)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}
