/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Incident, ScreenId } from './types';
import {
  INITIAL_INCIDENTS,
  MOCK_AST_FILES,
  MOCK_COMMIT_INDEXES,
  MOCK_WEBHOOK_LOGS,
  MOCK_TEAM_MEMBERS,
  MOCK_API_KEYS,
  MOCK_HOURLY_METRICS,
  MOCK_SYSTEM_HEALTH,
} from './data/mockData';

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
import { CheckCircle2, AlertTriangle, X } from 'lucide-react';

export default function App({ initialScreen = 'dashboard' }: { initialScreen?: ScreenId }) {
  const [currentScreen, setCurrentScreen] = useState<ScreenId>(initialScreen);
  const [incidents, setIncidents] = useState<Incident[]>(INITIAL_INCIDENTS);
  const [selectedIncidentId, setSelectedIncidentId] = useState<string>('INC-8094');
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Selected incident object
  const selectedIncident =
    incidents.find((i) => i.id === selectedIncidentId) || incidents[0] || INITIAL_INCIDENTS[0];

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
      const newId = `INC-${Math.floor(8100 + Math.random() * 100)}`;
      const newIncident: Incident = {
        id: newId,
        title: resp.analysis?.root_cause || 'nil pointer dereference in ChargeCart()',
        status: 'CRITICAL',
        triggeringFile: 'scripts/test-crash/main.go:21',
        triggeringLine: 21,
        latencyMs: 14,
        commitHash: '8f3a1b4',
        branch: 'main',
        timestamp: new Date().toISOString().replace('T', ' ').substring(0, 19) + ' UTC',
        goroutineId: 'goroutine 21 [running]',
        panicMessage: 'panic: runtime error: invalid memory address or nil pointer dereference',
        rawStackTrace: `goroutine 21 [running]:\nruntime/debug.Stack()\n\t/workspace/sdk/go/middleware.go:28 +0x68\nmain.main.func2({0x12995dae8, 0x102893268}, 0x0)\n\t/workspace/scripts/test-crash/main.go:21 +0x74`,
        astSnippet: {
          functionName: 'main.func2',
          file: 'scripts/test-crash/main.go',
          startLine: 18,
          lines: [
            { lineNum: 18, content: 'mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {' },
            { lineNum: 19, content: '	log.Println("Triggering nil pointer dereference panic...")' },
            { lineNum: 20, content: '	var ptr *int' },
            { lineNum: 21, content: '	*ptr = 42 // PANIC: nil pointer dereference', isTriggerLine: true },
            { lineNum: 22, content: '})' },
          ],
        },
      };

      setIncidents((prev) => [newIncident, ...prev]);
      setSelectedIncidentId(newId);
      setCurrentScreen('incident_detail');
      showToast(`[PANIC INGESTED] Engine Processed AST & Gemini Analysis: ${newId}`);
    } catch (e) {
      console.error('Telemetry dispatch error:', e);
      showToast(`[PANIC SIMULATED] Ingested Local Panic`);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans antialiased flex flex-col selection:bg-black selection:text-white">
      {/* Top Header */}
      <Header
        currentScreen={currentScreen}
        onNavigate={(screen) => setCurrentScreen(screen)}
        onSimulatePanic={handleSimulatePanic}
        criticalCount={criticalCount}
      />

      {/* Main Content Render Area */}
      <main className="flex-1">
        {currentScreen === 'login' && (
          <LoginPage
            onNavigate={(screen) => setCurrentScreen(screen)}
            onLoginSuccess={(user) => showToast(`Welcome back, ${user}!`)}
          />
        )}

        {currentScreen === 'new' && (
          <OnboardingPage onNavigate={(screen) => setCurrentScreen(screen)} />
        )}

        {currentScreen === 'dashboard' && (
          <DashboardPage
            incidents={incidents}
            onSelectIncident={(id) => {
              setSelectedIncidentId(id);
              setCurrentScreen('incident_detail');
            }}
            onNavigate={(screen) => setCurrentScreen(screen)}
            onSimulatePanic={handleSimulatePanic}
          />
        )}

        {currentScreen === 'incident_detail' && (
          <IncidentDetailPage
            incident={selectedIncident}
            allIncidents={incidents}
            onSelectIncident={(id) => setSelectedIncidentId(id)}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}

        {currentScreen === 'ast' && (
          <AstExplorerPage
            commitIndexes={MOCK_COMMIT_INDEXES}
            astFiles={MOCK_AST_FILES}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}

        {currentScreen === 'webhooks' && (
          <WebhooksPage
            logs={MOCK_WEBHOOK_LOGS}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}

        {currentScreen === 'team' && (
          <TeamPage
            members={MOCK_TEAM_MEMBERS}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}

        {currentScreen === 'status' && (
          <SystemStatusPage
            health={MOCK_SYSTEM_HEALTH}
            metrics={MOCK_HOURLY_METRICS}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}

        {currentScreen === 'settings' && (
          <SettingsPage
            apiKeys={MOCK_API_KEYS}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}
      </main>

      {/* Notification Toast */}
      {toastMessage && (
        <div className="fixed bottom-4 right-4 bg-black text-white font-mono text-xs px-4 py-3 rounded-sm shadow-none z-50 flex items-center gap-3 border border-slate-800 animate-in fade-in slide-in-from-bottom-2">
          <AlertTriangle className="w-4 h-4 text-red-400 shrink-0" />
          <span className="font-semibold">{toastMessage}</span>
          <button
            onClick={() => setToastMessage(null)}
            className="text-slate-400 hover:text-white ml-2"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}

      {/* Footer */}
      <footer className="bg-white border-t border-slate-200 mt-12 py-4">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between text-xs font-mono text-slate-500 gap-2">
          <div className="flex items-center gap-2">
            <span className="font-bold text-slate-900">[TRIAGE]</span>
            <span>Go Crash Detection & AST Isolation Engine</span>
          </div>

          <div className="flex items-center gap-4 text-[11px]">
            <span>Org: algotyrnt</span>
            <span>Repo: beacon-app</span>
            <span className="text-emerald-700 font-bold">● Operational</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
