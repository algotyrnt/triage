---
title: Overview & Concepts
description: Zero-overhead Go panic isolation, automated GitHub PRs/issues, and Gemini AI incident diagnostics
---

Welcome to **triage**—the zero-overhead Go panic isolation tool, automated GitHub issue & bugfix PR engine, and AI diagnostic platform.

```
Go HTTP Server (your app)
        │  panic → defer/recover
        ▼
 triage SDK Middleware
        │  async, non-blocking POST (<0.02ms overhead)
        ▼
 Triage Engine  (:8080)
  ├── Verify API key  (PostgreSQL)
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
 Studio Dashboard  (:3000)            — multi-project switcher & incident inspector
 Public Web / Docs (:4321)            — landing page & integration docs
```

---

## Why Triage?

Traditional application monitoring tools and crash loggers capture giant stack traces and attempt to send whole files to large language models. This has several drawbacks:

1. **High Token Costs & Prompt Saturation:** Ingesting whole 2,000-line source files into LLMs wastes thousands of tokens per incident.
2. **Context Dilution or Blindness:** Raw stack traces lack code context, while single-function fragments miss struct definitions and uninitialized constructors in adjacent files.
3. **Synchronous Request Overhead:** Heavy APM agents often add 0.8ms – 1.5ms of latency to each HTTP request.

### The Triage Solution

- **Multi-File Package AST Slicing:** Using Go's standard `go/parser` and `go/ast` packages, Triage isolates the exact crashing function along with referenced struct definitions, constructors, and helper functions across package files.
- **>90% Token Reduction:** Instead of sending entire repositories or 2,000-line source files, Triage delivers a selectively pruned multi-file context to Gemini AI, reducing AI diagnostic costs to less than $0.0001 per incident.
- **Sub-0.02ms Client Latency:** Telemetry is handed off to an asynchronous 4-worker pool backed by a 1,000-job queue. HTTP responses return immediately with zero blockage.
- **Automated Bugfix Pull Requests:** Creates dedicated fix branches, synthesizes clean fixes via Gemini AI, commits the changes, and opens a GitHub Pull Request with 1 click.
- **Automated GitHub Issue Triaging:** Automatically creates GitHub issues with structured root causes, formatted AST code, and drop-in git patches.
- **Multi-Project & Monorepo Support:** Track multiple Go services from a unified workspace with automatic `go.mod` module discovery and a project switcher in the dashboard header.
- **Single-Container Self-Hosting:** Run the engine and studio dashboard effortlessly using Docker Compose or pre-built GHCR containers.

---

## Architecture at a Glance

| Component               | Port     | Technology        | Purpose                                                       |
| :---------------------- | :------- | :---------------- | :------------------------------------------------------------ |
| **Go Client SDK**       | Embedded | Go 1.22+          | Non-blocking HTTP middleware with panic recovery              |
| **Triage Engine**       | `:8080`  | Go 1.26+          | Telemetry ingestion, AST slicing, Gemini AI client, REST APIs |
| **Studio Dashboard**    | `:3000`  | Next.js 16        | Real-time incident inspector, AST explorer, setup wizard      |
| **Documentation & Web** | `:4321`  | Astro & Starlight | Public landing site and technical reference                   |
| **PostgreSQL**          | `:5432`  | Postgres 16       | Persistent storage for incidents, API keys, and cache         |

---

## Next Steps

- [5-Minute Quickstart Guide](/docs/quickstart)
- [Go SDK Integration](/docs/sdk)
- [GitHub App & PR Automation](/docs/github-integration)
- [Self-Hosting with Docker](/docs/self-hosting)
