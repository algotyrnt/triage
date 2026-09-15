---
title: Documentation
description: Welcome to the triage Go Panic Isolation & AI Diagnostics Documentation
---

Welcome to the **triage** technical documentation.

## Quick Navigation

<div class="grid grid-cols-1 sm:grid-cols-2 gap-4 my-6">

<a href="/docs/overview" class="doc-card">
  <strong>Overview & Architecture</strong>
  <span>Learn how Triage achieves zero-overhead panic isolation and on-demand AST slicing.</span>
</a>

<a href="/docs/quickstart" class="doc-card">
  <strong>5-Minute Quickstart</strong>
  <span>Spin up a local stack and capture your first AI-diagnosed panic.</span>
</a>

<a href="/docs/sdk" class="doc-card">
  <strong>Go SDK Integration</strong>
  <span>Integrate Triage middleware into standard net/http, Chi, Gin, Echo, or Fiber.</span>
</a>

<a href="/docs/github-integration" class="doc-card">
  <strong>GitHub App & PR Automation</strong>
  <span>Set up automated GitHub issues and 1-click bugfix Pull Request generation.</span>
</a>

<a href="/docs/api-reference" class="doc-card">
  <strong>Engine REST API Reference</strong>
  <span>Complete REST API specifications for telemetry, projects, and AI diagnostics.</span>
</a>

<a href="/docs/ast-engine" class="doc-card">
  <strong>AST Engine Internals</strong>
  <span>Explore the Go parser, *ast.FuncDecl node extraction, and 3-tier caching.</span>
</a>

<a href="/docs/ai-diagnostics" class="doc-card">
  <strong>AI Incident Diagnostics</strong>
  <span>Configure your preferred AI model (Gemini, OpenAI, Claude, Ollama) for structured root-cause fixes.</span>
</a>

<a href="/docs/self-hosting" class="doc-card">
  <strong>Self-Hosting & Docker</strong>
  <span>Deploy zero-dependency single-container Triage in your private VPC or cluster.</span>
</a>

<a href="/docs/development" class="doc-card">
  <strong>Development & Releases</strong>
  <span>Makefile commands, testing matrix, and automated SemVer release pipeline.</span>
</a>

</div>

---

## What is Triage?

When a Go HTTP server panics at runtime, **triage**:

1. Non-blockingly catches the panic using standard Go `defer + recover()`.
2. Gathers the stack trace and slices the exact crash site along with cross-file package context (receiver structs, constructors, helper functions) using `go/parser`.
3. Streams the AST snippet to the configured **AI Diagnostic Engine** for instant root cause diagnosis and patch proposals.
4. Auto-creates detailed GitHub issues and enables 1-click **automated bugfix Pull Request generation**.
5. Displays live incidents in your self-hosted **Studio Dashboard UI** with an interactive project switcher.

---

## Ready to get started?

Jump right into the [Go SDK Integration Guide](/docs/sdk) or check the [Self-Hosting Guide](/docs/self-hosting).
