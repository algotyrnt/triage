---
title: Overview & Concepts
description: Zero-overhead Go panic isolation, automated GitHub PRs/issues, and AI incident diagnostics
---

Welcome to **triage**—the zero-overhead Go panic isolation tool, automated GitHub issue & bugfix PR engine, and AI diagnostic platform.

```
Go HTTP Server (your app)
        │  panic → defer/recover
        ▼
 triage SDK Middleware
        │  async, non-blocking POST (<0.02ms overhead)
        ▼
 Triage Server  (:8080) [Zero-Dependency Single Binary / Container]
  ├── Embedded Studio Dashboard (Vite + React 19 SPA)
  ├── Embedded SQLite Database  (Zero-config auto-migration)
  ├── Ingestion & In-Memory Auth (<1ms API key verification)
  ├── Server-Driven OAuth & JWT Session RBAC (Owner/Admin/Dev/Viewer)
  ├── Resolve Multi-File Package AST Context
  │     ├── Crash function *ast.FuncDecl
  │     ├── Receiver struct & type definitions (cross-file)
  │     └── Related constructors (New<Type>) & package helpers
  ├── Multi-Provider AI analysis       → root_cause + suggested_fix + git patch
  ├── Automated GitHub Actions
  │     ├── File GitHub Issue with AST snippets & telemetry
  │     └── Open Bugfix Pull Request with auto-applied git diff
  └── Return JSON response / Stream Live Incidents via SSE
```

---

## Why Triage?

Traditional application monitoring tools and crash loggers capture giant stack traces and attempt to send whole files to large language models. This has several drawbacks:

1. **High Token Costs & Prompt Saturation:** Ingesting whole 2,000-line source files into LLMs wastes thousands of tokens per incident.
2. **Context Dilution or Blindness:** Raw stack traces lack code context, while single-function fragments miss struct definitions and uninitialized constructors in adjacent files.
3. **Synchronous Request Overhead:** Heavy APM agents often add 0.8ms – 1.5ms of latency to each HTTP request.

### The Triage Solution

- **Multi-File Package AST Slicing:** Using Go's standard `go/parser` and `go/ast` packages, Triage isolates the exact crashing function along with referenced struct definitions, constructors, and helper functions across package files.
- **Domain-Aware AI Triage:** Enriches LLM prompts with optional architectural context and business invariants, ensuring diagnoses and generated patches respect your service boundaries.
- **>90% Token Reduction:** Instead of sending entire repositories or 2,000-line source files, Triage delivers a selectively pruned multi-file context to AI models, reducing diagnostic costs to less than $0.0001 per incident.
- **Sub-0.02ms Client Latency:** Telemetry is handed off to an asynchronous 4-worker pool backed by a 1,000-job queue. HTTP responses return immediately with zero blockage.
- **Automated Bugfix Pull Requests:** Creates dedicated fix branches, synthesizes clean fixes via the AI engine, commits the changes, and opens a GitHub Pull Request with 1 click.
- **Automated GitHub Issue Triaging:** Automatically creates GitHub issues with structured root causes, formatted AST code, and drop-in git patches.
- **Multi-Project & Monorepo Support:** Track multiple Go services from a unified workspace with automatic `go.mod` module discovery and a project switcher in the dashboard header.
- **Real-Time Telemetry Streaming (SSE):** Unidirectional Server-Sent Events stream newly ingested crashes and incident state updates directly to connected dashboards with zero page reloads.
- **Zero-Dependency Single-Container Self-Hosting:** Run the entire system with embedded SQLite and embedded React UI in a single lightweight container or standalone binary.

---

## Architecture at a Glance

| Component               | Port     | Technology              | Purpose                                                                         |
| :---------------------- | :------- | :---------------------- | :------------------------------------------------------------------------------ |
| **Triage Server**       | `:8080`  | Go 1.26+ (Embedded UI)  | Telemetry ingestion, AST slicing, Multi-provider AI, and React Studio Dashboard |
| **Go Client SDK**       | Embedded | Go 1.26+                | Non-blocking HTTP middleware with panic recovery                                |
| **Embedded Storage**    | Embedded | SQLite (WAL mode)       | Zero-config embedded persistence for incidents, keys, and settings              |
| **Documentation & Web** | `:4321`  | Astro & Starlight (Bun) | Public landing site and technical reference                                     |

---

## Next Steps

- [5-Minute Quickstart Guide](/docs/quickstart)
- [Go SDK Integration](/docs/sdk)
- [GitHub App & PR Automation](/docs/github-integration)
- [Self-Hosting with Docker](/docs/self-hosting)
