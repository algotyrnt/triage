/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Incident, ScreenId, Project } from '@/types';

import { Header } from '@/components/Header';
import { LoginPage } from '@/components/screens/LoginPage';
import { OnboardingPage } from '@/components/screens/OnboardingPage';
import { ProjectsPage } from '@/components/screens/ProjectsPage';
import { DashboardPage } from '@/components/screens/DashboardPage';
import { IncidentDetailPage } from '@/components/screens/IncidentDetailPage';
import { AstExplorerPage } from '@/components/screens/AstExplorerPage';
import { WebhooksPage } from '@/components/screens/WebhooksPage';
import { TeamPage } from '@/components/screens/TeamPage';
import { SystemStatusPage } from '@/components/screens/SystemStatusPage';
import { SettingsPage } from '@/components/screens/SettingsPage';
import { SetupWizardPage } from '@/components/screens/SetupWizardPage';

import { engineClient } from '@/services/engineClient';
import { logger } from '@/services/logger';
import { AlertTriangle, CheckCircle2, X } from 'lucide-react';

type ToastVariant = 'success' | 'error';

export default function App({ initialScreen = 'projects' }: { initialScreen?: ScreenId }) {
  const [currentScreen, setCurrentScreen] = useState<ScreenId>(initialScreen);
  const [projects, setProjects] = useState<Project[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedIncidentId, setSelectedIncidentId] = useState<string>('');
  const [activeRepo, setActiveRepo] = useState<string>('');
  const [activeRootDir, setActiveRootDir] = useState<string>('');
  const [activeApiKey, setActiveApiKey] = useState<string>('');
  const [isRefreshingProjects, setIsRefreshingProjects] = useState(false);
  const [currentUser, setCurrentUser] = useState<{
    username: string;
    avatarUrl?: string;
  } | null>(null);
  const [toast, setToast] = useState<{
    message: string;
    variant: ToastVariant;
  } | null>(null);
  const [isBootstrapping, setIsBootstrapping] = useState(true);

  const mapIncidents = (rawIncidents: any[]): Incident[] => {
    return rawIncidents.map((item: any) => ({
      id: item.id || `INC-${Math.random().toString(36).substr(2, 6).toUpperCase()}`,
      repositoryId: item.repository_id || '',
      repositoryName: item.repository_name || '',
      title: item.title || 'Runtime Go Panic',
      status: item.status || 'CRITICAL',
      triggeringFile: `${item.file}:${item.line}`,
      triggeringLine: item.line,
      latencyMs: 14,
      commitHash: '8f3a1b4',
      branch: 'main',
      timestamp:
        new Date(item.created_at || Date.now()).toISOString().replace('T', ' ').substring(0, 19) +
        ' UTC',
      goroutineId: 'goroutine [running]',
      panicMessage: item.panic_message,
      rawStackTrace: item.stack_trace,
      githubIssueUrl: item.github_issue_url || undefined,
      githubIssueNumber: item.github_issue_number ? Number(item.github_issue_number) : undefined,
      githubPrUrl: item.github_pr_url || undefined,
      githubPrNumber: item.github_pr_number ? Number(item.github_pr_number) : undefined,
      suggestedPatch: item.suggested_patch || undefined,
      astSnippet: {
        functionName: 'main',
        file: item.file,
        startLine: item.line,
        lines: [
          {
            lineNum: item.line,
            content: item.ast_snippet || item.panic_message,
            isTriggerLine: true,
          },
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
  };

  // Bootstrap: check setup status, restore session, load data
  React.useEffect(() => {
    async function bootstrap() {
      try {
        // Step 1: Immediately extract and persist OAuth callback token from URL
        const params = new URLSearchParams(window.location.search);
        const urlToken = params.get('token');
        const setupStep = params.get('setup_step');
        const targetProject = params.get('project');
        const targetScreen = params.get('screen') as ScreenId | null;

        if (urlToken) {
          localStorage.setItem('triage_session', urlToken);
          engineClient.setAuthToken(urlToken);
          window.history.replaceState({}, '', window.location.pathname);
        }

        const storedToken = urlToken || localStorage.getItem('triage_session');
        let authenticatedUser: any = null;

        // Step 2: If we have a stored token, verify the user session directly with GitHub API
        if (storedToken) {
          try {
            const res = await fetch('https://api.github.com/user', {
              headers: {
                Authorization: `Bearer ${storedToken}`,
                Accept: 'application/vnd.github+json',
                'X-GitHub-Api-Version': '2022-11-28',
              },
            });

            if (res.ok) {
              const user = await res.json();
              authenticatedUser = user;
              engineClient.setAuthToken(storedToken);
              setCurrentUser({
                username: user.login,
                avatarUrl: user.avatar_url,
              });
            } else {
              localStorage.removeItem('triage_session');
              engineClient.setAuthToken(null);
            }
          } catch (e) {
            console.error('Failed to verify GitHub token', e);
            localStorage.removeItem('triage_session');
            engineClient.setAuthToken(null);
          }
        }

        // Step 3: Check setup status
        const setupStatus = await engineClient.getSetupStatus();
        if (!setupStatus.configured) {
          setCurrentScreen('setup');
          setIsBootstrapping(false);
          return;
        }

        if (setupStep) {
          setCurrentScreen('setup');
          setIsBootstrapping(false);
          return;
        }

        // Step 4: If not authenticated, go to login page
        if (!authenticatedUser) {
          setCurrentScreen('login');
          setIsBootstrapping(false);
          return;
        }

        // Step 5: Authenticated — Load projects & incidents
        const loadedProjects = await engineClient.getProjects();
        if (loadedProjects && loadedProjects.length > 0) {
          setProjects(loadedProjects);

          // Find target project if specified in URL or default to first
          let selectedProject = loadedProjects[0];
          if (targetProject) {
            const matched = loadedProjects.find(
              (p) =>
                `${p.owner}/${p.repo}`.toLowerCase() === targetProject.toLowerCase() ||
                p.repo.toLowerCase() === targetProject.toLowerCase(),
            );
            if (matched) selectedProject = matched;
          }

          const owner = selectedProject.owner;
          const repo = selectedProject.repo;
          const rootDir = selectedProject.root_dir || '';
          setActiveRepo(`${owner}/${repo}`);
          setActiveRootDir(rootDir);

          const storageKey = `triage_key_${owner}_${repo}_${rootDir}`;
          const localStoredKey = localStorage.getItem(storageKey);
          const keyToUse = localStoredKey || selectedProject.api_key_masked || '';
          setActiveApiKey(keyToUse);

          // Load incidents
          const liveIncidents = await engineClient.getIncidents();
          if (liveIncidents && liveIncidents.length > 0) {
            const mapped = mapIncidents(liveIncidents);
            setIncidents(mapped);
            const targetIncidentId = params.get('incident') || params.get('incident_id');
            if (targetIncidentId && mapped.some((i) => i.id === targetIncidentId)) {
              setSelectedIncidentId(targetIncidentId);
              setCurrentScreen('incident_detail');
            } else if (targetProject) {
              setCurrentScreen('dashboard');
            } else if (targetScreen) {
              setCurrentScreen(targetScreen);
            } else {
              // Default landing page is projects overview allowing the user to select
              setCurrentScreen('projects');
            }
          } else {
            if (targetProject) {
              setCurrentScreen('dashboard');
            } else if (targetScreen) {
              setCurrentScreen(targetScreen);
            } else {
              setCurrentScreen('projects');
            }
          }
        } else {
          // No projects — go to onboarding
          setCurrentScreen('new');
        }
      } catch (e) {
        logger.warn('Bootstrap error:', e);
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
    setActiveRootDir('');
    setActiveApiKey('');
    setProjects([]);
    setIncidents([]);
    setCurrentScreen('login');
    showToast('Logged out of Triage Console', 'success');
  };

  const handleSelectProject = (project: Project, targetScreen: ScreenId = 'dashboard') => {
    const owner = project.owner;
    const repo = project.repo;
    const rootDir = project.root_dir || '';
    setActiveRepo(`${owner}/${repo}`);
    setActiveRootDir(rootDir);

    const storageKey = `triage_key_${owner}_${repo}_${rootDir}`;
    const localStoredKey = localStorage.getItem(storageKey);
    const keyToUse = localStoredKey || project.api_key_masked || '';
    setActiveApiKey(keyToUse);
    setCurrentScreen(targetScreen);
  };

  const handleRefreshProjects = async () => {
    setIsRefreshingProjects(true);
    try {
      const [loadedProjects, liveIncidents] = await Promise.all([
        engineClient.getProjects(),
        engineClient.getIncidents(),
      ]);
      if (loadedProjects) {
        setProjects(loadedProjects);
      }
      if (liveIncidents) {
        setIncidents(mapIncidents(liveIncidents));
      }
      showToast('Projects and telemetry updated', 'success');
    } catch (e) {
      logger.warn('Refresh error:', e);
      showToast('Failed to refresh projects', 'error');
    } finally {
      setIsRefreshingProjects(false);
    }
  };

  const handleProjectSetup = async (repo: string, apiKey: string, rootDir?: string) => {
    setActiveRepo(repo);
    setActiveRootDir(rootDir || '');
    let finalKey = apiKey;
    const parts = repo.split('/');
    const owner = parts[0] || currentUser?.username || 'algotyrnt';
    const repoName = parts[1] || repo;
    const cleanRoot = rootDir || '';
    const storageKey = `triage_key_${owner}_${repoName}_${cleanRoot}`;

    try {
      const res = await engineClient.createProject(repo, rootDir, currentUser?.username);
      if (res && res.api_key) {
        finalKey = res.api_key;
      }
    } catch {
      // Fallback to local generated key
    }
    setActiveApiKey(finalKey);
    localStorage.setItem(storageKey, finalKey);

    // Refresh projects list
    try {
      const updatedProjects = await engineClient.getProjects();
      if (updatedProjects && updatedProjects.length > 0) {
        setProjects(updatedProjects);
      }
    } catch (e) {
      logger.warn('Failed to reload projects list', e);
    }

    showToast(
      `Project ${repo}${cleanRoot ? ` (${cleanRoot})` : ''} setup complete with API Key ${finalKey.substring(0, 12)}...`,
      'success',
    );
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
          <button
            onClick={() => setToast(null)}
            className="text-slate-400 hover:text-white transition-colors cursor-pointer"
          >
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
          activeRootDir={activeRootDir}
          projects={projects}
          onSelectProject={handleSelectProject}
          currentUser={currentUser}
          onLogout={handleLogout}
        />
      )}

      <main className="flex-1 w-full">
        {isBootstrapping ? (
          <div className="flex-1 flex items-center justify-center py-20">
            <div className="text-center space-y-3">
              <div className="w-8 h-8 border-2 border-slate-300 border-t-slate-900 rounded-full animate-spin mx-auto" />
              <p className="text-sm font-mono text-slate-500">Initializing Triage Console...</p>
            </div>
          </div>
        ) : (
          <>
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

            {currentScreen === 'projects' && (
              <ProjectsPage
                projects={projects}
                incidents={incidents}
                onSelectProject={handleSelectProject}
                onNavigate={(screen) => setCurrentScreen(screen)}
                onRefresh={handleRefreshProjects}
                isRefreshing={isRefreshingProjects}
              />
            )}

            {currentScreen === 'new' && (
              <OnboardingPage
                currentUser={currentUser}
                onNavigate={(screen) => setCurrentScreen(screen)}
                onProjectSetup={(repo, key, rootDir) => {
                  handleProjectSetup(repo, key, rootDir);
                  setCurrentScreen('dashboard');
                }}
              />
            )}

            {currentScreen === 'dashboard' && (
              <DashboardPage
                incidents={incidents}
                onNavigate={(screen) => setCurrentScreen(screen)}
                activeRepo={activeRepo}
                rootDir={activeRootDir}
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
                  onIncidentUpdated={(updated) =>
                    setIncidents((prev) =>
                      prev.map((inc) => (inc.id === updated.id ? updated : inc)),
                    )
                  }
                  onNavigate={(screen) => setCurrentScreen(screen)}
                />
              ) : (
                <div className="text-center py-16 bg-white rounded-xl border border-slate-200 shadow-sm">
                  <h3 className="text-lg font-semibold text-slate-800">No incident selected</h3>
                  <p className="text-sm text-slate-500 mt-1">
                    Select an incident from the dashboard or simulate a panic to view details.
                  </p>
                </div>
              ))}

            {currentScreen === 'ast' && (
              <AstExplorerPage
                onNavigate={(screen) => setCurrentScreen(screen)}
                commitIndexes={[]}
                astFiles={[]}
              />
            )}
            {currentScreen === 'webhooks' && (
              <WebhooksPage onNavigate={(screen) => setCurrentScreen(screen)} logs={[]} />
            )}
            {currentScreen === 'team' && (
              <TeamPage teamMembers={[]} onNavigate={(screen) => setCurrentScreen(screen)} />
            )}
            {currentScreen === 'status' && (
              <SystemStatusPage
                onNavigate={(screen) => setCurrentScreen(screen)}
                health={[]}
                metrics={[]}
              />
            )}
            {currentScreen === 'settings' && (
              <SettingsPage
                apiKeys={[]}
                activeApiKey={activeApiKey}
                onKeyUpdated={(newKey) => setActiveApiKey(newKey)}
                onNavigate={(screen) => setCurrentScreen(screen)}
                activeRepo={activeRepo}
                activeRootDir={activeRootDir}
              />
            )}
          </>
        )}
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
              Powered by <span className="text-slate-900 font-medium">Gemini AI</span> &amp; AST
              Parser
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}
