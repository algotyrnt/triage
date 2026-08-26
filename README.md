# triage

[![GitHub Release](https://img.shields.io/github/v/release/algotyrnt/triage?include_prereleases&logo=github&color=6366f1)](https://github.com/algotyrnt/triage/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/algotyrnt/triage/sdk/go.svg)](https://pkg.go.dev/github.com/algotyrnt/triage/sdk/go)
[![Docker Engine](https://img.shields.io/badge/docker-ghcr.io%2Falgotyrnt%2Ftriage--engine-blue?logo=docker)](https://github.com/algotyrnt/triage/pkgs/container/triage-engine)
[![Docker Dashboard](https://img.shields.io/badge/docker-ghcr.io%2Falgotyrnt%2Ftriage--dashboard-blue?logo=docker)](https://github.com/algotyrnt/triage/pkgs/container/triage-dashboard)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Zero-overhead Go panic isolation, automated GitHub triage, and AI-powered incident diagnostics. When a panic occurs in a Go HTTP server, triage intercepts it non-blockingly, isolates the exact crash site along with cross-file package context (receiver struct definitions, referenced types, constructors, and helper functions), runs it through the pluggable AI engine (Gemini, OpenAI, Claude, Ollama) for root-cause analysis, and enables 1-click **automated GitHub issue filing** and **bugfix Pull Request generation** — all without blocking your server's response.

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
  ├── Multi-Provider AI analysis       → root_cause + suggested_fix + git patch
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
- **Pluggable Multi-Provider AI Engine:** Bring your preferred LLM provider—**Google Gemini**, **OpenAI** (GPT-4o/o3-mini), **Anthropic Claude** (Claude 3.5/3.7), or **100% Air-Gapped Local Models** (Ollama, vLLM, DeepSeek-Coder, Qwen2.5) with live connection testing and latency benchmarking.
- **Crash Fingerprinting & Frequency Deduplication:** Computes deterministic SHA-256 crash fingerprints from `file:line:panic` to aggregate repeating crashes into canonical incidents with live `occurrence_count` frequency badges and `last_seen_at` timestamps, preventing duplicate GitHub issues.
- **Zero-Config Database Auto-Migration:** Automatically creates and validates tables, columns, and performance indexes on startup using PostgreSQL transaction advisory locks (`pg_advisory_xact_lock`), enabling instant deployment on any Postgres instance without manual SQL execution.
- **Domain-Aware AI Triage:** Attach optional architectural context, domain rules, and ledger invariants during onboarding or project settings. The AI model automatically incorporates this context to ensure root-cause analyses and patch suggestions adhere to your business boundaries.
- **Sub-0.02ms Client Latency:** Bounded 4-goroutine worker pool with a 1,000-job buffer asynchronously dispatches telemetry without blocking user HTTP requests.
- **Engine-Driven OAuth & RBAC:** Zero frontend secret exposure. The Go engine performs GitHub OAuth code exchanges, manages user identity, issues signed 30-day HS256 JWTs, and enforces permissions across `Owner`, `Admin`, `Developer`, and `Viewer` tiers.
- **Automated Bugfix Pull Requests:** 1-click Pull Request generation directly from the Studio Dashboard. The engine creates a dedicated fix branch, applies the patch via the active LLM, commits the changes, and opens a linked PR on GitHub.
- **Automated GitHub Issue Filing:** Automatically creates GitHub issues with formatted AST code blocks, raw stack traces, AI diagnostic summaries, and triage labels.
- **Multi-Project & Monorepo Support:** Track multiple Go repositories and monorepos from a single dashboard with an instant project switcher. Includes automatic Go submodule detection (`go.mod` discovery) and path normalization.
- **Real-Time Live Telemetry (SSE):** Unidirectional Server-Sent Events stream freshly intercepted panics and state changes directly into the Studio Dashboard with automatic keep-alive heartbeats and zero page reloads.
- **Direct AI REST Endpoints:** Dedicated REST APIs for on-demand crash analysis (`/api/v1/llm/analyze-panic`), unified diff patch generation (`/api/v1/llm/generate-patch`), and live connection testing (`/api/v1/settings/llm/test`).
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

Open [http://localhost:3000](http://localhost:3000) to complete the 5-step Setup Wizard (GitHub App, OAuth, AI model configuration).

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
		"your_sample_api_key",
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
| [**Pluggable AI Diagnostics**](https://triage.algotyrnt.com/docs/ai-diagnostics)       | Root-cause analysis, prompt design, and patch synthesis            |
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

## Making a Release

Triage follows [Semantic Versioning](https://semver.org) (`vMAJOR.MINOR.PATCH`) with automated pre-flight quality checks, Go submodule dual-tagging, and continuous delivery via GitHub Actions.

### 1. Pre-flight & Dry Run (Optional)

Preview the release workflow, build artifacts, and simulated tag creation without making any git changes:

```bash
make release-dry-run VERSION=v0.2.0
```

### 2. Cut and Publish a Release

Releases must be initiated from the `main` branch with a clean git working tree. Use automated SemVer bump targets or specify an explicit version:

```bash
# Automated SemVer bumps:
make release-patch   # e.g. v0.1.0 -> v0.1.1
make release-minor   # e.g. v0.1.0 -> v0.2.0
make release-major   # e.g. v0.1.0 -> v1.0.0

# Or specify an explicit version:
make release VERSION=v0.2.0
```

### 3. Automated Release Lifecycle

When a release is triggered, the pipeline performs the following steps:

1. **Local Pre-flight Quality Gate:** Runs `make check` (format validation, `go vet`, engine/SDK test suites, and component builds).
2. **Dual Git Tagging:** Creates root tag `vX.Y.Z` and submodule tag `sdk/go/vX.Y.Z` (required for Go module proxy resolution).
3. **Push to Remote:** Pushes tags to GitHub to trigger the release workflow.
4. **CI/CD Pipeline (`.github/workflows/release.yml`):**
   - **Go SDK:** Publishes `sdk/go/vX.Y.Z` and warms the Go module proxy cache (`pkg.go.dev`).
   - **Docker Images:** Builds and publishes multi-arch (`linux/amd64`, `linux/arm64`) images to GitHub Container Registry:
     - `ghcr.io/algotyrnt/triage-engine:vX.Y.Z` (and `:latest`)
     - `ghcr.io/algotyrnt/triage-dashboard:vX.Y.Z` (and `:latest`)
   - **Web / Documentation:** Builds and deploys the documentation site live to Cloudflare Pages.
   - **GitHub Release:** Publishes the official immutable GitHub release with auto-generated release notes and attached production configuration (`docker-compose.prod.yml`).

---

## Repository Structure

```
.
├── apps/
│   ├── engine/               # Go 1.26+ core engine (AST slicing, Multi-Provider AI, OAuth & REST APIs)
│   ├── dashboard/            # Next.js 16 Studio Dashboard (Bun runtime / Pure static SPA)
│   └── web/                  # Astro & Starlight public site and documentation (Bun runtime)
├── sdk/go/                   # Official Go SDK (panic recovery middleware & async dispatch, Go 1.26+)
└── test-services/            # Simulation microservices for local validation (Go 1.26+)
```
