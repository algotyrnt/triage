---
title: 5-Minute Quickstart
description: Get from installation to your first AI-diagnosed Go panic in under 5 minutes
---

Follow this guide to spin up a local Triage stack and capture your first Go crash with AI diagnostics.

## Prerequisites

- **Go 1.26+**
- **Docker & Docker Compose**
- An API key for your preferred AI provider (**Google Gemini**, **OpenAI**, **Anthropic Claude**) or a local instance (**Ollama**, **vLLM**)

---

## 1. Start the Triage Stack

Clone the repository and start the Docker Compose cluster:

```bash
git clone https://github.com/algotyrnt/triage.git
cd triage
docker compose up --build -d
```

This starts:

- **`triage-db` (`:5432`)**: PostgreSQL 16 database.
- **`triage-engine` (`:8080`)**: Core engine serving telemetry and REST APIs.
- **`triage-dashboard` (`:3000`)**: Studio Dashboard UI.

---

## 2. Run the Setup Wizard

Open [http://localhost:3000](http://localhost:3000) in your browser. The initial setup wizard will guide you through:

1. **GitHub App Manifest Creation** (One-click app registration).
2. **Repository Installation** (Grant access to your Go repositories).
3. **OAuth Linking** (Configure GitHub login).
4. **AI Model Configuration** (Select Google Gemini, OpenAI, Claude, or local Ollama).
5. **Verification** (Engine self-test).

Once configured, copy your project API key (e.g. `tr_live_...`).

---

## 3. Add Triage SDK to Your Go App

Install the official Go SDK:

```bash
go get github.com/algotyrnt/triage/sdk/go
```

Wrap your HTTP handler:

```go
package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()

	// An example endpoint that panics:
	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		var ptr *int
		*ptr = 42 // Nil pointer dereference panic!
	})

	// Wrap mux with Triage middleware:
	handler := triage.Middleware(
		"tr_live_your_api_key",
		"http://localhost:8080/api/v1/telemetry",
	)(mux)

	http.ListenAndServe(":8081", handler)
}
```

---

## 4. Trigger a Panic & View Diagnostics

Run your Go application (using `-trimpath` to generate production-accurate relative stack traces):

```bash
# Run test-services/simple-service on :8081 with your API key
cd test-services/simple-service
TRIAGE_API_KEY=tr_live_your_key go run -trimpath main.go
```

In another terminal, trigger the crash:

```bash
curl http://localhost:8081/crash
```

Your HTTP client will receive a generic `500 Internal Server Error` without any sensitive internals exposed.

Now switch back to your Studio Dashboard at **http://localhost:3000**:

1. You will see a new **CRITICAL** incident under your active project.
2. The isolated `*ast.FuncDecl` code block will be highlighted along with resolved cross-file structs and constructors.
3. AI diagnostics will display the exact root cause and suggested drop-in git patch.
4. Click **Generate Fix (PR)** to automatically create a branch and open a verified Pull Request on GitHub.

---

## Next Steps

- [Complete Go SDK Integration Guide](/docs/sdk)
- [AST Engine & Node Slicing](/docs/ast-engine)
- [Self-Hosting in Production](/docs/self-hosting)
