/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import { Incident, AstCommitIndex, AstFile, WebhookLog, TeamMember, ApiKey, MetricHourly, SystemHealthComponent } from '../types';

export const GEMINI_MODEL_NAME = 'gemini-3.6-flash';

export const INITIAL_INCIDENTS: Incident[] = [
  {
    id: 'INC-8094',
    title: 'nil pointer dereference in ChargeCart()',
    status: 'CRITICAL',
    triggeringFile: 'pkg/handler/checkout.go:58',
    triggeringLine: 58,
    latencyMs: 14,
    commitHash: '8f3a1b4',
    branch: 'main',
    timestamp: '2026-07-28 14:22:04 UTC',
    goroutineId: 'goroutine 54 [running]',
    panicMessage: 'panic: runtime error: invalid memory address or nil pointer dereference',
    rawStackTrace: `goroutine 54 [running]:
pkg/handler.(*CheckoutHandler).ChargeCart(0x0, 0xc0000a2000)
	/workspace/pkg/handler/checkout.go:58 +0x42
net/http.HandlerFunc.ServeHTTP(0x1028e3b20?, {0x12995dae8, 0x102893268}, 0x1400018a000?)
	/usr/local/go/src/net/http/server.go:2166 +0x38
net/http.(*ServeMux).ServeHTTP(0x102894540?, {0x12995dae8, 0x102893268}, 0x1400018a000)
	/usr/local/go/src/net/http/server.go:2683 +0x1b8`,
    astSnippet: {
      functionName: 'ChargeCart',
      file: 'pkg/handler/checkout.go',
      startLine: 54,
      lines: [
        { lineNum: 54, content: 'func (c *CheckoutHandler) ChargeCart(w http.ResponseWriter, r *http.Request) {' },
        { lineNum: 55, content: '	cartID := r.Header.Get("X-Cart-ID")' },
        { lineNum: 56, content: '	ctx := r.Context()' },
        { lineNum: 57, content: '	// Unchecked pointer dereference on PaymentGateway' },
        { lineNum: 58, content: '	order, err := c.PaymentGateway.ChargeCart(ctx, cartID)', isTriggerLine: true },
        { lineNum: 59, content: '	if err != nil { http.Error(w, err.Error(), 500); return }' },
        { lineNum: 60, content: '	json.NewEncoder(w).Encode(order)' },
        { lineNum: 61, content: '}' },
      ],
    },
    geminiAnalysis: {
      rootCause: 'Uninitialized Receiver Pointer (PaymentGateway)',
      explanation:
        'The receiver method `ChargeCart` attempted to dereference field `c.PaymentGateway`, which was nil when invoked from HTTP route handler.',
      severity: 'CRITICAL',
      recommendedFix:
        'Add a defensive nil check for `c.PaymentGateway` before dereferencing, or ensure dependency injection initializes `PaymentGateway` during `NewCheckoutHandler()` bootstrap.',
    },
    suggestedPatch: `--- a/pkg/handler/checkout.go
+++ b/pkg/handler/checkout.go
@@ -57,3 +57,6 @@ func (c *CheckoutHandler) ChargeCart(w http.ResponseWriter, r *http.Request) {
+	if c.PaymentGateway == nil {
+		http.Error(w, "payment gateway uninitialized", http.StatusInternalServerError)
+		return
+	}
 	order, err := c.PaymentGateway.ChargeCart(ctx, cartID)`,
    githubIssueUrl: 'https://github.com/algotyrnt/beacon-app/issues/104',
    githubIssueNumber: 104,
  },
  {
    id: 'INC-8091',
    title: 'index out of range [3] with length 3 in ParseHeaders()',
    status: 'INVESTIGATING',
    triggeringFile: 'pkg/api/router.go:112',
    triggeringLine: 112,
    latencyMs: 18,
    commitHash: '8f3a1b4',
    branch: 'main',
    timestamp: '2026-07-28 12:08:19 UTC',
    goroutineId: 'goroutine 12 [running]',
    panicMessage: 'panic: runtime error: index out of range [3] with length 3',
    rawStackTrace: `goroutine 12 [running]:
pkg/api.ParseHeaders({0xc000120100, 0x3, 0x3})
	/workspace/pkg/api/router.go:112 +0x11a
pkg/api.(*Router).ServeHTTP(0xc000098000, {0x12995dae8, 0x102893268}, 0x1400018a000)
	/workspace/pkg/api/router.go:45 +0x62`,
    astSnippet: {
      functionName: 'ParseHeaders',
      file: 'pkg/api/router.go',
      startLine: 108,
      lines: [
        { lineNum: 108, content: 'func ParseHeaders(parts []string) map[string]string {' },
        { lineNum: 109, content: '	headers := make(map[string]string)' },
        { lineNum: 110, content: '	for i := 0; i <= len(parts); i++ {' },
        { lineNum: 111, content: '		// Off-by-one comparison (i <= len(parts))' },
        { lineNum: 112, content: '		headers[fmt.Sprintf("H-%d", i)] = parts[i]', isTriggerLine: true },
        { lineNum: 113, content: '	}' },
        { lineNum: 114, content: '	return headers' },
        { lineNum: 115, content: '}' },
      ],
    },
    geminiAnalysis: {
      rootCause: 'Off-By-One Slice Bounds Access',
      explanation:
        'The loop condition `i <= len(parts)` evaluates to true when `i == len(parts)`, causing a slice index out of bounds panic.',
      severity: 'HIGH',
      recommendedFix: 'Change loop bounds condition to `i < len(parts)`.',
    },
    suggestedPatch: `--- a/pkg/api/router.go
+++ b/pkg/api/router.go
@@ -110,1 +110,1 @@ func ParseHeaders(parts []string) map[string]string {
-	for i := 0; i <= len(parts); i++ {
+	for i := 0; i < len(parts); i++ {`,
    githubIssueUrl: 'https://github.com/algotyrnt/beacon-app/issues/102',
    githubIssueNumber: 102,
  },
  {
    id: 'INC-8088',
    title: 'interface conversion: interface {} is nil, not string',
    status: 'RESOLVED',
    triggeringFile: 'pkg/auth/token.go:42',
    triggeringLine: 42,
    latencyMs: 9,
    commitHash: '2c9e4a1',
    branch: 'main',
    timestamp: '2026-07-27 18:45:00 UTC',
    goroutineId: 'goroutine 88 [running]',
    panicMessage: 'panic: interface conversion: interface {} is nil, not string',
    rawStackTrace: `goroutine 88 [running]:
pkg/auth.ExtractUserID({0x129910a20, 0x0})
	/workspace/pkg/auth/token.go:42 +0x88`,
    astSnippet: {
      functionName: 'ExtractUserID',
      file: 'pkg/auth/token.go',
      startLine: 38,
      lines: [
        { lineNum: 38, content: 'func ExtractUserID(ctx context.Context) string {' },
        { lineNum: 39, content: '	val := ctx.Value("user_id")' },
        { lineNum: 40, content: '	// Unsafe type assertion without comma-ok check' },
        { lineNum: 41, content: '	return val.(string)' },
        { lineNum: 42, content: '}', isTriggerLine: true },
      ],
    },
    geminiAnalysis: {
      rootCause: 'Unchecked Type Assertion on Nil Interface',
      explanation: 'The context value `user_id` was nil, causing a panic during string type assertion.',
      severity: 'MEDIUM',
      recommendedFix: 'Use comma-ok type assertion `uid, ok := val.(string)` and handle `!ok`.',
    },
    suggestedPatch: `--- a/pkg/auth/token.go
+++ b/pkg/auth/token.go
@@ -41,1 +41,4 @@ func ExtractUserID(ctx context.Context) string {
-	return val.(string)
+	if uid, ok := val.(string); ok {
+		return uid
+	}
+	return ""`,
    githubIssueUrl: 'https://github.com/algotyrnt/beacon-app/issues/98',
    githubIssueNumber: 98,
  },
];

