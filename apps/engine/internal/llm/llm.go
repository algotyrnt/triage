// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"
)

type AnalysisResult struct {
	RootCause    string `json:"root_cause"`
	SuggestedFix string `json:"suggested_fix"`
}

// AnalyzeCrash sends the crash stack trace and isolated AST snippet to the Gemini model
// and returns a structured root cause analysis and suggested fix.
func AnalyzeCrash(ctx context.Context, stackTrace, astSnippet, apiKey, modelName string) (*AnalysisResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is missing or empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("GEMINI_MODEL_NAME is missing or empty")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	prompt := fmt.Sprintf(`You are an expert Go backend engineer and automated incident diagnostician.
Analyze the following Go panic crash telemetry and isolated function AST node.

### Stack Trace:
%s

### Surrounding Function AST Node:
%s

Respond ONLY with a valid JSON object with the following schema:
{
  "root_cause": "Explanation of what caused the crash",
  "suggested_fix": "Detailed solution or code modification to fix the issue"
}`, stackTrace, astSnippet)

	resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), &genai.GenerateContentConfig{
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

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(rawText), &analysis); err != nil {
		log.Printf("[GEMINI UNMARSHAL ERROR] Raw response: %s", rawText)
		return nil, fmt.Errorf("failed to parse Gemini JSON response: %w", err)
	}

	return &analysis, nil
}
