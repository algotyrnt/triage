---
title: Troubleshooting & FAQ
description: Solutions for common edge cases including -trimpath build flags, AST symbolication, CORS errors, and reverse proxy setup
---

This guide provides direct solutions for common setup, telemetry, and AST extraction issues.

---

## 1. Using `-trimpath` (Recommended Best Practice)

While the Triage Engine includes smart path normalization to automatically match absolute file paths to repository files, compiling with `-trimpath` is a recommended Go best practice:

```bash
# During local testing:
go run -trimpath main.go

# When building container images or production binaries:
go build -trimpath -o server .
```

`-trimpath` strips developer-specific absolute file system prefixes from compiled binaries, producing clean relative package paths (e.g. `handlers/payment.go`) and ensuring reproducible stack traces across all development and production environments.

---

## 2. Empty AST Snippets in Incidents

### Symptoms

An incident is recorded, but the **AST Context** panel says _"No AST nodes found"_ or shows raw fallback code.

### Causes & Fixes

1. **GitHub App Permissions:** Ensure your GitHub App has **Repository Contents: Read & Write** permissions.
2. **Repository Access:** Ensure the GitHub App is installed on the specific organization and repository.
3. **Monorepo `root_dir` Mismatch:** If your Go service is in a subfolder, verify the project's `root_dir` matches your repository folder structure (e.g. `services/order-service`).
4. **Git Branch / Commit:** If telemetry is sent with a commit SHA that has not been pushed to GitHub, the GitHub Contents API cannot locate the file. Push your branch to GitHub before testing.

---

## 3. Telemetry Not Appearing in Dashboard

### Checklist

1. **Verify API Key:** Ensure the API key in your SDK matches an active, unrevoked key in your Triage project settings.
2. **Check Engine Connectivity:** Test whether your Go app can reach the engine:
   ```bash
   curl -I https://triage.yourcompany.com/health
   ```
3. **Inspect Server Logs:** Look for engine logs with `level=WARN` or `level=ERROR`. The engine logs unauthorized telemetry attempts with client IP addresses.

---

## 4. CORS Errors in Studio Dashboard

### Symptoms

Browser console reports: `Access to fetch has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present`.

### Cause

The Triage Server locks down CORS access strictly to your configured `instance_url` stored in the database.

### Fix

1. Open the Studio Dashboard **Settings** page.
2. Verify that **Instance URL** matches your actual browser domain (e.g. `https://triage.yourcompany.com` or `http://localhost:8080`).
3. If running locally, loopback origins (`localhost`, `127.0.0.1`) are permitted by default.

---

## 5. GitHub OAuth Redirect Loop or "Invalid State"

### Cause

When running behind a reverse proxy (e.g. Nginx, Caddy, Cloudflare, Traefik), SSL termination may strip headers or drop the CSRF state cookie if HTTPS headers are missing.

### Fix

Ensure your reverse proxy forwards standard proxy headers:

```nginx
# Nginx example:
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```
