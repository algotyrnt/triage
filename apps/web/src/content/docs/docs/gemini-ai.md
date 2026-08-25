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

You can select your preferred model during the Studio Dashboard setup wizard or update it dynamically at any time in **Settings > Gemini AI**.

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

## Domain-Aware AI Triage & Project Context

To ensure Gemini diagnoses panics with high precision without violating business invariants, Triage supports optional **Project & Architectural Domain Context**.

### How It Works

1. During project onboarding (or later in **Settings > General**), you can provide domain descriptions such as ledger idempotency rules, state machine transitions, concurrency invariants, or database transaction semantics.
2. When a panic occurs, the engine automatically injects your domain context into the prompt:
   ```markdown
   ### Project & Domain Context:

   High-throughput payment gateway processing Stripe and crypto webhooks with strict ledger idempotency and database transaction rollbacks.
   ```
3. Gemini uses this context in `AnalyzeCrash`, `GeneratePatch`, and `ApplyFixToFile` to recommend fixes that respect your application's domain logic rather than making generic assumptions.

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

Open your self-hosted Studio Dashboard, navigate to **Settings > Gemini AI** (or complete the initial Setup Wizard), and enter your Google AI Studio API key and desired model name (e.g. `gemini-1.5-flash` or `gemini-2.5-flash`). Credentials are automatically stored and encrypted in PostgreSQL.
