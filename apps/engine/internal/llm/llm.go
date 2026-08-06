// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
)

type AnalysisResult struct {
	RootCause    string `json:"root_cause"`
	SuggestedFix string `json:"suggested_fix"`
}

// AnalyzeCrash queries Gemini 3.6 Flash using google.golang.org/genai to diagnose the stack trace and AST node.
func AnalyzeCrash(ctx context.Context, stackTrace string, astSnippet string) (*AnalysisResult, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	clientConfig := &genai.ClientConfig{
		APIKey: apiKey,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	prompt := fmt.Sprintf(`You are an expert Go crash triage system.
Analyze the following panic stack trace and surrounding function source code AST node:

### STACK TRACE:
%s

### SURROUNDING AST NODE:
%s

Respond ONLY with a valid JSON object with the following schema:
{
  "root_cause": "Explanation of what caused the crash",
  "suggested_fix": "Detailed solution or code modification to fix the issue"
}`, stackTrace, astSnippet)

	resp, err := client.Models.GenerateContent(ctx, "gemini-3.6-flash", genai.Text(prompt), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("gemini api call failed: %w", err)
	}

	rawText := resp.Text()
	rawText = strings.TrimSpace(rawText)
	if strings.HasPrefix(rawText, "```json") {
		rawText = strings.TrimPrefix(rawText, "```json")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```") {
		rawText = strings.TrimPrefix(rawText, "```")
		rawText = strings.TrimSuffix(rawText, "```")
	}
	rawText = strings.TrimSpace(rawText)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(rawText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini JSON output: %w (raw response: %s)", err, rawText)
	}

	return &result, nil
}
