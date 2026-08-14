---
title: Go SDK Integration Guide
description: Integrate triage panic interceptor middleware into any Go HTTP router
---

Add the `triage/sdk/go` panic interceptor middleware to your HTTP routers in a single line.

## Installation

```bash
go get github.com/algotyrnt/triage/sdk/go
```

## Middleware Setup

```go
package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Managed Usage:
	handler := triage.Middleware("tr_live_key_123", triage.WithRepo("myorg/myrepo"))(mux)

	// Self-Hosted Usage:
	// handler := triage.Middleware(
	// 	"tr_selfhosted_key",
	// 	triage.WithGatewayURL("http://localhost:8080/api/v1/telemetry"),
	// 	triage.WithRepo("myorg/myrepo"),
	// )(mux)

	http.ListenAndServe(":8081", handler)
}
```
