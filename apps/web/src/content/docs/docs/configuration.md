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

## Go SDK Middleware Reference

The Go SDK (`sdk/go`) middleware is initialized with just your API key and Engine URL:

```go
handler := triage.Middleware(
    apiKey string,
    engineURL string,
)(next http.Handler)
```

| Parameter   | Type     | Description                                                                                     |
| :---------- | :------- | :---------------------------------------------------------------------------------------------- |
| `apiKey`    | `string` | Telemetry ingestion API key. Identifies repository, monorepo subfolder, and permissions.        |
| `engineURL` | `string` | Full HTTP telemetry endpoint of your Triage engine (`https://.../api/v1/telemetry`).           |

All other metadata (Git commit SHA, GitHub repository, monorepo directory) is auto-resolved by the Go binary runtime and the Triage engine.
