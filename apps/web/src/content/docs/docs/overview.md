---
title: Overview & Concepts
description: Zero-overhead Go panic isolation and Gemini AI incident diagnostics
---

Welcome to **triage**—the zero-overhead Go panic isolation tool, automated GitHub issue triaging engine, and AI diagnostic platform.

```
Go HTTP Server (your app)
        │  panic → defer/recover
        ▼
 triage SDK Middleware
        │  async, non-blocking POST (<0.02ms overhead)
        ▼
 Triage Engine  (:8080)
  ├── Verify API key  (PostgreSQL)
  ├── Resolve AST snippet
  │     1. In-memory KV cache          → hit: <1.5ms
  │     2. PostgreSQL ast_nodes table  → pre-indexed lookup: <5ms
  │     3. GitHub Contents API         → on-demand fetch + parse: <25ms
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

## Why Triage?

Traditional application monitoring tools and crash loggers capture giant stack traces and attempt to send whole files to large language models. This has several drawbacks:

1. **High Token Costs & Prompt Saturation:** Ingesting whole 2,000-line source files into LLMs wastes thousands of tokens per incident.
2. **Synchronous Request Overhead:** Heavy APM agents often add 0.8ms – 1.5ms of latency to each HTTP request.
3. **Complex Pre-indexing:** Many symbolication services require background database pre-indexing pipelines that break during fast-moving CI/CD deploys.

### The Triage Solution

- **On-Demand Synchronous AST Slicing:** Zero pre-indexing required. Using Go's standard `go/parser` and `go/ast` packages, Triage extracts only the enclosing `*ast.FuncDecl` node (the function containing the panicking line).
- **94% Token Reduction:** Instead of sending whole files, Triage sends only 8–25 relevant lines of AST code to Gemini AI, reducing AI diagnostic costs to less than $0.0001 per incident.
- **Sub-0.02ms Client Latency:** Telemetry is handed off to an asynchronous 4-worker pool backed by a 1,000-job queue. HTTP responses return immediately with zero blockage.
- **Automated GitHub Issue Triaging:** Automatically creates GitHub issues with structured root causes, formatted AST code, and drop-in git patches.
- **Single-Container Self-Hosting:** Run the engine and studio dashboard in 1 single Docker container (`triage/engine:latest`).

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
- [Self-Hosting with Docker](/docs/self-hosting)
