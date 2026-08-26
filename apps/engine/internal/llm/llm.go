// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"context"
	"net/http"
	"strings"
)

// ProviderType represents supported AI model backends.
type ProviderType string

const (
	ProviderGemini    ProviderType = "gemini"
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderOllama    ProviderType = "ollama"
	ProviderCustom    ProviderType = "custom"
)

// Config encapsulates configuration required to initialize an LLM provider.
type Config struct {
	Provider   string       `json:"provider"`
	APIKey     string       `json:"api_key"`
	Model      string       `json:"model"`
	BaseURL    string       `json:"base_url,omitempty"`
	HTTPClient *http.Client `json:"-"`
}

// AnalysisResult contains structured incident diagnosis output.
type AnalysisResult struct {
	RootCause    string `json:"root_cause"`
	SuggestedFix string `json:"suggested_fix"`
	Severity     string `json:"severity,omitempty"`
	Explanation  string `json:"explanation,omitempty"`
}

// Provider represents a pluggable LLM backend for crash diagnosis and automated patching.
type Provider interface {
	AnalyzeCrash(ctx context.Context, stackTrace, astSnippet string, projectContext ...string) (*AnalysisResult, error)
	GeneratePatch(ctx context.Context, file, panicMessage, astSnippet, stackTrace, rootCause string, projectContext ...string) (string, error)
	ApplyFixToFile(ctx context.Context, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch string, projectContext ...string) (string, error)
	TestConnection(ctx context.Context) error
}

// NewProvider instantiates the appropriate LLM provider based on the provided configuration.
func NewProvider(cfg Config) (Provider, error) {
	providerType := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerType == "" {
		providerType = string(ProviderGemini)
	}

	switch ProviderType(providerType) {
	case ProviderGemini:
		return NewGeminiProvider(cfg)
	case ProviderOpenAI:
		return NewOpenAIProvider(cfg)
	case ProviderAnthropic:
		return NewAnthropicProvider(cfg)
	case ProviderOllama, ProviderCustom:
		return NewOpenAICompatibleProvider(cfg)
	default:
		// Default to OpenAI-compatible provider if unknown or custom
		return NewOpenAICompatibleProvider(cfg)
	}
}

// Standalone convenience functions that create an ephemeral provider on the fly.

func AnalyzeCrash(ctx context.Context, cfg Config, stackTrace, astSnippet string, projectContext ...string) (*AnalysisResult, error) {
	p, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}
	return p.AnalyzeCrash(ctx, stackTrace, astSnippet, projectContext...)
}

func GeneratePatch(ctx context.Context, cfg Config, file, panicMessage, astSnippet, stackTrace, rootCause string, projectContext ...string) (string, error) {
	p, err := NewProvider(cfg)
	if err != nil {
		return "", err
	}
	return p.GeneratePatch(ctx, file, panicMessage, astSnippet, stackTrace, rootCause, projectContext...)
}

func ApplyFixToFile(ctx context.Context, cfg Config, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch string, projectContext ...string) (string, error) {
	p, err := NewProvider(cfg)
	if err != nil {
		return "", err
	}
	return p.ApplyFixToFile(ctx, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch, projectContext...)
}
