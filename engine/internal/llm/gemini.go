// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

// GeminiProvider implements Provider using the official Google GenAI Go SDK.
type GeminiProvider struct {
	apiKey    string
	modelName string
}

// NewGeminiProvider creates a new Google Gemini LLM provider.
func NewGeminiProvider(cfg Config) (*GeminiProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{
		apiKey:    apiKey,
		modelName: model,
	}, nil
}

func (p *GeminiProvider) newClient(ctx context.Context) (*genai.Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: p.apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gemini client: %w", err)
	}
	return client, nil
}

func (p *GeminiProvider) AnalyzeCrash(ctx context.Context, stackTrace, astSnippet string, projectContext ...string) (*AnalysisResult, error) {
	client, err := p.newClient(ctx)
	if err != nil {
		return nil, err
	}

	prompt := BuildAnalyzeCrashPrompt(stackTrace, astSnippet, projectContext...)
	resp, err := client.Models.GenerateContent(ctx, p.modelName, genai.Text(prompt), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("gemini api call failed: %w", err)
	}

	rawText := CleanMarkdownCodeBlock(resp.Text(), "json")

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(rawText), &analysis); err != nil {
		slog.Error("failed to unmarshal Gemini response JSON", "error", err, "raw_response", rawText)
		return nil, fmt.Errorf("failed to parse Gemini JSON response: %w", err)
	}

	return &analysis, nil
}

func (p *GeminiProvider) GeneratePatch(ctx context.Context, file, panicMessage, astSnippet, stackTrace, rootCause string, projectContext ...string) (string, error) {
	client, err := p.newClient(ctx)
	if err != nil {
		return "", err
	}

	prompt := BuildGeneratePatchPrompt(file, panicMessage, astSnippet, stackTrace, rootCause, projectContext...)
	resp, err := client.Models.GenerateContent(ctx, p.modelName, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini patch generation failed: %w", err)
	}

	rawText := CleanMarkdownCodeBlock(resp.Text(), "diff")
	return rawText, nil
}

func (p *GeminiProvider) ApplyFixToFile(ctx context.Context, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch string, projectContext ...string) (string, error) {
	client, err := p.newClient(ctx)
	if err != nil {
		return "", err
	}

	prompt := BuildApplyFixPrompt(file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch, projectContext...)
	resp, err := client.Models.GenerateContent(ctx, p.modelName, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini apply fix failed: %w", err)
	}

	rawText := CleanMarkdownCodeBlock(resp.Text(), "go")

	if formatted, err := format.Source([]byte(rawText)); err == nil {
		return string(formatted), nil
	}

	if !strings.HasSuffix(rawText, "\n") {
		rawText += "\n"
	}
	return rawText, nil
}

func (p *GeminiProvider) TestConnection(ctx context.Context) error {
	client, err := p.newClient(ctx)
	if err != nil {
		return err
	}

	resp, err := client.Models.GenerateContent(ctx, p.modelName, genai.Text("Respond with 'OK'"), nil)
	if err != nil {
		return fmt.Errorf("gemini connection test failed: %w", err)
	}
	if strings.TrimSpace(resp.Text()) == "" {
		return fmt.Errorf("gemini returned empty response")
	}
	return nil
}
