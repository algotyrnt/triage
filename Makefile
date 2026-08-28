# ==============================================================================
# Triage Project Makefile
# ==============================================================================
# Provides automated workflows for releases, testing, builds, linting,
# Docker containers, and local development.
# ==============================================================================

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

# ------------------------------------------------------------------------------
# Configuration & Metadata
# ------------------------------------------------------------------------------
BIN_DIR         := bin
REPO            := github.com/algotyrnt/triage
RELEASE_BRANCH  ?= main
ALLOW_DIRTY     ?= 0
ALLOW_BRANCH    ?= 0

# Versioning (matches root tag format v*.*.*)
LATEST_TAG      := $(shell git describe --tags --match="v[0-9]*" --abbrev=0 2>/dev/null || (git tag -l "v[0-9]*" --sort=-v:refname 2>/dev/null | head -n1) || echo "v0.1.0")
VERSION         ?= $(LATEST_TAG)
COMMIT          := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE      := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Toolchain
GO              ?= go
BUN             ?= bun
DOCKER          ?= docker

# Go Build Flags
LDFLAGS         := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)

# Docker Image
DOCKER_IMAGE    ?= ghcr.io/algotyrnt/triage
DOCKER_ENGINE_IMAGE ?= $(DOCKER_IMAGE)