export const MOCK_COMMIT_INDEXES: AstCommitIndex[] = [
  {
    commitHash: '8f3a1b4',
    branch: 'main',
    parsedFilesCount: 42,
    totalFunctionsCount: 1420,
    indexedAt: '2026-07-28 14:20:11 UTC',
    status: 'INDEXED',
  },
  {
    commitHash: '2c9e4a1',
    branch: 'main',
    parsedFilesCount: 41,
    totalFunctionsCount: 1408,
    indexedAt: '2026-07-27 18:00:00 UTC',
    status: 'INDEXED',
  },
  {
    commitHash: '1a8f902',
    branch: 'dev/auth-v2',
    parsedFilesCount: 39,
    totalFunctionsCount: 1395,
    indexedAt: '2026-07-25 09:12:44 UTC',
    status: 'INDEXED',
  },
];

export const MOCK_AST_FILES: AstFile[] = [
  {
    name: 'pkg',
    path: 'pkg',
    isDir: true,
    children: [
      {
        name: 'handler',
        path: 'pkg/handler',
        isDir: true,
        children: [
          {
            name: 'checkout.go',
            path: 'pkg/handler/checkout.go',
            isDir: false,
            totalLines: 120,
            totalFuncs: 4,
            sizeBytes: 3840,
            nodes: [
              {
                id: 'ast-node-1',
                kind: 'FuncDecl',
                name: 'ChargeCart',
                receiver: '(c *CheckoutHandler)',
                signature: 'func (c *CheckoutHandler) ChargeCart(w http.ResponseWriter, r *http.Request)',
                line: 54,
                pos: 1480,
                end: 1820,
                children: [
                  { id: 'ast-sub-1', kind: 'CallExpr', name: 'r.Header.Get("X-Cart-ID")', line: 55, pos: 1510, end: 1545 },
                  { id: 'ast-sub-2', kind: 'SelectorExpr', name: 'c.PaymentGateway.ChargeCart', line: 58, pos: 1600, end: 1650 },
                ],
              },
              {
                id: 'ast-node-2',
                kind: 'FuncDecl',
                name: 'NewCheckoutHandler',
                signature: 'func NewCheckoutHandler(gw PaymentGateway) *CheckoutHandler',
                line: 18,
                pos: 420,
                end: 680,
              },
            ],
          },
          {
            name: 'user.go',
            path: 'pkg/handler/user.go',
            isDir: false,
            totalLines: 185,
            totalFuncs: 6,
            sizeBytes: 5210,
            nodes: [
              {
                id: 'ast-node-3',
                kind: 'FuncDecl',
                name: 'GetUserProfile',
                receiver: '(h *UserHandler)',
                signature: 'func (h *UserHandler) GetUserProfile(w http.ResponseWriter, r *http.Request)',
                line: 32,
                pos: 890,
                end: 1240,
              },
            ],
          },
        ],
      },
      {
        name: 'api',
        path: 'pkg/api',
        isDir: true,
        children: [
          {
            name: 'router.go',
            path: 'pkg/api/router.go',
            isDir: false,
            totalLines: 140,
            totalFuncs: 5,
            sizeBytes: 4120,
            nodes: [
              {
                id: 'ast-node-4',
                kind: 'FuncDecl',
                name: 'ParseHeaders',
                signature: 'func ParseHeaders(parts []string) map[string]string',
                line: 108,
                pos: 3100,
                end: 3340,
              },
            ],
          },
        ],
      },
      {
        name: 'auth',
        path: 'pkg/auth',
        isDir: true,
        children: [
          {
            name: 'token.go',
            path: 'pkg/auth/token.go',
            isDir: false,
            totalLines: 95,
            totalFuncs: 3,
            sizeBytes: 2890,
            nodes: [
              {
                id: 'ast-node-5',
                kind: 'FuncDecl',
                name: 'ExtractUserID',
                signature: 'func ExtractUserID(ctx context.Context) string',
                line: 38,
                pos: 1100,
                end: 1280,
              },
            ],
          },
        ],
      },
    ],
  },
  {
    name: 'main.go',
    path: 'main.go',
    isDir: false,
    totalLines: 48,
    totalFuncs: 1,
    sizeBytes: 1240,
    nodes: [
      {
        id: 'ast-node-6',
        kind: 'FuncDecl',
        name: 'main',
        signature: 'func main()',
        line: 12,
        pos: 240,
        end: 890,
      },
    ],
  },
];

