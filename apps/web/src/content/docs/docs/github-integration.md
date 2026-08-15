---
title: GitHub App & Issue Automation
description: Set up automated GitHub issue creation, repository permissions, and OAuth authentication
---

Triage connects to GitHub as an official GitHub App to fetch source code for on-demand AST parsing and automatically file triage issues with full stack traces.

## Capabilities

- **On-Demand Source Fetching:** Access private and public repository source code via GitHub Contents API without storing source trees locally.
- **Automated Issue Creation:** Automatically open issues when new panics occur, populated with AST code blocks, Gemini root-cause analyses, and git patches.
- **OAuth User Login:** Secure team authentication with GitHub OAuth.

---

## 1-Click GitHub App Creation

The Studio Dashboard provides an automated manifest-based GitHub App setup:

1. Open your self-hosted Studio Dashboard and start the **Setup Wizard**.
2. Click **Create GitHub App** — Triage generates a GitHub App manifest with the exact required permissions.
3. You will be redirected to GitHub to confirm the App creation.
4. GitHub redirects back with the App ID, Client ID, Client Secret, and Private Key, which Triage securely stores in your PostgreSQL database.

---

## Required Permissions

If creating a GitHub App manually, ensure the following permissions are configured:

| Permission              | Access       | Purpose                              |
| :---------------------- | :----------- | :----------------------------------- |
| **Repository Contents** | Read-only    | On-demand AST fetching by commit SHA |
| **Issues**              | Read & Write | Automated triage issue filing        |
| **Metadata**            | Read-only    | Repository metadata lookup           |
| **User Email**          | Read-only    | GitHub OAuth authentication          |

---

## Automated Issue Template

When a panic occurs, Triage opens an issue with the following format:

```markdown
## 💥 Panic Intercepted: `payment.go:28`

**Trace ID:** `tr_7f9c2d1e8a4b0c3d9a1f`  
**Commit:** `7f8b9e1`  
**Goroutine:** `goroutine 42 [running]`

### 🔍 Gemini AI Diagnostic Summary

> **Root Cause:** Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.  
> **Suggested Fix:** Allocate memory before access: `req := &PaymentPayload{}`

### 🌲 Enclosing AST Node (`ProcessTransaction`)

\`\`\`go
func ProcessTransaction(w http.ResponseWriter, r *http.Request) {
var req *PaymentPayload
if req.Amount <= 0 { // <--- PANIC TRIGGER [LINE 28]
...
}
}
\`\`\`
```
