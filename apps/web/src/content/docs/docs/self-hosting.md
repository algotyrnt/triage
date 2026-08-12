---
title: Self-Hosting Guide
description: Deploy single-container triage engine on GCP Cloud Run, Docker, or Kubernetes
---

Self-hosting **triage** requires only **1 single Docker container** (`triage/engine`).

## Docker Quickstart

```bash
docker run -d \
  --name triage-engine \
  -p 8080:8080 \
  -e GEMINI_API_KEY="your_gemini_api_key" \
  -e TRIAGE_API_KEY="tr_test_key_9042" \
  triage/engine:latest
```

Open `http://localhost:8080/dashboard` in your browser to access the Studio Dashboard UI!
