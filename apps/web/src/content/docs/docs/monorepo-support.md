---
title: Monorepos & Multi-Module Architecture
description: Guide on setting up Go monorepos, nested go.mod detection, root_dir path normalization, and service-scoped API keys
---

Many modern Go engineering teams operate within a **monorepo**, where a single GitHub repository hosts multiple independent Go services (e.g. `services/order-service`, `apps/engine`, `workers/analytics`).

Triage provides native, first-class support for monorepos, automatically discovering submodules and normalizing file paths between Git commits and runtime stack traces.

---

## The Monorepo Problem in Crash Reporting

In a monorepo, Go runtime stack traces often look like this:

```
goroutine 42 [running]:
main.ProcessOrder(...)
    pkg/orders/order.go:45 +0x84
```

However, inside your Git repository, the actual file is located at:

```
services/order-service/pkg/orders/order.go
```

Without monorepo awareness, AST extractors and automated fixers fail to locate the file on GitHub.

### How Triage Solves This

Triage introduces **`root_dir` Path Normalization**:

1. When tracking a service, you specify its `root_dir` (e.g. `services/order-service`).
2. When a panic occurs, Triage normalizes the path (`root_dir` + `file_path`) before querying GitHub.
3. When Gemini AI generates a fix or opens a Pull Request, the diff is committed directly to the correct repository path (`services/order-service/pkg/orders/order.go`).

---

## Automatic Go Submodule Detection

When adding a new project through the Studio Dashboard Setup Wizard:

1. Enter your GitHub repository (e.g. `myorg/backend-platform`).
2. Triage automatically scans the repository for all nested `go.mod` files via `/api/v1/repos/detect-modules`.
3. A list of detected modules is presented:
   - `Root (/)`
   - `services/auth/`
   - `services/payment/`
   - `apps/engine/`
4. Select the specific service you want to track. Triage automatically sets the `root_dir` and provisions an isolated API key for that service.

---

## Tracking Multiple Services from One Repository

You can track as many independent services as you like from the same GitHub repository:

```
myorg/platform-monorepo (GitHub Repository)
    ├── services/auth-service     ──> Tracked as Project 1 (API Key: tr_live_auth_...)
    ├── services/order-service    ──> Tracked as Project 2 (API Key: tr_live_order_...)
    └── services/payment-service  ──> Tracked as Project 3 (API Key: tr_live_payment_...)
```

### Benefits of Service Scoping

- **Isolated Incident Timelines:** The dashboard incident feed can be filtered by specific service.
- **Independent API Key Revocation:** Revoking or rotating a key for `order-service` will not impact `payment-service`.
- **Targeted Pull Requests:** Automated bugfix PRs only modify files within the affected service's directory.

---

## SDK Configuration for Monorepos

Each service simply uses its assigned API key in the SDK middleware:

```go
package main

import (
    "net/http"
    triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
    mux := http.NewServeMux()
    // Register routes...

    // Use the specific API key provisioned for this monorepo service:
    handler := triage.Middleware(
        "tr_live_orders_1724123456",
        "https://triage.yourcompany.com/api/v1/telemetry",
    )(mux)

    http.ListenAndServe(":8081", handler)
}
```

The Triage engine automatically resolves the associated repository, `root_dir`, and permissions directly from the API key.
