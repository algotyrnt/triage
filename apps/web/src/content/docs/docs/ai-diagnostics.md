---
title: Pluggable Multi-Provider LLM Diagnostics
description: Multi-provider AI triage with Google Gemini, OpenAI, Anthropic Claude, and Local Ollama/vLLM models
---

Triage features a pluggable, multi-provider LLM architecture. Whether your team relies on cloud frontier models (**Google Gemini**, **OpenAI**, **Anthropic Claude**) or runs **100% air-gapped on-premise local models** (**Ollama**, **vLLM**, **LocalAI**, **LM Studio**), Triage delivers structured incident diagnostics, unified git diff generation, and clean source-level fix application.

---

## Supported AI Providers

| Provider             | Supported Models                                                               | Connection Type                                          |
| :------------------- | :----------------------------------------------------------------------------- | :------------------------------------------------------- |
| **Google Gemini**    | `gemini-2.0-flash`, `gemini-2.5-pro`, `gemini-1.5-flash`                       | Official Google GenAI SDK                                |
| **OpenAI**           | `gpt-4o`, `gpt-4o-mini`, `o3-mini`, `o1`, `gpt-4.5-preview`                    | OpenAI Chat Completions API                              |
| **Anthropic Claude** | `claude-3-5-sonnet-20241022`, `claude-3-7-sonnet-20250219`, `claude-3-5-haiku` | Claude Messages API                                      |
| **Local / Ollama**   | `deepseek-coder-v2`, `qwen2.5-coder:7b`, `llama3.3`                            | OpenAI-compatible Base URL (`http://localhost:11434/v1`) |

---

## Air-Gapped & Local Model Setup (Ollama / vLLM)

For high-security or on-premise deployments requiring zero outbound telemetry:

1. Start your local Ollama instance with a code-specialized model:
   ```bash
   ollama run deepseek-coder-v2
   ```
2. In the Triage Studio Dashboard, navigate to **Settings > AI Configuration** (or Step 4 of the Setup Wizard).
3. Select **Local / Ollama**:
   - **Base URL:** `http://localhost:11434/v1` (or your vLLM / LocalAI endpoint)
   - **Model Name:** `deepseek-coder-v2` or `qwen2.5-coder:7b`
   - **API Key:** _(leave blank if authentication is not required)_
4. Click **Test Connection** to benchmark endpoint latency and verify model availability before saving.

---

## Structured Output Schema

The Triage Engine guarantees strict JSON schema output across all providers:

```json
{
  "type": "object",
  "properties": {
    "root_cause": {
      "type": "string",
      "description": "Precise explanation of why the Go runtime panicked on the triggering line."
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

To ensure LLMs diagnose crashes accurately without violating business rules, Triage supports optional **Project & Architectural Domain Context**.

### How It Works

1. During project onboarding (or later in **Settings > General**), specify domain constraints such as ledger idempotency rules, state machine transitions, concurrency invariants, or database transaction semantics.
2. When a panic occurs, the engine automatically injects your domain context into the prompt:
   ```markdown
   ### Project & Domain Context:

   High-throughput payment gateway processing Stripe and crypto webhooks with strict ledger idempotency and database transaction rollbacks.
   ```
3. The active provider uses this context in `AnalyzeCrash`, `GeneratePatch`, and `ApplyFixToFile` to synthesize patches that strictly adhere to your business boundaries.

---

## Automated File Patching (`ApplyFixToFile`)

When generating a bugfix Pull Request, the engine uses the configured LLM's code reasoning to merge the suggested patch into the latest source file retrieved from GitHub:

1. Preserves existing package headers, imports, comments, and project indentation.
2. Applies necessary nil guards, error validations, or constructor initializations.
3. Automatically formats the resulting Go code with standard `go/format` before committing directly to the bugfix branch (`triage/fix-...`).

---

## Direct REST Endpoints

The Triage engine provides direct HTTP endpoints for on-demand AI operations:

### 1. Test Provider Connection

```bash
curl -X POST http://localhost:8080/api/v1/settings/llm/test \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "api_key": "sk-...",
    "model": "gpt-4o"
  }'
```

### 2. Panic Root Cause Analysis

```bash
curl -X POST http://localhost:8080/api/v1/llm/analyze-panic \
  -H "Content-Type: application/json" \
  -d '{
    "triggeringFile": "handlers/payment.go",
    "panicMessage": "runtime error: invalid memory address or nil pointer dereference",
    "astCode": "func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n..."
  }'
```

### 3. Git Diff Patch Generation

```bash
curl -X POST http://localhost:8080/api/v1/llm/generate-patch \
  -H "Content-Type: application/json" \
  -d '{
    "triggeringFile": "handlers/payment.go",
    "panicMessage": "runtime error: invalid memory address or nil pointer dereference",
    "astCode": "func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n...",
    "rootCause": "Attempted to evaluate req.Amount on an uninitialized nil pointer on line 28."
  }'
```

## Dynamic Configuration & Key Management

AI provider credentials, models, and custom endpoint URLs are managed dynamically through the Setup Wizard and Studio Dashboard (**Settings > AI Configuration**).

- **Database-Backed:** Settings are stored securely in PostgreSQL (`instance_config` table).
- **Hot-Swappable:** You can switch providers (e.g. from Gemini to local Ollama or Claude) or update models on the fly without restarting the engine container.
- **Connection Testing:** The built-in `/api/v1/settings/llm/test` endpoint lets you verify authentication and benchmark latency before saving changes.
