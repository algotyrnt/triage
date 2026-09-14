// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func jsonHTTPResponse(status int, body interface{}) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(http.Header),
	}
}

func TestPromptBuilders(t *testing.T) {
	crashPrompt := BuildAnalyzeCrashPrompt("goroutine 1 [running]:\nmain.go:10", "func main() {}", "Payment service context")
	if crashPrompt == "" {
		t.Fatal("expected non-empty crash prompt")
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
		{"```json\n{\"tag\":\"none\"}\n```", "nonexistent", "{\"tag\":\"none\"}"},
		{"```go\npackage sample\n```", "nonexistent", "package sample"},
		{"```diff\n--- a\n+++ b\n```", "nonexistent", "--- a\n+++ b"},
		{"```\nplain text\n```", "nonexistent", "plain text"},
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

func TestNewProvider_Factory(t *testing.T) {
	// Gemini
	g, err := NewProvider(Config{Provider: "gemini", APIKey: "test-key"})
	if err != nil || g == nil {
		t.Fatalf("expected gemini provider: %v", err)
	}
	_, err = NewProvider(Config{Provider: "gemini", APIKey: ""})
	if err == nil {
		t.Errorf("expected error for gemini without api key")
	}

	// OpenAI
	o, err := NewProvider(Config{Provider: "openai", APIKey: "test-key"})
	if err != nil || o == nil {
		t.Fatalf("expected openai provider: %v", err)
	}
	_, err = NewProvider(Config{Provider: "openai", APIKey: ""})
	if err == nil {
		t.Errorf("expected error for openai without api key")
	}

	// Anthropic
	a, err := NewProvider(Config{Provider: "anthropic", APIKey: "test-key"})
	if err != nil || a == nil {
		t.Fatalf("expected anthropic provider: %v", err)
	}
	_, err = NewProvider(Config{Provider: "anthropic", APIKey: ""})
	if err == nil {
		t.Errorf("expected error for anthropic without api key")
	}

	// Ollama
	ol, err := NewProvider(Config{Provider: "ollama", Model: "deepseek"})
	if err != nil || ol == nil {
		t.Fatalf("expected ollama provider: %v", err)
	}

	// Custom
	c, err := NewProvider(Config{Provider: "custom", Model: "custom-model"})
	if err != nil || c == nil {
		t.Fatalf("expected custom provider: %v", err)
	}

	// Empty defaults to Gemini
	gDef, err := NewProvider(Config{Provider: "", APIKey: "key"})
	if err != nil || gDef == nil {
		t.Fatalf("expected default to gemini: %v", err)
	}
}

func TestOpenAIProvider_Mock(t *testing.T) {
	client := mockClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer test-openai-key" {
			return jsonHTTPResponse(http.StatusUnauthorized, map[string]string{"error": "unauthorized"}), nil
		}

		var reqBody openAIChatRequest
		_ = json.NewDecoder(req.Body).Decode(&reqBody)

		userMsg := ""
		if len(reqBody.Messages) > 0 {
			userMsg = reqBody.Messages[len(reqBody.Messages)-1].Content
		}

		reply := ""
		if strings.Contains(userMsg, "Respond with 'OK'") {
			reply = "OK"
		} else if strings.Contains(userMsg, "Existing Full File Content") {
			reply = "```go\npackage main\n\nfunc main() {}\n```"
		} else if strings.Contains(userMsg, "unified git diff patch") {
			reply = "```diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n- old\n+ new\n```"
		} else {
			reply = "```json\n{\"root_cause\":\"nil pointer dereference\",\"suggested_fix\":\"initialize pointer\"}\n```"
		}

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
					}{Content: reply},
					FinishReason: "stop",
				},
			},
		}
		return jsonHTTPResponse(http.StatusOK, resp), nil
	})

	provider, err := NewOpenAIProvider(Config{
		Provider:   "openai",
		APIKey:     "test-openai-key",
		Model:      "gpt-4o",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("failed to create OpenAI provider: %v", err)
	}

	ctx := context.Background()

	// 1. AnalyzeCrash
	res, err := provider.AnalyzeCrash(ctx, "goroutine 1", "func test() {}")
	if err != nil || res.RootCause != "nil pointer dereference" {
		t.Fatalf("unexpected AnalyzeCrash: %v, %+v", err, res)
	}

	// 2. GeneratePatch
	patch, err := provider.GeneratePatch(ctx, "main.go", "panic", "ast", "stack", "cause")
	if err != nil || !strings.Contains(patch, "--- a/main.go") {
		t.Fatalf("unexpected GeneratePatch: %v, patch=%s", err, patch)
	}

	// 3. ApplyFixToFile
	code, err := provider.ApplyFixToFile(ctx, "main.go", "package main", "panic", "ast", "stack", "cause", "fix", "patch")
	if err != nil || !strings.Contains(code, "package main") {
		t.Fatalf("unexpected ApplyFixToFile: %v, code=%s", err, code)
	}

	// 4. TestConnection
	if err := provider.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
}

