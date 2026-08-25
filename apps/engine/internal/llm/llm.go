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

type AnalysisResult struct {
	RootCause    string `json:"root_cause"`
	SuggestedFix string `json:"suggested_fix"`
}

// AnalyzeCrash sends the crash stack trace, isolated AST snippet, and optional project domain context to Gemini
// and returns a structured root cause analysis and suggested fix.
func AnalyzeCrash(ctx context.Context, stackTrace, astSnippet, apiKey, modelName string, projectContext ...string) (*AnalysisResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}
	if modelName == "" {
		return nil, fmt.Errorf("gemini model name is not configured")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	contextSection := ""
	if len(projectContext) > 0 && strings.TrimSpace(projectContext[0]) != "" {
		contextSection = fmt.Sprintf("\n### Project & Domain Context:\n%s\n", strings.TrimSpace(projectContext[0]))
	}

	prompt := fmt.Sprintf(`You are an expert Go backend engineer and automated incident diagnostician.
Analyze the following Go panic crash telemetry and surrounding Go AST package context (including the crash site, struct/type definitions, constructors, and package helpers).%s
### Stack Trace:
%s

### Surrounding AST & Package Context:
%s

Respond ONLY with a valid JSON object with the following schema:
{
  "root_cause": "Explanation of what caused the crash",
  "suggested_fix": "Detailed solution or code modification to fix the issue"
}`, contextSection, stackTrace, astSnippet)

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
		slog.Error("failed to unmarshal Gemini response JSON", "error", err, "raw_response", rawText)
		return nil, fmt.Errorf("failed to parse Gemini JSON response: %w", err)
	}

	return &analysis, nil
}

// GeneratePatch sends the crash details, AST context, analysis, and optional project domain context to Gemini to generate a unified git diff patch.
func GeneratePatch(ctx context.Context, file, panicMessage, astSnippet, stackTrace, rootCause, apiKey, modelName string, projectContext ...string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("gemini api key is not configured")
	}
	if modelName == "" {
		return "", fmt.Errorf("gemini model name is not configured")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	contextSection := ""
	if len(projectContext) > 0 && strings.TrimSpace(projectContext[0]) != "" {
		contextSection = fmt.Sprintf("\n### Project & Domain Context:\n%s\n", strings.TrimSpace(projectContext[0]))
	}

	prompt := fmt.Sprintf(`You are an expert Go systems engineer and automated patch generator.
Create a unified git diff patch to resolve the following Go panic crash.%s
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
3. Return ONLY the patch text. Do not wrap in markdown backticks or add introductory commentary.`, contextSection, file, panicMessage, rootCause, stackTrace, astSnippet)

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

// ApplyFixToFile merges the suggested patch / panic fix into the complete source file and returns the full updated file content.
func ApplyFixToFile(ctx context.Context, file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch, apiKey, modelName string, projectContext ...string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("gemini api key is not configured")
	}
	if modelName == "" {
		return "", fmt.Errorf("gemini model name is not configured")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	contextSection := ""
	if len(projectContext) > 0 && strings.TrimSpace(projectContext[0]) != "" {
		contextSection = fmt.Sprintf("\n### Project & Domain Context:\n%s\n", strings.TrimSpace(projectContext[0]))
	}

	prompt := fmt.Sprintf(`You are an expert Go systems engineer.
You are tasked with applying a bugfix for a Go panic into the full source code file.%s
### File Path:
%s

### Panic Message:
%s

### Root Cause:
%s

### Recommended Solution:
%s

### Suggested Patch / Fix:
%s

### Existing Full File Content:
%s

Instructions:
1. Apply the bugfix to the file accurately and cleanly.
2. Ensure valid Go syntax, correct imports, and proper formatting.
3. Return ONLY the complete, updated Go file source code. Do NOT wrap in markdown backticks and do NOT add conversational explanations.`, contextSection, file, panicMessage, rootCause, suggestedFix, patch, currentContent)

	resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini apply fix failed: %w", err)
	}

	rawText := strings.TrimSpace(resp.Text())
	if strings.HasPrefix(rawText, "```go") {
		rawText = strings.TrimPrefix(rawText, "```go")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```") {
		rawText = strings.TrimPrefix(rawText, "```")
		rawText = strings.TrimSuffix(rawText, "```")
	}
	rawText = strings.TrimSpace(rawText)

	// Format code using standard gofmt conventions (which includes POSIX trailing newline)
	if formatted, err := format.Source([]byte(rawText)); err == nil {
		return string(formatted), nil
	}

	// Fallback guarantee: ensure file ends with standard newline
	if !strings.HasSuffix(rawText, "\n") {
		rawText += "\n"
	}

	return rawText, nil
}
