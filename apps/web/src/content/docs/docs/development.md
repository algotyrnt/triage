---
title: Development & Release Guide
description: Local development commands, testing workflows, and release automation with Make
---

Triage includes a centralized `Makefile` to streamline local development, multi-package testing, linting, Docker operations, and SemVer release management.

## Prerequisites

- **Go 1.22+**
- **Bun** (latest)
- **Docker & Docker Compose**
- **Make** (`/usr/bin/make` on macOS / Linux)

---

## Quick Reference

Run `make help` at the root of the repository to display all available targets categorized by workflow:

```bash
make help
```

---

## Development Workflows

### 1. Install Dependencies

Download Go modules and install frontend dependencies across the monorepo:

```bash
make install
```

### 2. Local Service Development

Run individual services locally outside Docker with hot reloading and live logs:

```bash
# Start Go engine on :8080
make dev-engine

# Start Studio Dashboard (Next.js) on :3000
make dev-dashboard

# Start Documentation & Landing Page (Astro) on :4321
make dev-web

# Start local panic simulation test service on :8081
make dev-test-service
```

### 3. Docker Compose Stack

Manage the local Docker development cluster:

```bash
# Start PostgreSQL, Engine, and Dashboard
make up

# View container logs
make logs

# Stop the stack
make down
```

---

## Testing & Quality Assurance

Run the comprehensive pre-flight verification gate before submitting Pull Requests:

```bash
# Run linters, test suites, and application builds
make check
```

### Individual Test Targets

```bash
# Run all Go test suites (Engine + SDK)
make test

# Run Engine tests
make test-engine

# Run Go SDK tests
make test-sdk

# Generate code coverage profile (coverage.html)
make test-coverage
```

### Formatting & Linting

```bash
# Verify Go formatting, go vet, and Prettier checks
make lint

# Auto-format all Go, Astro, and Next.js code
make format
```

---

## Release Automation

Releases in Triage follow [Semantic Versioning](https://semver.org) and publish multi-arch Docker images, Go SDK packages, and web distribution bundles.

### 1. Dry-Run Verification

Preview the release process and verify all build assets without creating git tags or modifying git history:

```bash
make release-dry-run VERSION=v0.1.1
```

### 2. Cutting a Release

Execute a one-command release with automated pre-flight checks, dual-tagging (`vX.Y.Z` and `sdk/go/vX.Y.Z`), and remote push:

```bash
# Bump patch version: e.g. v0.1.0 -> v0.1.1
make release-patch

# Bump minor version: e.g. v0.1.0 -> v0.2.0
make release-minor

# Bump major version: e.g. v0.1.0 -> v1.0.0
make release-major

# Or specify an explicit version:
make release VERSION=v0.1.1
```

### 3. Release Pipeline Lifecycle

```
Local `make release`
  │  1. Verifies git working directory is clean
  │  2. Runs full test & lint matrix (`make check`)
  │  3. Creates tags `v0.1.1` and `sdk/go/v0.1.1`
  │  4. Pushes tags to GitHub
  ▼
GitHub Actions (`release.yml`)
  ├── Warms Go module proxy cache (pkg.go.dev indexing)
  ├── Builds & pushes multi-arch Docker images to GHCR
  ├── Packages web distribution bundle (`triage-web-v0.1.1.tar.gz`)
  └── Publishes GitHub Release with automated notes
```
