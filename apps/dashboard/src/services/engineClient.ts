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

const DEFAULT_ENGINE_URL =
  process.env.NEXT_PUBLIC_ENGINE_URL || 'http://localhost:8080/api/v1/telemetry';
const TEST_HARNESS_URL = 'http://localhost:8081/crash';

export class EngineClient {
  private engineUrl: string;
  private authToken: string | null = null;

  constructor(engineUrl: string = DEFAULT_ENGINE_URL) {
    this.engineUrl = engineUrl;
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
      const response = await fetch(this.engineUrl, {
        method: 'OPTIONS',
      });
      const latencyMs = Date.now() - startTime;
      return {
        online: response.ok || response.status === 405 || response.status === 200,
        url: this.engineUrl,
        latencyMs,
      };
    } catch {
      return {
        online: false,
        url: this.engineUrl,
        latencyMs: 0,
      };
    }
  }

  async sendTelemetry(payload: TelemetryPayload): Promise<TelemetryResponse> {
    const response = await fetch(this.engineUrl, {
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/incidents`);
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/settings/llm`, {
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/settings/llm`, {
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/projects`);
      if (!res.ok) return [];
      const data = await res.json();
      return data.projects || [];
    } catch {
      return [];
    }
  }

  async createProject(
    repo: string,
    ownerUsername?: string,
  ): Promise<{ success: boolean; repo: string; api_key: string }> {
    const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
    const res = await fetch(`${baseUrl}/projects`, {
      method: 'POST',
      headers: this.getAuthHeaders(),
      body: JSON.stringify({ repo, owner_username: ownerUsername || '' }),
    });
    if (!res.ok) {
      throw new Error(`Failed to create project: ${await res.text()}`);
    }
    return await res.json();
  }

  async getStats(): Promise<any> {
    try {
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/stats`);
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/setup/status`);
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/setup/llm`, {
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
    const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
    const res = await fetch(`${baseUrl}/setup/manifest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ instance_url: instanceUrl }),
    });
    if (!res.ok) throw new Error(`Failed to get manifest: ${await res.text()}`);
    return await res.json();
  }

  async getInstallUrl(): Promise<{ url: string }> {
    const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
    const res = await fetch(`${baseUrl}/setup/install`);
    if (!res.ok) throw new Error(`Failed to get install URL: ${await res.text()}`);
    return await res.json();
  }

  async saveOAuthConfig(clientId: string, clientSecret: string): Promise<{ success: boolean }> {
    const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
    const res = await fetch(`${baseUrl}/setup/oauth`, {
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

  async getSetupRepos(): Promise<{ owner: string; repo: string }[]> {
    try {
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/setup/repos`);
      if (!res.ok) return [];
      const data = await res.json();
      return data.repos || [];
    } catch {
      return [];
    }
  }

  async testSetupConnection(): Promise<{
    success: boolean;
    app_name?: string;
    error?: string;
  }> {
    const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
    const res = await fetch(`${baseUrl}/setup/test`, { method: 'POST' });
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
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/auth/me`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) return { valid: false };
      const data = await res.json();
      return { valid: true, user: data };
    } catch {
      return { valid: false };
    }
  }

  async getInstalledRepos(): Promise<{ owner: string; repo: string }[]> {
    try {
      const baseUrl = this.engineUrl.replace(/\/telemetry$/, '');
      const res = await fetch(`${baseUrl}/setup/repos`, {
        headers: this.getAuthHeaders(),
      });
      if (!res.ok) return [];
      const data = await res.json();
      return data.repos || [];
    } catch {
      return [];
    }
  }
}

export const engineClient = new EngineClient();
