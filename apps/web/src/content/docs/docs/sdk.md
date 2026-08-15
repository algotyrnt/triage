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
		triage.WithGatewayURL("https://triage.yourcompany.com/api/v1/telemetry"),
		triage.WithRepo("myorg/myrepo"),
	)(mux)

	http.ListenAndServe(":8080", handler)
}
```

---

## Configuration Options

Pass functional options to `triage.Middleware()`:

| Option | Type | Description |
| :--- | :--- | :--- |
| `WithRepo(repo string)` | `string` | Sets the GitHub repository (`owner/repo`) for on-demand AST fetching. |
| `WithGatewayURL(url string)` | `string` | Custom engine telemetry URL (defaults to production managed gateway). |
| `WithCommit(sha string)` | `string` | Explicit Git commit SHA override (auto-detected via `debug.ReadBuildInfo()` if omitted). |
| `WithWorkerPool(workers, queue int)` | `int, int` | Configures async dispatch worker pool size (default: 4 workers, 1000 queue depth). |
| `WithCustomLogger(logger Logger)` | `Logger` | Custom logger interface for SDK debug logs. |

### Example with Full Options:

```go
handler := triage.Middleware(
	"tr_live_your_api_key",
	triage.WithRepo("myorg/myrepo"),
	triage.WithGatewayURL("http://localhost:8080/api/v1/telemetry"),
	triage.WithCommit("8f3a1b4c5d6e7f8091a2b3c4"),
	triage.WithWorkerPool(8, 2000),
)(mux)
```

---

## Router Integrations

### 1. Go Standard Library (`net/http`)

```go
mux := http.NewServeMux()
mux.HandleFunc("/process", processHandler)

handler := triage.Middleware("tr_live_key")(mux)
http.ListenAndServe(":8080", handler)
```

### 2. Chi (`go-chi/chi/v5`)

```go
r := chi.NewRouter()
r.Use(triage.Middleware("tr_live_key", triage.WithRepo("myorg/myrepo")))

r.Get("/items", getItemsHandler)
http.ListenAndServe(":8080", r)
```

### 3. Gin (`gin-gonic/gin`)

```go
r := gin.New()
// Use Gin's WrapH helper to adapt standard http.Handler middleware
r.Use(gin.WrapH(triage.Middleware("tr_live_key", triage.WithRepo("myorg/myrepo"))(r)))

r.GET("/api/users", handleUsers)
r.Run(":8080")
```

### 4. Echo (`labstack/echo/v4`)

```go
e := echo.New()
e.Use(echo.WrapMiddleware(triage.Middleware("tr_live_key", triage.WithRepo("myorg/myrepo"))))

e.GET("/ping", handlePing)
e.Start(":8080")
```

### 5. Fiber (`gofiber/fiber/v2`)

```go
app := fiber.New()
app.Use(adaptor.HTTPMiddleware(triage.Middleware("tr_live_key", triage.WithRepo("myorg/myrepo"))))

app.Get("/stats", handleStats)
app.Listen(":8080")
```

---

## Trace Context & OpenTelemetry

Each recovered panic automatically propagates an OpenTelemetry-compatible trace ID:
- Header: `X-Triage-Trace-ID`
- Standard W3C header: `traceparent`

This allows you to correlate client error reports with the corresponding Triage dashboard incident.
