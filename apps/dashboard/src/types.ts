/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

export type ScreenId =
  | 'setup'
  | 'login'
  | 'new'
  | 'dashboard'
  | 'incident_detail'
  | 'ast'
  | 'webhooks'
  | 'team'
  | 'status'
  | 'settings';

export type IncidentStatus = 'CRITICAL' | 'INVESTIGATING' | 'RESOLVED';

export interface Incident {
  id: string; // e.g. "INC-8094"
  title: string; // e.g. "nil pointer dereference in GetProfile()"
  status: IncidentStatus;
  triggeringFile: string; // e.g. "pkg/handler/user.go:42"
  triggeringLine: number; // e.g. 42
  latencyMs: number; // e.g. 740
  commitHash: string; // e.g. "8f3a1b4"
  branch: string; // e.g. "main"
  timestamp: string; // e.g. "2026-07-28 12:04:18 UTC"
  goroutineId: string; // e.g. "goroutine 42 [running]"
  panicMessage: string; // e.g. "panic: runtime error: invalid memory address or nil pointer dereference"
  rawStackTrace: string;
  githubIssueUrl?: string;
  githubIssueNumber?: number; // e.g. 104
  githubPrUrl?: string;
  githubPrNumber?: number; // e.g. 12
  astSnippet: {
    functionName: string; // e.g. "GetProfile"
    file: string;
    startLine: number;
    lines: { lineNum: number; content: string; isTriggerLine?: boolean }[];
  };
  geminiAnalysis?: {
    rootCause: string;
    explanation: string;
    severity: 'CRITICAL' | 'HIGH' | 'MEDIUM';
    recommendedFix: string;
  };
  suggestedPatch?: string;
}

export type AstNodeKind =
  | 'FuncDecl'
  | 'TypeSpec'
  | 'FieldList'
  | 'BlockStmt'
  | 'AssignStmt'
  | 'CallExpr'
  | 'SelectorExpr'
  | 'ReturnStmt';

export interface AstNode {
  id: string;
  name: string;
  kind: AstNodeKind;
  pos: number;
  end: number;
  line: number;
  signature?: string;
  receiver?: string;
  children?: AstNode[];
}

export interface AstFile {
  path: string;
  name: string;
  isDir?: boolean;
  totalFuncs?: number;
  totalLines?: number;
  sizeBytes?: number;
  nodes?: AstNode[];
  children?: AstFile[];
}

export interface AstCommitIndex {
  commitHash: string;
  branch: string;
  parsedFilesCount: number;
  totalFunctionsCount: number;
  status: 'INDEXED' | 'PARSING' | 'FAILED';
  indexedAt: string;
}

export interface WebhookLog {
  id: string;
  status: 'SUCCESS' | 'UNAUTHORIZED' | 'ERROR';
  statusCode: number; // e.g. 200, 401, 502
  eventType: 'panic.ingested' | 'ast.reindexed' | 'alert.triggered';
  sourceIp: string;
  timestamp: string;
  latencyMs: number;
  headers: Record<string, string>;
  requestBody: string;
  responseBody: string;
}

export interface TeamMember {
  id: string;
  name: string;
  githubUsername: string;
  avatarUrl?: string;
  role: 'Owner' | 'Admin' | 'Member' | 'Read-Only';
  scopes: string[];
  lastActive: string;
  mfaEnabled: boolean;
}

export interface ApiKey {
  id: string;
  name: string;
  keyMasked: string;
  fullKey?: string;
  createdAt: string;
  lastUsed?: string;
  status: 'ACTIVE' | 'REVOKED' | 'EXPIRED';
}

export interface MetricHourly {
  hourLabel: string; // e.g. "08:00"
  panicCount: number;
  avgLatencyMs: number;
  astIndexTimeMs: number;
}

export interface SystemHealthComponent {
  name: string;
  service: string;
  status: 'OPERATIONAL' | 'DEGRADED' | 'DOWN';
  latency: string;
  detail: string;
}

export interface Project {
  id: string;
  owner: string;
  repo: string;
  root_dir?: string;
  installation_id?: number;
  api_key_masked?: string;
  created_at?: string;
}

export interface DetectedModule {
  path: string;
  name: string;
  is_root: boolean;
}

export interface RepositoryItem {
  owner: string;
  repo: string;
  name: string;
  installed: boolean;
  branch?: string;
  lang?: string;
  visibility?: string;
  private?: boolean;
}
