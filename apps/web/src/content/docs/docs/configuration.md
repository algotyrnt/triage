---
title: Environment & Configuration Reference
description: Complete environment variable and configuration reference for Engine, SDK, and Dashboard
---

This page documents all configuration options and environment variables across the Triage ecosystem.

## Triage Engine Configuration

The engine (`apps/engine`) supports the following environment variables:

| Variable       | Type     | Default | Description                                                  |
| :------------- | :------- | :------ | :----------------------------------------------------------- |
| `DATABASE_URL` | `string` | —       | PostgreSQL connection string (**required**).                 |
| `PORT`         | `string` | `8080`  | Engine HTTP listen port.                                     |
| `LOG_LEVEL`    | `string` | `info`  | Engine logging verbosity (`debug`, `info`, `warn`, `error`). |

> **Note:** Ingestion API keys, Gemini AI credentials, GitHub App configs, and the Dashboard origin URL (`instance_url`) are managed and validated strictly through PostgreSQL via the Setup Wizard and Dashboard Settings UI.

---

## Studio Dashboard Configuration

The Next.js dashboard (`apps/dashboard`) supports:

| Variable            | Type     | Default                 | Description                 |
| :------------------ | :------- | :---------------------- | :-------------------------- |
| `PORT`              | `string` | `3000`                  | Dashboard HTTP listen port. |
| `TRIAGE_ENGINE_URL` | `string` | `http://localhost:8080` | Engine backend URL.         |

> **Note:** The dashboard automatically discovers its public origin directly from incoming HTTP requests without requiring an explicit dashboard URL variable.

---

## Go SDK Middleware Reference

The Go SDK (`sdk/go`) middleware is initialized with just your API key and Engine URL:

```go
handler := triage.Middleware(
    apiKey string,
    engineURL string,
)(next http.Handler)
```

| Parameter   | Type     | Description                                                                              |
| :---------- | :------- | :--------------------------------------------------------------------------------------- |
| `apiKey`    | `string` | Telemetry ingestion API key. Identifies repository, monorepo subfolder, and permissions. |
| `engineURL` | `string` | Full HTTP telemetry endpoint of your Triage engine (`https://.../api/v1/telemetry`).     |

All other metadata (Git commit SHA, GitHub repository, monorepo directory) is auto-resolved by the Go binary runtime and the Triage engine.

---

## Test Services Configuration (`test-services/*`)

The dummy test simulation microservices (`test-services/*`) support the following configuration:

| Variable            | Type     | Required? | Default                                  | Description                                        |
| :------------------ | :------- | :-------- | :--------------------------------------- | :------------------------------------------------- |
| `TRIAGE_API_KEY`    | `string` | **Yes**   | —                                        | Project-scoped ingestion key from Triage Dashboard |
| `TRIAGE_ENGINE_URL` | `string` | No        | `http://localhost:8080/api/v1/telemetry` | Telemetry endpoint on the Triage engine            |
| `PORT`              | `string` | No        | Service-specific (`8081`-`8084`)         | Port the HTTP service listens on                   |

---

## CORS & Origin Security

The Triage Engine enforces dynamic, origin-restricted Cross-Origin Resource Sharing (CORS):

- **Setup Phase (Pre-Configuration):** When `instance_url` is not yet saved in PostgreSQL, the engine reflects the caller's origin, enabling seamless Setup Wizard access on first boot.
- **Operational Phase (Post-Setup):** Once configured, the engine restricts browser cross-origin requests strictly to the database-configured `instance_url` and local development loopback addresses (`localhost`, `127.0.0.1`).
- **Server-to-Server Ingestion:** The Go SDK uses direct HTTP client calls which do not enforce browser CORS rules, ensuring zero-overhead telemetry ingestion.
- **Credentials & Headers:** Permitted origins receive `Access-Control-Allow-Credentials: true` and `Vary: Origin`.
