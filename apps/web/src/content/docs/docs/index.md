---
title: Documentation
description: Welcome to the triage Go Panic Isolation & Gemini AI Diagnostics Documentation
---

Welcome to the **triage** technical documentation.

## Quick Navigation

<div class="grid grid-cols-1 sm:grid-cols-2 gap-4 my-6">

<a href="/docs/overview" class="doc-card">
  <strong>📖 Overview & Architecture</strong>
  <span>Learn how Triage achieves zero-overhead panic isolation and on-demand AST slicing.</span>
</a>

<a href="/docs/quickstart" class="doc-card">
  <strong>⚡ 5-Minute Quickstart</strong>
  <span>Spin up a local stack and capture your first AI-diagnosed panic.</span>
</a>

<a href="/docs/sdk" class="doc-card">
  <strong>🛠️ Go SDK Integration</strong>
  <span>Integrate Triage middleware into standard net/http, Chi, Gin, Echo, or Fiber.</span>
</a>

<a href="/docs/ast-engine" class="doc-card">
  <strong>🌲 AST Engine Internals</strong>
  <span>Explore the Go parser, *ast.FuncDecl node extraction, and 3-tier caching.</span>
</a>

<a href="/docs/gemini-ai" class="doc-card">
  <strong>🤖 Gemini AI Diagnostics</strong>
  <span>Configure your preferred Gemini model for structured root-cause fixes.</span>
</a>

<a href="/docs/self-hosting" class="doc-card">
  <strong>🐳 Self-Hosting & Docker</strong>
  <span>Deploy the single-container Triage engine in your private VPC or cluster.</span>
</a>

</div>

---

## What is Triage?

When a Go HTTP server panics at runtime, **triage**:
1. Non-blockingly catches the panic using standard Go `defer + recover()`.
2. Gathers the stack trace and slices only the enclosing `*ast.FuncDecl` function declaration using `go/parser`.
3. Streams the AST snippet to **Google Gemini AI** for instant root cause diagnosis and patch proposals.
4. Auto-creates a detailed GitHub issue with labels and telemetry IDs.
5. Displays live incidents in your self-hosted **Studio Dashboard UI**.

---

## Ready to get started?

Jump right into the [Go SDK Integration Guide](/docs/sdk) or check the [Self-Hosting Guide](/docs/self-hosting).
