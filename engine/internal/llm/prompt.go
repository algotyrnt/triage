// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package llm

import (
	"fmt"
	"strings"
)

// BuildAnalyzeCrashPrompt builds the standard crash diagnosis prompt.
func BuildAnalyzeCrashPrompt(stackTrace, astSnippet string, projectContext ...string) string {
	contextSection := ""
	if len(projectContext) > 0 && strings.TrimSpace(projectContext[0]) != "" {
		contextSection = fmt.Sprintf("\n### Project & Domain Context:\n%s\n", strings.TrimSpace(projectContext[0]))
	}

	return fmt.Sprintf(`You are an expert Go backend engineer and automated incident diagnostician.
Analyze the following Go panic crash telemetry and surrounding Go AST package context (including the crash site, struct/type definitions, constructors, and package helpers).%s
### Stack Trace:
%s

### Surrounding AST & Package Context:
%s

Respond ONLY with a valid JSON object with the following schema:
{
  "root_cause": "Explanation of what caused the crash",
  "suggested_fix": "Detailed solution or code modification to fix the issue",
  "severity": "CRITICAL, HIGH, or MEDIUM based on impact (CRITICAL for fatal crashes / nil pointers / memory races, HIGH for application panic errors / type assertion bugs, MEDIUM for recovered handler warnings)"
}`, contextSection, stackTrace, astSnippet)
}

// BuildGeneratePatchPrompt builds the unified git diff patch prompt.
func BuildGeneratePatchPrompt(file, panicMessage, astSnippet, stackTrace, rootCause string, projectContext ...string) string {
	contextSection := ""
	if len(projectContext) > 0 && strings.TrimSpace(projectContext[0]) != "" {
		contextSection = fmt.Sprintf("\n### Project & Domain Context:\n%s\n", strings.TrimSpace(projectContext[0]))
	}

	return fmt.Sprintf(`You are an expert Go systems engineer and automated patch generator.
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
}

// BuildApplyFixPrompt builds the prompt to apply the patch directly to the complete file.
func BuildApplyFixPrompt(file, currentContent, panicMessage, astSnippet, stackTrace, rootCause, suggestedFix, patch string, projectContext ...string) string {
	contextSection := ""
	if len(projectContext) > 0 && strings.TrimSpace(projectContext[0]) != "" {
		contextSection = fmt.Sprintf("\n### Project & Domain Context:\n%s\n", strings.TrimSpace(projectContext[0]))
	}

	return fmt.Sprintf(`You are an expert Go systems engineer.
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
}

// CleanMarkdownCodeBlock strips markdown code fence wrappers (```json, ```go, ```diff, ```).
func CleanMarkdownCodeBlock(raw string, preferredTag string) string {
	rawText := strings.TrimSpace(raw)
	if preferredTag != "" && strings.HasPrefix(rawText, "```"+preferredTag) {
		rawText = strings.TrimPrefix(rawText, "```"+preferredTag)
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```json") {
		rawText = strings.TrimPrefix(rawText, "```json")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```go") {
		rawText = strings.TrimPrefix(rawText, "```go")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```diff") {
		rawText = strings.TrimPrefix(rawText, "```diff")
		rawText = strings.TrimSuffix(rawText, "```")
	} else if strings.HasPrefix(rawText, "```") {
		rawText = strings.TrimPrefix(rawText, "```")
		rawText = strings.TrimSuffix(rawText, "```")
	}
	return strings.TrimSpace(rawText)
}