func TestAnthropicProvider_Mock(t *testing.T) {
	client := mockClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("x-api-key") != "test-claude-key" {
			return jsonHTTPResponse(http.StatusUnauthorized, map[string]string{"error": "unauthorized"}), nil
		}

		var reqBody anthropicRequest
		_ = json.NewDecoder(req.Body).Decode(&reqBody)

		userMsg := ""
		if len(reqBody.Messages) > 0 {
			userMsg = reqBody.Messages[len(reqBody.Messages)-1].Content
		}

		reply := ""
		if strings.Contains(userMsg, "Respond with 'OK'") {
			reply = "OK"
		} else if strings.Contains(userMsg, "Existing Full File Content") {
			reply = "```go\npackage main\n\nfunc main() {}\n```"
		} else if strings.Contains(userMsg, "unified git diff patch") {
			reply = "```diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n- old\n+ new\n```"
		} else {
			reply = "```json\n{\"root_cause\":\"anthropic root cause\",\"suggested_fix\":\"fix it\"}\n```"
		}

		resp := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{Type: "text", Text: reply},
			},
		}
		return jsonHTTPResponse(http.StatusOK, resp), nil
	})

	provider, err := NewAnthropicProvider(Config{
		Provider:   "anthropic",
		APIKey:     "test-claude-key",
		Model:      "claude-3-5-sonnet-20241022",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("failed to create Anthropic provider: %v", err)
	}

	ctx := context.Background()

	// 1. AnalyzeCrash
	res, err := provider.AnalyzeCrash(ctx, "stack", "ast")
	if err != nil || res.RootCause != "anthropic root cause" {
		t.Fatalf("unexpected AnalyzeCrash: %v, %+v", err, res)
	}

	// 2. GeneratePatch
	patch, err := provider.GeneratePatch(ctx, "main.go", "panic", "ast", "stack", "cause")
	if err != nil || !strings.Contains(patch, "--- a/main.go") {
		t.Fatalf("unexpected GeneratePatch: %v, patch=%s", err, patch)
	}

	// 3. ApplyFixToFile
	code, err := provider.ApplyFixToFile(ctx, "main.go", "package main", "panic", "ast", "stack", "cause", "fix", "patch")
	if err != nil || !strings.Contains(code, "package main") {
		t.Fatalf("unexpected ApplyFixToFile: %v, code=%s", err, code)
	}

	// 4. TestConnection
	if err := provider.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
}

