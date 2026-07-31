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

  // Simulate dynamic Go runtime panic
  const handleSimulatePanic = async () => {
    try {
      const res = await fetch('/api/simulate-panic', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          file: 'pkg/handler/checkout.go:58',
          panicType: Math.random() > 0.5 ? 'nil_pointer' : 'slice_out_of_bounds',
        }),
      });
      const data = await res.json();
      if (data.incident) {
        setIncidents((prev) => [data.incident, ...prev]);
        setSelectedIncidentId(data.incident.id);
        setCurrentScreen('incident_detail');
        showToast(`[PANIC] Go Runtime Panic Ingested: ${data.incident.id} (${data.incident.triggeringFile})`);
      }
    } catch (e) {
      // Local fallback simulation if server API isn't responding
      const fallbackId = `INC-${Math.floor(8100 + Math.random() * 100)}`;
      const fallbackIncident: Incident = {
        id: fallbackId,
        title: 'nil pointer dereference in ChargeCart()',
        status: 'CRITICAL',
        triggeringFile: 'pkg/handler/checkout.go:58',
        triggeringLine: 58,
        latencyMs: 14,
        commitHash: '8f3a1b4',
        branch: 'main',
        timestamp: new Date().toISOString().replace('T', ' ').substring(0, 19) + ' UTC',
        goroutineId: 'goroutine 54 [running]',
        panicMessage: 'panic: runtime error: invalid memory address or nil pointer dereference',
        rawStackTrace: `goroutine 54 [running]:\npkg/handler.(*CheckoutHandler).ChargeCart(0x0, 0xc0000a2000)\n\t/workspace/pkg/handler/checkout.go:58 +0x42`,
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
      };

      setIncidents((prev) => [fallbackIncident, ...prev]);
      setSelectedIncidentId(fallbackId);
      setCurrentScreen('incident_detail');
      showToast(`[PANIC] Go Runtime Panic Ingested: ${fallbackId} (pkg/handler/checkout.go:58)`);
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
