# Triage: Go Crash Isolation & AI Diagnostic Engine

> **[IMPORTANT] Work In Progress (WIP)**
> Triage is currently in active early development. Interfaces, telemetry protocols, and SDK signatures may evolve as features are added.

**Triage** is a zero-overhead Go crash isolation tool, automated GitHub issue triaging engine, and AI-powered diagnostic platform. When a panic occurs in a Go web application, Triage intercepts the crash non-blockingly, uses Go standard parser to isolate **ONLY** the surrounding function's AST node (`*ast.FuncDecl`), queries Google Gemini for instant root-cause analysis, and automatically posts a detailed issue to your GitHub repository.

---

## Architecture & System Data Flow

```text
                     ┌──────────────────────────────────────────────┐
                     │          Your Go Application / Service       │
                     └──────────────────────┬───────────────────────┘
                                            │
                                  Panic Intercepted
                              triage.Middleware (defer/recover)
                                            │
                                Async Non-Blocking POST
                                            ▼
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 Triage Engine (:8080)                                      │
│                                                                                            │
│  1. Parse Stack Trace  ──►  2. Extract Func AST  ──►  3. Gemini 3.5 Flash  ──►  4. GitHub App│
│     (Top App Frame)            (internal/ast)            (genai SDK)            Issue Post │
└───────────────────────────────────────────┬────────────────────────────────────────────────┘
                                            │
                                    JSON Telemetry Stream
                                            ▼
                     ┌──────────────────────────────────────────────┐
                     │         Triage Studio Dashboard (/dashboard) │
                     │   - AI Root Cause Analysis                   │
                     │   - Isolated Code AST Viewer                 │
                     │   - Suggested Code Fix Diff                  │
                     └──────────────────────────────────────────────┘
```

### 1. Client SDK (`sdk/go`)

The SDK provides a lightweight HTTP middleware (`triage.Middleware`) wrapping standard `http.Handler` routes. On panic:

- Intercepts crash using `defer + recover()`.
- Captures stack trace using `debug.Stack()`.
- Slices top application stack frame line and file path (ignoring standard library & middleware frames).
- Dispatches an **asynchronous, non-blocking** HTTP POST request to the Triage Engine.

### 2. Core Go Engine (`apps/engine`)

- **AST Node Isolation (`internal/ast`)**: Reads the local `.go` file and uses `go/parser`, `go/token`, `go/ast`, and `go/printer` to extract **ONLY** the `*ast.FuncDecl` enclosing the target crash line.
- **AI Diagnostics (`internal/llm`)**: Uses the official Google Gemini SDK (`google.golang.org/genai`) to send the stack trace and isolated AST node to Gemini 3.5 Flash, generating structured JSON with `root_cause` and `suggested_fix`.
- **GitHub App Subsystem (`internal/github`)**: Uses pure Go standard library RSA/JWT signing (`RS256`) to exchange installation tokens (`POST /app/installations/{id}/access_tokens`), create formatted Markdown issues (`POST /repos/{owner}/{repo}/issues`), and verify `X-Hub-Signature-256` HMAC webhook signatures.

### 3. Studio Web App (`apps/web`)

- Built with **Next.js 16 App Router** and **Bun**.
- **`/`**: Product landing page and SDK integration guide.
- **`/dashboard`**: Real-time crash isolation dashboard displaying live panic feeds, AST code viewers, and AI fix diffs.
- **`engineClient` Integration**: Client API service module ([`apps/web/src/services/engineClient.ts`](file:///Users/punjitha/projects/triage/apps/web/src/services/engineClient.ts)) and Next.js API proxy route ([`apps/web/src/app/api/telemetry/route.ts`](file:///Users/punjitha/projects/triage/apps/web/src/app/api/telemetry/route.ts)) connecting the dashboard directly to the Go Engine.

---

## Repository Structure

```text
.
├── PROJECT_CONTEXT.md    # Master architecture & security reference specification
├── apps/
│   ├── engine/           # Go 1.26+ Core Engine (AST parser, Gemini SDK & GitHub App)
│   │   ├── internal/ast/ # go/ast function node extractor
│   │   ├── internal/github/ # GitHub App JWT auth, issue poster & webhook handler
│   │   ├── internal/llm/ # Gemini 3.5 Flash SDK integration
│   │   └── main.go       # HTTP Telemetry server listening on :8080
│   └── web/              # Next.js 16 + Bun Web App (Landing page & Dashboard)
│       └── src/
│           ├── app/
│           │   ├── page.tsx            # Landing Page (domain.com)
│           │   ├── dashboard/page.tsx  # Studio Dashboard (domain.com/dashboard)
│           │   └── api/telemetry/      # Next.js API proxy route for engine
│           └── services/engineClient.ts # Engine client API service
├── sdk/
│   └── go/               # Go Client SDK (panic middleware & telemetry dispatcher)
└── scripts/
    └── test-crash/       # Local test harness triggering panic simulations (:8081)
```

---

## Quickstart

### Prerequisites

- **Go**: 1.26 or higher
- **Bun**: 1.0 or higher
- **Gemini API Key**: Set `GEMINI_API_KEY` in `apps/engine/.env.local`

### 1. Start the Triage Engine

Configure `apps/engine/.env.local`:

```env
PORT=8080
TRIAGE_API_KEY=tr_test_key_9042
GEMINI_API_KEY=your_gemini_3.5_flash_key
```

Run the engine server:

```bash
cd apps/engine
go run main.go
# Listening on http://localhost:8080
```

### 2. Start the Web Dashboard

Configure `apps/web/.env.local`:

```env
PORT=3000
NEXT_PUBLIC_ENGINE_URL=http://localhost:8080/api/v1/telemetry
```

Run the web dashboard:

```bash
cd apps/web
bun install
bun dev
# Dashboard running on http://localhost:3000
```

### 3. Run Test Crash Simulation

```bash
cd scripts/test-crash
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

	// Wrap handler with Triage Middleware
	telemetryURL := "http://localhost:8080/api/v1/telemetry"
	wrappedHandler := triage.Middleware("tr_api_key_123", telemetryURL)(mux)

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
