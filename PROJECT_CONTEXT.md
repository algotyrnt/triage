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
│   ├── manager/            # Control Plane, Gateway & DB Manager (Go 1.26+, Cloud Run :8000)
│   │   ├── db/             # Database Models & Helper Methods (package db)
│   │   └── migrations/     # SQL DDL Migration Scripts
│   └── web/                # Web Platform, Docs & Studio Dashboard (Next.js 16 :3000)
├── sdk/
│   └── go/                 # Lightweight Panic Interceptor Middleware
└── test-service/           # Local dummy web app to trigger panics for testing (:8081)
```

| Component | Technology | Deployment Target | Cost Model |
| --- | --- | --- | --- |
| **Triage Manager (`apps/manager`)** | Go 1.26+ (`net/http`, `pgxpool`) | GCP Cloud Run Function | Free Tier (< 15MB RAM, :8000) |
| **Triage Engine (`apps/engine`)** | Go 1.26+ (`go/ast`, Gemini SDK) | GCP Cloud Run | 2 Million Invocations/mo Free (:8080) |
| **Web Platform & Dashboard (`apps/web`)** | Next.js 16 + Bun + React 19 | GCP Cloud Run / Vercel | Free Tier (:3000) |
| **Test Service (`test-service`)** | Go 1.26+ Dummy HTTP App | Docker Container | Testing (:8081) |
| **Database Module (`apps/manager/db`)** | Go Package (`package db`) | PostgreSQL (Cloud SQL / Neon) | Free Tier |
| **Triage SDK (`sdk/go`)** | Pure Go Standard Library | Go Package Import | N/A |

---

## 3. High-Level Data Flow

```text
[ Target Go Application (test-service) ] 
       │ 1. Intercept Panic (sdk/go)
       │    Dispatches HTTP POST to Triage Manager
       ▼
[ Go Triage Manager Cloud Function (apps/manager :8000) ]
       │ 2. Validates API Key in PostgreSQL DB (api_keys table)
       │ 3. Fetches pre-parsed AST snippet from PostgreSQL DB (ast_nodes table)
       │ 4. Proxies payload to Core AI Engine (:8080/api/v1/telemetry)
       ▼
[ GCP Cloud Run Engine (apps/engine :8080) ]
       │ 5. Queries Gemini 3.5 Flash (Root cause + Suggested Fix)
       │ 6. Returns JSON diagnostics to Triage Manager
       ▼
[ Studio Dashboard & GitHub ]
       │ 7. Persists Incident to PostgreSQL DB (incidents table)
       │ 8. Updates Real-Time Studio Dashboard (/dashboard)
       │ 9. POST GitHub Issue via GitHub App API
       ▼
[ GitHub Repository ] <--- Issue Created (#104)
```

---

## 4. Subsystem Breakdown

### A. Client SDK (`sdk/go`)
* **Purpose:** Non-intrusive HTTP middleware wrapping standard Go routers (`net/http`, `chi`, `gin`).
* **Behavior:** Captures panics via `defer` + `recover()`, pulls raw trace via `runtime/debug.Stack()`, extracts caller line/file, generates OpenTelemetry Trace ID (`X-Triage-Trace-ID`), and dispatches an asynchronous, non-blockingly queued HTTP POST request to the Triage Manager.
* **Usage:** `triage.Middleware(apiKey, opts...)` (supports optional `triage.WithGatewayURL(...)` for self-hosted deployments).

### B. Triage Engine (`apps/engine`)
* **Purpose:** Stateless pipeline that processes incoming crashes and handles LLM analysis.
* **Key Modules:**
  * `internal/ast`: PostgreSQL-backed AST manager (`pgxpool`) and background repository indexer (`POST /api/v1/ast/index`).
  * `internal/llm`: Uses `google.golang.org/genai` to analyze the stack trace alongside the isolated AST snippet. Requests structured JSON output containing root cause, severity, and suggested code fix.

### C. Triage Manager (`apps/manager`)
* **Purpose:** High-performance Go Cloud Run control-plane function handling crash ingestion (`/api/telemetry`), API key verification, PostgreSQL connection pooling (`apps/manager/db`), and GitHub App webhooks (`/api/github/webhook`). Single `go.mod` file module.
* **Performance:** Default port **`:8000`**, boot time **< 50ms**, memory footprint **< 15 MB**. Enforces **Webhook Delivery Idempotency** (`webhook_logs` table).

### D. Web Platform & Studio Dashboard (`apps/web`)
* **Purpose:** Unified Next.js 16 web application integrating the product marketing landing page (`/`), SDK integration docs (`/docs`), and Studio Dashboard UI (`/dashboard`).

### E. Test Service (`test-service`)
* **Purpose:** Dummy Go web application harness on `:8081` designed to simulate panics (`GET /crash`) and test end-to-end telemetry ingestion.

---

## 5. Security & Edge Case Policies

1. **Path Traversal Protection:** Sanitize all incoming file paths. Reject any path containing `..` or leading slashes outside the project root context.
2. **Zero Storage of Secrets in Engine:** Storage of repository access tokens, GitHub RSA keys, and database passwords is strictly restricted to `apps/manager` and `apps/web`.
3. **Payload Truncation:** Limit request body sizes to 1MB to prevent Denial of Service (DoS) memory exhaustion.
