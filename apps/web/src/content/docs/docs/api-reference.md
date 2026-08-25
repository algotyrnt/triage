---
title: Engine REST API Reference
description: Complete REST API specification for telemetry ingestion, incidents, GitHub PR/issue automation, projects, and Gemini AI endpoints
---

All Triage Engine endpoints are served by default on port `:8080`.

## Health Check

### `GET /health`

Returns the operational health of the Triage engine and PostgreSQL database connection.

**Response (200 OK):**

```json
{
  "status": "healthy",
  "database": "connected"
}
```

---

## Telemetry Ingestion

### `POST /api/v1/telemetry`

Receives panic telemetry payloads dispatched asynchronously by the Go SDK middleware.

**Headers:**

- `Content-Type: application/json`

**Request Body:**

```json
{
  "api_key": "tr_live_payment_1724123456",
  "commit": "7f8b9e1a2c3d4e5f60718293",
  "file": "handlers/payment.go",
  "line": 28,
  "stack_trace": "goroutine 42 [running]:\n...",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

**Response (200 OK):**

```json
{
  "incident_id": "INC-8094",
  "status": "PROCESSED",
  "root_cause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
  "suggested_fix": "Allocate memory before assignment: req := &PaymentPayload{}",
  "latency_ms": 14
}
```

---

## Real-Time Event Streaming (SSE)

### `GET /api/v1/events/stream`

Streams live telemetry events, new panic incidents, and incident updates to connected dashboard clients using standard **Server-Sent Events (SSE)**.

**Headers:**

- `Accept: text/event-stream`
- `Authorization: Bearer <JWT>` _(optional, if session token is not provided via query)_

**Query Parameters:**

- `token` _(optional)_: JWT session token when connecting from browser `EventSource` (which cannot send custom headers).

**Response (200 OK — `text/event-stream`):**

The stream maintains a continuous HTTP connection with periodic 15-second heartbeat comments (`: keepalive\n\n`) to prevent proxy timeouts.

#### Event Types:

##### 1. Initial Handshake (`connected`)

Sent immediately when a client establishes a connection:

```json
data: {"type":"connected","timestamp":"2026-08-25T14:30:00Z"}
```

##### 2. Incident Ingested (`incident_created`)

Dispatched immediately when a runtime Go panic is intercepted and recorded:

```json
data: {"type":"incident_created","data":{"id":"INC-8094","title":"nil pointer dereference","status":"CRITICAL","file":"handlers/payment.go","line":28,"created_at":"2026-08-25T14:30:05Z"},"timestamp":"2026-08-25T14:30:05Z"}
```

##### 3. Incident Updated (`incident_updated`)

Dispatched when an automated GitHub Issue or Pull Request is attached to an incident:

```json
data: {"type":"incident_updated","data":{"id":"INC-8094","github_issue_url":"https://github.com/myorg/payments-service/issues/42","github_pr_url":"https://github.com/myorg/payments-service/pull/43"},"timestamp":"2026-08-25T14:30:10Z"}
```

---

## Incidents Management

### `GET /api/v1/incidents`

Lists recent recorded incidents with optional filtering by repository ID or repository slug.

**Query Parameters:**

- `repository_id` _(optional)_: Filter incidents by internal repository UUID.
- `repo` _(optional)_: Filter incidents by repository slug (e.g. `owner/repo` or `repo`).

**Response (200 OK):**

```json
{
  "incidents": [
    {
      "id": "INC-8094",
      "repository_id": "repo_1a2b3c",
      "repository_name": "myorg/payments-service",
      "title": "nil pointer dereference in ProcessTransaction()",
      "status": "CRITICAL",
      "file": "handlers/payment.go",
      "line": 28,
      "panic_message": "runtime error: invalid memory address or nil pointer dereference",
      "root_cause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
      "suggested_fix": "Allocate memory with req := &PaymentPayload{} before field access.",
      "suggested_patch": "@@ -26,3 +26,4 @@\n- var req *PaymentPayload\n+ req := &PaymentPayload{}",
      "github_issue_number": 42,
      "github_issue_url": "https://github.com/myorg/payments-service/issues/42",
      "github_pr_number": 43,
      "github_pr_url": "https://github.com/myorg/payments-service/pull/43",
      "created_at": "2026-08-18T14:32:10Z"
    }
  ]
}
```

---

## GitHub Automation

### `POST /api/v1/incidents/create-issue`

Creates a detailed GitHub Issue on the incident's target repository with AST code snippets, Gemini AI root cause analysis, stack trace, and reproduction links.

**Request Body:**

```json
{
  "incident_id": "INC-8094"
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "github_issue": {
    "number": 42,
    "html_url": "https://github.com/myorg/payments-service/issues/42"
  }
}
```

---

### `POST /api/v1/incidents/create-pr`

Generates and opens an automated bugfix Pull Request on GitHub. The engine fetches the target file, synthesizes the fix using Gemini AI, creates a new Git branch (`triage/fix-...`), commits the updated file, and opens a linked Pull Request.

**Request Body:**

```json
{
  "incident_id": "INC-8094",
  "patch_code": "@@ -26,3 +26,4 @@..." // optional override
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "pull_request": {
    "number": 43,
    "html_url": "https://github.com/myorg/payments-service/pull/43",
    "branch": "triage/fix-inc8094-1724123456"
  }
}
```

---

## Projects & Monorepo Modules

### `GET /api/v1/projects`

Lists all tracked repositories and configured Go services.

**Response (200 OK):**

```json
{
  "projects": [
    {
      "id": "proj_9042",
      "owner": "myorg",
      "repo": "payments-service",
      "root_dir": "backend",
      "installation_id": 12345678,
      "context": "High-throughput payment gateway processing Stripe webhooks.",
      "api_key_masked": "tr_live_...9042",
      "created_at": "2026-08-15T10:00:00Z"
    }
  ]
}
```

---

### `POST /api/v1/projects`

Registers a new repository or Go monorepo service, saves domain context, and issues an initial API key.

**Request Body:**

```json
{
  "owner": "myorg",
  "repo": "payments-service",
  "root_dir": "backend",
  "owner_username": "octocat",
  "context": "High-throughput payment gateway processing Stripe webhooks."
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "repo": "myorg/payments-service",
  "root_dir": "backend",
  "context": "High-throughput payment gateway processing Stripe webhooks.",
  "api_key": "tr_live_payments-service_1724123456",
  "key_masked": "tr_live_...3456"
}
```

---

### `PUT /api/v1/projects/context`

Updates the architectural and domain context for an existing project.

**Request Body:**

```json
{
  "owner": "myorg",
  "repo": "payments-service",
  "root_dir": "backend",
  "context": "Updated architectural context and domain invariants for payment service."
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "owner": "myorg",
  "repo": "payments-service",
  "root_dir": "backend",
  "context": "Updated architectural context and domain invariants for payment service."
}
```

---

### `GET /api/v1/repos/detect-modules`

Automatically scans a GitHub repository or local workspace for nested `go.mod` files to configure monorepo submodules.

**Query Parameters:**

- `owner`: Repository owner (e.g. `myorg`)
- `repo`: Repository name (e.g. `myrepo`)

**Response (200 OK):**

```json
{
  "modules": [
    {
      "path": "",
      "name": "Repository Root (/)",
      "is_root": true
    },
    {
      "path": "apps/api",
      "name": "apps/api/ (Go Module)",
      "is_root": false
    },
    {
      "path": "services/worker",
      "name": "services/worker/ (Go Module)",
      "is_root": false
    }
  ]
}
```

---

### `GET /api/v1/projects/keys`

Lists API keys for a specific project.

**Query Parameters:**

- `owner`: Repository owner
- `repo`: Repository name
- `root_dir`: _(optional)_ Monorepo subfolder

**Response (200 OK):**

```json
{
  "keys": [
    {
      "id": "key_1724123456",
      "name": "Production Service Key",
      "key_masked": "tr_live_...3456",
      "status": "ACTIVE",
      "created_at": "2026-08-15T10:00:00Z"
    }
  ]
}
```

---

### `POST /api/v1/projects/keys/revoke`

Revokes a project API key.

**Request Body:**

```json
{
  "key_id": "key_1724123456"
}
```

**Response (200 OK):**

```json
{
  "success": true
}
```

---

## Gemini AI Diagnostics

### `POST /api/v1/gemini/analyze-panic`

Runs on-demand structured root cause analysis on a panic snippet using Google Gemini AI.

**Request Body:**

```json
{
  "panicMessage": "runtime error: invalid memory address or nil pointer dereference",
  "rawStackTrace": "goroutine 42 [running]:\n...",
  "triggeringFile": "handlers/payment.go",
  "astCode": "func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n..."
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "rootCause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
  "explanation": "The variable req is declared as a pointer but never initialized before dereference.",
  "severity": "CRITICAL",
  "recommendedFix": "Allocate memory with req := &PaymentPayload{} and validate JSON payload before field access."
}
```

---

### `POST /api/v1/gemini/generate-patch`

Generates a unified git diff format patch for a diagnosed crash.

**Request Body:**

```json
{
  "triggeringFile": "handlers/payment.go",
  "panicMessage": "runtime error: invalid memory address or nil pointer dereference",
  "astCode": "func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n...",
  "rootCause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
  "stackTrace": "goroutine 42 [running]:\n..."
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "patch": "@@ -26,3 +26,4 @@ func ProcessTransaction(w http.ResponseWriter, r *http.Request) {\n-    var req *PaymentPayload\n-    if req.Amount <= 0 {\n+    req := &PaymentPayload{}\n+    if err := json.NewDecoder(r.Body).Decode(req); err != nil || req.Amount <= 0 {"
}
```

---

### `GET/POST /api/v1/settings/llm`

View or update the active Gemini model and API key configuration.

**GET Response (200 OK):**

```json
{
  "gemini_model": "gemini-2.5-flash",
  "has_api_key": true
}
```

---

## AST Pre-Indexing

### `POST /api/v1/ast/index`

Pre-indexes a repository's package AST declarations into PostgreSQL for sub-millisecond `< 5ms` symbolication.

**Request Body:**

```json
{
  "repo": "myorg/payments-service",
  "root_dir": "backend",
  "commit_sha": "main"
}
```

**Response (200 OK):**

```json
{
  "status": "INDEXED",
  "functions_indexed": 142,
  "files_processed": 18
}
```

---

## Engine Stats

### `GET /api/v1/stats`

Returns runtime statistics including total projects, total panics ingested, and database connection status.

**Response (200 OK):**

```json
{
  "projects_count": 4,
  "incidents_count": 12,
  "critical_count": 2,
  "status": "operational"
}
```

---

## Authentication & RBAC Identity

Triage uses Go engine-driven GitHub OAuth and issues cryptographically signed JWT sessions.

### `GET /api/v1/auth/github`

Initiates the GitHub OAuth authorization redirect with cryptographic CSRF state cookies.

---

### `GET /api/v1/auth/github/callback`

Receives GitHub OAuth authorization code, exchanges it securely with GitHub, upserts user profile, assigns role, and redirects to dashboard with session JWT.

---

### `GET /api/v1/auth/me`

Returns the current authenticated caller profile and role.

**Headers:**

- `Authorization: Bearer <JWT>`

**Response (200 OK):**

```json
{
  "user": {
    "id": "usr_12345",
    "github_id": "12345",
    "username": "octocat",
    "email": "octocat@github.com",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345",
    "role": "Owner"
  }
}
```

---

## Team & Member Access Control

Endpoints to manage team members, roles (`Owner`, `Admin`, `Developer`, `Viewer`), and pending invitations.

### `GET /api/v1/team/members`

Lists all active team members in the organization.

**Headers:**

- `Authorization: Bearer <JWT>`

**Response (200 OK):**

```json
{
  "members": [
    {
      "id": "usr_12345",
      "github_id": "12345",
      "username": "octocat",
      "email": "octocat@github.com",
      "avatar_url": "https://avatars.githubusercontent.com/u/12345",
      "role": "Owner",
      "created_at": "2026-08-15T10:00:00Z"
    }
  ]
}
```

---

### `PUT /api/v1/team/members/role`

Updates a team member's assigned role. Requires `Owner` or `Admin` permissions.

**Request Body:**

```json
{
  "id": "usr_67890",
  "role": "Admin"
}
```

**Response (200 OK):**

```json
{
  "status": "success",
  "id": "usr_67890",
  "role": "Admin"
}
```

---

### `DELETE /api/v1/team/members`

Revokes a team member's access. Requires `Owner` permission.

**Query Parameters:**

- `id`: Target user ID (e.g. `usr_67890`)

**Response (200 OK):**

```json
{
  "status": "success",
  "id": "usr_67890"
}
```

---

### `GET /api/v1/team/invites`

Lists all pending team invitations.

**Response (200 OK):**

```json
{
  "invitations": [
    {
      "id": "inv_1724123456",
      "github_username": "torvalds",
      "role": "Developer",
      "invited_by": "usr_12345",
      "created_at": "2026-08-20T12:00:00Z"
    }
  ]
}
```

---

### `POST /api/v1/team/invites`

Creates a pending invitation for a GitHub username. Requires `Owner` or `Admin` permissions.

**Request Body:**

```json
{
  "github_username": "torvalds",
  "role": "Developer"
}
```

**Response (201 Created):**

```json
{
  "status": "created",
  "invitation": {
    "id": "inv_1724123456",
    "github_username": "torvalds",
    "role": "Developer",
    "invited_by": "usr_12345",
    "created_at": "2026-08-20T12:00:00Z"
  }
}
```

---

### `DELETE /api/v1/team/invites`

Revokes a pending invitation.

**Query Parameters:**

- `id`: Invitation ID or GitHub username

**Response (200 OK):**

```json
{
  "status": "success",
  "id": "inv_1724123456"
}
```