# Colors for terminal output
COLOR_RESET   := \033[0m
COLOR_BOLD    := \033[1m
COLOR_GREEN   := \033[32m
COLOR_YELLOW  := \033[33m
COLOR_BLUE    := \033[34m
COLOR_CYAN    := \033[36m
COLOR_RED     := \033[31m

# ==============================================================================
# Help Target
# ==============================================================================

.PHONY: help
help: ## Show this help message
	@printf "\n$(COLOR_BOLD)$(COLOR_CYAN)Triage Build & Release Toolkit$(COLOR_RESET)\n"
	@printf "$(COLOR_YELLOW)Current Version:$(COLOR_RESET) $(VERSION) (latest tag: $(LATEST_TAG), commit: $(COMMIT))\n\n"
	@printf "$(COLOR_BOLD)Usage:$(COLOR_RESET) make $(COLOR_GREEN)<target>$(COLOR_RESET) [VERSION=vX.Y.Z]\n\n"
	@awk 'BEGIN {FS = ":.*?## "} \
		/^[a-zA-Z0-9_-]+:.*?## / { \
			target = $$1; \
			desc = $$2; \
			if (target ~ /^release/) { category = "Release & Versioning"; } \
			else if (target ~ /^tag/) { category = "Release & Versioning"; } \
			else if (target ~ /^push-tags/) { category = "Release & Versioning"; } \
			else if (target ~ /^package/) { category = "Release & Versioning"; } \
			else if (target ~ /^build/) { category = "Build Artifacts"; } \
			else if (target ~ /^test/) { category = "Testing & QA"; } \
			else if (target ~ /^(lint|format|fmt|check)/) { category = "Testing & QA"; } \
			else if (target ~ /^(docker|up|down|logs|prod-up)/) { category = "Docker & Containers"; } \
			else if (target ~ /^(dev|install|deps)/) { category = "Development"; } \
			else if (target ~ /^clean/) { category = "Utilities"; } \
			else { category = "General"; } \
			categories[category] = categories[category] sprintf("  $(COLOR_GREEN)%-24s$(COLOR_RESET) %s\n", target, desc); \
		} \
		END { \
			order[1] = "Release & Versioning"; \
			order[2] = "Testing & QA"; \
			order[3] = "Build Artifacts"; \
			order[4] = "Docker & Containers"; \
			order[5] = "Development"; \
			order[6] = "Utilities"; \
			for (i = 1; i <= 6; i++) { \
				c = order[i]; \
				if (c in categories) { \
					printf "$(COLOR_BOLD)$(COLOR_BLUE)[%s]$(COLOR_RESET)\n%s\n", c, categories[c]; \
				} \
			} \
		}' $(MAKEFILE_LIST)

# ==============================================================================
# Release & Versioning
# ==============================================================================

.PHONY: version
version: ## Display current version metadata
	@printf "$(COLOR_BOLD)Version:$(COLOR_RESET)    %s\n" "$(VERSION)"
	@printf "$(COLOR_BOLD)Latest Tag:$(COLOR_RESET) %s\n" "$(LATEST_TAG)"
	@printf "$(COLOR_BOLD)Commit:$(COLOR_RESET)     %s\n" "$(COMMIT)"
	@printf "$(COLOR_BOLD)Build Date:$(COLOR_RESET) %s\n" "$(BUILD_DATE)"

.PHONY: check-version-var
check-version-var:
	@if ! echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$$'; then \
		printf "$(COLOR_RED)[ERROR] Invalid version '$(VERSION)'. Must match SemVer format 'vX.Y.Z' (e.g. v0.1.1 or v1.0.0-rc.1)$(COLOR_RESET)\n"; \
		exit 1; \
	fi

.PHONY: check-git-clean
check-git-clean:
	@if [ "$(ALLOW_DIRTY)" != "1" ]; then \
		if ! git diff --quiet || ! git diff --cached --quiet; then \
			printf "$(COLOR_RED)[ERROR] Git working tree has uncommitted changes. Commit or stash them, or run with ALLOW_DIRTY=1$(COLOR_RESET)\n"; \
			git status -s; \
			exit 1; \
		fi; \
	fi

.PHONY: check-git-branch
check-git-branch:
	@if [ "$(ALLOW_BRANCH)" != "1" ]; then \
		CURRENT_BRANCH=$$(git branch --show-current 2>/dev/null || echo ""); \
		if [ -n "$$CURRENT_BRANCH" ] && [ "$$CURRENT_BRANCH" != "$(RELEASE_BRANCH)" ]; then \
			printf "$(COLOR_RED)[ERROR] Current branch is '$$CURRENT_BRANCH'. Releases must be cut from '$(RELEASE_BRANCH)' (or run with ALLOW_BRANCH=1)$(COLOR_RESET)\n"; \
			exit 1; \
		fi; \
	fi

.PHONY: release-dry-run
release-dry-run: check-version-var ## Simulate release workflow without modifying git or pushing
	@printf "$(COLOR_BOLD)$(COLOR_CYAN)==> Simulating release for $(VERSION)...$(COLOR_RESET)\n"
	@printf "$(COLOR_YELLOW)[1/3] Running pre-flight checks (lint + test + build)...$(COLOR_RESET)\n"
	@$(MAKE) check
	@printf "$(COLOR_YELLOW)[2/3] Simulating tag creation:$(COLOR_RESET)\n"
	@printf "  - Would create git tag: $(COLOR_GREEN)%s$(COLOR_RESET)\n" "$(VERSION)"
	@printf "  - Would create git tag: $(COLOR_GREEN)sdk/go/%s$(COLOR_RESET)\n" "$(VERSION)"
	@printf "$(COLOR_YELLOW)[3/3] Simulating remote push:$(COLOR_RESET)\n"
	@printf "  - Would run: git push origin %s\n" "$(VERSION)"
	@printf "  - Would run: git push origin sdk/go/%s\n" "$(VERSION)"
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)Dry run completed successfully!$(COLOR_RESET)\n"

.PHONY: tag
tag: check-version-var ## Create local git tags for root and sdk/go
	@printf "$(COLOR_CYAN)==> Creating release tags for $(VERSION)...$(COLOR_RESET)\n"
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		printf "$(COLOR_YELLOW)Tag $(VERSION) already exists locally.$(COLOR_RESET)\n"; \
	else \
		git tag -a "$(VERSION)" -m "Release $(VERSION)"; \
		printf "$(COLOR_GREEN)Created tag: $(VERSION)$(COLOR_RESET)\n"; \
	fi
	@if git rev-parse "sdk/go/$(VERSION)" >/dev/null 2>&1; then \
		printf "$(COLOR_YELLOW)Tag sdk/go/$(VERSION) already exists locally.$(COLOR_RESET)\n"; \
	else \
		git tag -a "sdk/go/$(VERSION)" -m "Release Go SDK $(VERSION)"; \
		printf "$(COLOR_GREEN)Created tag: sdk/go/$(VERSION)$(COLOR_RESET)\n"; \
	fi

.PHONY: push-tags
push-tags: check-version-var ## Push release tags to origin to trigger CI/CD pipeline
	@printf "$(COLOR_CYAN)==> Pushing release tags to origin...$(COLOR_RESET)\n"
	git push origin "$(VERSION)"
	git push origin "sdk/go/$(VERSION)"
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)Pushed tags $(VERSION) and sdk/go/$(VERSION) to origin.$(COLOR_RESET)\n"

.PHONY: release
release: check-git-branch check-git-clean check-version-var ## Execute full release: verify, tag, and push (e.g. make release VERSION=v0.1.1)
	@printf "$(COLOR_BOLD)$(COLOR_CYAN)==> Initiating Triage Release $(VERSION)$(COLOR_RESET)\n"
	@printf "$(COLOR_YELLOW)[1/3] Running pre-flight verification (lint, test, build)...$(COLOR_RESET)\n"
	@$(MAKE) check VERSION=$(VERSION)
	@printf "$(COLOR_YELLOW)[2/3] Creating release tags...$(COLOR_RESET)\n"
	@$(MAKE) tag VERSION=$(VERSION)
	@printf "$(COLOR_YELLOW)[3/3] Pushing tags to remote...$(COLOR_RESET)\n"
	@$(MAKE) push-tags VERSION=$(VERSION)
	@printf "\n$(COLOR_BOLD)$(COLOR_GREEN)Release $(VERSION) successfully published!$(COLOR_RESET)\n"
	@printf "GitHub Actions release pipeline will now build containers and publish the GitHub Release.\n"

.PHONY: release-patch
release-patch: ## Bump patch version and trigger release (e.g. v0.1.0 -> v0.1.1)
	@NEXT_VERSION=$$(echo "$(LATEST_TAG)" | awk -F. '{print $$1 "." $$2 "." $$3+1}'); \
	printf "$(COLOR_CYAN)Bumping patch version: $(LATEST_TAG) -> $$NEXT_VERSION$(COLOR_RESET)\n"; \
	$(MAKE) release VERSION=$$NEXT_VERSION

.PHONY: release-minor
release-minor: ## Bump minor version and trigger release (e.g. v0.1.0 -> v0.2.0)
	@NEXT_VERSION=$$(echo "$(LATEST_TAG)" | awk -F. '{print $$1 "." $$2+1 ".0"}'); \
	printf "$(COLOR_CYAN)Bumping minor version: $(LATEST_TAG) -> $$NEXT_VERSION$(COLOR_RESET)\n"; \
	$(MAKE) release VERSION=$$NEXT_VERSION

.PHONY: release-major
release-major: ## Bump major version and trigger release (e.g. v0.1.0 -> v1.0.0)
	@NEXT_VERSION=$$(echo "$(LATEST_TAG)" | awk -F. '{v=substr($$1,2); print "v" v+1 ".0.0"}'); \
	printf "$(COLOR_CYAN)Bumping major version: $(LATEST_TAG) -> $$NEXT_VERSION$(COLOR_RESET)\n"; \
	$(MAKE) release VERSION=$$NEXT_VERSION

# ==============================================================================
# Testing & Quality Assurance
# ==============================================================================

.PHONY: check
check: lint test build ## Run full pre-flight quality gate (lint + test + build)
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)==> All pre-flight checks passed!$(COLOR_RESET)\n"

.PHONY: test
test: test-engine test-sdk ## Run all Go test suites
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)==> All test suites passed!$(COLOR_RESET)\n"

.PHONY: test-engine
test-engine: ## Run Engine unit tests
	@printf "$(COLOR_CYAN)==> Testing Engine (engine)...$(COLOR_RESET)\n"
	@cd engine && $(GO) test -v ./...

.PHONY: test-sdk
test-sdk: ## Run Go SDK unit tests
	@printf "$(COLOR_CYAN)==> Testing Go SDK (sdk/go)...$(COLOR_RESET)\n"
	@cd sdk/go && $(GO) test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with code coverage report
	@printf "$(COLOR_CYAN)==> Running Go tests with coverage profiling...$(COLOR_RESET)\n"
	@cd engine && $(GO) test -coverprofile=../coverage-engine.out ./...
	@cd sdk/go && $(GO) test -coverprofile=../../coverage-sdk.out ./...
	@echo "mode: set" > coverage.out
	@grep -h -v "^mode:" coverage-engine.out coverage-sdk.out >> coverage.out 2>/dev/null || true
	@rm -f coverage-engine.out coverage-sdk.out
	@$(GO) tool cover -func=coverage.out
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@printf "$(COLOR_GREEN)Generated coverage report at coverage.html$(COLOR_RESET)\n"

.PHONY: lint
lint: lint-go lint-web lint-dashboard ## Run all code linters and formatting checks
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)==> Linting & formatting checks passed!$(COLOR_RESET)\n"

.PHONY: lint-go
lint-go: lint-engine lint-sdk ## Verify all Go formatting and static analysis

.PHONY: lint-engine
lint-engine: ## Verify Engine formatting and vet
	@printf "$(COLOR_CYAN)==> Checking Engine formatting and vet...$(COLOR_RESET)\n"
	@UNFORMATTED=$$(gofmt -l engine); \
	if [ -n "$$UNFORMATTED" ]; then \
		printf "$(COLOR_RED)[ERROR] Unformatted Go files in engine:\n$$UNFORMATTED$(COLOR_RESET)\n"; \
		printf "Run '$(COLOR_YELLOW)make format-go$(COLOR_RESET)' to auto-fix.\n"; \
		exit 1; \
	fi
	@cd engine && $(GO) vet ./...

.PHONY: lint-sdk
lint-sdk: ## Verify Go SDK formatting and vet
	@printf "$(COLOR_CYAN)==> Checking Go SDK formatting and vet...$(COLOR_RESET)\n"
	@UNFORMATTED=$$(gofmt -l sdk/go); \
	if [ -n "$$UNFORMATTED" ]; then \
		printf "$(COLOR_RED)[ERROR] Unformatted Go files in sdk/go:\n$$UNFORMATTED$(COLOR_RESET)\n"; \
		printf "Run '$(COLOR_YELLOW)make format-go$(COLOR_RESET)' to auto-fix.\n"; \
		exit 1; \
	fi
	@cd sdk/go && $(GO) vet ./...

.PHONY: lint-web
lint-web: ## Check Astro web formatting
	@printf "$(COLOR_CYAN)==> Checking Web formatting...$(COLOR_RESET)\n"
	@cd web && $(BUN) run format:check

.PHONY: lint-dashboard
lint-dashboard: ## Check Vite dashboard formatting
	@printf "$(COLOR_CYAN)==> Checking Dashboard formatting...$(COLOR_RESET)\n"
	@cd dashboard && $(BUN) run format:check

.PHONY: format fmt
format: format-go format-web format-dashboard ## Auto-format all Go, Web, and Dashboard code
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)==> All files formatted!$(COLOR_RESET)\n"

fmt: format

.PHONY: format-go
format-go: ## Auto-format Go code with gofmt
	@printf "$(COLOR_CYAN)==> Formatting Go files...$(COLOR_RESET)\n"
	@gofmt -w -s .

.PHONY: format-web
format-web: ## Auto-format Astro web code
	@printf "$(COLOR_CYAN)==> Formatting Web files...$(COLOR_RESET)\n"
	@cd web && $(BUN) run format

.PHONY: format-dashboard
format-dashboard: ## Auto-format Vite dashboard code
	@printf "$(COLOR_CYAN)==> Formatting Dashboard files...$(COLOR_RESET)\n"
	@cd dashboard && $(BUN) run format

# ==============================================================================
# Build Targets
# ==============================================================================

.PHONY: build
build: build-dashboard build-triage build-sdk build-web ## Build all components
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)==> All components built successfully!$(COLOR_RESET)\n"

