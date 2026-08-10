/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Incident, ScreenId } from './types';

import { Header } from './components/Header';
import { LoginPage } from './components/screens/LoginPage';
import { OnboardingPage } from './components/screens/OnboardingPage';
import { DashboardPage } from './components/screens/DashboardPage';
import { IncidentDetailPage } from './components/screens/IncidentDetailPage';
import { AstExplorerPage } from './components/screens/AstExplorerPage';
import { WebhooksPage } from './components/screens/WebhooksPage';
import { TeamPage } from './components/screens/TeamPage';
import { SystemStatusPage } from './components/screens/SystemStatusPage';
import { SettingsPage } from './components/screens/SettingsPage';

import { engineClient } from './services/engineClient';
import { CheckCircle2, X } from 'lucide-react';

export default function App({ initialScreen = 'dashboard' }: { initialScreen?: ScreenId }) {
  const [currentScreen, setCurrentScreen] = useState<ScreenId>(initialScreen);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedIncidentId, setSelectedIncidentId] = useState<string>('');
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Selected incident object
  const selectedIncident = incidents.find((i) => i.id === selectedIncidentId) || incidents[0];

  const criticalCount = incidents.filter((i) => i.status === 'CRITICAL').length;

  // Show Toast
  const showToast = (msg: string) => {
    setToastMessage(msg);
    setTimeout(() => setToastMessage(null), 3500);
  };

  // Simulate dynamic Go runtime panic via engineClient
  const handleSimulatePanic = async () => {
    try {
      const resp = await engineClient.triggerTestPanic();
      const uuidSuffix = typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID().slice(0, 8).toUpperCase() : `${Date.now()}`;
      const newId = `INC-${uuidSuffix}`;
      const newIncident: Incident = {
        id: newId,
        title: resp.analysis?.root_cause || 'nil pointer dereference in ChargeCart()',
        status: 'CRITICAL',
        triggeringFile: 'test-service/main.go:21',
        triggeringLine: 21,
        latencyMs: 14,
        commitHash: '8f3a1b4',
        branch: 'main',
        timestamp: new Date().toISOString().replace('T', ' ').substring(0, 19) + ' UTC',
        goroutineId: 'goroutine 21 [running]',
        panicMessage: 'panic: runtime error: invalid memory address or nil pointer dereference',
        rawStackTrace: `goroutine 21 [running]:\nruntime/debug.Stack()\n\t/workspace/sdk/go/middleware.go:28 +0x68\nmain.main.func2({0x12995dae8, 0x102893268}, 0x0)\n\t/workspace/test-service/main.go:21 +0x74`,
        astSnippet: {
          functionName: 'main.func2',
          file: 'test-service/main.go',
          startLine: 18,
          lines: [
            { lineNum: 18, content: 'func main() {' },
            { lineNum: 19, content: '	http.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {' },
            { lineNum: 20, content: '		var ptr *string' },
            { lineNum: 21, content: '		_ = *ptr // nil pointer dereference', isTriggerLine: true },
            { lineNum: 22, content: '	})' },
            { lineNum: 23, content: '}' },
          ],
        },
        geminiAnalysis: resp.analysis
          ? {
              rootCause: resp.analysis.root_cause,
              explanation: resp.analysis.root_cause,
              severity: 'CRITICAL',
              recommendedFix: resp.analysis.suggested_fix,
            }
          : undefined,
        suggestedPatch: resp.analysis?.suggested_fix,
        githubIssueUrl: resp.github_issue?.html_url,
        githubIssueNumber: resp.github_issue?.number,
      };

      setIncidents((prev) => [newIncident, ...prev]);
      setSelectedIncidentId(newId);
      showToast(`Live Panic Telemetry Ingested: ${newId}`);
    } catch (err) {
      console.error(err);
      showToast('Engine Ingestion Error: Ensure apps/engine is running');
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 flex flex-col font-sans selection:bg-indigo-500 selection:text-white">
      {/* Global Toast Notification */}
      {toastMessage && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center space-x-3 bg-slate-900 border border-slate-800 text-slate-100 px-4 py-3 rounded-lg shadow-xl animate-in fade-in slide-in-from-bottom-5">
          <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
          <span className="text-sm font-medium">{toastMessage}</span>
          <button onClick={() => setToastMessage(null)} className="text-slate-400 hover:text-white transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      <Header
        currentScreen={currentScreen}
        onNavigate={(screen) => setCurrentScreen(screen)}
        criticalCount={criticalCount}
        onSimulatePanic={handleSimulatePanic}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {currentScreen === 'login' && <LoginPage onLoginSuccess={() => setCurrentScreen('dashboard')} onNavigate={(screen) => setCurrentScreen(screen)} />}
        {currentScreen === 'new' && <OnboardingPage onNavigate={(screen) => setCurrentScreen(screen)} />}

        {currentScreen === 'dashboard' && (
          <DashboardPage
            incidents={incidents}
            onNavigate={(screen) => setCurrentScreen(screen)}
            onSimulatePanic={handleSimulatePanic}
            onSelectIncident={(id) => {
              setSelectedIncidentId(id);
              setCurrentScreen('incident_detail');
            }}
          />
        )}

        {currentScreen === 'incident_detail' && selectedIncident && (
          <IncidentDetailPage
            incident={selectedIncident}
            allIncidents={incidents}
            onSelectIncident={(id) => setSelectedIncidentId(id)}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}

        {currentScreen === 'ast' && <AstExplorerPage onNavigate={(screen) => setCurrentScreen(screen)} commitIndexes={[]} astFiles={[]} />}
        {currentScreen === 'webhooks' && <WebhooksPage onNavigate={(screen) => setCurrentScreen(screen)} logs={[]} />}
        {currentScreen === 'team' && <TeamPage teamMembers={[]} onNavigate={(screen) => setCurrentScreen(screen)} />}
        {currentScreen === 'status' && <SystemStatusPage onNavigate={(screen) => setCurrentScreen(screen)} health={[]} metrics={[]} />}
        {currentScreen === 'settings' && <SettingsPage apiKeys={[]} onNavigate={(screen) => setCurrentScreen(screen)} />}
      </main>

      <footer className="border-t border-slate-200 bg-white py-6">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 gap-4">
          <div className="flex items-center space-x-2">
            <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            <span>triage Core Engine Active</span>
            <span className="text-slate-300">•</span>
            <span>Zero-Overhead Go Crash Isolation</span>
          </div>
          <div>
            Powered by{' '}
            <span className="text-slate-900 font-medium">Google Gemini 3.5 Flash</span> &amp; AST Parser
          </div>
        </div>
      </footer>
    </div>
  );
}
