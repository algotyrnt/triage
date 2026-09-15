---
title: CLI & Configuration Reference
description: Complete CLI flags, embedded storage options, and SDK configuration reference
---

Triage follows the **Prometheus zero-config model**: it runs as a single binary or container with sensible defaults, requires no environment files or external database servers, and stores all instance configuration dynamically in embedded SQLite.

## Triage Server CLI Flags

The `triage` binary supports the following command-line flags:

| Flag          | Type     | Default | Description                                                     |
| :------------ | :------- | :------ | :-------------------------------------------------------------- |
| `--port`      | `string` | `8080`  | HTTP listen port for both the API and embedded Studio Dashboard |
| `--data-dir`  | `string` | `data`  | Directory where embedded SQLite database (`triage.db`) is saved |
| `--db`        | `string` | —       | Explicit SQLite database file path (overrides `--data-dir`)     |
| `--log-level` | `string` | `info`  | Logging verbosity (`debug`, `info`, `warn`, `error`)            |

### Example Usages

```bash
# Default zero-config startup (listens on :8080, writes to data/triage.db)
./bin/triage

# Custom port and data directory
./bin/triage --port=9090 --data-dir=/var/lib/triage --log-level=debug
```

> **Note:** Ingestion API keys, AI provider credentials, GitHub App configs, and the Dashboard origin URL (`instance_url`) are managed and validated strictly through the embedded SQLite database via the Setup Wizard and Dashboard Settings UI.

---

## Go SDK Middleware Reference

The Go SDK (`sdk/go`) middleware is initialized with just your API key and Server URL:

```go
handler := triage.Middleware(
    apiKey string,
    triageServerURL string,
)(next http.Handler)
```

| Parameter         | Type     | Description                                                                                    |
| :---------------- | :------- | :--------------------------------------------------------------------------------------------- |
| `apiKey`          | `string` | Telemetry ingestion API key. Identifies repository, monorepo subfolder, and permissions.       |
| `triageServerURL` | `string` | Full HTTP telemetry endpoint of your Triage server (`http://localhost:8080/api/v1/telemetry`). |

All other metadata (Git commit SHA, GitHub repository, monorepo directory) is auto-resolved by the Go binary runtime and the Triage engine.

---

## Test Services Configuration (`test-services/*`)

The dummy test simulation microservices (`test-services/*`) support the following configuration:

| Variable            | Type     | Required? | Default                                  | Description                                        |
| :------------------ | :------- | :-------- | :--------------------------------------- | :------------------------------------------------- |
| `TRIAGE_API_KEY`    | `string` | **Yes**   | —                                        | Project-scoped ingestion key from Triage Dashboard |
| `TRIAGE_ENGINE_URL` | `string` | No        | `http://localhost:8080/api/v1/telemetry` | Telemetry endpoint on the Triage server            |
| `PORT`              | `string` | No        | Service-specific (`8081`-`8084`)         | Port the HTTP service listens on                   |

---

## CORS & Origin Security

The Triage Server enforces dynamic, origin-restricted Cross-Origin Resource Sharing (CORS):

- **Setup Phase (Pre-Configuration):** When `instance_url` is not yet saved, the server reflects the caller's origin, enabling seamless Setup Wizard access on first boot.
- **Operational Phase (Post-Setup):** Once configured, the server restricts browser cross-origin requests strictly to the configured `instance_url` and local development loopback addresses (`localhost`, `127.0.0.1`).
- **Server-to-Server Ingestion:** The Go SDK uses direct HTTP client calls which do not enforce browser CORS rules, ensuring zero-overhead telemetry ingestion.
