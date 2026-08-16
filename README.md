# triage

[![GitHub Release](https://img.shields.io/github/v/release/algotyrnt/triage?include_prereleases&logo=github&color=6366f1)](https://github.com/algotyrnt/triage/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/algotyrnt/triage/sdk/go.svg)](https://pkg.go.dev/github.com/algotyrnt/triage/sdk/go)
[![Docker Engine](https://img.shields.io/badge/docker-ghcr.io%2Falgotyrnt%2Ftriage--engine-blue?logo=docker)](https://github.com/algotyrnt/triage/pkgs/container/triage-engine)
[![Docker Dashboard](https://img.shields.io/badge/docker-ghcr.io%2Falgotyrnt%2Ftriage--dashboard-blue?logo=docker)](https://github.com/algotyrnt/triage/pkgs/container/triage-dashboard)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> **Work in Progress (Pre-v1.0)** — interfaces, telemetry protocols, and SDK signatures are in active development. Releases follow SemVer starting at `v0.1.0`.

Zero-overhead Go panic isolation and AI-powered incident diagnostics. When a panic occurs in a Go HTTP server, triage intercepts it non-blockingly, isolates the exact `*ast.FuncDecl` surrounding the crash site, runs it through Gemini AI for root-cause analysis, and automatically files a GitHub issue — all without blocking your server's response.

---

## Releases & Installation

### Go SDK

Install the latest release via `go get`:

```bash
go get github.com/algotyrnt/triage/sdk/go@latest
```

Or pin a specific version:

```bash
go get github.com/algotyrnt/triage/sdk/go@v0.1.0
```

### Pre-Built Containers (GHCR)

Multi-architecture images (`linux/amd64`, `linux/arm64`) are published automatically on every release:

- **Engine:** `docker pull ghcr.io/algotyrnt/triage-engine:latest` (or `:v0.1.0`)
- **Dashboard:** `docker pull ghcr.io/algotyrnt/triage-dashboard:latest` (or `:v0.1.0`)

---

```
Go HTTP Server (your app)
        │  panic → defer/recover
        ▼
 triage SDK Middleware
        │  async, non-blocking POST
        ▼
 Triage Engine  (:8080)
  ├── Verify API key  (PostgreSQL)
  ├── Resolve AST snippet
  │     1. In-memory KV cache          → hit: <2ms
  │     2. PostgreSQL ast_nodes table  → pre-indexed lookup
  │     3. GitHub Contents API         → on-demand fetch + parse
  │     └── Local workspace fallback
  ├── Gemini AI analysis               → root_cause + suggested_fix
  ├── Persist incident                 (PostgreSQL)
  └── Return JSON response
        │
        ▼
 Studio Dashboard  (:3000)            — real-time incident viewer
 Public Web / Docs (:4321)            — landing page & integration docs
```

---

## SDK

### Install

```bash
go get github.com/algotyrnt/triage/sdk/go
```

### Usage

Wrap any standard `http.Handler` or `http.ServeMux`:

```go
package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/data", handleData)

	engineURL := "https://triage.yourcompany.com/api/v1/telemetry"
	handler := triage.Middleware("tr_live_your_api_key", engineURL)(mux)

	http.ListenAndServe(":8080", handler)
}
```

### Zero Configuration by Default

Because the Triage engine automatically resolves your repository, monorepo root directory, and Git commit hash directly from your API key and binary build metadata, the 2-argument signature is all you need:

```go
handler := triage.Middleware(
	"tr_live_your_api_key",
	"https://triage.yourcompany.com/api/v1/telemetry",
)(mux)
```

The Triage engine automatically resolves your repository, monorepo root directory, and permissions from the API key, while the SDK automatically detects the Git commit SHA via `debug.ReadBuildInfo()` and propagates OpenTelemetry W3C trace headers (`traceparent`, `X-Triage-Trace-ID`).

On panic, the middleware:

1. Recovers from the panic using `defer + recover()`
2. Captures the full stack trace via `debug.Stack()`
3. Parses the top application frame (skipping runtime/stdlib/middleware frames)
4. Generates an OpenTelemetry-compatible trace ID (`X-Triage-Trace-ID` / `traceparent`)
5. Dispatches an **async, non-blocking** POST to the engine via a bounded worker pool (4 workers, 1000-job queue)
6. Returns a generic `500 Internal Server Error` to the caller — no internal detail is leaked

---

## Self-Hosting

### Prerequisites

- Docker & Docker Compose
- A Google AI Studio API key ([aistudio.google.com](https://aistudio.google.com))
- A GitHub account (for GitHub App creation and OAuth)

### 1. Start the stack

#### Option A: Production with Pre-Built Images (Recommended)

```bash
# Set release version to latest
export TRIAGE_VERSION=latest

mkdir -p db
curl -sSL "https://raw.githubusercontent.com/algotyrnt/triage/main/db/schema.sql" -o db/schema.sql
curl -sSL "https://raw.githubusercontent.com/algotyrnt/triage/main/docker-compose.prod.yml" -o docker-compose.prod.yml

# Start the stack (POSTGRES_PASSWORD is required for production security)
POSTGRES_PASSWORD=your_secure_password TRIAGE_VERSION=latest docker compose -f docker-compose.prod.yml up -d
```

#### Option B: Local Development from Source

```bash
git clone https://github.com/algotyrnt/triage.git
cd triage
docker compose up --build -d
```

This starts three containers:

| Container          | Port   | Description                                              |
| ------------------ | ------ | -------------------------------------------------------- |
| `triage-db`        | `5432` | PostgreSQL 16 — schema auto-applied from `db/schema.sql` |
| `triage-engine`    | `8080` | Go engine — all API, setup, and auth routes              |
| `triage-dashboard` | `3000` | Next.js Studio Dashboard                                 |

### 2. Run the setup wizard

Open **http://localhost:3000** and follow the 5-step wizard:

1. **GitHub App** — create and install a GitHub App for repository access and automated issue filing
2. **Installation** — install the app on your GitHub org or personal account
3. **OAuth** — link GitHub OAuth for secure dashboard logins
4. **Gemini AI** — enter your Google AI Studio API key and model name (e.g. `gemini-2.5-flash`)
5. **Verify** — confirm the full pipeline is operational

The engine auto-generates and persists a cryptographic session secret on first boot — no manual configuration required.

### 3. Create a project and get an API key

In the dashboard, add your repository to get an API key (`tr_live_...`). Use that key in the SDK.

### 4. Test the pipeline locally

```bash
cd test-service

# Copy and fill in your API key and engine URL
cp .env.example .env.local

go run main.go
# → Starting test-service on :8081 ...

# In another terminal — trigger a nil pointer dereference panic:
curl http://localhost:8081/crash
```

Open the dashboard at **http://localhost:3000** to see the isolated AST snippet and AI root-cause analysis.

---

## Environment Reference

> **When using Docker Compose, you don't need to set any of these.** The compose file already wires all inter-service configuration. These variables are only relevant when running services natively (e.g. during engine development with `go run`).

### Engine Configuration (Optional Overrides)

| Variable              | Default                 | Description                                              |
| --------------------- | ----------------------- | -------------------------------------------------------- |
| `DATABASE_URL`        | —                       | PostgreSQL connection string (**required**)              |
| `PORT`                | `8080`                  | Engine listen port                                       |
| `TRIAGE_API_KEY`      | —                       | Fallback static API key (used when DB is unreachable)    |
| `GEMINI_API_KEY`      | —                       | Overrides the Gemini key stored in DB                    |
| `GEMINI_MODEL_NAME`   | —                       | Overrides the model name stored in DB                    |
| `AST_WORKSPACE_ROOT`  | `.`                     | Local path used as fallback for on-demand AST resolution |
| `NEXT_PUBLIC_APP_URL` | `http://localhost:3000` | Dashboard origin (used for OAuth redirect URLs)          |

### Test Service (`test-service/.env.example`)

| Variable            | Default            | Description                           |
| ------------------- | ------------------ | ------------------------------------- |
| `TRIAGE_API_KEY`    | `tr_test_key_9042` | API key sent with telemetry payloads  |
| `TRIAGE_ENGINE_URL` | SDK default        | Engine URL override for local testing |
| `PORT`              | `8081`             | Test service listen port              |

---

## Repository Structure

```
.
├── db/
│   └── schema.sql            # PostgreSQL DDL — all tables, indexes & constraints
├── apps/
│   ├── engine/               # Go 1.26+ core engine
│   │   ├── internal/
│   │   │   ├── ast/          # On-demand AST fetcher, in-memory KV cache, PostgreSQL node manager
│   │   │   ├── db/           # PostgreSQL pool, incident store, API key verification
│   │   │   ├── github/       # GitHub App JWT auth, installation tokens, Contents API & issue filing
│   │   │   ├── llm/          # Gemini AI SDK integration
│   │   │   └── session/      # JWT session minting & validation
│   │   ├── main.go           # HTTP server (:8080) — all API, setup & auth routes
│   │   └── Dockerfile
│   ├── dashboard/            # Next.js 16 Studio Dashboard (:3000)
│   │   ├── src/
│   │   └── Dockerfile
│   └── web/                  # Astro public landing page & Starlight docs (:4321)
│       └── src/
├── sdk/
│   └── go/               # Go client SDK (panic middleware & async telemetry dispatch)
│       ├── middleware.go
│       └── middleware_test.go
├── test-service/             # Local panic simulation harness (:8081)
│   └── main.go
└── docker-compose.yml
```

---

## Engine API

All routes are served by the engine on `:8080`.

| Method     | Route                            | Description                                   |
| ---------- | -------------------------------- | --------------------------------------------- |
| `GET`      | `/health`                        | Health check                                  |
| `POST`     | `/api/v1/telemetry`              | Receive SDK panic telemetry                   |
| `POST`     | `/api/v1/ast/index`              | Pre-index a repository's AST into PostgreSQL  |
| `GET`      | `/api/v1/incidents`              | List recent incidents                         |
| `GET/POST` | `/api/v1/projects`               | List / create tracked repositories            |
| `GET`      | `/api/v1/stats`                  | Engine stats                                  |
| `GET`      | `/api/v1/setup/status`           | Setup wizard completion status                |
| `POST`     | `/api/v1/setup/manifest`         | Generate GitHub App manifest                  |
| `GET`      | `/api/v1/setup/callback`         | GitHub App manifest conversion callback       |
| `GET`      | `/api/v1/setup/install`          | Get GitHub App installation URL               |
| `GET`      | `/api/v1/setup/install/callback` | GitHub App installation callback              |
| `POST`     | `/api/v1/setup/oauth`            | Save GitHub OAuth credentials                 |
| `POST`     | `/api/v1/setup/llm`              | Save Gemini AI configuration                  |
| `POST`     | `/api/v1/setup/test`             | Verify GitHub App connectivity                |
| `GET`      | `/api/v1/setup/repos`            | List installed repositories                   |
| `GET/POST` | `/api/v1/settings/llm`           | View / update Gemini settings (authenticated) |
| `GET`      | `/api/v1/auth/github`            | Initiate GitHub OAuth flow                    |
| `GET`      | `/api/v1/auth/github/callback`   | GitHub OAuth callback                         |
| `GET`      | `/api/v1/auth/me`                | Get current session user                      |

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Created by [Punjitha Bandara](https://algotyrnt.com).