.PHONY: build-dashboard
build-dashboard: ## Build Vite Studio Dashboard into Engine embed directory
	@printf "$(COLOR_CYAN)==> Building Vite Dashboard ($(VERSION))...$(COLOR_RESET)\n"
	@cd dashboard && $(BUN) run build

.PHONY: build-triage build-engine
build-triage: ## Compile Triage server binary to bin/triage
	@printf "$(COLOR_CYAN)==> Building Triage binary ($(VERSION))...$(COLOR_RESET)\n"
	@mkdir -p $(BIN_DIR)
	@cd engine && CGO_ENABLED=0 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o ../$(BIN_DIR)/triage main.go
	@printf "$(COLOR_GREEN)Built $(BIN_DIR)/triage$(COLOR_RESET)\n"

build-engine: build-triage

.PHONY: build-sdk
build-sdk: ## Build & verify Go SDK
	@printf "$(COLOR_CYAN)==> Building Go SDK...$(COLOR_RESET)\n"
	@cd sdk/go && $(GO) build ./...

.PHONY: build-web
build-web: ## Build Astro web documentation & landing page
	@printf "$(COLOR_CYAN)==> Building Astro Web bundle ($(VERSION))...$(COLOR_RESET)\n"
	@cd web && PUBLIC_TRIAGE_VERSION=$(VERSION) $(BUN) run build

