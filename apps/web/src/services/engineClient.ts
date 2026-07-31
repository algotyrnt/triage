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

const DEFAULT_ENGINE_URL = process.env.NEXT_PUBLIC_ENGINE_URL || 'http://localhost:8080/api/v1/telemetry';
const TEST_HARNESS_URL = 'http://localhost:8081/crash';

export class EngineClient {
  private engineUrl: string;

  constructor(engineUrl: string = DEFAULT_ENGINE_URL) {
    this.engineUrl = engineUrl;
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

  async triggerTestPanic(): Promise<TelemetryResponse> {
    try {
      const harnessResp = await fetch(TEST_HARNESS_URL);
      const text = await harnessResp.text();
      console.log('Test harness response:', text);
    } catch {
      console.warn('Test harness on :8081 unavailable, dispatching direct panic payload to engine');
    }

    const mockPanicPayload: TelemetryPayload = {
      api_key: 'tr_live_key_9042',
      file: 'scripts/test-crash/main.go',
      line: 21,
      stack_trace: `goroutine 21 [running]:
runtime/debug.Stack()
	/workspace/sdk/go/middleware.go:28 +0x68
main.main.func2({0x12995dae8, 0x102893268}, 0x0)
	/workspace/scripts/test-crash/main.go:21 +0x74
net/http.HandlerFunc.ServeHTTP(0x1028e3b20?, {0x12995dae8, 0x102893268}, 0x1400018a000?)
	/usr/local/go/src/net/http/server.go:2166 +0x38`,
      github_owner: 'algotyrnt',
      github_repo: 'triage',
    };

    return this.sendTelemetry(mockPanicPayload);
  }
}

export const engineClient = new EngineClient();
