---
title: Environment & Configuration Reference
description: Complete environment variable and configuration reference for Engine, SDK, and Dashboard
---

This page documents all configuration options and environment variables across the Triage ecosystem.

## Triage Engine Configuration

The engine (`apps/engine`) supports the following environment variables:

| Variable              | Type     | Default                 | Description                                                          |
| :-------------------- | :------- | :---------------------- | :------------------------------------------------------------------- |
| `DATABASE_URL`        | `string` | —                       | PostgreSQL connection string (**required**).                         |
| `PORT`                | `string` | `8080`                  | Engine HTTP listen port.                                             |
| `TRIAGE_API_KEY`      | `string` | —                       | Fallback static API key (used if PostgreSQL is temporarily offline). |
| `GEMINI_API_KEY`      | `string` | —                       | Google AI Studio API key (overrides database setting).               |
| `GEMINI_MODEL_NAME`   | `string` | —                       | Configured Gemini model name for diagnostic inference.               |
| `AST_WORKSPACE_ROOT`  | `string` | `.`                     | Local fallback filesystem path for offline AST resolution.           |
| `NEXT_PUBLIC_APP_URL` | `string` | `http://localhost:3000` | Dashboard origin for OAuth redirect URLs.                            |
| `LOG_LEVEL`           | `string` | `info`                  | Engine logging verbosity (`debug`, `info`, `warn`, `error`).         |

---

## Studio Dashboard Configuration

The Next.js dashboard (`apps/dashboard`) supports:

| Variable                 | Type     | Default                                  | Description                                      |
| :----------------------- | :------- | :--------------------------------------- | :----------------------------------------------- |
| `PORT`                   | `string` | `3000`                                   | Dashboard HTTP listen port.                      |
| `NEXT_PUBLIC_ENGINE_URL` | `string` | `http://localhost:8080/api/v1/telemetry` | Gateway URL displayed in project setup snippets. |
| `NEXT_PUBLIC_API_URL`    | `string` | `http://localhost:8080`                  | Internal engine proxy URL.                       |

---

## Go SDK Options Reference

The Go SDK (`sdk/go`) options configured via `triage.Middleware()`:

```go
handler := triage.Middleware(
    apiKey string,
    opts ...Option,
)(next http.Handler)
```

| Option             | Signature                            | Description                                           |
| :----------------- | :----------------------------------- | :---------------------------------------------------- |
| `WithRepo`         | `WithRepo(repo string)`              | GitHub repository (`owner/repo`) for AST resolution.  |
| `WithRootPath`     | `WithRootPath(path string)`          | Go backend subdirectory for monorepo setups.          |
| `WithGatewayURL`   | `WithGatewayURL(url string)`         | Engine telemetry endpoint.                            |
| `WithCommit`       | `WithCommit(sha string)`             | Explicit git commit SHA override.                     |
| `WithWorkerPool`   | `WithWorkerPool(workers, queue int)` | Adjust async telemetry worker count and queue buffer. |
| `WithCustomLogger` | `WithCustomLogger(l Logger)`         | Inject a custom logger for SDK output.                |
