## Description

<!-- Provide a brief explanation of the motivation and context for this change. -->

## Related Issue(s)

<!-- Fixes #123, Closes #456 -->

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Test coverage expansion
- [ ] Build / CI / Tooling improvement

## Component(s) Affected

- [ ] `engine` (Core Go Server / AST Slicing / SQLite / LLM)
- [ ] `dashboard` (Vite + React 19 Studio UI)
- [ ] `sdk/go` (Go panic middleware & telemetry client)
- [ ] `web` (Documentation & Landing Site)
- [ ] `docker` (Dockerfile & Container deployment)
- [ ] `ci` (GitHub Actions workflows)

## Verification & Testing

<!-- Describe the tests you ran to verify your changes. Include commands, sample outputs, or screenshots if applicable. -->

- [ ] `make check` passed locally
- [ ] `make test-race` passed locally (if touching Go concurrency)
- [ ] Unit tests added or updated

## Checklist

- [ ] My code adheres to the project's coding style (`make lint` / `make format`)
- [ ] I have read and followed the [Contributing Guidelines](CONTRIBUTING.md)
- [ ] I have added appropriate documentation where necessary
- [ ] My PR title follows the project commit message guidelines (e.g. `Add`, `Update`, `Refactor`, `Fix`)
- [ ] If this is a breaking change, I have documented the migration path
