---
title: Self-Hosting Guide
description: Deploy single-container Triage engine on Docker, Kubernetes, or Cloud Run
---

Triage is designed for frictionless self-hosting. You can run the entire platform as **1 single Docker container** or deploy with Docker Compose for local development.

## Production Deployment with Pre-Built Images (Recommended)

You can run the official production containers published to GitHub Container Registry (`ghcr.io`):

```bash
mkdir -p db
curl -fsSL https://raw.githubusercontent.com/algotyrnt/triage/main/db/schema.sql -o db/schema.sql
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
  -e GEMINI_API_KEY="your_gemini_api_key" \
  -e TRIAGE_API_KEY="tr_live_production_key" \
  ghcr.io/algotyrnt/triage-engine:latest
```

---

## Database Migration & Schema

The PostgreSQL schema is located in `db/schema.sql`. It contains:

- `projects`: Configured Go repositories and API keys.
- `incidents`: Captured panics, stack traces, and AI root causes.
- `ast_nodes`: Cached/pre-indexed AST function declarations.
- `system_config`: Encrypted GitHub App credentials and settings.
- `users` & `sessions`: Authenticated dashboard team members.

---

## Production Checklist

1. **Postgres Password:** Always supply an explicit, strong `POSTGRES_PASSWORD` when launching `docker-compose.prod.yml`.
2. **Session Secrets:** The engine auto-generates a cryptographic session secret on first boot and stores it in PostgreSQL.
3. **HTTPS / Reverse Proxy:** Place Caddy, Nginx, or Cloudflare in front of `:8080` and `:3000` for SSL termination.
