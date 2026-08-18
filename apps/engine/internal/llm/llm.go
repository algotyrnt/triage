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
Analyze the following Go panic crash telemetry and surrounding Go AST package context (including the crash site, struct/type definitions, constructors, and package helpers).

### Stack Trace:
%s

### Surrounding AST & Package Context:
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

// GeneratePatch sends the crash details, AST context, and analysis to Gemini to generate a unified git diff patch.
func GeneratePatch(ctx context.Context, file, panicMessage, astSnippet, stackTrace, rootCause, apiKey, modelName string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is missing or empty")
	}
	if modelName == "" {
		return "", fmt.Errorf("GEMINI_MODEL_NAME is missing or empty")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	prompt := fmt.Sprintf(`You are an expert Go systems engineer and automated patch generator.
Create a unified git diff patch to resolve the following Go panic crash.

### Triggering File:
%s

### Panic Message:
%s

### Root Cause:
%s

### Stack Trace:
%s

### Surrounding AST / Code Context:
%s

Instructions:
1. Provide a precise, valid unified diff patch (starting with "--- a/..." and "+++ b/..." with @@ chunk headers) or clear code modification fix.
2. Only modify what is strictly necessary to guard against nil pointers, bounds errors, or invalid state.
3. Return ONLY the patch text. Do not wrap in markdown backticks or add introductory commentary.`, file, panicMessage, rootCause, stackTrace, astSnippet)

	resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini patch generation failed: %w", err)
	}

	rawText := strings.TrimSpace(resp.Text())
	if strings.HasPrefix(rawText, "```diff") {
		rawText = strings.TrimPrefix(rawText, "```diff")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```go") {
		rawText = strings.TrimPrefix(rawText, "```go")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```") {
		rawText = strings.TrimPrefix(rawText, "```")
		rawText = strings.TrimSuffix(rawText, "```")
	}
	rawText = strings.TrimSpace(rawText)

	return rawText, nil
}
