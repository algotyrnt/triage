# triage: Go Crash Isolation & AI Diagnostic Engine

> **[IMPORTANT] Work In Progress (WIP)**
> triage is currently in active early development. Interfaces, telemetry protocols, and SDK signatures may evolve as features are added.

**triage** is a zero-overhead Go crash isolation tool, automated GitHub issue triaging engine, and AI-powered diagnostic platform. When a panic occurs in a Go web application, triage intercepts the crash non-blockingly, queries pre-parsed function AST nodes (`*ast.FuncDecl`) directly from PostgreSQL, queries Google Gemini for instant root-cause analysis, and automatically posts a detailed issue to your GitHub repository.

---

## Architecture & System Data Flow

```text
Panic Crash Telemetry & On-Demand Synchronous AST Isolation Flow
   │
   ▼
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                           Go Triage Manager (:8000) Ingestion                              │
│                                                                                            │
│  1. Receive SDK Telemetry  ──►  2. Verify API Key  ──►  3. Proxy Payload to Triage Engine  │
│     (Commit + Stack Trace)         (PostgreSQL DB)            (:8080)                      │
└───────────────────────────────────────────┬────────────────────────────────────────────────┘
                                            │
                                            ▼
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 Triage Engine (:8080)                                      │
│                                                                                            │
│  1. Check In-Memory  ──►  2. Fetch Source Code  ──►  3. Parse AST Node  ──► 4. Gemini AI  │
│     AST KV Cache             (GitHub API / Local)       (*ast.FuncDecl)         Diagnostics│
└───────────────────────────────────────────┬────────────────────────────────────────────────┘
                                            │
                                  Engine JSON Response
                                            ▼
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                      Manager Routing & Production Ecosystem                                │
│                                                                                            │
│  1. Go Triage Manager (:8000)   ──► 2. Web Platform & Studio Dashboard (:3000)            │
│     (Save Incident Audit Log)        (Landing page /, Docs, /dashboard UI)                 │
└────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Architecture

### 1. Client SDK (`sdk/go`)

The Triage SDK provides a lightweight HTTP middleware (`triage.Middleware`) wrapping standard `http.Handler` routes. On panic:

- Intercepts crash using `defer + recover()`.
- Captures stack trace using `debug.Stack()`.
- Automatically extracts Git commit SHA via `debug.ReadBuildInfo()` (Go 1.18+ embedded VCS info), with options `WithCommit("...")` and `WithRepo("owner/repo")`.
- Slices top application stack frame line and file path.
- Generates OpenTelemetry Trace ID (`X-Triage-Trace-ID` and `traceparent` headers).
- Dispatches an **asynchronous, non-blocking** HTTP POST request to the Triage Manager.

### 2. Core Go Engine (`apps/engine`)

- **On-Demand Source Fetcher & AST Parser ([`internal/ast`](apps/engine/internal/ast))**: On telemetry panic ingestion, fetches the exact source code for the reported `commit_sha` on-demand (via GitHub API / Raw URL or local workspace context), parses the Go AST in memory, and isolates the panicking `*ast.FuncDecl` snippet.
- **In-Memory AST KV Cache**: Caches extracted function AST snippets in a thread-safe `sync.Map` store keyed by `owner/repo@commit:file:line` for **< 2ms** instant lookups on repeat panics.
- **AI Diagnostics (`internal/llm`)**: Uses official Google Gemini SDK (`google.golang.org/genai`) to send stack trace + AST snippet to Gemini 3.5 Flash (`GEMINI_MODEL_NAME`), generating structured JSON with `root_cause` and `suggested_fix`.

### 3. Go Triage Manager (`apps/manager`)

- Built in **Go 1.26+** (`net/http`, `pgxpool`). Default port **`:8000`**. Single `go.mod` file module.
- **50x Faster Cold Starts (< 50ms)** and **< 15 MB RAM footprint**.
- Handles incoming telemetry proxies (`/api/telemetry`), API key verification, PostgreSQL connection pooling (`apps/manager/db`), and GitHub App webhooks (`/api/github/webhook`), verifying `X-Hub-Signature-256` HMAC signatures with **Webhook Delivery Idempotency**.

### 4. Web Platform & Studio Dashboard (`apps/web`)

- Built with **Next.js 16 App Router**, **Bun**, and **React 19**.
- Integrates the product marketing landing page (`/`), SDK integration docs, and the real-time Studio Dashboard UI (`/dashboard`).

---

## PostgreSQL Database Schema (`apps/engine/migrations/0001_init.sql`)

The database DDL migration script provides full PostgreSQL schema definitions:

```sql
-- 1. Users & OAuth Sessions
CREATE TABLE users (
  id VARCHAR(64) PRIMARY KEY,
  github_id VARCHAR(64) UNIQUE NOT NULL,
  username VARCHAR(255) NOT NULL,
  avatar_url TEXT,
  role VARCHAR(32) DEFAULT 'Member',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tracked Repositories & GitHub App Installations
CREATE TABLE repositories (
  id VARCHAR(64) PRIMARY KEY,
  owner VARCHAR(255) NOT NULL,
  repo VARCHAR(255) NOT NULL,
  installation_id BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3a. AST Function Symbol Storage (Indexed for < 1ms lookups)
CREATE TABLE ast_nodes (
  id VARCHAR(64) PRIMARY KEY,
  owner VARCHAR(255) NOT NULL,
  repo VARCHAR(255) NOT NULL,
  commit_sha VARCHAR(64) NOT NULL,
  file_path TEXT NOT NULL,
  line_number INT NOT NULL,
  function_name VARCHAR(255),
  ast_snippet TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ast_nodes_lookup ON ast_nodes (owner, repo, file_path, line_number);

-- 3b. Repository AST Indexing Jobs & Enum Status Tracker
CREATE TABLE ast_indexes (
  id VARCHAR(64) PRIMARY KEY,
  owner VARCHAR(255) NOT NULL,
  repo VARCHAR(255) NOT NULL,
  commit_sha VARCHAR(64) NOT NULL,
  branch VARCHAR(128) DEFAULT 'main',
  status VARCHAR(32) DEFAULT 'PENDING', -- PENDING, PROCESSING, INDEXED, FAILED
  parsed_files_count INT DEFAULT 0,
  total_functions_count INT DEFAULT 0,
  error_message TEXT,
  indexed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Ingested Incidents & AI Analysis Log
CREATE TABLE incidents (
  id VARCHAR(64) PRIMARY KEY,
  repository_id VARCHAR(64) REFERENCES repositories(id),
  title TEXT NOT NULL,
  status VARCHAR(32) DEFAULT 'CRITICAL',
  file TEXT NOT NULL,
  line INT NOT NULL,
  panic_message TEXT NOT NULL,
  stack_trace TEXT NOT NULL,
  ast_snippet TEXT,
  root_cause TEXT,
  suggested_fix TEXT,
  github_issue_url TEXT,
  github_issue_number INT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 5. API Keys
CREATE TABLE api_keys (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) REFERENCES users(id),
  repository_id VARCHAR(64) REFERENCES repositories(id),
  name VARCHAR(255) NOT NULL,
  key_hash VARCHAR(255) UNIQUE NOT NULL,
  key_masked VARCHAR(64) NOT NULL,
  revoked_at TIMESTAMP WITH TIME ZONE,
  expires_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 6. Webhook Delivery Idempotency & Audit Logs
CREATE TABLE webhook_logs (
  id VARCHAR(64) PRIMARY KEY,
  delivery_id VARCHAR(255) UNIQUE,
  event_type VARCHAR(128) NOT NULL,
  status VARCHAR(32) DEFAULT 'SUCCESS',
  status_code INT DEFAULT 200,
  request_body TEXT,
  response_body TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

---

## Repository Structure

```text
.
├── PROJECT_CONTEXT.md    # Master architecture & security reference specification
├── apps/
│   ├── dashboard/        # Next.js 16 Studio Dashboard App (Port :3000, Dockerfile)
│   │   ├── src/          # React Studio Dashboard UI, components & engine client
│   │   └── Dockerfile    # Multi-stage production container image
│   ├── engine/           # Go 1.26+ Core Engine (On-demand AST extractor, DB manager & Gemini SDK)
│   │   ├── internal/ast/ # On-demand AST fetcher & in-memory cache
│   │   ├── internal/db/  # PostgreSQL database pool & incident store
│   │   ├── internal/llm/ # Gemini 3.5 Flash SDK integration
│   │   ├── migrations/   # PostgreSQL SQL DDL migration scripts
│   │   ├── Dockerfile    # Multi-stage production container image
│   │   ├── .env.example  # Engine environment template
│   │   └── main.go       # HTTP Telemetry server listening on :8080
│   └── web/              # Public Marketing Landing Page & Starlight Docs (Astro static site, NO Dockerfile)
│       └── src/content/  # Starlight documentation markdown files
├── sdk/
│   └── go/               # Go Client SDK (panic middleware & telemetry dispatcher)
└── test-service/         # Local test harness triggering panic simulations (:8081)
```

---

## Quickstart

### Prerequisites

- **Docker** and **Docker Compose**
- **Go**: 1.26+ (If running the SDK simulator natively)

### 1. Boot the Triage Stack

Triage runs locally via Docker Compose. The only required environment configuration is the `DATABASE_URL`, all other settings are handled securely in the UI.

```bash
# Set up environment variables
cp .env.example .env

# Start the stack (Postgres, Go Engine, Next.js Dashboard)
docker-compose up --build -d
```

### 2. Complete the Setup Wizard

Once the containers are running, navigate to the Dashboard to securely configure your instance:

1. Open **[http://localhost:3000](http://localhost:3000)** in your browser.
2. The 5-step **Setup Wizard** will automatically guide you through:
   - Generating and installing a **GitHub App** (for repository AST access and bug reporting).
   - Linking **GitHub OAuth** (for secure dashboard logins).
   - Setting up your **Gemini AI Configuration** (API Key and Model Name).
   - Note: The Engine will automatically generate and persist a secure cryptographic session secret on its first boot—no manual configuration required.
3. Once the final verification step succeeds, log in with your GitHub account!

### 3. Run a Crash Simulation

You can test the crash analysis pipeline using the local SDK test harness:

```bash
cd test-service
go run main.go
# Server listening on http://localhost:8081

# Trigger a test nil pointer dereference panic:
curl http://localhost:8081/crash
```

Check the engine logs or open `http://localhost:3000/dashboard` to inspect the isolated AST node and Gemini AI root cause analysis!

---

## SDK Integration Example

Add Triage middleware to any standard Go HTTP multiplexer:

```go
package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/data", handleData)

	// Managed Cloud Usage (Zero config needed):
	wrappedHandler := triage.Middleware("tr_cloud_key_123")(mux)

	// Self-Hosted Enterprise Usage (Custom Gateway URL):
	// wrappedHandler := triage.Middleware(
	// 	"tr_selfhosted_key",
	// 	triage.WithGatewayURL("https://triage.mycompany.internal/api/telemetry"),
	// )(mux)

	http.ListenAndServe(":8081", wrappedHandler)
}
```

---

## WIP Roadmap & Issue Tracker

Track all roadmap items on the **[Milestone v1.0.0](https://github.com/algotyrnt/triage/milestone/1)** and **[GitHub Project Board](https://github.com/algotyrnt/triage/projects)**.

---

## Author & License

Created by **Punjitha Bandara (algotyrnt)** - [https://algotyrnt.com](https://algotyrnt.com)

Licensed under the **Apache License, Version 2.0**. See [LICENSE](LICENSE) for the full license text.
