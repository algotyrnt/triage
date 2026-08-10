# triage: Go Crash Isolation & AI Diagnostic Engine

> **[IMPORTANT] Work In Progress (WIP)**
> triage is currently in active early development. Interfaces, telemetry protocols, and SDK signatures may evolve as features are added.

**triage** is a zero-overhead Go crash isolation tool, automated GitHub issue triaging engine, and AI-powered diagnostic platform. When a panic occurs in a Go web application, triage intercepts the crash non-blockingly, queries pre-parsed function AST nodes (`*ast.FuncDecl`) directly from PostgreSQL, queries Google Gemini for instant root-cause analysis, and automatically posts a detailed issue to your GitHub repository.

---

## Architecture & System Data Flow

```text
1. Asynchronous Repository AST Indexing (git push webhook)
   │
   ▼
[ Go Triage Manager (apps/manager :8000) ]
   │ Marks DB status = 'PENDING' in PostgreSQL (ast_indexes)
   │ Triggers Engine AST Indexer
   ▼
[ Go Triage Engine (apps/engine :8080) ]
   │ Parses .go files ONCE using Go's native go/ast parser
   │ Writes AST function nodes to PostgreSQL ast_nodes table
   │ Updates DB status = 'INDEXED'
   ▼
[ PostgreSQL DB ] (ast_nodes table: indexed & ready)


2. Panic Crash Telemetry Ingestion (0ms Disk I/O)
   │
   ▼
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 Triage Engine (:8080)                                      │
│                                                                                            │
│  1. Parse Stack Trace  ──►  2. Query Pre-Parsed AST  ──►  3. Gemini 3.5 Flash ──► 4. Return │
│     (Top App Frame)            From PostgreSQL DB           (genai SDK)            JSON    │
└───────────────────────────────────────────┬────────────────────────────────────────────────┘
                                            │
                                    JSON Telemetry Stream
                                            ▼
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                           Triage Production Ecosystem                                      │
│                                                                                            │
│  1. Go Triage Manager (:8000)   ──► 2. Web Platform & Studio Dashboard (:3000)            │
│     (/api/telemetry, webhook)        (Landing page /, Docs, /dashboard UI)                 │
└────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Architecture

### 1. Client SDK (`sdk/go`)

The Triage SDK provides a lightweight HTTP middleware (`triage.Middleware`) wrapping standard `http.Handler` routes. On panic:

- Intercepts crash using `defer + recover()`.
- Captures stack trace using `debug.Stack()`.
- Slices top application stack frame line and file path (ignoring standard library & middleware frames).
- Generates OpenTelemetry Trace ID (`X-Triage-Trace-ID` and `traceparent` headers).
- Dispatches an **asynchronous, non-blocking** HTTP POST request to the Triage Manager.

### 2. Core Go Engine (`apps/engine`)

- **PostgreSQL Pre-Parsed AST Querying ([`internal/ast`](file:///Users/punjitha/projects/triage/apps/engine/internal/ast/node.go))**: Connects directly to PostgreSQL database via `pgxpool`. Queries pre-parsed `*ast.FuncDecl` snippets directly from the database (**0 disk I/O, zero on-demand file parsing, minimal RAM usage**).
- **Background AST Indexer**: `POST /api/v1/ast/index` endpoint parses repository `.go` files once, extracts function nodes, and populates PostgreSQL `ast_nodes`.
- **AI Diagnostics (`internal/llm`)**: Uses official Google Gemini SDK (`google.golang.org/genai`) to send stack trace + AST snippet to Gemini 3.5 Flash (`GEMINI_MODEL_NAME`), generating structured JSON with `root_cause` and `suggested_fix`.

### 3. Go Triage Manager (`apps/manager`)

- Built in **Go 1.26+** (`net/http`, `pgxpool`). Default port **`:8000`**. Single `go.mod` file module.
- **50x Faster Cold Starts (< 50ms)** and **< 15 MB RAM footprint**.
- Handles incoming telemetry proxies (`/api/telemetry`), API key verification, PostgreSQL connection pooling (`apps/manager/db`), and GitHub App webhooks (`/api/github/webhook`), verifying `X-Hub-Signature-256` HMAC signatures with **Webhook Delivery Idempotency**.

### 4. Web Platform & Studio Dashboard (`apps/web`)

- Built with **Next.js 16 App Router**, **Bun**, and **React 19**.
- Integrates the product marketing landing page (`/`), SDK integration docs, and the real-time Studio Dashboard UI (`/dashboard`).

---

## PostgreSQL Database Schema (`apps/manager/migrations/0001_init.sql`)

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
  name VARCHAR(255) NOT NULL,
  key_hash VARCHAR(255) NOT NULL,
  key_masked VARCHAR(64) NOT NULL,
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
│   ├── engine/           # Go 1.26+ Core Engine (PostgreSQL AST querier & Gemini 3.5 Flash SDK)
│   │   ├── internal/ast/ # go/ast function node extractor & PostgreSQL AST manager
│   │   ├── internal/llm/ # Gemini 3.5 Flash SDK integration
│   │   ├── Dockerfile    # Multi-stage production container image
│   │   ├── .env.example  # Engine environment template
│   │   └── main.go       # HTTP Telemetry server listening on :8080
│   ├── manager/          # Go 1.26+ Control Plane & Manager Cloud Function (:8000)
│   │   ├── db/           # Database Models & Query Methods (package db)
│   │   │   ├── db.go
│   │   │   └── migrations/ 0001_init.sql
│   │   ├── main.go       # Ingestion & webhook proxy handler
│   │   ├── Dockerfile    # Multi-stage production container image
│   │   └── .env.example  # Manager environment template
│   └── web/              # Web Platform, Docs & Studio Dashboard (:3000)
│       ├── src/app/      # Landing page, docs & /dashboard UI
│       ├── Dockerfile    # Multi-stage production container image
│       └── .env.example  # Web environment template
├── sdk/
│   └── go/               # Go Client SDK (panic middleware & telemetry dispatcher)
└── test-service/         # Local test harness triggering panic simulations (:8081)
```

---

## Quickstart

### Prerequisites

- **Go**: 1.26 or higher
- **Bun**: 1.0 or higher
- **PostgreSQL**: 14 or higher (or Cloud SQL / Neon Serverless Postgres)
- **Gemini API Key**: Set `GEMINI_API_KEY` in environment

### 1. Start the Triage Engine

Configure `apps/engine/.env.local` (see [`apps/engine/.env.example`](file:///Users/punjitha/projects/triage/apps/engine/.env.example)):

```env
PORT=8080
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/triage_db
TRIAGE_API_KEY=tr_test_key_9042
GEMINI_API_KEY=your_google_ai_studio_key_here
GEMINI_MODEL_NAME=gemini-3.5-flash
AST_WORKSPACE_ROOT=../../
```

Run the engine server:

```bash
cd apps/engine
go run main.go
# Listening on http://localhost:8080
```

### 2. Start the Manager & Web Platform

Configure `apps/manager/.env.local` and `apps/web/.env.local`:

```bash
# Start Go Manager Function (:8000)
cd apps/manager && go run main.go

# Start Web Platform & Studio Dashboard (:3000)
cd apps/web && bun dev
```

### 3. Run Test Crash Simulation

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
