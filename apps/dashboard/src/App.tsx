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
import { SetupWizardPage } from './components/screens/SetupWizardPage';

import { engineClient } from './services/engineClient';
import { AlertTriangle, CheckCircle2, X } from 'lucide-react';

type ToastVariant = 'success' | 'error';

export default function App({ initialScreen = 'dashboard' }: { initialScreen?: ScreenId }) {
  const [currentScreen, setCurrentScreen] = useState<ScreenId>(initialScreen);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedIncidentId, setSelectedIncidentId] = useState<string>('');
  const [activeRepo, setActiveRepo] = useState<string>('');
  const [activeApiKey, setActiveApiKey] = useState<string>('');
  const [currentUser, setCurrentUser] = useState<{ username: string; avatarUrl?: string } | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: ToastVariant } | null>(null);
  const [isBootstrapping, setIsBootstrapping] = useState(true);

  // Bootstrap: check setup status, restore session, load data
  React.useEffect(() => {
    async function bootstrap() {
      try {
        // Step 1: Check if instance is configured
        const setupStatus = await engineClient.getSetupStatus();
        if (!setupStatus.configured) {
          setCurrentScreen('setup');
          setIsBootstrapping(false);
          return;
        }

        // Step 2: Check for OAuth callback token in URL
        const params = new URLSearchParams(window.location.search);
        const urlToken = params.get('token');
        const authStatus = params.get('auth');

        if (urlToken && authStatus === 'success') {
          // Store token from OAuth callback
          localStorage.setItem('triage_session', urlToken);
          // Clean URL
          window.history.replaceState({}, '', window.location.pathname);
        }

        // Step 3: Check for setup wizard redirect params
        const setupStep = params.get('setup_step');
        if (setupStep) {
          setCurrentScreen('setup');
          setIsBootstrapping(false);
          return;
        }

        // Step 4: Validate existing session
        const storedToken = urlToken || localStorage.getItem('triage_session');
        if (!storedToken) {
          setCurrentScreen('login');
          setIsBootstrapping(false);
          return;
        }

        const sessionResult = await engineClient.verifySession(storedToken);
        if (!sessionResult.valid || !sessionResult.user) {
          localStorage.removeItem('triage_session');
          setCurrentScreen('login');
          setIsBootstrapping(false);
          return;
        }

        // Session is valid — set user
        engineClient.setAuthToken(storedToken);
        setCurrentUser({
          username: sessionResult.user.username,
          avatarUrl: sessionResult.user.avatar_url,
        });

        // Step 5: Load projects
        const projects = await engineClient.getProjects();
        if (projects && projects.length > 0) {
          const firstProject = projects[0];
          setActiveRepo(`${firstProject.owner}/${firstProject.repo}`);
          // Load incidents
          const liveIncidents = await engineClient.getIncidents();
          if (liveIncidents && liveIncidents.length > 0) {
            const mapped: Incident[] = liveIncidents.map((item: any) => ({
              id: item.id || `INC-${Math.random().toString(36).substr(2, 6).toUpperCase()}`,
              title: item.title || 'Runtime Go Panic',
              status: item.status || 'CRITICAL',
              triggeringFile: `${item.file}:${item.line}`,
              triggeringLine: item.line,
              latencyMs: 14,
              commitHash: '8f3a1b4',
              branch: 'main',
              timestamp: new Date(item.created_at || Date.now()).toISOString().replace('T', ' ').substring(0, 19) + ' UTC',
              goroutineId: 'goroutine [running]',
              panicMessage: item.panic_message,
              rawStackTrace: item.stack_trace,
              astSnippet: {
                functionName: 'main',
                file: item.file,
                startLine: item.line,
                lines: [
                  { lineNum: item.line, content: item.ast_snippet || item.panic_message, isTriggerLine: true },
                ],
              },
              geminiAnalysis: item.root_cause
                ? {
                    rootCause: item.root_cause,
                    explanation: item.root_cause,
                    severity: 'CRITICAL',
                    recommendedFix: item.suggested_fix,
                  }
                : undefined,
            }));
            setIncidents(mapped);
            if (mapped[0]) setSelectedIncidentId(mapped[0].id);
          }
          setCurrentScreen('dashboard');
        } else {
          // No projects — go to onboarding
          setCurrentScreen('new');
        }
      } catch (e) {
        console.warn('Bootstrap error:', e);
        setCurrentScreen('login');
      } finally {
        setIsBootstrapping(false);
      }
    }
    bootstrap();
  }, []);

  // Selected incident object
  const selectedIncident = incidents.find((i) => i.id === selectedIncidentId) || incidents[0];

  const criticalCount = incidents.filter((i) => i.status === 'CRITICAL').length;

  // Show Toast
  const showToast = (message: string, variant: ToastVariant = 'success') => {
    setToast({ message, variant });
    setTimeout(() => setToast(null), 3500);
  };

  const handleLoginSuccess = (user: { username: string; avatarUrl?: string }) => {
    setCurrentUser(user);
    showToast(`Authenticated as @${user.username} via GitHub`, 'success');
  };

  const handleLogout = () => {
    localStorage.removeItem('triage_session');
    engineClient.setAuthToken(null);
    setCurrentUser(null);
    setActiveRepo('');
    setActiveApiKey('');
    setIncidents([]);
    setCurrentScreen('login');
    showToast('Logged out of Triage Console', 'success');
  };

  const handleProjectSetup = async (repo: string, apiKey: string) => {
    setActiveRepo(repo);
    let finalKey = apiKey;
    try {
      const res = await engineClient.createProject(repo, currentUser?.username);
      if (res && res.api_key) {
        finalKey = res.api_key;
      }
    } catch {
      // Fallback to local generated key
    }
    setActiveApiKey(finalKey);
    showToast(`Project ${repo} setup complete with API Key ${finalKey.substring(0, 12)}...`, 'success');
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 flex flex-col font-sans selection:bg-indigo-500 selection:text-white">
      {/* Global Toast Notification */}
      {toast && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center space-x-3 bg-slate-900 border border-slate-800 text-slate-100 px-4 py-3 rounded-lg shadow-xl animate-in fade-in slide-in-from-bottom-5">
          {toast.variant === 'error' ? (
            <AlertTriangle className="w-5 h-5 text-red-400 shrink-0" />
          ) : (
            <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
          )}
          <span className="text-sm font-medium">{toast.message}</span>
          <button onClick={() => setToast(null)} className="text-slate-400 hover:text-white transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {!isBootstrapping && currentScreen !== 'setup' && currentScreen !== 'login' && (
        <Header
          currentScreen={currentScreen}
          onNavigate={(screen) => setCurrentScreen(screen)}
          criticalCount={criticalCount}
          activeRepo={activeRepo}
          currentUser={currentUser}
          onLogout={handleLogout}
        />
      )}

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {isBootstrapping && (
          <div className="flex-1 flex items-center justify-center py-20">
            <div className="text-center space-y-3">
              <div className="w-8 h-8 border-2 border-slate-300 border-t-slate-900 rounded-full animate-spin mx-auto" />
              <p className="text-sm font-mono text-slate-500">Initializing Triage Console...</p>
            </div>
          </div>
        )}

        {currentScreen === 'setup' && (
          <SetupWizardPage onNavigate={(screen) => setCurrentScreen(screen)} />
        )}

        {currentScreen === 'login' && (
          <LoginPage
            onLoginSuccess={(user) => {
              handleLoginSuccess(user);
              setCurrentScreen('new');
            }}
            onNavigate={(screen) => setCurrentScreen(screen)}
          />
        )}
        {currentScreen === 'new' && (
          <OnboardingPage
            currentUser={currentUser}
            onNavigate={(screen) => setCurrentScreen(screen)}
            onProjectSetup={(repo, key) => {
              handleProjectSetup(repo, key);
              setCurrentScreen('dashboard');
            }}
          />
        )}

        {currentScreen === 'dashboard' && (
          <DashboardPage
            incidents={incidents}
            onNavigate={(screen) => setCurrentScreen(screen)}
            activeRepo={activeRepo}
            apiKey={activeApiKey}
            onSelectIncident={(id) => {
              setSelectedIncidentId(id);
              setCurrentScreen('incident_detail');
            }}
          />
        )}

        {currentScreen === 'incident_detail' &&
          (selectedIncident ? (
            <IncidentDetailPage
              incident={selectedIncident}
              allIncidents={incidents}
              onSelectIncident={(id) => setSelectedIncidentId(id)}
              onNavigate={(screen) => setCurrentScreen(screen)}
            />
          ) : (
            <div className="text-center py-16 bg-white rounded-xl border border-slate-200 shadow-sm">
              <h3 className="text-lg font-semibold text-slate-800">No incident selected</h3>
              <p className="text-sm text-slate-500 mt-1">Select an incident from the dashboard or simulate a panic to view details.</p>
            </div>
          ))}

        {currentScreen === 'ast' && <AstExplorerPage onNavigate={(screen) => setCurrentScreen(screen)} commitIndexes={[]} astFiles={[]} />}
        {currentScreen === 'webhooks' && <WebhooksPage onNavigate={(screen) => setCurrentScreen(screen)} logs={[]} />}
        {currentScreen === 'team' && <TeamPage teamMembers={[]} onNavigate={(screen) => setCurrentScreen(screen)} />}
        {currentScreen === 'status' && <SystemStatusPage onNavigate={(screen) => setCurrentScreen(screen)} health={[]} metrics={[]} />}
        {currentScreen === 'settings' && <SettingsPage apiKeys={[]} onNavigate={(screen) => setCurrentScreen(screen)} />}
      </main>

      {!isBootstrapping && currentScreen !== 'setup' && currentScreen !== 'login' && (
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
              <span className="text-slate-900 font-medium">gemini-3.5-flash</span> &amp; AST Parser
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}
