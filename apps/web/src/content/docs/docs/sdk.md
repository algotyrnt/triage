---
title: Go SDK Integration Guide
description: Complete guide for integrating triage panic isolation middleware with any Go HTTP router
---

The `github.com/algotyrnt/triage/sdk/go` package provides non-blocking panic recovery middleware compatible with Go standard library `net/http` and all popular HTTP frameworks.

## Installation

```bash
go get github.com/algotyrnt/triage/sdk/go
```

---

## Basic Middleware Usage

By default, the SDK only requires your **API key** and the **Engine URL**. Repository metadata, monorepo subdirectories, and Git commit hashes are automatically resolved by the backend and the Go runtime:

```go
package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/orders", handleOrders)

	// Wrap handler with triage panic isolation middleware
	handler := triage.Middleware(
		"tr_live_your_api_key",
		"https://triage.yourcompany.com/api/v1/telemetry",
	)(mux)

	http.ListenAndServe(":8080", handler)
}
```

---

## How Zero-Configuration Works

The Triage SDK is designed with **zero client-side configuration boilerplate**. You only pass `apiKey` and `engineURL`:

1. **Repository & Monorepo Subfolder Resolution:** When you register a repository in the Triage dashboard (even as a subfolder in a monorepo like `/backend` or `/apps/api`), your API key is uniquely bound to that project. The Triage engine looks up repository metadata automatically upon telemetry arrival.
2. **Git Commit Detection:** The SDK automatically extracts the Git commit SHA from Go's embedded binary build info via `debug.ReadBuildInfo()`.
3. **Trace Context Propagation:** The SDK automatically extracts or generates OpenTelemetry-compatible W3C `traceparent` headers (`00-<trace_id>-0000000000000001-01`) and `X-Triage-Trace-ID`.

---

## Router Integrations

### 1. Go Standard Library (`net/http`)

```go
mux := http.NewServeMux()
mux.HandleFunc("/process", processHandler)

handler := triage.Middleware("tr_live_key", "http://localhost:8080/api/v1/telemetry")(mux)
http.ListenAndServe(":8080", handler)
```

### 2. Chi (`go-chi/chi/v5`)

```go
r := chi.NewRouter()
r.Use(triage.Middleware("tr_live_key", "http://localhost:8080/api/v1/telemetry"))

r.Get("/items", getItemsHandler)
http.ListenAndServe(":8080", r)
```

### 3. Gin (`gin-gonic/gin`)

```go
r := gin.New()
// Use Gin's WrapH helper to adapt standard http.Handler middleware
r.Use(gin.WrapH(triage.Middleware("tr_live_key", "http://localhost:8080/api/v1/telemetry")(r)))

r.GET("/api/users", handleUsers)
r.Run(":8080")
```

### 4. Echo (`labstack/echo/v4`)

```go
e := echo.New()
e.Use(echo.WrapMiddleware(triage.Middleware("tr_live_key", "http://localhost:8080/api/v1/telemetry")))

e.GET("/ping", handlePing)
e.Start(":8080")
```

### 5. Fiber (`gofiber/fiber/v2`)

```go
app := fiber.New()
app.Use(adaptor.HTTPMiddleware(triage.Middleware("tr_live_key", "http://localhost:8080/api/v1/telemetry")))

app.Get("/stats", handleStats)
app.Listen(":8080")
```

---

## Trace Context & OpenTelemetry

Each recovered panic automatically propagates an OpenTelemetry-compatible trace ID:

- Header: `X-Triage-Trace-ID`
- Standard W3C header: `traceparent`

This allows you to correlate client error reports with the corresponding Triage dashboard incident.