.PHONY: deploy-web
deploy-web: build-web ## Deploy Astro web site to Cloudflare via Wrangler
	@printf "$(COLOR_CYAN)==> Deploying Web to Cloudflare...$(COLOR_RESET)\n"
	@cd web && $(BUN) x wrangler deploy

# ==============================================================================
# Docker & Containers
# ==============================================================================

.PHONY: docker-build
docker-build: ## Build unified Triage Docker image locally
	@printf "$(COLOR_CYAN)==> Building Triage Docker image ($(DOCKER_IMAGE):$(VERSION))...$(COLOR_RESET)\n"
	@$(DOCKER) build \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		.
	@printf "$(COLOR_GREEN)Triage Docker image built successfully.$(COLOR_RESET)\n"

.PHONY: docker-run run
docker-run: ## Run Triage in a single Docker container
	@printf "$(COLOR_CYAN)==> Starting Triage container on :8080...$(COLOR_RESET)\n"
	@$(DOCKER) run -d --rm \
		--name triage \
		-p 8080:8080 \
		-v triage_data:/data \
		$(DOCKER_IMAGE):latest
	@printf "$(COLOR_GREEN)Triage running at http://localhost:8080$(COLOR_RESET)\n"

run: docker-run

.PHONY: docker-stop stop
docker-stop: ## Stop running Triage container
	@printf "$(COLOR_CYAN)==> Stopping Triage container...$(COLOR_RESET)\n"
	@$(DOCKER) stop triage 2>/dev/null || true
	@printf "$(COLOR_GREEN)Triage container stopped.$(COLOR_RESET)\n"

