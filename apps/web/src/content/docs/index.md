---
title: triage Documentation
description: Go Crash Isolation & AI Diagnostic Engine Documentation
---

Welcome to **triage**—the zero-overhead Go crash isolation tool, automated GitHub issue triaging engine, and AI diagnostic platform.

## Key Features

- **On-Demand Synchronous AST Parsing:** Zero background database pre-indexing required. Fetches exact source code by `commit_sha` on demand and isolates the panicking `*ast.FuncDecl` function block.
- **AI Diagnostics:** Leverages **Google Gemini 3.5 Flash** for instant root-cause analysis and suggested code fixes.
- **Single-Container Self-Hosting:** Self-hosters run **1 single Docker container** (`triage/engine:latest` on `:8080`) that serves both diagnostic API endpoints and the Studio Dashboard UI (`:8080/dashboard`).

## Quick Links

- [Go SDK Integration Guide](/docs/sdk)
- [Self-Hosting Guide](/docs/self-hosting)
