/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import { logger } from './logger';

export interface TelemetryPayload {
  api_key: string;
  file: string;
  line: number;
  stack_trace: string;
  github_owner?: string;
  github_repo?: string;
  installation_id?: number;
}

export interface AnalysisResult {
  root_cause: string;
  suggested_fix: string;
}

export interface GithubIssueResponse {
  number: number;
  html_url: string;
  state: string;
  title: string;
}

export interface TelemetryResponse {
  status: string;
  ast?: string;
  analysis?: AnalysisResult;
  github_issue?: GithubIssueResponse;
  error?: string;
}

export interface EngineStatus {
  online: boolean;
  url: string;
  latencyMs: number;
}

export class EngineApiError extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly data?: any;

  constructor(message: string, status = 500, statusText = 'Internal Error', data?: any) {
    super(message);
    this.name = 'EngineApiError';
    this.status = status;
    this.statusText = statusText;
    this.data = data;
    Object.setPrototypeOf(this, EngineApiError.prototype);
  }
}

function resolveDefaultBaseUrl(providedUrl?: string): string {
  if (providedUrl && providedUrl !== 'undefined' && !providedUrl.startsWith('undefined')) {
    return providedUrl.endsWith('/api/v1')
      ? providedUrl
      : `${providedUrl.replace(/\/$/, '')}/api/v1`;
  }

  if (typeof window !== 'undefined') {
    try {
      const stored = localStorage.getItem('triage_engine_url');
      if (stored && stored !== 'undefined') {
        return stored.endsWith('/api/v1') ? stored : `${stored.replace(/\/$/, '')}/api/v1`;
      }
    } catch (e) {
      logger.debug('Failed to read triage_engine_url from localStorage', e);
    }

    // In browser (embedded in Go engine or via Vite proxy), relative '/api/v1' works natively
    return '/api/v1';
  }

  return '/api/v1';
}

export class EngineClient {
  private baseUrl: string;
  private authToken: string | null = null;

  constructor(baseUrl?: string) {
    this.baseUrl = resolveDefaultBaseUrl(baseUrl);
  }