export const MOCK_WEBHOOK_LOGS: WebhookLog[] = [
  {
    id: 'wh-9022',
    status: 'SUCCESS',
    statusCode: 200,
    eventType: 'panic.ingested',
    sourceIp: '34.120.45.12',
    timestamp: '2026-07-28 14:22:05.142 UTC',
    latencyMs: 14,
    headers: {
      'Content-Type': 'application/json',
      'X-Triage-Signature': 'sha256=8f9a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a',
      'User-Agent': 'Triage-Go-SDK/v1.2.0',
    },
    requestBody: JSON.stringify(
      {
        api_key: 'trj_demo_XXXXXXXXXXXX',
        repo: 'algotyrnt/beacon-app',
        panic_message: 'panic: runtime error: invalid memory address or nil pointer dereference',
        file: 'pkg/handler/checkout.go',
        line: 58,
        goroutine: 'goroutine 54 [running]',
      },
      null,
      2
    ),
    responseBody: JSON.stringify(
      {
        incident_id: 'INC-8094',
        status: 'CRITICAL',
        ast_symbolicated: true,
        gemini_model: GEMINI_MODEL_NAME,
        github_issue_created: 104,
      },
      null,
      2
    ),
  },
  {
    id: 'wh-9021',
    status: 'SUCCESS',
    statusCode: 200,
    eventType: 'ast.reindexed',
    sourceIp: '140.82.112.4',
    timestamp: '2026-07-28 14:20:12.004 UTC',
    latencyMs: 85,
    headers: {
      'Content-Type': 'application/json',
      'X-GitHub-Event': 'push',
      'X-Hub-Signature-256': 'sha256=4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b',
    },
    requestBody: JSON.stringify(
      {
        ref: 'refs/heads/main',
        head_commit: {
          id: '8f3a1b4',
          message: 'fix: update user handler dependencies',
          author: { name: 'algotyrnt', email: 'algotyrnt@example.com' },
        },
      },
      null,
      2
    ),
    responseBody: JSON.stringify({ status: 'reindexed', total_funcs: 1420 }, null, 2),
  },
  {
    id: 'wh-9019',
    status: 'UNAUTHORIZED',
    statusCode: 401,
    eventType: 'panic.ingested',
    sourceIp: '198.51.100.44',
    timestamp: '2026-07-28 10:15:22.080 UTC',
    latencyMs: 3,
    headers: {
      'Content-Type': 'application/json',
      'X-Triage-Signature': 'invalid_secret_key_hash',
    },
    requestBody: JSON.stringify({ error: 'invalid payload key signature' }, null, 2),
    responseBody: JSON.stringify({ error: '401 Unauthorized: Invalid API Key Signature' }, null, 2),
  },
];

