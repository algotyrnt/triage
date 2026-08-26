---
title: Authentication, RBAC & Team Onboarding
description: Comprehensive guide on Go engine-driven OAuth, cryptographic JWT session security, RBAC role tiers, and team member onboarding
---

Triage features a zero-trust, engine-driven identity and Role-Based Access Control (RBAC) system. All OAuth exchanges and token signing occur strictly on the backend Go Engine, ensuring GitHub Client Secrets and private keys never touch the browser.

---

## Architecture & Zero-Trust Security

```
Browser (Studio Dashboard)               Triage Go Engine (:8080)                  GitHub OAuth / API
          │                                         │                                      │
          │── 1. GET /api/v1/auth/github ──────────>│                                      │
          │<── 2. Set CSRF Cookie + Redirect ───────│                                      │
          │                                         │── 3. Authorize via GitHub ──────────>│
          │<────────────────────────────────────────┼── 4. Callback with ?code= & state ───│
          │── 5. GET /api/v1/auth/github/callback ─>│                                      │
          │                                         │── 6. Exchange code for access_token ─>│
          │                                         │<── 7. Access token + User Profile ───│
          │                                         │                                      │
          │                                         ├── 8. Upsert user in PostgreSQL       │
          │                                         ├── 9. Determine RBAC Role Tier        │
          │                                         ├── 10. Issue 30-day signed HS256 JWT  │
          │<── 11. Redirect with ?token=<JWT> ──────│                                      │
          │                                         │                                      │
          │── 12. GET /api/v1/auth/me (Bearer JWT) >│── Verify signature & return role ───>│
```

### Security Highlights

- **Zero Frontend Secret Exposure:** The frontend is a 100% static Single Page Application (SPA). OAuth client secrets and credentials remain securely locked in PostgreSQL.
- **Cryptographic CSRF Protection:** Authorization redirects set a random, short-lived, `HttpOnly` state nonce cookie validated during callback.
- **Stateless 30-Day HS256 JWTs:** The Engine signs user sessions using the instance's 256-bit `session_secret`. Tokens are validated in microseconds on each API request.

---

## Role-Based Access Control (RBAC) Tiers

Triage defines 4 distinct role levels to enforce least-privilege access:

| Capability                                       | Owner | Admin | Developer | Viewer |
| :----------------------------------------------- | :---: | :---: | :-------: | :----: |
| **View Incidents & Panic Timeline**              |  ✅   |  ✅   |    ✅     |   ✅   |
| **Inspect Isolated AST Snippets & Stack Traces** |  ✅   |  ✅   |    ✅     |   ✅   |
| **Simulate Crashes & Run Diagnostics**           |  ✅   |  ✅   |    ✅     |   ✅   |
| **Trigger AI Fixes & Open Bugfix PRs**           |  ✅   |  ✅   |    ✅     |   ❌   |
| **Create & Revoke Project API Keys**             |  ✅   |  ✅   |    ❌     |   ❌   |
| **Add / Delete Projects & Monorepo Modules**     |  ✅   |  ✅   |    ❌     |   ❌   |
| **Invite Team Members & Modify Roles**           |  ✅   |  ✅   |    ❌     |   ❌   |
| **Assign `Owner` Role & Revoke Member Access**   |  ✅   |  ❌   |    ❌     |   ❌   |

---

## First-User Setup (Owner Bootstrap)

When a fresh Triage instance is deployed:

1. The first team member who completes GitHub OAuth login is **automatically assigned the `Owner` role**.
2. The instance records this user as the primary administrator.
3. **Owner Protection Safeguards:** The engine enforces that the last remaining `Owner` cannot be demoted or deleted, preventing administrator lockouts.

---

## Onboarding Team Members

Organization administrators (`Owner` and `Admin`) can invite additional developers to the workspace.

### Step 1: Send an Invitation

1. Navigate to the **Team** screen in the top navigation bar.
2. Click **Invite GitHub User**.
3. Enter the collaborator's GitHub handle (e.g. `torvalds` or `devonvance`).
4. Select their starting role:
   - **Developer:** Can inspect panics, test modules, and trigger automated bugfix PRs.
   - **Admin:** Can manage projects, issue ingestion keys, and invite members.
   - **Viewer:** Read-only access to dashboards and incident metrics.
5. Click **Send Invitation**.

### Step 2: Automatic Invitation Claiming

- The invitation is persisted in PostgreSQL under the `invitations` table.
- When the invited user navigates to your Triage dashboard and clicks **Sign in with GitHub**, the Go Engine automatically matches their verified GitHub username against pending invitations.
- The user account is provisioned with the pre-assigned role, and the invitation is consumed.

---

## Managing Members & Permissions

Owners and Admins can manage organization access directly from the **Team** screen:

- **Role Updates:** Use the inline role dropdown in the member roster to promote or adjust permissions (`Admin` ↔ `Developer` ↔ `Viewer`).
- **Revoking Access:** Click **Revoke Access** on any member row to immediately remove their access and invalidate permissions.
- **Revoking Pending Invites:** In the _Pending Invitations_ banner, click **Revoke Invite** to cancel unaccepted invites before login.

---

## Programmatic API Access (Bearer Tokens)

All protected REST endpoints accept user session JWTs via standard HTTP headers:

```bash
curl -H "Authorization: Bearer <your_session_jwt>" \
     https://triage.yourcompany.com/api/v1/team/members
```
