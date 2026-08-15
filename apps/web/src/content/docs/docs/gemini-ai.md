---
title: Gemini AI Diagnostics
description: Structured JSON panic diagnostics and patch generation using Google Gemini AI
---

Triage integrates with Google's official `google.golang.org/genai` SDK to run sub-second structured panic diagnostics and patch generation.

## Model Configuration

Triage is completely model-agnostic. You can configure any model available in your Google AI Studio account (including Flash, Pro, or future releases).

- **High Speed & Low Latency:** Rapid round-trip structured inference.
- **Deterministic Schema Support:** Guaranteed JSON schema compliance without parsing failures.
- **Cost Efficiency:** Combined with Triage's 94% AST token reduction, incident analysis uses minimal tokens.

You can set your model during the Studio Dashboard setup wizard or dynamically via the `GEMINI_MODEL_NAME` environment variable.

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

## Example AI Diagnostic Output

Given a nil pointer panic in `payment.go:28`:

```json
{
  "root_cause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
  "suggested_fix": "Allocate memory with req := &PaymentPayload{} and validate JSON decode errors before field access.",
  "severity": "CRITICAL",
  "suggested_patch": "@@ -26,3 +26,4 @@\n-    var req *PaymentPayload\n-    if req.Amount <= 0 {\n+    req := &PaymentPayload{}\n+    if err := json.NewDecoder(r.Body).Decode(req); err != nil || req.Amount <= 0 {"
}
```

---

## Configuring Your Gemini API Key & Model

### Option 1: Studio Dashboard Wizard

Open your self-hosted Studio Dashboard and enter your Google AI Studio API key and desired model name in **Step 4: Gemini AI**.

### Option 2: Environment Variables

Pass your API key and model name when launching the Docker container:

```bash
docker run -d \
  -p 8080:8080 \
  -e GEMINI_API_KEY="your_api_key_here" \
  -e GEMINI_MODEL_NAME="your_preferred_model_name" \
  triage/engine:latest
```
