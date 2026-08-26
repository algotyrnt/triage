// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPromptBuilders(t *testing.T) {
	crashPrompt := BuildAnalyzeCrashPrompt("goroutine 1 [running]:\nmain.go:10", "func main() {}", "Payment service context")
	if crashPrompt == "" {
		t.Fatal("expected non-empty crash prompt")
	}
	if !testing.Short() && len(crashPrompt) < 50 {
		t.Errorf("prompt unexpectedly short: %s", crashPrompt)
	}

	patchPrompt := BuildGeneratePatchPrompt("main.go", "nil pointer", "func main() {}", "stack trace", "nil ptr", "Domain context")
	if patchPrompt == "" {
		t.Fatal("expected non-empty patch prompt")
	}

	applyPrompt := BuildApplyFixPrompt("main.go", "package main", "nil pointer", "func main() {}", "stack", "nil ptr", "fix", "diff", "Domain context")
	if applyPrompt == "" {
		t.Fatal("expected non-empty apply prompt")
	}
}

func TestCleanMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		input    string
		tag      string
		expected string
	}{
		{"```json\n{\"root_cause\":\"test\"}\n```", "json", "{\"root_cause\":\"test\"}"},
		{"```diff\n--- a/file.go\n+++ b/file.go\n```", "diff", "--- a/file.go\n+++ b/file.go"},
		{"```go\npackage main\n```", "go", "package main"},
		{"```\nplain text\n```", "", "plain text"},
		{"raw content without fence", "", "raw content without fence"},
	}

	for _, tt := range tests {
		got := CleanMarkdownCodeBlock(tt.input, tt.tag)
		if got != tt.expected {
			t.Errorf("CleanMarkdownCodeBlock(%q, %q) = %q; want %q", tt.input, tt.tag, got, tt.expected)
		}
	}
}

func TestOpenAIProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-openai-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var req openAIChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		resp := openAIChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "```json\n{\"root_cause\": \"nil pointer dereference\", \"suggested_fix\": \"initialize pointer\"}\n```",
					},
					FinishReason: "stop",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(Config{
		Provider:   "openai",
		APIKey:     "test-openai-key",
		Model:      "gpt-4o",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("failed to create OpenAI provider: %v", err)
	}

	res, err := provider.AnalyzeCrash(context.Background(), "goroutine 1", "func test() {}")
	if err != nil {
		t.Fatalf("AnalyzeCrash failed: %v", err)
	}
	if res.RootCause != "nil pointer dereference" {
		t.Errorf("unexpected root cause: %s", res.RootCause)
	}
}

func TestAnthropicProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-claude-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		resp := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{
					Type: "text",
					Text: "```diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n- old\n+ new\n```",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewAnthropicProvider(Config{
		Provider:   "anthropic",
		APIKey:     "test-claude-key",
		Model:      "claude-3-5-sonnet-20241022",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("failed to create Anthropic provider: %v", err)
	}

	patch, err := provider.GeneratePatch(context.Background(), "main.go", "panic", "ast", "stack", "cause")
	if err != nil {
		t.Fatalf("GeneratePatch failed: %v", err)
	}
	if patch != "--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n- old\n+ new" {
		t.Errorf("unexpected patch content: %s", patch)
	}
}

func TestOllamaProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "```go\npackage main\n\nfunc main() {}\n```",
					},
					FinishReason: "stop",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider(Config{
		Provider:   "ollama",
		Model:      "deepseek-coder-v2",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("failed to create Ollama provider: %v", err)
	}

	code, err := provider.ApplyFixToFile(context.Background(), "main.go", "package main", "panic", "ast", "stack", "cause", "fix", "patch")
	if err != nil {
		t.Fatalf("ApplyFixToFile failed: %v", err)
	}
	if code != "package main\n\nfunc main() {}\n" {
		t.Errorf("unexpected applied code: %q", code)
	}
}
