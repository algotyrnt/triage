# Triage — Project Context & System Architecture

## 1. Executive Summary
**Triage** is a lightweight, zero-log automated crash detection and root-cause triaging engine for Go web applications. When a Go service panics in production, Triage intercepts the stack trace, extracts the specific failing function's Abstract Syntax Tree (AST) node, performs root-cause analysis via Gemini, and automatically opens a detailed issue on the project's GitHub repository.

### Key Value Propositions
* **Zero Log Retention:** Crash logs and source code snippets pass through temporary memory only—never stored in a database.
* **Minimal DB Footprint:** Operates strictly on project metadata (GitHub installation IDs and hashed API keys).
* **AST Isolation:** Sends only the isolated, failing Go function snippet to Gemini rather than entire source files, keeping context tight, fast, and precise.

---

## 2. Monorepo Architecture & Tech Stack

```text
triage/
├── PROJECT_CONTEXT.md       # Master project reference
├── apps/
│   ├── web/                # Dashboard (Next.js 15 App Router, Tailwind CSS, Firebase)
│   └── engine/             # Core Telemetry Engine (Go 1.22+, Cloud Run)
├── sdk/
│   └── go/                 # Lightweight Panic Interceptor Middleware
└── scripts/
    └── test-crash/         # Local dummy web app to trigger panics for testing
```

| Component | Technology | Deployment Target | Cost Model |
| --- | --- | --- | --- |
| **Dashboard (`apps/web`)** | Next.js 15 App Router + Tailwind | Firebase App Hosting | Free Tier (Scales to $0) |
| **Auth** | Firebase Auth (GitHub OAuth) | Firebase | 50k MAUs Free |
| **Database** | Cloud Firestore | GCP | 50k Reads / 20k Writes daily Free |
| **AST Storage** | Google Cloud Storage (GCS) | GCP | 5 GB Free |
| **Telemetry Engine (`apps/engine`)** | Go 1.22+ (`go/ast`, Gemini SDK) | GCP Cloud Run | 2 Million Invocations/mo Free |
| **SDK (`sdk/go`)** | Pure Go Standard Library | Go Package Import | N/A |

---

## 3. High-Level Data Flow

```text
[ Target Go Application ] 
       │ 1. Intercept Panic (sdk/go)
       │    Sends: Stack Trace + API Key
       ▼
[ GCP Cloud Run Engine (apps/engine) ]
       │ 2. Validate API Key (In-Memory LRU Cache)
       │ 3. Fetch pre-parsed AST snippet from GCS (or local file)
       │ 4. Isolate failing *ast.FuncDecl via line number
       │ 5. Query Gemini 2.5 API (Root cause + Suggested Fix)
       │ 6. POST GitHub Issue via GitHub App API
       ▼
[ GitHub Repository ] <--- Issue Created (#104)
[ Memory Garbage Collected ] <--- Zero Logs Stored in DB
```

---

## 4. Subsystem Breakdown

### A. Client SDK (`sdk/go`)
* **Purpose:** Non-intrusive HTTP middleware wrapping standard Go routers (`net/http`, `chi`, `gin`).
* **Behavior:** Captures panics via `defer` + `recover()`, pulls raw trace via `runtime/debug.Stack()`, extracts caller line/file, and dispatches an asynchronous, non-blocking HTTP POST request to the Triage Engine.

### B. Core Telemetry Engine (`apps/engine`)
* **Purpose:** Stateless pipeline that processes incoming crashes and handles LLM analysis.
* **Key Modules:**
  * `internal/ast`: Uses Go standard packages (`go/parser`, `go/token`, `go/ast`) to isolate the exact function declaration (`*ast.FuncDecl`) surrounding the line where the panic occurred.
  * `internal/llm`: Uses `google.golang.org/genai` to analyze the stack trace alongside the isolated AST snippet. Requests structured JSON output containing root cause, severity, and suggested code fix.
  * `internal/github`: Uses GitHub App Installation Tokens to construct and post formatted Markdown issues (`POST /repos/{owner}/{repo}/issues`).

### C. Web Dashboard (`apps/web`)
* **Purpose:** Minimalist developer dashboard for connecting GitHub repositories, managing API keys, and monitoring webhook delivery health.
* **Design Guidelines (Strict):**
  * **Theme:** Light mode canvas (`#F8FAFC`), pure white cards (`#FFFFFF`), crisp 1px light gray borders (`#E2E8F0`).
  * **Palette:** Pitch Black (`#000000`) primary actions, Charcoal (`#111827`) text, Emerald Green (`#059669`) for operational states, Red (`#DC2626`) for panics/danger zone. Zero secondary accent colors.
  * **Typography:** Inter / System Sans-Serif for UI; JetBrains Mono / SF Mono for code, hashes, and line numbers.

---

## 5. Security & Edge Case Policies

1. **Unique Project Indexing:** Firestore documents are keyed by `{github_owner}_{github_repo}`. Duplicate initialization attempts by team members are blocked with a notification that the repository is already configured.
2. **Automated Cleanup:** When a user uninstalls the Triage GitHub App, GitHub dispatches an `installation.deleted` webhook. The engine verifies the `X-Hub-Signature-256` HMAC signature, revokes all associated API keys, and removes stored AST artifacts from Cloud Storage.
3. **In-Memory Caching:** To prevent Firestore pay-per-read billing spikes, valid API keys are cached in memory inside the Cloud Run Go engine using an LRU cache.
