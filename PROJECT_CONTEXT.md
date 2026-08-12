# triage: Architectural & Context Reference Specification

## 1. Executive Summary & Purpose

**triage** is an enterprise-grade developer tool designed to isolate Go application crashes, extract function-level Abstract Syntax Tree (AST) context, and leverage **Google Gemini 3.5 Flash** to diagnose root causes and post automated GitHub issues.

This document serves as the ground-truth technical specification for the repository architecture, data flows, security boundaries, and design systems.

---

## 2. System Architecture & Monorepo Topology

```text
algotyrnt/triage
├── apps/
│   ├── engine/             # Core AI Telemetry & AST Engine (Go 1.26+, Cloud Run :8080)
│   │   ├── internal/ast/   # On-demand AST extractor & in-memory cache
│   │   ├── internal/db/    # PostgreSQL connection pool & incident logger
│   │   ├── internal/llm/   # Google Gemini 3.5 Flash SDK integration
│   │   └── migrations/     # SQL DDL Migration Scripts
│   └── web/                # Web Platform, Docs & Studio Dashboard (Next.js 16 :3000)
├── sdk/
│   └── go/                 # Lightweight Panic Interceptor Middleware
└── test-service/           # Local dummy web app to trigger panics for testing (:8081)
```

| Component                                       | Technology                                 | Deployment Target             | Cost Model                            |
| ----------------------------------------------- | ------------------------------------------ | ----------------------------- | ------------------------------------- |
| **Triage Engine (`apps/engine`)**               | Go 1.26+ (`go/ast`, `pgxpool`, Gemini SDK) | GCP Cloud Run                 | 2 Million Invocations/mo Free (:8080) |
| **Web Platform & Dashboard (`apps/web`)**       | Next.js 16 + Bun + React 19                | GCP Cloud Run / Vercel        | Free Tier (:3000)                     |
| **Test Service (`test-service`)**               | Go 1.26+ Dummy HTTP App                    | Docker Container              | Testing (:8081)                       |
| **Database Module (`apps/engine/internal/db`)** | Go Package (`package db`)                  | PostgreSQL (Cloud SQL / Neon) | Free Tier                             |
| **Triage SDK (`sdk/go`)**                       | Pure Go Standard Library                   | Go Package Import             | N/A                                   |

---

## 3. High-Level Data Flow

```text
[ Target Go Application (test-service) ]
       │ 1. Intercept Panic (sdk/go)
       │    Extracts commit SHA & stack trace, dispatches HTTP POST to Triage Engine (:8080)
       ▼
[ GCP Cloud Run Engine (apps/engine :8080) ]
       │ 2. Validates API Key (PostgreSQL DB / Env Fallback)
       │ 3. Checks In-Memory AST Cache (sync.Map key = owner/repo@commit:file:line)
       │ 4. On Cache Miss: On-demand fetches source code via GitHub API / Raw URL / Workspace fallback
       │ 5. Parses Go AST in memory to isolate panicking function node (*ast.FuncDecl)
       │ 6. Queries Gemini 3.5 Flash with stack trace + isolated function snippet
       │ 7. Persists Incident to PostgreSQL DB (incidents table)
       ▼
[ Studio Dashboard & GitHub ]
       │ 8. Updates Real-Time Studio Dashboard (/dashboard)
       │ 9. POST GitHub Issue via GitHub App API
       ▼
[ GitHub Repository ] <--- Issue Created (#104)
```

---

## 4. Subsystem Breakdown

### A. Client SDK (`sdk/go`)

- **Purpose:** Non-intrusive HTTP middleware wrapping standard Go routers (`net/http`, `chi`, `gin`).
- **Behavior:** Captures panics via `defer` + `recover()`, pulls raw trace via `runtime/debug.Stack()`, extracts Git commit SHA via `debug.ReadBuildInfo()`, generates OpenTelemetry Trace ID (`X-Triage-Trace-ID`), and dispatches an asynchronous, non-blocking HTTP POST request to the Triage Engine (`:8080/api/v1/telemetry`).
- **Usage:** `triage.Middleware(apiKey, opts...)` (supports optional `triage.WithGatewayURL(...)` for self-hosted deployments).

### B. Triage Engine (`apps/engine`)

- **Purpose:** Unified Go backend pipeline that authenticates incoming telemetry, fetches source code on demand, isolates function AST nodes, handles Gemini AI diagnostics, and persists incident audit logs.
- **Key Modules:**
  - `internal/ast`: On-demand source code fetcher, in-memory KV AST snippet cache, and Go `go/ast` parser.
  - `internal/db`: PostgreSQL connection pool (`pgxpool`), API key verifier, and incident record storage.
  - `internal/llm`: Uses `google.golang.org/genai` to analyze the stack trace alongside the isolated AST snippet.

### C. Web Platform & Studio Dashboard (`apps/web`)

- **Purpose:** Unified Next.js 16 web application integrating the product marketing landing page (`/`), SDK integration docs (`/docs`), and Studio Dashboard UI (`/dashboard`).

### D. Test Service (`test-service`)

- **Purpose:** Dummy Go web application harness on `:8081` designed to simulate panics (`GET /crash`) and test end-to-end telemetry ingestion.

---

## 5. Security & Edge Case Policies

1. **Path Traversal Protection:** Sanitize all incoming file paths. Reject any path containing `..` or leading slashes outside the project root context.
2. **Payload Truncation:** Limit request body sizes to 1MB to prevent Denial of Service (DoS) memory exhaustion.
