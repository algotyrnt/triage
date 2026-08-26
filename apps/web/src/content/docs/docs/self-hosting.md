---
title: Self-Hosting Guide
description: Deploy single-container Triage engine on Docker, Kubernetes, or Cloud Run
---

Triage is designed for frictionless self-hosting. You can run the entire platform as **1 single Docker container** or deploy with Docker Compose for local development.

## Production Deployment with Pre-Built Images (Recommended)

You can run the official production containers published to GitHub Container Registry (`ghcr.io`):

```bash
curl -fsSL https://raw.githubusercontent.com/algotyrnt/triage/main/docker-compose.prod.yml -o docker-compose.prod.yml

POSTGRES_PASSWORD=your_secure_password docker compose -f docker-compose.prod.yml up -d
```

---

## Local Development from Source

To build from source for local development, use `docker-compose.yml`:

```bash
git clone https://github.com/algotyrnt/triage.git
cd triage
docker compose up --build -d
```

Open [http://localhost:3000](http://localhost:3000) to complete the setup wizard.

---

## Single Container Docker Run

You can also run just the `ghcr.io/algotyrnt/triage-engine` container against an existing PostgreSQL instance:

```bash
docker run -d \
  --name triage-engine \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@db.internal:5432/triage_db" \
  ghcr.io/algotyrnt/triage-engine:latest
```

---

## Database Auto-Migration & Schema

Triage features **zero-config startup auto-migration**. When the Engine connects to PostgreSQL on boot, it automatically verifies and provisions all required tables, columns, and performance indexes using transaction-level advisory locks (`pg_advisory_xact_lock`). No manual migration scripts or CLI tools are required.

The PostgreSQL schema is defined in [`apps/engine/internal/db/schema.sql`](https://github.com/algotyrnt/triage/blob/main/apps/engine/internal/db/schema.sql):

- `users` & `invitations`: RBAC identity and team access tiers (`Owner`, `Admin`, `Developer`, `Viewer`).
- `repositories`: Configured Go repositories, monorepo root directory submodules, and domain architecture context.
- `incidents`: Captured panics, SHA-256 crash fingerprints for frequency deduplication (`occurrence_count`, `last_seen_at`), AI provider/model metadata, and generated bugfix patches.
- `ast_nodes` & `ast_indexes`: Cached and pre-indexed AST function declarations for sub-millisecond symbolication.
- `api_keys`: Ingestion API key hashes (`tr_live_...`) with constant-time SHA-256 authentication.
- `instance_config`: Dynamic settings (GitHub App, AI provider credentials, model/base URL, instance URL, session secrets).
- `github_installations` & `installation_repos`: GitHub App org and repository access maps.

---

## Production Checklist

1. **Postgres Password:** Always supply an explicit, strong `POSTGRES_PASSWORD` when launching `docker-compose.prod.yml`.
2. **Session Secrets:** The engine loads `session_secret` from PostgreSQL (or falls back to an internal secret in development) for signing 30-day HS256 JWT tokens.
3. **CORS & Origin Security:** Browser cross-origin access is dynamically locked to your configured dashboard origin (`instance_url`) upon completing the setup wizard.
4. **HTTPS / Reverse Proxy:** Place Caddy, Nginx, or Cloudflare in front of `:8080` and `:3000` for SSL termination.