stop: docker-stop

.PHONY: docker-logs logs
docker-logs: ## Follow Triage Docker container logs
	@$(DOCKER) logs -f triage

logs: docker-logs

# ==============================================================================
# Development & Setup
# ==============================================================================

.PHONY: install deps
install: ## Install all dependencies (Go modules & Bun packages)
	@printf "$(COLOR_CYAN)==> Downloading Go modules and installing Bun packages...$(COLOR_RESET)\n"
	@cd engine && $(GO) mod download
	@cd sdk/go && $(GO) mod download
	@cd web && $(BUN) install && $(BUN) x astro sync
	@cd dashboard && $(BUN) install
	@printf "$(COLOR_BOLD)$(COLOR_GREEN)==> Dependencies installed and types synchronized!$(COLOR_RESET)\n"

deps: install

.PHONY: dev-engine
dev-engine: ## Run Engine server locally with live logging
	@printf "$(COLOR_CYAN)==> Starting Engine on :8080...$(COLOR_RESET)\n"
	@cd engine && $(GO) run main.go

.PHONY: dev-dashboard
dev-dashboard: ## Run Vite Dashboard in development mode (proxying API to :8080)
	@printf "$(COLOR_CYAN)==> Starting Dashboard on :3000...$(COLOR_RESET)\n"
	@cd dashboard && $(BUN) run dev

.PHONY: dev-web
dev-web: ## Run Astro Web & docs in development mode
	@printf "$(COLOR_CYAN)==> Starting Astro Web on :4321...$(COLOR_RESET)\n"
	@cd web && $(BUN) run dev

# ==============================================================================
# Utilities & Cleanup
# ==============================================================================

.PHONY: clean
clean: ## Clean binaries, coverage reports, tarballs, and build artifacts
	@printf "$(COLOR_CYAN)==> Cleaning build outputs...$(COLOR_RESET)\n"
	@rm -rf $(BIN_DIR)
	@rm -rf coverage.out coverage.html coverage-engine.out coverage-sdk.out
	@rm -rf web/dist
	@rm -rf engine/internal/ui/dist
	@printf "$(COLOR_GREEN)Clean completed.$(COLOR_RESET)\n"
