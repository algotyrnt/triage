---
title: GitHub App, Issue & PR Automation
description: Set up automated GitHub issue creation, bugfix Pull Request generation, permissions, and OAuth authentication
---

Triage connects to GitHub as an official GitHub App to fetch source code for on-demand AST parsing, automatically file triage issues with full stack traces, and open verified bugfix Pull Requests.

## Capabilities

- **On-Demand Source Fetching:** Access private and public repository source code via GitHub Contents API without storing source trees locally.
- **Automated Issue Creation:** Automatically open issues when new panics occur, populated with AST code blocks, AI root-cause analyses, and reproduction details.
- **Automated Bugfix Pull Requests (PRs):** Generate complete bugfix PRs with 1 click directly from the Studio Dashboard or API.
- **Go Monorepo & Multi-Module Detection:** Automatically detect nested `go.mod` files and normalize paths for multi-service repos.
- **Engine-Driven OAuth & RBAC:** Secure authentication with zero secret exposure on the frontend. Automatically grants `Owner` to the first user, honors invitations, and issues cryptographically signed JWT sessions.

---

## 1-Click GitHub App Creation

The Studio Dashboard provides an automated manifest-based GitHub App setup:

1. Open your self-hosted Studio Dashboard and start the **Setup Wizard**.
2. Click **Create GitHub App** — Triage generates a GitHub App manifest with the exact required permissions.
3. You will be redirected to GitHub to confirm the App creation.
4. GitHub redirects back with the App ID, Client ID, Client Secret, and Private Key, which Triage securely stores in your embedded SQLite database.

---

## Required Permissions

When configuring your GitHub App (or creating it manually), ensure the following permissions are granted:

| Permission              | Access       | Purpose                                                     |
| :---------------------- | :----------- | :---------------------------------------------------------- |
| **Repository Contents** | Read & Write | On-demand AST fetching & committing bugfix code to branches |
| **Pull Requests**       | Read & Write | Automated bugfix Pull Request generation                    |
| **Issues**              | Read & Write | Automated triage issue filing                               |
| **Metadata**            | Read-only    | Repository metadata lookup                                  |
| **User Email**          | Read-only    | GitHub OAuth team member login                              |

---

## Automated Bugfix Pull Request Workflow

When a crash is diagnosed, you can trigger an automated Pull Request from the Incident Detail page or via `POST /api/v1/incidents/create-pr`:

```
1. Fetch latest file content & verify base commit SHA via GitHub Contents API
   (with Monorepo path normalization if root_dir is configured)
              │
              ▼
2. Validate patch target file against security policies
   (rejects workflows, Dockerfiles, .env, private keys, path traversal)
              │
              ▼
3. AI Engine synthesizes bugfix via ApplyFixToFile & verifies changes exist
              │
              ▼
4. Create dedicated Git branch: triage/fix-<incident_id>-<random_hex>
              │
              ▼
5. Commit updated file to branch via GitHub Contents API
              │
              ▼
6. Open Pull Request on GitHub referencing incident & closing issue
              │
              ▼
7. Update incident record in database with PR number and PR URL
```

---

## Automated Issue Template

When an incident issue is created, Triage opens an issue with structured diagnostic information:

````markdown
## 🚨 Panic in handlers/payment.go:28: runtime error: invalid memory address or nil pointer dereference

A runtime panic was intercepted by **Triage** in `handlers/payment.go:28`.

---

### Panic Details

- **Incident ID**: `INC-8094`
- **File**: `handlers/payment.go:28`
- **Timestamp**: `2026-08-18 14:32:10 UTC`

---

### Root Cause Analysis (AI Engine)

Attempted to evaluate `req.Amount` on an uninitialized nil pointer (`*PaymentPayload`) on line 28.

---

### Recommended Fix

Allocate memory before access: `req := &PaymentPayload{}` and validate JSON decoding errors.

[**Generate Bugfix PR via Triage Studio**](http://localhost:8080/?incident=INC-8094)

---

### Isolated Function AST Context

<details open>
<summary><b>View Isolated Code Context around line 28</b></summary>

```go
func ProcessTransaction(w http.ResponseWriter, r *http.Request) {
    var req *PaymentPayload
    if req.Amount <= 0 { // <--- PANIC TRIGGER [LINE 28]
        ...
    }
}
```

</details>
````

---

### Stack Trace

```
goroutine 42 [running]:
...
```
