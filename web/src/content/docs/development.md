---
title: Development & Release Guide
description: Local development commands, testing workflows, and release automation with Make
---

Triage includes a centralized `Makefile` to streamline local development, multi-package testing, linting, Docker operations, and SemVer release management.

## Prerequisites

- **Go 1.26+** (Required for Server and Go SDK)
- **Bun** (latest / 1.x+, used for Web & Dashboard build tooling)
- **Docker** (For single-container builds and local testing)
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

Download Go modules and install frontend dependencies across the repository:

```bash
make install
```

### 2. Local Service Development

Run individual components locally outside Docker with hot reloading and live logs:

```bash
# Start Triage server (with embedded SQLite and API) on :8080
make dev-engine

# Start Studio Dashboard (Vite + React 19 with hot reload proxying to :8080) on :3000
make dev-dashboard

# Start Documentation & Landing Page (Astro) on :4321
make dev-web

# Run a test simulation service with your API key and -trimpath (e.g. Order Service on :8082)
cd test-services/order-service
TRIAGE_API_KEY=your_sample_api_key go run -trimpath .
```

### 3. Single-Container Docker Workflow

Run and manage Triage in a single Docker container:

```bash
# Build the unified Docker image locally
make docker-build

# Run Triage in a container (listens on :8080 with volume triage_data:/data)
make run

# View container logs
make logs

# Stop the container
make stop
```

---

## Testing & Quality Assurance

Run the comprehensive pre-flight verification gate before submitting Pull Requests:

```bash
# Run linters, test suites, and component builds
make check
```

### Individual Test Targets

```bash
# Run all Go test suites (Server + SDK)
make test

# Run Server tests
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

# Auto-format all Go, Astro, and Vite code
make format
```

### Packaging Standalone Release Binaries

To locally cross-compile and generate standalone release archives (`.tar.gz` for Linux and macOS, `.zip` for Windows) along with SHA256 checksums:

```bash
make package VERSION=vX.Y.Z
```

Outputs are bundled cleanly into `dist-bin/`.

---

## Release Automation

Releases in Triage follow [Semantic Versioning](https://semver.org) and publish standalone cross-platform binaries, multi-arch Docker images, Go SDK packages, and web distribution bundles.

### 1. Dry-Run Verification

Preview the release process and verify all build assets without creating git tags or modifying git history:

```bash
make release-dry-run VERSION=vX.Y.Z
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
make release VERSION=vX.Y.Z
```

### 3. Release Pipeline Lifecycle

```
Local `make release`
  │  1. Verifies git working directory is clean
  │  2. Runs full test & lint matrix (`make check`)
  │  3. Creates tags `vX.Y.Z` and `sdk/go/vX.Y.Z`
  │  4. Pushes tags to GitHub
  ▼
GitHub Actions (`release.yml`)
  ├── Cross-compiles standalone binaries (Linux, macOS, Windows) & SHA256 checksums
  ├── Builds & pushes multi-arch Docker image (ghcr.io/algotyrnt/triage)
  ├── Warms Go module proxy cache (pkg.go.dev indexing)
  ├── Deploys documentation to Cloudflare Pages
  └── Publishes GitHub Release with automated release notes and binary assets
```
