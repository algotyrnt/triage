# Contributing to Triage

Thank you for your interest in contributing to Triage. We welcome contributions of all kinds: bug reports, feature requests, documentation improvements, test coverage expansions, and code changes.

This document outlines the guidelines and best practices to help you get started quickly and ensure your contributions can be smoothly reviewed and merged.

---

## Table of Contents

1. [Repository Architecture](#repository-architecture)
2. [Prerequisites](#prerequisites)
3. [Local Development Setup](#local-development-setup)
4. [Development Workflows](#development-workflows)
5. [Coding Guidelines and Standards](#coding-guidelines-and-standards)
6. [Commit Message Guidelines](#commit-message-guidelines)
7. [Submitting a Pull Request](#submitting-a-pull-request)
8. [Release Lifecycle](#release-lifecycle)
9. [Getting Help](#getting-help)

---

## Repository Architecture

Triage is structured as a modular monorepo containing the engine, studio dashboard, client SDK, and documentation:

```
.
├── Dockerfile          # Multi-stage production container build (Bun -> Go -> Alpine)
├── Makefile            # Centralized build, lint, test, and release automation
├── go.work             # Multi-module Go workspace
├── engine/             # Core server (Go 1.26+)
│   ├── cmd/            # Entrypoints
│   ├── internal/api/   # HTTP REST routes, SSE live stream and middleware
│   ├── internal/ast/   # Multi-file Go AST parsing, symbol indexing and slicing
│   ├── internal/config/# CLI flags and runtime configuration
│   ├── internal/db/    # Embedded SQLite storage with WAL auto-migration
│   ├── internal/github/# GitHub App authentication, issue filing and automated PRs
│   ├── internal/llm/   # Pluggable AI engine (Gemini, OpenAI, Claude, Ollama)
│   └── internal/ui/    # Embedded Vite dashboard static bundle (go:embed)
├── dashboard/          # Vite + React 19 Studio Dashboard (TypeScript, Tailwind CSS)
├── sdk/go/             # Official Go SDK panic recovery middleware (<0.02ms overhead)
├── web/                # Starlight + Astro public documentation and marketing site
└── test-services/      # Simulation microservices for local panic testing
```

---

## Prerequisites

Ensure you have the following installed on your machine:

- **Go**: 1.24+ (Go 1.26+ recommended)
- **Bun**: latest (used for dashboard and documentation workflows)
- **golangci-lint**: 1.64+ (install via `make tools` or your package manager)
- **Docker**: Optional, for containerized local testing
- **Make**: Standard POSIX make
- **Git**: 2.30+

---

## Local Development Setup

1. **Fork and Clone the Repository:**

   ```bash
   git clone https://github.com/<your-username>/triage.git
   cd triage
   ```

2. **Add Upstream Remote:**

   ```bash
   git remote add upstream https://github.com/algotyrnt/triage.git
   git fetch upstream
   ```

3. **Install All Dependencies and Tools:**

   This installs Go module dependencies, Bun packages, and Go development tools:

   ```bash
   make install
   make tools
   ```

4. **Verify Your Environment:**

   Run the full pre-flight verification gate to ensure everything builds and tests pass out of the box:

   ```bash
   make check
   ```

---

## Development Workflows

The repository provides granular make targets for isolated component development:

### 1. Running the Core Server (Engine)

```bash
make dev-engine
```
The server will start on http://localhost:8080 with embedded SQLite storage in `./data`.

### 2. Running the Studio Dashboard (Vite Dev Server)

```bash
make dev-dashboard
```
Starts the React 19 dashboard on http://localhost:3000 with hot-module reloading. It automatically proxies API requests to http://localhost:8080.

### 3. Running Documentation and Marketing Site (Astro)

```bash
make dev-web
```
Serves the Starlight documentation site on http://localhost:4321.

### 4. Running Test Suites

```bash
# Run all unit tests
make test

# Run tests with the Go data race detector (-race)
make test-race

# Generate code coverage reports
make test-coverage
```

### 5. Running Linters and Formatters

```bash
# Verify formatting across Go, Dashboard, and Web
make lint

# Automatically format all Go files (gofmt) and TypeScript/CSS (Prettier)
make format
```

---

## Coding Guidelines and Standards

### Go Code (engine/ and sdk/go/)

- **Idiomatic Go**: Follow Effective Go and Go Code Review Comments.
- **Concurrency Safety**: Always run `make test-race` when touching goroutines, sync primitives, or channels.
- **Error Handling**: Wrap errors with meaningful context (`fmt.Errorf("failed to parse ast: %w", err)`). Do not discard errors silently.
- **Zero Allocations and Performance**: The `sdk/go` middleware runs in hot request paths. Keep client allocations minimal and avoid blocking HTTP handlers under any circumstance.

### Frontend Code (dashboard/ and web/)

- **TypeScript**: Use strict typing. Avoid `any` unless absolutely necessary with explanatory comments.
- **Styling**: Use Tailwind CSS utility classes adhering to existing design tokens.
- **Formatting**: Run `bun run format` before committing.

---

## Commit Message Guidelines

This repository follows a clean, descriptive imperative commit message style matching the project git history. Do not use emoji prefixes.

### Format

- **First line (Subject)**: Start with an imperative, capitalized action verb (e.g. `Add`, `Update`, `Refactor`, `Enhance`, `Fix`, `Remove`). Keep the subject line concise (under 72 characters) and describe what was changed and why. Do not end with a period.
- **Optional body**: For multi-faceted or significant changes, add a blank line followed by a concise explanation or bulleted list of key technical details.

### Standard Action Verbs

- `Add`: Introduce a new feature, endpoint, test suite, or configuration file.
- `Update`: Modify existing functionality, documentation, dependencies, or workflows.
- `Refactor`: Restructure code without changing its external behavior.
- `Enhance`: Improve performance, error handling, test coverage, or edge-case handling.
- `Fix`: Correct a bug, regression, race condition, or broken test.
- `Remove`: Delete deprecated or dead code, unused endpoints, or obsolete assets.

### Examples from the Repository

```git
Add SetHTTPClient function for customizable HTTP client in tests and improve error handling in SignAppJWT
```

```git
Refactor main function into run for improved argument handling and error management
```

```git
Enhance tests for LLM providers, logger initialization, UI handlers, version population, and main application run scenarios

- Add comprehensive tests for OpenAI, Anthropic, and Gemini providers with mock responses.
- Introduce tests for logger initialization at various log levels (WARN, ERROR).
- Create new tests for UI handler to cover all branches including static assets and SPA routes.
- Implement tests for version population from build info.
```

```git
Update CI configuration to include workflow files and codecov.yml in change detection, and remove caching of Go build artifacts
```

---

## Submitting a Pull Request

1. **Create a Feature Branch:**

   Always branch from an up-to-date `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feat/my-new-feature
   ```

2. **Make Your Changes and Add Tests:**
   Ensure any new functionality or bug fix includes corresponding unit tests.

3. **Run Pre-Flight Quality Gate:**
   ```bash
   make check
   ```
   All tests and lint checks must pass before opening your PR.

4. **Submit Your Pull Request:**
   - Push your branch: `git push origin feat/my-new-feature`
   - Open a PR against `algotyrnt/triage:main`.
   - Complete the [Pull Request Template](.github/pull_request_template.md).
   - Link any related issues (`Closes #123` or `Fixes #123`).

5. **Code Review:**
   - Maintainers will review your PR. Address any review comments with follow-up commits.
   - Once approved and CI passes, a maintainer will squash and merge your PR.

---

## Release Lifecycle

Triage follows Semantic Versioning 2.0.0 (vMAJOR.MINOR.PATCH).
- Releases are automated via GitHub Actions (`.github/workflows/release.yml`).
- Standalone binaries are cross-compiled for Linux, macOS, and Windows with SHA-256 checksums.
- Multi-arch Docker images are pushed to GitHub Container Registry (`ghcr.io/algotyrnt/triage`).
- The Go SDK is dual-tagged (`vX.Y.Z` and `sdk/go/vX.Y.Z`) and indexed on [pkg.go.dev](https://pkg.go.dev/github.com/algotyrnt/triage/sdk/go).

---

## Getting Help

- **Documentation**: [triage.algotyrnt.com](https://triage.algotyrnt.com)
- **GitHub Discussions**: [github.com/algotyrnt/triage/discussions](https://github.com/algotyrnt/triage/discussions)
- **Issue Tracker**: [github.com/algotyrnt/triage/issues](https://github.com/algotyrnt/triage/issues)
- **Security Inquiries**: See [SECURITY.md](SECURITY.md)