func TestGeminiProvider_Mock(t *testing.T) {
	client := mockClient(func(req *http.Request) (*http.Response, error) {
		bodyBytes, _ := io.ReadAll(req.Body)
		bodyStr := string(bodyBytes)

		reply := ""
		if strings.Contains(bodyStr, "Respond with 'OK'") {
			reply = "OK"
		} else if strings.Contains(bodyStr, "EMPTY_CONNECTION") {
			reply = ""
		} else if strings.Contains(bodyStr, "NON_GO_CODE") {
			reply = "plain text without go syntax"
		} else if strings.Contains(bodyStr, "Existing Full File Content") {
			reply = "```go\npackage main\n\nfunc main() {}\n```"
		} else if strings.Contains(bodyStr, "unified git diff patch") {
			reply = "```diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n- old\n+ new\n```"
		} else if strings.Contains(bodyStr, "INVALID_JSON") {
			reply = "not a valid json string"
		} else {
			reply = `{"root_cause":"gemini root cause","suggested_fix":"gemini fix"}`
		}

		respJSON := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%q}]}}]}`, reply)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(respJSON)),
			Header:     make(http.Header),
		}, nil
	})

	provider, err := NewGeminiProvider(Config{
		Provider:   "gemini",
		APIKey:     "fake-gemini-key",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("failed to create Gemini provider: %v", err)
	}

	ctx := context.Background()

	// 1. AnalyzeCrash
	res, err := provider.AnalyzeCrash(ctx, "stack", "ast")
	if err != nil || res.RootCause != "gemini root cause" {
		t.Fatalf("unexpected Gemini AnalyzeCrash: %v, %+v", err, res)
	}

	// 1b. AnalyzeCrash with invalid JSON
	_, err = provider.AnalyzeCrash(ctx, "INVALID_JSON", "ast")
	if err == nil {
		t.Errorf("expected error for invalid json in AnalyzeCrash")
	}

	// 2. GeneratePatch
	patch, err := provider.GeneratePatch(ctx, "main.go", "panic", "ast", "stack", "cause")
	if err != nil || !strings.Contains(patch, "--- a/main.go") {
		t.Fatalf("unexpected Gemini GeneratePatch: %v, patch=%s", err, patch)
	}

	// 3. ApplyFixToFile - valid go code
	code, err := provider.ApplyFixToFile(ctx, "main.go", "package main", "panic", "ast", "stack", "cause", "fix", "patch")
	if err != nil || !strings.Contains(code, "package main") {
		t.Fatalf("unexpected Gemini ApplyFixToFile: %v, code=%s", err, code)
	}

	// 3b. ApplyFixToFile - non-go code fallback
	nonGo, err := provider.ApplyFixToFile(ctx, "main.go", "NON_GO_CODE", "panic", "ast", "stack", "cause", "fix", "patch")
	if err != nil || !strings.Contains(nonGo, "plain text without go syntax") {
		t.Fatalf("unexpected Gemini non-go fallback: %v, code=%s", err, nonGo)
	}

	// 4. TestConnection
	if err := provider.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
}

func TestOpenAIProvider_Errors(t *testing.T) {
	ctx := context.Background()

	// Error response from API (500)
	errClient := mockClient(func(req *http.Request) (*http.Response, error) {
		errResp := openAIChatResponse{
			Error: &struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			}{Message: "rate limit exceeded", Type: "rate_limit_error", Code: "rate_limit"},
		}
		return jsonHTTPResponse(http.StatusTooManyRequests, errResp), nil
	})

	provider, err := NewOpenAIProvider(Config{
		Provider:   "openai",
		APIKey:     "test-key",
		HTTPClient: errClient,
	})
	if err != nil {
		t.Fatalf("failed to init provider: %v", err)
	}

	if _, err := provider.AnalyzeCrash(ctx, "s", "a"); err == nil {
		t.Errorf("expected error from AnalyzeCrash on 429")
	}
	if err := provider.TestConnection(ctx); err == nil {
		t.Errorf("expected error from TestConnection on 429")
	}

	// Empty choices returned
	emptyChoicesClient := mockClient(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, openAIChatResponse{Choices: nil}), nil
	})
	provider2, _ := NewOpenAIProvider(Config{Provider: "openai", APIKey: "test", HTTPClient: emptyChoicesClient})
	if _, err := provider2.AnalyzeCrash(ctx, "s", "a"); err == nil {
		t.Errorf("expected error on empty choices")
	}
	if err := provider2.TestConnection(ctx); err == nil {
		t.Errorf("expected error on empty choices in TestConnection")
	}

	// Non-Go code fallback in ApplyFixToFile
	rawTextClient := mockClient(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, openAIChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "some non go code"}},
			},
		}), nil
	})
	provider3, _ := NewOpenAIProvider(Config{Provider: "openai", APIKey: "test", HTTPClient: rawTextClient})
	res, err := provider3.ApplyFixToFile(ctx, "main.go", "curr", "p", "a", "s", "r", "f", "pt")
	if err != nil || !strings.Contains(res, "some non go code") {
		t.Errorf("expected fallback text, got %v, %s", err, res)
	}

	// Invalid JSON decode in AnalyzeCrash
	provider4, _ := NewOpenAIProvider(Config{Provider: "openai", APIKey: "test", HTTPClient: rawTextClient})
	if _, err := provider4.AnalyzeCrash(ctx, "s", "a"); err == nil {
		t.Errorf("expected JSON unmarshal error")
	}
}

func TestAnthropicProvider_Errors(t *testing.T) {
	ctx := context.Background()

	// Error response (500)
	errClient := mockClient(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusInternalServerError, anthropicResponse{
			Type: "error",
			Error: &struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{Type: "api_error", Message: "internal server error"},
		}), nil
	})

	provider, _ := NewAnthropicProvider(Config{
		Provider:   "anthropic",
		APIKey:     "test",
		HTTPClient: errClient,
	})

	if _, err := provider.AnalyzeCrash(ctx, "s", "a"); err == nil {
		t.Errorf("expected error on 500")
	}
	if err := provider.TestConnection(ctx); err == nil {
		t.Errorf("expected error on TestConnection 500")
	}

	// Empty content
	emptyClient := mockClient(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, anthropicResponse{Content: nil}), nil
	})
	provider2, _ := NewAnthropicProvider(Config{Provider: "anthropic", APIKey: "test", HTTPClient: emptyClient})
	if _, err := provider2.AnalyzeCrash(ctx, "s", "a"); err == nil {
		t.Errorf("expected error on empty content")
	}
	if err := provider2.TestConnection(ctx); err == nil {
		t.Errorf("expected error on empty content for TestConnection")
	}

	// Non-Go code fallback in ApplyFixToFile
	rawClient := mockClient(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, anthropicResponse{
			Content: []anthropicContentBlock{{Type: "text", Text: "non go text"}},
		}), nil
	})
	provider3, _ := NewAnthropicProvider(Config{Provider: "anthropic", APIKey: "test", HTTPClient: rawClient})
	res, err := provider3.ApplyFixToFile(ctx, "main.go", "curr", "p", "a", "s", "r", "f", "pt")
	if err != nil || !strings.Contains(res, "non go text") {
		t.Errorf("expected fallback text, got %v, %s", err, res)
	}

	// Invalid JSON decode in AnalyzeCrash
	if _, err := provider3.AnalyzeCrash(ctx, "s", "a"); err == nil {
		t.Errorf("expected JSON unmarshal error")
	}
}

func TestOpenAICompatibleProvider_Options(t *testing.T) {
	// Ollama defaults
	o, err := NewOpenAICompatibleProvider(Config{Provider: "ollama"})
	if err != nil || o.baseURL != "http://localhost:11434/v1" || o.modelName != "deepseek-coder-v2" {
		t.Errorf("unexpected ollama config: %+v, err=%v", o, err)
	}

	// Custom defaults with trailing slash stripped
	c, err := NewOpenAICompatibleProvider(Config{
		Provider: "custom",
		BaseURL:  "https://custom.endpoint.com/v1/",
		Model:    "custom-model",
	})
	if err != nil || c.baseURL != "https://custom.endpoint.com/v1" || c.modelName != "custom-model" {
		t.Errorf("unexpected custom config: %+v, err=%v", c, err)
	}
}

func TestStandaloneConvenienceFunctions(t *testing.T) {
	ctx := context.Background()

	// Error when provider is unconfigured
	badCfg := Config{Provider: "gemini", APIKey: ""}
	if _, err := AnalyzeCrash(ctx, badCfg, "s", "a"); err == nil {
		t.Errorf("expected error from AnalyzeCrash with empty key")
	}
	if _, err := GeneratePatch(ctx, badCfg, "f", "p", "a", "s", "r"); err == nil {
		t.Errorf("expected error from GeneratePatch with empty key")
	}
	if _, err := ApplyFixToFile(ctx, badCfg, "f", "c", "p", "a", "s", "r", "sf", "pt"); err == nil {
		t.Errorf("expected error from ApplyFixToFile with empty key")
	}
}
