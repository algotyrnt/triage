# triage

[![GitHub Release](https://img.shields.io/github/v/release/algotyrnt/triage?include_prereleases&logo=github&color=6366f1)](https://github.com/algotyrnt/triage/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/algotyrnt/triage/sdk/go.svg)](https://pkg.go.dev/github.com/algotyrnt/triage/sdk/go)
[![Docker Engine](https://img.shields.io/badge/docker-ghcr.io%2Falgotyrnt%2Ftriage--engine-blue?logo=docker)](https://github.com/algotyrnt/triage/pkgs/container/triage-engine)
[![Docker Dashboard](https://img.shields.io/badge/docker-ghcr.io%2Falgotyrnt%2Ftriage--dashboard-blue?logo=docker)](https://github.com/algotyrnt/triage/pkgs/container/triage-dashboard)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Zero-overhead Go panic isolation, automated GitHub triage, and AI-powered incident diagnostics. When a panic occurs in a Go HTTP server, triage intercepts it non-blockingly, isolates the exact crash site along with cross-file package context (receiver struct definitions, referenced types, constructors, and helper functions), runs it through Gemini AI for root-cause analysis, and enables 1-click **automated GitHub issue filing** and **bugfix Pull Request generation** — all without blocking your server's response.

---

```
Go HTTP Server (your app)
        │  panic → defer/recover
        ▼
 triage SDK Middleware
        │  async, non-blocking POST (<0.02ms overhead)
        ▼
 Triage Engine  (:8080)
  ├── Ingestion & In-Memory Auth (<1ms API key verification)
  ├── Engine-Driven OAuth & JWT Session RBAC (Owner/Admin/Dev/Viewer)
  ├── Resolve Multi-File Package AST Context
  │     ├── Crash function *ast.FuncDecl
  │     ├── Receiver struct & type definitions (cross-file)
  │     ├── Related constructors (New<Type>) & package helpers
  │     └── 3-tier cache (In-memory <1.5ms → Postgres pre-index → GitHub on-demand)
  ├── Gemini AI analysis               → root_cause + suggested_fix + git patch
  ├── Persist incident                 (PostgreSQL)
  ├── Automated GitHub Actions
  │     ├── File GitHub Issue with AST snippets & telemetry
  │     └── Open Bugfix Pull Request with auto-applied git diff
  └── Return JSON response
        │
        ▼
 Studio Dashboard  (:3000)            — multi-project switcher, RBAC team & incident inspector
 Public Web / Docs (:4321)            — landing page & integration docs
```

---

## Key Highlights

- **Multi-File Package AST Slicing:** Extracts the exact crashing function alongside cross-file receiver structs, referenced types, constructors, and package helpers using Go's standard `go/parser` and `go/ast`. Eliminates >90% of token overhead while providing 100% semantic clarity.
- **Sub-0.02ms Client Latency:** Bounded 4-goroutine worker pool with a 1,000-job buffer asynchronously dispatches telemetry without blocking user HTTP requests.
- **Engine-Driven OAuth & RBAC:** Zero frontend secret exposure. The Go engine performs GitHub OAuth code exchanges, manages user identity, issues signed 30-day HS256 JWTs, and enforces permissions across `Owner`, `Admin`, `Developer`, and `Viewer` tiers.
- **Automated Bugfix Pull Requests:** 1-click Pull Request generation directly from the Studio Dashboard. The engine creates a dedicated fix branch, applies the patch via Gemini AI, commits the changes, and opens a linked PR on GitHub.
- **Automated GitHub Issue Filing:** Automatically creates GitHub issues with formatted AST code blocks, raw stack traces, Gemini diagnostic summaries, and triage labels.
- **Multi-Project & Monorepo Support:** Track multiple Go repositories and monorepos from a single dashboard with an instant project switcher. Includes automatic Go submodule detection (`go.mod` discovery) and path normalization.
- **Direct Gemini AI Endpoints:** Dedicated REST APIs for on-demand crash analysis (`/api/v1/gemini/analyze-panic`) and unified diff patch generation (`/api/v1/gemini/generate-patch`).
- **Dynamic Origin-Restricted CORS:** Protects proprietary AST context and stack traces by locking down browser access strictly to your configured dashboard domain.
- **Single-Container Self-Hosting:** Deploy the engine and dashboard effortlessly with Docker Compose or pre-built GHCR images.

---

## Quickstart

### 1. Start Triage Stack

```bash
git clone https://github.com/algotyrnt/triage.git
cd triage
docker compose up --build -d
```

Open [http://localhost:3000](http://localhost:3000) to complete the 5-step Setup Wizard (GitHub App, OAuth, Gemini AI API key).

### 2. Add SDK Middleware to Your Go App

```bash
go get github.com/algotyrnt/triage/sdk/go@latest
```

```go
package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/data", handleData)

	// Wrap your standard http.Handler with Triage panic middleware:
	handler := triage.Middleware(
		"tr_live_your_api_key",
		"http://localhost:8080/api/v1/telemetry",
	)(mux)

	http.ListenAndServe(":8080", handler)
}
```

> **Tip:** When running or building your Go service, use `-trimpath` to generate clean relative stack traces that match your repository structure on GitHub.

---

## Documentation

Full technical documentation, architectural deep dives, and API specifications are available at [**triage.algotyrnt.com**](https://triage.algotyrnt.com):

| Guide                                                                                  | Description                                                        |
| :------------------------------------------------------------------------------------- | :----------------------------------------------------------------- |
| [**Overview & Concepts**](https://triage.algotyrnt.com/docs/overview)                  | Platform architecture and core design principles                   |
| [**5-Minute Quickstart**](https://triage.algotyrnt.com/docs/quickstart)                | End-to-end setup and first panic crash simulation                  |
| [**Go SDK Integration**](https://triage.algotyrnt.com/docs/sdk)                        | Zero-config middleware setup and trace propagation                 |
| [**Monorepos & Multi-Module**](https://triage.algotyrnt.com/docs/monorepo-support)     | Managing nested `go.mod` submodules and path normalization         |
| [**AST Engine & Node Slicing**](https://triage.algotyrnt.com/docs/ast-engine)          | AST parsing, multi-file symbol resolution, and 3-tier caching      |
| [**Gemini AI Diagnostics**](https://triage.algotyrnt.com/docs/gemini-ai)               | Root-cause analysis, prompt design, and patch synthesis            |
| [**GitHub App & PR Automation**](https://triage.algotyrnt.com/docs/github-integration) | 1-click GitHub App creation, issue filing, and automated PRs       |
| [**Authentication & Team RBAC**](https://triage.algotyrnt.com/docs/team-and-rbac)      | Engine-driven OAuth, JWT sessions, and member onboarding           |
| [**Self-Hosting Guide**](https://triage.algotyrnt.com/docs/self-hosting)               | Docker Compose, GHCR images, and database migrations               |
| [**Environment & Configuration**](https://triage.algotyrnt.com/docs/configuration)     | Engine, Dashboard, and SDK environment reference                   |
| [**Development & Releases**](https://triage.algotyrnt.com/docs/development)            | Makefile targets, test suites, and SemVer automation               |
| [**Engine REST API Reference**](https://triage.algotyrnt.com/docs/api-reference)       | Complete HTTP REST endpoints and telemetry protocols               |
| [**Troubleshooting & FAQ**](https://triage.algotyrnt.com/docs/troubleshooting)         | Solutions for `-trimpath`, CORS, reverse proxies, and missing ASTs |

---

## Development Toolkit

```bash
# View all available targets
make help

# Run full pre-flight verification gate (lint + test + build)
make check

# Start local services with hot-reloading
make dev-engine       # Engine on :8080
make dev-dashboard    # Dashboard on :3000
make dev-web          # Docs & Landing on :4321

# Docker stack management
make up               # Start Docker Compose cluster
make logs             # Tail container logs
make down             # Stop cluster
```

---

## Repository Structure

```
.
├── db/schema.sql             # PostgreSQL DDL — schema, RBAC tables & performance indexes
├── apps/
│   ├── engine/               # Go 1.26+ core engine (AST slicing, Gemini AI, OAuth & REST APIs)
│   ├── dashboard/            # Next.js 16 Studio Dashboard (Bun runtime / Pure static SPA)
│   └── web/                  # Astro & Starlight public site and documentation (Bun runtime)
├── sdk/go/                   # Official Go SDK (panic recovery middleware & async dispatch, Go 1.26+)
└── test-services/            # Simulation microservices for local validation (Go 1.26+)
```

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Created by [Punjitha Bandara](https://algotyrnt.com).
