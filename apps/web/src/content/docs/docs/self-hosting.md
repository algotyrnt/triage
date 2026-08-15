---
title: Self-Hosting Guide
description: Deploy single-container Triage engine on Docker, Kubernetes, or Cloud Run
---

Triage is designed for frictionless self-hosting. You can run the entire platform as **1 single Docker container** or deploy with Docker Compose for local development.

## Docker Compose Quickstart (Recommended)

The easiest way to self-host Triage is with the official `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: triage-db
    restart: always
    environment:
      POSTGRES_USER: triage
      POSTGRES_PASSWORD: triage_secret_password
      POSTGRES_DB: triage_db
    ports:
      - '5432:5432'
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/schema.sql:/docker-entrypoint-initdb.d/schema.sql

  engine:
    image: triage/engine:latest
    container_name: triage-engine
    restart: always
    ports:
      - '8080:8080'
    environment:
      PORT: '8080'
      DATABASE_URL: 'postgres://triage:triage_secret_password@postgres:5432/triage_db?sslmode=disable'
    depends_on:
      - postgres

  dashboard:
    image: triage/dashboard:latest
    container_name: triage-dashboard
    restart: always
    ports:
      - '3000:3000'
    environment:
      PORT: '3000'
      NEXT_PUBLIC_ENGINE_URL: 'http://localhost:8080/api/v1/telemetry'
    depends_on:
      - engine

volumes:
  postgres_data:
```

### Start the Cluster

```bash
docker compose up -d
```

Open [http://localhost:3000](http://localhost:3000) to complete the setup wizard.

---

## Single Container Docker Run

You can also run just the `triage/engine` container against an existing PostgreSQL instance:

```bash
docker run -d \
  --name triage-engine \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@db.internal:5432/triage_db" \
  -e GEMINI_API_KEY="your_gemini_api_key" \
  -e TRIAGE_API_KEY="tr_live_production_key" \
  triage/engine:latest
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

1. **Postgres Credentials:** Change the default `POSTGRES_PASSWORD` in `docker-compose.yml`.
2. **Session Secrets:** The engine auto-generates a cryptographic session secret on first boot and stores it in PostgreSQL.
3. **HTTPS / Reverse Proxy:** Place Caddy, Nginx, or Cloudflare in front of `:8080` and `:3000` for SSL termination.
