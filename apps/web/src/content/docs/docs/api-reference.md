---
title: Engine REST API Reference
description: Complete REST API specification for telemetry ingestion, AST indexing, and incident management
---

All Triage Engine endpoints are served on port `:8080`.

## Health Check

### `GET /health`

Returns the operational health of the Triage engine and PostgreSQL connection.

**Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "database": "connected"
}
```

---

## Telemetry Ingestion

### `POST /api/v1/telemetry`

Receives panic telemetry payloads from Go SDK middleware.

**Headers:**

- `Content-Type: application/json`

**Request Body:**

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

**Response (200 OK):**

```json
{
  "incident_id": "inc_9f8a7b6c5d4e",
  "status": "PROCESSED",
  "root_cause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
  "suggested_fix": "Allocate memory before assignment.",
  "latency_ms": 14
}
```

---

## Incidents Management

### `GET /api/v1/incidents`

Lists recent incidents (requires authentication token).

**Headers:**

- `Authorization: Bearer <session_token>`

**Response:**

```json
[
  {
    "id": "inc_9f8a7b6c5d4e",
    "repo": "myorg/myrepo",
    "file": "handlers/payment.go",
    "line": 28,
    "panic_message": "runtime error: invalid memory address or nil pointer dereference",
    "severity": "CRITICAL",
    "root_cause": "Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.",
    "created_at": "2026-08-15T10:30:00Z"
  }
]
```

---

## AST Pre-Indexing

### `POST /api/v1/ast/index`

Triggers synchronous pre-indexing of a repository's AST into PostgreSQL.

**Headers:**

- `Authorization: Bearer <session_token>`
- `Content-Type: application/json`

**Request Body:**

```json
{
  "repo": "myorg/myrepo",
  "commit_sha": "main"
}
```

**Response:**

```json
{
  "status": "INDEXED",
  "functions_indexed": 142,
  "files_processed": 18
}
```
