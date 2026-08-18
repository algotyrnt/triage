/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

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

const DEFAULT_BASE_URL = (
  process.env.NEXT_PUBLIC_ENGINE_URL || 'http://localhost:8080/api/v1'
).replace(/\/telemetry$/, '');

export class EngineClient {
  private baseUrl: string;
  private telemetryUrl: string;
  private authToken: string | null = null;

  constructor(engineUrl: string = DEFAULT_BASE_URL) {
    this.baseUrl = engineUrl.replace(/\/telemetry$/, '');
    this.telemetryUrl = `${this.baseUrl}/telemetry`;
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  getTelemetryUrl(): string {
    return this.telemetryUrl;
  }

  getAuthGitHubUrl(): string {
    return `${this.baseUrl}/auth/github`;
  }

  setAuthToken(token: string | null) {
    this.authToken = token;
  }

  private getAuthHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }
    return headers;
  }

  async checkStatus(): Promise<EngineStatus> {
    const startTime = Date.now();
    try {
      const response = await fetch(this.telemetryUrl, {
        method: 'OPTIONS',
      });
      const latencyMs = Date.now() - startTime;
      return {
        online: response.ok || response.status === 405 || response.status === 200,
        url: this.telemetryUrl,
        latencyMs,
      };
    } catch {
      return {
        online: false,
        url: this.telemetryUrl,
        latencyMs: 0,
      };
    }
  }

  async sendTelemetry(payload: TelemetryPayload): Promise<TelemetryResponse> {
    const response = await fetch(this.telemetryUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Engine telemetry call failed (${response.status}): ${errorText}`);
    }

    return (await response.json()) as TelemetryResponse;
  }

  async getIncidents(): Promise<any[]> {
    try {
      const res = await fetch(`${this.baseUrl}/incidents`);
      if (!res.ok) return [];
      const data = await res.json();
      return data.incidents || [];
    } catch {
      return [];
    }
  }

  async getLlmConfig(): Promise<{
    gemini_api_key?: string;
    gemini_model?: string;
  }> {
    try {
      const res = await fetch(`${this.baseUrl}/settings/llm`, {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return {};
      return await res.json();
    } catch {
      return {};
    }
  }

  async updateLlmConfig(apiKey: string, model: string): Promise<boolean> {
    try {
      const res = await fetch(`${this.baseUrl}/settings/llm`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify({ gemini_api_key: apiKey, gemini_model: model }),
      });
      return res.ok;
    } catch {
      return false;
    }
  }

  async getProjects(): Promise<any[]> {
    try {
      const res = await fetch(`${this.baseUrl}/projects`);
      if (!res.ok) return [];
      const data = await res.json();
      return data.projects || [];
    } catch {
      return [];
    }
  }

  async createProject(
    repo: string,
    rootDir?: string,
    ownerUsername?: string,
  ): Promise<{
    success: boolean;
    repo: string;
    root_dir?: string;
    api_key: string;
    key_masked?: string;
  }> {
    const res = await fetch(`${this.baseUrl}/projects`, {
      method: 'POST',
      headers: this.getAuthHeaders(),
      body: JSON.stringify({
        repo,
        root_dir: rootDir || '',
        owner_username: ownerUsername || '',
      }),
    });
    if (!res.ok) {
      throw new Error(`Failed to create project: ${await res.text()}`);
    }
    return await res.json();
  }

  async detectGoModules(
    owner: string,
    repo: string,
  ): Promise<{ path: string; name: string; is_root: boolean }[]> {
    try {
      const params = new URLSearchParams({ owner, repo });
      const res = await fetch(`${this.baseUrl}/repos/detect-modules?${params.toString()}`, {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return [{ path: '', name: 'Repository Root (/)', is_root: true }];
      const data = await res.json();
      return Array.isArray(data.modules) && data.modules.length > 0
        ? data.modules
        : [{ path: '', name: 'Repository Root (/)', is_root: true }];
    } catch {
      return [{ path: '', name: 'Repository Root (/)', is_root: true }];
    }
  }

  async getStats(): Promise<any> {
    try {
      const res = await fetch(`${this.baseUrl}/stats`);
      if (!res.ok) return null;
      return await res.json();
    } catch {
      return null;
    }
  }

  async getSetupStatus(): Promise<{
    configured: boolean;
    github_app: boolean;
    installation: boolean;
    oauth: boolean;
    llm: boolean;
  }> {
    try {
      const res = await fetch(`${this.baseUrl}/setup/status`);
      if (!res.ok)
        return {
          configured: false,
          github_app: false,
          installation: false,
          oauth: false,
          llm: false,
        };
      return await res.json();
    } catch {
      return {
        configured: false,
        github_app: false,
        installation: false,
        oauth: false,
        llm: false,
      };
    }
  }

  async saveLlmSetupConfig(apiKey: string, modelName: string): Promise<boolean> {
    try {
      const res = await fetch(`${this.baseUrl}/setup/llm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gemini_api_key: apiKey,
          gemini_model: modelName,
        }),
      });
      return res.ok;
    } catch {
      return false;
    }
  }

  async getSetupManifest(instanceUrl: string): Promise<{ manifest: any; url: string }> {
    const res = await fetch(`${this.baseUrl}/setup/manifest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ instance_url: instanceUrl }),
    });
    if (!res.ok) throw new Error(`Failed to get manifest: ${await res.text()}`);
    return await res.json();
  }

  async getInstallUrl(): Promise<{ url: string }> {
    const res = await fetch(`${this.baseUrl}/setup/install`);
    if (!res.ok) throw new Error(`Failed to get install URL: ${await res.text()}`);
    return await res.json();
  }

  async saveOAuthConfig(clientId: string, clientSecret: string): Promise<{ success: boolean }> {
    const res = await fetch(`${this.baseUrl}/setup/oauth`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: clientId,
        client_secret: clientSecret,
      }),
    });
    if (!res.ok) throw new Error(`Failed to save OAuth config: ${await res.text()}`);
    return await res.json();
  }

  async getSetupRepos(username?: string): Promise<import('@/types').RepositoryItem[]> {
    try {
      const url = new URL(`${this.baseUrl}/setup/repos`);
      if (username) {
        url.searchParams.set('username', username);
      }
      const res = await fetch(url.toString(), {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return [];
      const data = await res.json();
      return data.repos || [];
    } catch {
      return [];
    }
  }

  async checkRepoInstalled(
    owner: string,
    repo: string,
  ): Promise<{ installed: boolean; installation_id?: number; owner?: string; repo?: string }> {
    try {
      const url = new URL(`${this.baseUrl}/setup/check-repo`);
      url.searchParams.set('owner', owner);
      url.searchParams.set('repo', repo);
      const res = await fetch(url.toString(), {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return { installed: false };
      return await res.json();
    } catch {
      return { installed: false };
    }
  }

  async testSetupConnection(): Promise<{
    success: boolean;
    app_name?: string;
    error?: string;
  }> {
    const res = await fetch(`${this.baseUrl}/setup/test`, { method: 'POST' });
    if (!res.ok) {
      const data = await res.json().catch(() => ({ error: 'Connection failed' }));
      return { success: false, error: data.error || 'Connection test failed' };
    }
    return await res.json();
  }

  async verifySession(token: string): Promise<{
    valid: boolean;
    user?: {
      id: string;
      username: string;
      avatar_url: string;
      github_id: string;
    };
  }> {
    try {
      const res = await fetch(`${this.baseUrl}/auth/me`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) return { valid: false };
      const data = await res.json();
      return { valid: true, user: data };
    } catch {
      return { valid: false };
    }
  }

  async getInstalledRepos(): Promise<import('@/types').RepositoryItem[]> {
    try {
      const res = await fetch(`${this.baseUrl}/setup/repos`, {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return [];
      const data = await res.json();
      return data.repos || [];
    } catch {
      return [];
    }
  }

  async getProjectKeys(
    owner?: string,
    repo?: string,
    rootDir?: string,
  ): Promise<import('@/types').ApiKey[]> {
    try {
      const url = new URL(`${this.baseUrl}/projects/keys`);
      if (owner) url.searchParams.set('owner', owner);
      if (repo) url.searchParams.set('repo', repo);
      if (rootDir) url.searchParams.set('root_dir', rootDir);
      const res = await fetch(url.toString(), {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return [];
      const data = await res.json();
      if (!Array.isArray(data.keys)) return [];
      return data.keys.map((k: any) => ({
        id: k.id,
        name: k.name || 'API Key',
        keyMasked: k.key_masked || 'tr_live_...xxxx',
        fullKey: k.raw_key || undefined,
        createdAt: k.created_at ? new Date(k.created_at).toISOString().split('T')[0] : 'Recently',
        lastUsed: 'Recently',
        status: (k.status === 'REVOKED' ? 'REVOKED' : 'ACTIVE') as 'ACTIVE' | 'REVOKED',
      }));
    } catch {
      return [];
    }
  }

  async createApiKey(
    owner: string,
    repo: string,
    rootDir?: string,
    name?: string,
  ): Promise<{ success: boolean; key?: import('@/types').ApiKey & { raw_key?: string } }> {
    try {
      const res = await fetch(`${this.baseUrl}/projects/keys`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify({
          owner,
          repo,
          root_dir: rootDir || '',
          name: name || `Key for ${repo}`,
        }),
      });
      if (!res.ok) return { success: false };
      const data = await res.json();
      if (data.key) {
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
      }
      return { success: false };
    } catch {
      return { success: false };
    }
  }

  async revokeApiKey(keyId: string): Promise<boolean> {
    try {
      const res = await fetch(`${this.baseUrl}/projects/keys/revoke`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify({ key_id: keyId }),
      });
      return res.ok;
    } catch {
      return false;
    }
  }

  async createIncidentIssue(
    incidentId: string,
  ): Promise<{ success: boolean; issue_number?: number; issue_url?: string; error?: string }> {
    try {
      const res = await fetch(`${this.baseUrl}/incidents/create-issue`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify({ incident_id: incidentId }),
      });
      const data = await res.json();
      if (res.ok && data.success && data.github_issue) {
        return {
          success: true,
          issue_number: data.github_issue.number,
          issue_url: data.github_issue.html_url,
        };
      }
      return { success: false, error: data.error || 'Failed to create GitHub issue' };
    } catch (e: any) {
      return { success: false, error: e?.message || 'Network error' };
    }
  }

  async analyzePanic(params: {
    panicMessage: string;
    rawStackTrace: string;
    triggeringFile: string;
    astCode: string;
  }): Promise<{
    success: boolean;
    rootCause?: string;
    explanation?: string;
    severity?: string;
    recommendedFix?: string;
    error?: string;
  }> {
    try {
      const res = await fetch(`${this.baseUrl}/gemini/analyze-panic`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify(params),
      });
      const data = await res.json();
      if (res.ok && data.success) {
        return {
          success: true,
          rootCause: data.rootCause,
          explanation: data.explanation,
          severity: data.severity || 'CRITICAL',
          recommendedFix: data.recommendedFix,
        };
      }
      return { success: false, error: data.error || `Server error (${res.status})` };
    } catch (e: any) {
      return { success: false, error: e?.message || 'Failed to connect to AI engine' };
    }
  }

  async generateFixPatch(params: {
    triggeringFile: string;
    panicMessage: string;
    astCode: string;
    rootCause?: string;
    stackTrace?: string;
  }): Promise<{ success: boolean; patch?: string; error?: string }> {
    try {
      const res = await fetch(`${this.baseUrl}/gemini/generate-patch`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify(params),
      });
      const data = await res.json();
      if (res.ok && data.success && data.patch) {
        return { success: true, patch: data.patch };
      }
      return { success: false, error: data.error || `Server error (${res.status})` };
    } catch (e: any) {
      return { success: false, error: e?.message || 'Failed to connect to AI engine' };
    }
  }
}

export const engineClient = new EngineClient();
