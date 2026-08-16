---
title: SDK Architecture & Telemetry
description: Deep-dive into Triage's non-blocking panic recovery lifecycle and telemetry payload schema
---

This document outlines the internal architecture of the Triage Go client SDK and how it guarantees zero blocking on user requests.

## Panic Recovery Lifecycle

When a panic occurs in an HTTP handler wrapped by `triage.Middleware`:

```
┌─────────────────────────────────────────────────────────┐
│ 1. defer + recover() catches panic                      │
└──────────────────────────┬──────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────┐
│ 2. Captures debug.Stack() & extracts top app frame      │
└──────────────────────────┬──────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────┐
│ 3. Generates trace ID & writes generic HTTP 500         │
└──────────────────────────┬──────────────────────────────┘
                           │ (Non-blocking queue enqueue)
┌──────────────────────────▼──────────────────────────────┐
│ 4. Bounded Worker Pool (4 goroutines, 1000 buffer)      │
└──────────────────────────┬──────────────────────────────┘
                           │ (Async HTTP POST)
┌──────────────────────────▼──────────────────────────────┐
│ 5. Triage Engine API (/api/v1/telemetry)                │
└─────────────────────────────────────────────────────────┘
```

---

## 1. Frame Pruning & Symbolication

Standard Go `runtime/debug.Stack()` outputs internal runtime frames:

- `runtime/debug.Stack()`
- `github.com/algotyrnt/triage/sdk/go.Middleware.func1.1()`
- `runtime.gopanic()`

The Triage SDK parses the stack trace and skips runtime/stdlib/middleware frames to identify the **topmost application frame**. This extracts:

- `File`: Target file path (e.g. `handlers/payment.go`)
- `Line`: Exact triggering line number (e.g. `28`)
- `FunctionName`: Enclosing function name (e.g. `ProcessTransaction`)
- `GoroutineID`: Goroutine metadata (e.g. `goroutine 42 [running]`)

---

## 2. Bounded Worker Pool

To ensure memory safety and zero throughput degradation:

- The SDK initializes a buffered channel of size `1,000`.
- A fixed pool of 4 worker goroutines processes queued jobs.
- If the engine is temporarily unreachable or the queue is full under extreme panic storms, excess payloads are non-blockingly dropped to protect host memory.

---

## 3. Telemetry Payload Schema

The SDK dispatches the following JSON payload to `POST /api/v1/telemetry`:

```json
{
  "api_key": "tr_live_key_9042",
  "commit": "7f8b9e1a2c3d4e5f60718293",
  "file": "handlers/payment.go",
  "line": 28,
  "stack_trace": "goroutine 42 [running]:\n...",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

The Triage engine resolves `owner`, `repo`, and monorepo `root_dir` directly from the `api_key` in PostgreSQL, keeping the SDK payload minimal.

---

## 4. Secret Sanitization

The SDK automatically sanitizes sensitive headers before transmitting stack traces:

- `Authorization`
- `Cookie`
- `Set-Cookie`
- `X-Api-Key`
- Any header matching `(?i)(token|secret|password|bearer|auth)`