  setBaseUrl(url: string) {
    this.baseUrl = resolveDefaultBaseUrl(url);
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  getTelemetryUrl(): string {
    return `${this.baseUrl}/telemetry`;
  }

  getEventsStreamUrl(): string {
    return `${this.baseUrl}/events/stream`;
  }

  setAuthToken(token: string | null) {
    this.authToken = token;
  }

  private getAuthHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    const token =
      this.authToken ||
      (typeof window !== 'undefined' ? localStorage.getItem('triage_session') : null);
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  /**
   * Centralized, type-safe HTTP client with built-in error extraction,
   * authentication headers, query parameters, and optional fallbacks.
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit & {
      params?: Record<string, string | number | boolean | undefined>;
      skipAuth?: boolean;
      fallback?: T;
    } = {},
  ): Promise<T> {
    const { params, skipAuth, fallback, ...fetchOptions } = options;

    let url = endpoint.startsWith('http')
      ? endpoint
      : `${this.baseUrl}${endpoint.startsWith('/') ? '' : '/'}${endpoint}`;
    if (params) {
      const searchParams = new URLSearchParams();
      for (const [key, val] of Object.entries(params)) {
        if (val !== undefined && val !== '') {
          searchParams.set(key, String(val));
        }
      }
      const qs = searchParams.toString();
      if (qs) {
        url += (url.includes('?') ? '&' : '?') + qs;
      }
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(skipAuth ? {} : this.getAuthHeaders()),
      ...((fetchOptions.headers as Record<string, string>) || {}),
    };

    try {
      const res = await fetch(url, {
        ...fetchOptions,
        headers,
      });

      // Parse JSON safely
      let data: any = {};
      const text = await res.text();
      if (text && text.trim()) {
        try {
          data = JSON.parse(text);
        } catch {
          data = { text };
        }
      }

      if (!res.ok) {
        const errorMsg =
          data?.error ||
          data?.message ||
          data?.text ||
          `${res.statusText || 'Error'} (HTTP ${res.status})`;
        if (fallback !== undefined) {
          logger.warn(`API ${endpoint} (${res.status}): ${errorMsg}`);
          return fallback;
        }
        throw new EngineApiError(errorMsg, res.status, res.statusText, data);
      }

      return data as T;
    } catch (err: any) {
      if (fallback !== undefined) {
        logger.warn(`API ${endpoint} request failed:`, err?.message || err);
        return fallback;
      }
      if (err instanceof EngineApiError) throw err;
      logger.error(`API ${endpoint} error:`, err);
      throw new EngineApiError(err?.message || 'Network request failed', 0, 'Network Error');
    }
  }

  // ---------------------------------------------------------------------------
  // Core Diagnostics & Telemetry
  // ---------------------------------------------------------------------------

  async checkStatus(): Promise<EngineStatus> {
    const startTime = Date.now();
    const telemetryUrl = this.getTelemetryUrl();
    try {
      const response = await fetch(telemetryUrl, { method: 'OPTIONS' });
      return {
        online: response.ok || response.status === 405 || response.status === 200,
        url: telemetryUrl,
        latencyMs: Date.now() - startTime,
      };
    } catch {
      return { online: false, url: telemetryUrl, latencyMs: 0 };
    }
  }

  async sendTelemetry(payload: TelemetryPayload): Promise<TelemetryResponse> {
    return this.request<TelemetryResponse>('/telemetry', {
      method: 'POST',
      body: JSON.stringify(payload),
      skipAuth: true,
    });
  }

  async getIncidents(): Promise<any[]> {
    const res = await this.request<{ incidents?: any[] }>('/incidents', {
      fallback: { incidents: [] },
    });
    return res.incidents || [];
  }

  async getStats(): Promise<any> {
    return this.request('/stats', { fallback: null });
  }

  async getHealth(): Promise<{ status: string; version?: string; database?: string } | null> {
    const healthUrl = `${this.baseUrl.replace(/\/api\/v1\/?$/, '')}/health`;
    return this.request(healthUrl, {
      fallback: null,
      skipAuth: true,
    });
  }

  async getEngineVersion(): Promise<string | null> {
    const stats = await this.getStats();
    if (stats?.version) return stats.version;
    const health = await this.getHealth();
    return health?.version || null;
  }

  // ---------------------------------------------------------------------------
  // Projects & API Keys
  // ---------------------------------------------------------------------------

  async getProjects(): Promise<any[]> {
    const res = await this.request<{ projects?: any[] }>('/projects', {
      fallback: { projects: [] },
    });
    return res.projects || [];
  }

  async createProject(
    repo: string,
    rootDir?: string,
    ownerUsername?: string,
    projectContext?: string,
  ): Promise<{
    success: boolean;
    repo: string;
    root_dir?: string;
    context?: string;
    api_key: string;
    key_masked?: string;
  }> {
    return this.request('/projects', {
      method: 'POST',
      body: JSON.stringify({
        repo,
        root_dir: rootDir || '',
        owner_username: ownerUsername || '',
        context: projectContext || '',
      }),
    });
  }

  async updateProjectContext(
    owner: string,
    repo: string,
    rootDir?: string,
    projectContext?: string,
  ): Promise<{ success: boolean; error?: string }> {
    try {
      await this.request('/projects/context', {
        method: 'PUT',
        body: JSON.stringify({
          owner,
          repo,
          root_dir: rootDir || '',
          context: projectContext || '',
        }),
      });
      return { success: true };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async getProjectKeys(
    owner?: string,
    repo?: string,
    rootDir?: string,
  ): Promise<import('@/types').ApiKey[]> {
    const res = await this.request<{ keys?: any[] }>('/projects/keys', {
      params: { owner, repo, root_dir: rootDir },
      fallback: { keys: [] },
    });
    if (!Array.isArray(res.keys)) return [];
    return res.keys.map((k: any) => ({
      id: k.id,
      name: k.name || 'API Key',
      keyMasked: k.key_masked || '...xxxx',
      fullKey: k.raw_key || undefined,
      createdAt: k.created_at ? new Date(k.created_at).toISOString().split('T')[0] : 'Recently',
      lastUsed: 'Recently',
      status: (k.status === 'REVOKED' ? 'REVOKED' : 'ACTIVE') as 'ACTIVE' | 'REVOKED',
    }));
  }

  async createApiKey(
    owner: string,
    repo: string,
    rootDir?: string,
    name?: string,
  ): Promise<{
    success: boolean;
    key?: import('@/types').ApiKey & { raw_key?: string };
    error?: string;
  }> {
    try {
      const data = await this.request<{ key: any }>('/projects/keys', {
        method: 'POST',
        body: JSON.stringify({
          owner,
          repo,
          root_dir: rootDir || '',
          name: name || `Key for ${repo}`,
        }),
      });
      return {
        success: true,
        key: {
          id: data.key.id,
          name: data.key.name,
          keyMasked: data.key.key_masked,
          fullKey: data.key.raw_key,
          raw_key: data.key.raw_key,
          createdAt: data.key.created_at
            ? new Date(data.key.created_at).toISOString().split('T')[0]
            : 'Today',
          lastUsed: 'Never',
          status: 'ACTIVE',
        },
      };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async revokeApiKey(keyId: string): Promise<boolean> {
    try {
      await this.request('/projects/keys/revoke', {
        method: 'POST',
        body: JSON.stringify({ key_id: keyId }),
      });
      return true;
    } catch {
      return false;
    }
  }

  // ---------------------------------------------------------------------------
  // AST & Codebase Analysis
  // ---------------------------------------------------------------------------

  async detectGoModules(
    owner: string,
    repo: string,
  ): Promise<{ path: string; name: string; is_root: boolean }[]> {
    const fallback = [{ path: '', name: 'Repository Root (/)', is_root: true }];
    const res = await this.request<{ modules?: any[] }>('/repos/detect-modules', {
      params: { owner, repo },
      fallback: { modules: fallback },
    });
    return Array.isArray(res.modules) && res.modules.length > 0 ? res.modules : fallback;
  }

  async getASTTree(
    owner: string,
    repo: string,
    rootDir?: string,
  ): Promise<{ status: string; files: any[]; total: number } | null> {
    return this.request('/ast/tree', {
      params: { owner, repo, root_dir: rootDir },
      fallback: null,
    });
  }

  async indexAST(
    owner: string,
    repo: string,
    commit = 'main',
    rootDir = '',
  ): Promise<{ status: string; indexed_count?: number; error?: string } | null> {
    try {
      return await this.request('/ast/index', {
        method: 'POST',
        body: JSON.stringify({ owner, repo, commit, root_dir: rootDir }),
      });
    } catch (err: any) {
      return { status: 'error', error: err.message };
    }
  }

  // ---------------------------------------------------------------------------
  // AI Diagnostics & Remediation
  // ---------------------------------------------------------------------------

  async analyzePanic(params: {
    panicMessage: string;
    rawStackTrace: string;
    triggeringFile: string;
    astCode: string;
    projectContext?: string;
  }): Promise<{
    success: boolean;
    rootCause?: string;
    explanation?: string;
    severity?: string;
    recommendedFix?: string;
    error?: string;
  }> {
    try {
      const data = await this.request<any>('/llm/analyze-panic', {
        method: 'POST',
        body: JSON.stringify(params),
      });
      return {
        success: true,
        rootCause: data.rootCause,
        explanation: data.explanation,
        severity: data.severity || 'CRITICAL',
        recommendedFix: data.recommendedFix,
      };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async generateFixPatch(params: {
    triggeringFile: string;
    panicMessage: string;
    astCode: string;
    rootCause?: string;
    stackTrace?: string;
    projectContext?: string;
  }): Promise<{ success: boolean; patch?: string; error?: string }> {
    try {
      const data = await this.request<any>('/llm/generate-patch', {
        method: 'POST',
        body: JSON.stringify(params),
      });
      return { success: true, patch: data.patch };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async createIncidentIssue(
    incidentId: string,
  ): Promise<{ success: boolean; issue_number?: number; issue_url?: string; error?: string }> {
    try {
      const data = await this.request<any>('/incidents/create-issue', {
        method: 'POST',
        body: JSON.stringify({ incident_id: incidentId }),
      });
      return {
        success: true,
        issue_number: data.github_issue?.number,
        issue_url: data.github_issue?.html_url,
      };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async createPullRequest(params: { incidentId: string; patchCode?: string }): Promise<{
    success: boolean;
    pr_number?: number;
    pr_url?: string;
    branch?: string;
    error?: string;
  }> {
    try {
      const data = await this.request<any>('/incidents/create-pr', {
        method: 'POST',
        body: JSON.stringify({ incident_id: params.incidentId, patch_code: params.patchCode }),
      });
      return {
        success: true,
        pr_number: data.pull_request?.number,
        pr_url: data.pull_request?.html_url,
        branch: data.pull_request?.branch,
      };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  // ---------------------------------------------------------------------------
  // Runtime Settings (Admin / Owner Authenticated)
  // ---------------------------------------------------------------------------

  async getLlmConfig(): Promise<{
    provider?: string;
    api_key?: string;
    model?: string;
    base_url?: string;
  }> {
    return this.request('/settings/llm', { fallback: {} });
  }

  async updateLlmConfig(config: {
    provider?: string;
    apiKey?: string;
    model?: string;
    baseUrl?: string;
  }): Promise<boolean> {
    try {
      await this.request('/settings/llm', {
        method: 'POST',
        body: JSON.stringify({
          provider: config.provider || 'gemini',
          api_key: config.apiKey || '',
          model: config.model || '',
          base_url: config.baseUrl || '',
        }),
      });
      return true;
    } catch {
      return false;
    }
  }

  async testLlmConfig(config: {
    provider?: string;
    apiKey?: string;
    model?: string;
    baseUrl?: string;
  }): Promise<{ success: boolean; latency_ms?: number; error?: string; provider?: string }> {
    try {
      const data = await this.request<any>('/settings/llm/test', {
        method: 'POST',
        body: JSON.stringify({
          provider: config.provider || 'gemini',
          api_key: config.apiKey || '',
          model: config.model || '',
          base_url: config.baseUrl || '',
        }),
      });
      return {
        success: true,
        latency_ms: data.latency_ms,
        provider: data.provider || config.provider,
      };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  // ---------------------------------------------------------------------------
  // Setup Wizard (Public Onboarding Endpoints)
  // ---------------------------------------------------------------------------

  async getSetupStatus(): Promise<{
    configured: boolean;
    github_app: boolean;
    installation: boolean;
    oauth: boolean;
    llm: boolean;
  }> {
    return this.request('/setup/status', {
      skipAuth: true,
      fallback: {
        configured: false,
        github_app: false,
        installation: false,
        oauth: false,
        llm: false,
      },
    });
  }

  async getSetupManifest(instanceUrl: string): Promise<{ manifest: any; url: string }> {
    return this.request('/setup/manifest', {
      method: 'POST',
      body: JSON.stringify({ instance_url: instanceUrl }),
      skipAuth: true,
    });
  }

  async getInstallUrl(): Promise<{ url: string }> {
    return this.request('/setup/install', { skipAuth: true });
  }

  async saveOAuthConfig(clientId: string, clientSecret: string): Promise<{ success: boolean }> {
    return this.request('/setup/oauth', {
      method: 'POST',
      body: JSON.stringify({ client_id: clientId, client_secret: clientSecret }),
      skipAuth: true,
    });
  }

  async testSetupConnection(): Promise<{ success: boolean; app_name?: string; error?: string }> {
    try {
      const data = await this.request<any>('/setup/test', { method: 'POST', skipAuth: true });
      return { success: true, app_name: data.app_name };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async saveLlmSetupConfig(config: {
    provider?: string;
    apiKey?: string;
    model?: string;
    baseUrl?: string;
  }): Promise<boolean> {
    try {
      await this.request('/setup/llm', {
        method: 'POST',
        body: JSON.stringify({
          provider: config.provider || 'gemini',
          api_key: config.apiKey || '',
          model: config.model || '',
          base_url: config.baseUrl || '',
        }),
        skipAuth: true,
      });
      return true;
    } catch {
      return false;
    }
  }

  async testSetupLlmConfig(config: {
    provider?: string;
    apiKey?: string;
    model?: string;
    baseUrl?: string;
  }): Promise<{ success: boolean; latency_ms?: number; error?: string; provider?: string }> {
    try {
      const data = await this.request<any>('/setup/llm/test', {
        method: 'POST',
        body: JSON.stringify({
          provider: config.provider || 'gemini',
          api_key: config.apiKey || '',
          model: config.model || '',
          base_url: config.baseUrl || '',
        }),
        skipAuth: true,
      });
      return {
        success: true,
        latency_ms: data.latency_ms,
        provider: data.provider || config.provider,
      };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async getSetupRepos(username?: string): Promise<import('@/types').RepositoryItem[]> {
    const res = await this.request<{ repos?: any[] }>('/setup/repos', {
      params: { username },
      skipAuth: true,
      fallback: { repos: [] },
    });
    return res.repos || [];
  }

  async getInstalledRepos(): Promise<import('@/types').RepositoryItem[]> {
    const res = await this.request<{ repos?: any[] }>('/setup/repos', {
      skipAuth: true,
      fallback: { repos: [] },
    });
    return res.repos || [];
  }

  async getInstalledRepoSlugs(): Promise<string[]> {
    const res = await this.request<{ installed_repos?: string[] }>('/setup/installed-repos', {
      skipAuth: true,
      fallback: { installed_repos: [] },
    });
    return res.installed_repos || [];
  }

  async checkRepoInstalled(
    owner: string,
    repo: string,
  ): Promise<{ installed: boolean; installation_id?: number; owner?: string; repo?: string }> {
    const res = await this.request<any>('/setup/check-repo', {
      params: { owner, repo },
      skipAuth: true,
      fallback: { installed: false },
    });
    return {
      installed: Boolean(res.installed),
      installation_id: res.installation_id,
      owner: res.owner || owner,
      repo: res.repo || repo,
    };
  }

  // ---------------------------------------------------------------------------
  // Auth & Team Management
  // ---------------------------------------------------------------------------

  async getAuthUser(): Promise<{
    id: string;
    username: string;
    email?: string;
    avatar_url?: string;
    role: string;
  } | null> {
    const res = await this.request<{ user?: any }>('/auth/me', { fallback: { user: null } });
    return res.user || null;
  }

  async logout(): Promise<void> {
    try {
      await this.request('/auth/logout', { method: 'POST' });
    } catch (err) {
      logger.debug('Logout request failed', err);
    }
  }

  async getTeamMembers(): Promise<any[]> {
    const res = await this.request<{ members?: any[] }>('/team/members', {
      fallback: { members: [] },
    });
    return res.members || [];
  }

  async updateMemberRole(id: string, role: string): Promise<{ success: boolean; error?: string }> {
    try {
      await this.request('/team/members/role', {
        method: 'PUT',
        body: JSON.stringify({ id, role }),
      });
      return { success: true };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async removeMember(id: string): Promise<{ success: boolean; error?: string }> {
    try {
      await this.request('/team/members', {
        method: 'DELETE',
        params: { id },
      });
      return { success: true };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async getTeamInvites(): Promise<any[]> {
    const res = await this.request<{ invitations?: any[] }>('/team/invites', {
      fallback: { invitations: [] },
    });
    return res.invitations || [];
  }

  async createInvite(
    githubUsername: string,
    role: string,
  ): Promise<{ success: boolean; error?: string }> {
    try {
      await this.request('/team/invites', {
        method: 'POST',
        body: JSON.stringify({ github_username: githubUsername, role }),
      });
      return { success: true };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }

  async cancelInvite(id: string): Promise<{ success: boolean; error?: string }> {
    try {
      await this.request('/team/invites', {
        method: 'DELETE',
        params: { id },
      });
      return { success: true };
    } catch (err: any) {
      return { success: false, error: err.message };
    }
  }
}

export const engineClient = new EngineClient();
