---
title: Gemini AI Diagnostics & Patches
description: Structured JSON panic diagnostics, patch generation, and automated file fixing using Google Gemini AI
---

Triage integrates with Google's official `google.golang.org/genai` SDK to run sub-second structured panic diagnostics, unified git diff generation, and clean source-level fix application.

## Model Configuration

Triage is completely model-agnostic. You can configure any model available in your Google AI Studio account (including `gemini-2.5-flash`, `gemini-1.5-pro`, or `gemini-2.0-flash`):

- **High Speed & Low Latency:** Rapid round-trip structured inference in under 200ms.
- **Deterministic Schema Support:** Guaranteed JSON schema compliance without hallucinated fields or markdown formatting glitches.
- **Cost Efficiency:** Combined with Triage's 94% AST token reduction, incident analysis uses minimal tokens (< $0.0001 per incident).

You can set your model during the Studio Dashboard setup wizard, update it dynamically in **Settings > Gemini AI**, or pass the `GEMINI_MODEL_NAME` environment variable.

---

## Structured Output Schema

The Triage Engine enforces the following strict JSON schema via Gemini's `ResponseSchema`:

```json
{
  "type": "object",
  "properties": {
    "root_cause": {
      "type": "string",
      "description": "Precise one-sentence explanation of why the Go runtime panicked on the triggering line."
    },
    "suggested_fix": {
      "type": "string",
      "description": "Exact code correction required to prevent the panic."
    },
    "severity": {
      "type": "string",
      "enum": ["CRITICAL", "HIGH", "MEDIUM", "LOW"]
    },
    "suggested_patch": {
      "type": "string",
      "description": "Unified Git diff format patch."
    }
  },
  "required": ["root_cause", "suggested_fix", "severity"]
}
```

---

## Automated File Patching (`ApplyFixToFile`)

When generating a bugfix Pull Request, the engine uses Gemini AI's code reasoning to merge the suggested patch into the latest source file retrieved from GitHub:

1. Preserves existing package headers, imports, comments, and project indentation.
2. Applies necessary nil guards, error validations, or constructor initializations.
3. Produces a clean, compilation-ready file that is committed directly to the bugfix branch (`triage/fix-...`).

---

## Direct REST Endpoints

The Triage engine provides direct HTTP endpoints for on-demand Gemini AI operations:

### 1. Panic Root Cause Analysis

```bash
curl -X POST http://localhost:8080/api/v1/gemini/analyze-panic \
  -H "Content-Type: application/json" \
  -d '{
    "triggeringFile": "handlers/payment.go",
    "panicMessage": "runtime error: invalid memory address or nil pointer dereference",
    "astCode": "func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n..."
  }'
```

### 2. Git Diff Patch Generation

```bash
curl -X POST http://localhost:8080/api/v1/gemini/generate-patch \
  -H "Content-Type: application/json" \
  -d '{
    "triggeringFile": "handlers/payment.go",
    "panicMessage": "runtime error: invalid memory address or nil pointer dereference",
    "astCode": "func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n...",
    "rootCause": "Attempted to evaluate req.Amount on an uninitialized nil pointer on line 28."
  }'
```

---

## Configuring Your Gemini API Key & Model

### Option 1: Studio Dashboard Settings

Open your self-hosted Studio Dashboard, navigate to **Settings > Gemini AI**, and enter your Google AI Studio API key and desired model name.

### Option 2: Environment Variables

Pass your API key and model name when launching the engine:

```bash
docker run -d \
  -p 8080:8080 \
  -e GEMINI_API_KEY="your_api_key_here" \
  -e GEMINI_MODEL_NAME="gemini-2.5-flash" \
  ghcr.io/algotyrnt/triage-engine:latest
```
