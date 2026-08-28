---
title: Self-Hosting Guide
description: Deploy zero-dependency single-container Triage on Docker, Kubernetes, or Cloud Run
---

Triage is designed for frictionless self-hosting. Just like Prometheus or PocketBase, the entire platform runs as a **single, zero-dependency container** (or standalone binary) containing both the server backend and the embedded React Studio Dashboard.

## Production Deployment with Docker (Preferred & Recommended)

Run the official multi-architecture single-container image published to GitHub Container Registry (`ghcr.io`):

```bash
docker run -d \
  --name triage \
  -p 8080:8080 \
  -v triage_data:/data \
  ghcr.io/algotyrnt/triage:latest
```

Open [http://localhost:8080](http://localhost:8080) to access your instance and complete the initial setup wizard.

---

## Zero-Config Embedded Storage

Triage automatically initializes an embedded SQLite database in the `/data` directory (with Write-Ahead Logging enabled for high-concurrency performance). No database servers or configuration files are required.

To persist your incidents and configurations across container restarts, mount a host directory or named volume to `/data`:

```bash
docker run -d \
  --name triage \
  -p 8080:8080 \
  -v triage_data:/data \
  ghcr.io/algotyrnt/triage:latest
```

---

## Local Development from Source

To build and run Triage locally from source:

```bash
git clone https://github.com/algotyrnt/triage.git
cd triage

# Build all components (Dashboard + Engine binary + SDK + Web)
make build

# Run the standalone binary
./bin/triage --port=8080 --data-dir=./data
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

---

## Zero-Config Startup Auto-Migration

Triage automatically verifies and provisions all required tables, columns, and performance indexes on boot:

- `users` & `invitations`: RBAC identity and team access tiers (`Owner`, `Admin`, `Developer`, `Viewer`).
- `repositories`: Configured Go repositories, monorepo root directory submodules, and domain architecture context.
- `incidents`: Captured panics, SHA-256 crash fingerprints for frequency deduplication (`occurrence_count`, `last_seen_at`), AI provider/model metadata, and generated bugfix patches.
- `ast_nodes`: Pre-indexed AST function declarations for sub-millisecond symbolication.
- `api_keys`: Ingestion API key hashes with constant-time SHA-256 authentication.
- `instance_config`: Dynamic settings (GitHub App, AI provider credentials, model/base URL, instance URL, session secrets).
- `github_installations` & `installation_repos`: GitHub App org and repository access maps.

---

## Production Checklist

1. **Persistent Volume:** Ensure `/data` is mounted to a persistent volume when using the default SQLite storage.
2. **Reverse Proxy & SSL:** Place Caddy, Nginx, Traefik, or Cloudflare in front of port `8080` for HTTPS termination.
3. **Session Secrets:** Handled automatically in instance configuration (signed 30-day HS256 JWT tokens).