export const MOCK_TEAM_MEMBERS: TeamMember[] = [
  {
    id: 'usr-1',
    name: 'Punjitha (algotyrnt)',
    githubUsername: 'algotyrnt',
    avatarUrl: 'https://github.com/algotyrnt.png',
    role: 'Owner',
    scopes: ['repo:read', 'ast:write', 'incidents:manage', 'billing:admin'],
    lastActive: 'Active now',
    mfaEnabled: true,
  },
  {
    id: 'usr-2',
    name: 'Devon Vance',
    githubUsername: 'devonvance-go',
    role: 'Admin',
    scopes: ['repo:read', 'ast:write', 'incidents:manage'],
    lastActive: '2 hours ago',
    mfaEnabled: true,
  },
  {
    id: 'usr-3',
    name: 'Elena Rostova',
    githubUsername: 'erostova-sys',
    role: 'Member',
    scopes: ['repo:read', 'incidents:read'],
    lastActive: 'Yesterday',
    mfaEnabled: false,
  },
];

export const MOCK_API_KEYS: ApiKey[] = [
  {
    id: 'key-1',
    name: 'Production Cloud Run Engine Key',
    keyMasked: 'trj_demo_XXXXXXXX...8f',
    fullKey: 'trj_demo_XXXXXXXXXXXX',
    createdAt: '2026-05-10',
    lastUsed: '14 seconds ago',
    status: 'ACTIVE',
  },
  {
    id: 'key-2',
    name: 'Staging K8s Ingress Key',
    keyMasked: 'trj_stage_4a1b2c3d...90e',
    fullKey: 'trj_stage_4a1b2c3d4e5f6a7b8c90e',
    createdAt: '2026-06-15',
    lastUsed: '3 days ago',
    status: 'ACTIVE',
  },
];

export const MOCK_HOURLY_METRICS: MetricHourly[] = [
  { hourLabel: '00:00', panicCount: 1, avgLatencyMs: 140, astIndexTimeMs: 24 },
  { hourLabel: '02:00', panicCount: 0, avgLatencyMs: 0, astIndexTimeMs: 22 },
  { hourLabel: '04:00', panicCount: 2, avgLatencyMs: 210, astIndexTimeMs: 26 },
  { hourLabel: '06:00', panicCount: 0, avgLatencyMs: 0, astIndexTimeMs: 20 },
  { hourLabel: '08:00', panicCount: 4, avgLatencyMs: 180, astIndexTimeMs: 25 },
  { hourLabel: '10:00', panicCount: 3, avgLatencyMs: 320, astIndexTimeMs: 29 },
  { hourLabel: '12:00', panicCount: 2, avgLatencyMs: 740, astIndexTimeMs: 28 },
];

export const MOCK_SYSTEM_HEALTH: SystemHealthComponent[] = [
  {
    name: 'Go Runtime Ingestion Engine',
    service: 'Cloud Run / GCP us-central1',
    status: 'OPERATIONAL',
    latency: '14ms',
    detail: 'Symbolication worker pool healthy (12/12 pods active)',
  },
  {
    name: 'GCS AST Storage & Byte Indexer',
    service: 'Google Cloud Storage / AST Cache',
    status: 'OPERATIONAL',
    latency: '8ms',
    detail: '1,420 Go function signatures indexed for commit 8f3a1b4',
  },
  {
    name: `${GEMINI_MODEL_NAME} Diagnostic Client`,
    service: 'Gemini AI API Proxy',
    status: 'OPERATIONAL',
    latency: '420ms',
    detail: 'Root Cause & Patch Generator operational',
  },
];
