/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Incident, IncidentStatus, ScreenId, Project } from '@/types';

import { Header } from '@/components/Header';
import { LoginPage } from '@/components/screens/LoginPage';
import { OnboardingPage } from '@/components/screens/OnboardingPage';
import { ProjectsPage } from '@/components/screens/ProjectsPage';
import { DashboardPage } from '@/components/screens/DashboardPage';
import { IncidentDetailPage } from '@/components/screens/IncidentDetailPage';
import { TeamPage } from '@/components/screens/TeamPage';
import { SettingsPage } from '@/components/screens/SettingsPage';
import { SetupWizardPage } from '@/components/screens/SetupWizardPage';

import { engineClient } from '@/services/engineClient';
import { logger } from '@/services/logger';
import { useLatestRelease } from '@/components/useLatestRelease';
import { AlertTriangle, CheckCircle2, X, BookOpen, ExternalLink, Sparkles } from 'lucide-react';

type ToastVariant = 'success' | 'error';

export default function App({ initialScreen = 'projects' }: { initialScreen?: ScreenId }) {
  const release = useLatestRelease();
  const [currentScreen, setCurrentScreen] = useState<ScreenId>(initialScreen);
  const [projects, setProjects] = useState<Project[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedIncidentId, setSelectedIncidentId] = useState<string>('');
  const [activeRepo, setActiveRepo] = useState<string>('');
  const [activeRootDir, setActiveRootDir] = useState<string>('');
  const [activeApiKey, setActiveApiKey] = useState<string>('');
  const [isRefreshingProjects, setIsRefreshingProjects] = useState(false);
  const [currentUser, setCurrentUser] = useState<{
    id?: string;
    username: string;
    avatarUrl?: string;
    role?: string;
  } | null>(null);
  const [toast, setToast] = useState<{
    message: string;
    variant: ToastVariant;
  } | null>(null);
  const [isBootstrapping, setIsBootstrapping] = useState(true);

  const navigate = (screen: ScreenId) => {
    setCurrentScreen(screen);
  };

  const mapIncidents = (rawIncidents: any[]): Incident[] => {
    if (!Array.isArray(rawIncidents)) return [];
    return rawIncidents
      .filter((item: any) => item && (item.id || item.file || item.panic_message))
      .map((item: any) => ({
        id: item.id || `INC-${item.fingerprint ? item.fingerprint.substring(0, 8) : 'EVENT'}`,
        repositoryId: item.repository_id || '',
        repositoryName: item.repository_name || '',
        title: item.title || item.panic_message || 'Runtime Go Panic',
        status: (item.status === 'RESOLVED' ? 'RESOLVED' : 'OPEN') as IncidentStatus,
        triggeringFile: item.file ? `${item.file}:${item.line || 1}` : 'unknown:0',
        triggeringLine: item.line || 1,
        latencyMs: item.latency_ms || 14,
        commitHash: item.commit_hash || 'main',
        branch: item.branch || 'main',
        timestamp:
          new Date(item.created_at || Date.now()).toISOString().replace('T', ' ').substring(0, 19) +
          ' UTC',
        goroutineId: 'goroutine [running]',
        fingerprint: item.fingerprint || undefined,
        occurrenceCount: item.occurrence_count || 1,
        lastSeenAt: item.last_seen_at ? new Date(item.last_seen_at).toUTCString() : undefined,
        severity: item.severity || undefined,
        aiProvider: item.ai_provider || undefined,
        aiModel: item.ai_model || undefined,
        panicMessage: item.panic_message || item.title || '',
        rawStackTrace: item.stack_trace || '',
        githubIssueUrl: item.github_issue_url || undefined,
        githubIssueNumber: item.github_issue_number ? Number(item.github_issue_number) : undefined,
        githubPrUrl: item.github_pr_url || undefined,
        githubPrNumber: item.github_pr_number ? Number(item.github_pr_number) : undefined,
        suggestedPatch: item.suggested_patch || undefined,
        astSnippet: {
          functionName: item.function_name || 'main',
          file: item.file || '',
          startLine: item.line || 1,
          lines: [
            {
              lineNum: item.line || 1,
              content: item.ast_snippet || item.panic_message || '',
              isTriggerLine: true,
            },
          ],
        },
        aiAnalysis: item.root_cause
          ? {
              rootCause: item.root_cause,
              explanation: item.root_cause,
              severity: item.severity || undefined,
              recommendedFix: item.suggested_fix || '',
            }
          : undefined,
      }));
  };

  // Bootstrap: check setup status, restore session, load data
  React.useEffect(() => {
    async function bootstrap() {
      try {
        // Step 1: Clean up one-time URL query parameters
        const params = new URLSearchParams(window.location.search);
        const setupStep = params.get('setup_step');
        const targetProject = params.get('project');
        const targetScreen = params.get('screen') as ScreenId | null;
        const isInstalledRedirect = params.get('installed') === 'true';

        if (
          params.get('auth') ||
          isInstalledRedirect ||
          params.get('setup_error') ||
          params.get('installed') ||
          params.get('app_created')
        ) {
          window.history.replaceState({}, '', window.location.pathname);
        }

        let authenticatedUser: any = null;

        // Step 2: Verify user session with Engine backend (cookie or Authorization header)
        try {
          const user = await engineClient.getAuthUser();
          if (user) {
            authenticatedUser = user;
            setCurrentUser({
              id: user.id,
              username: user.username,
              avatarUrl: user.avatar_url,
              role: user.role,
            });
          } else {
            localStorage.removeItem('triage_session');
            engineClient.setAuthToken(null);
          }
        } catch (e) {
          console.error('Failed to verify session with Engine', e);
          localStorage.removeItem('triage_session');
          engineClient.setAuthToken(null);
        }

        // Step 3: Check setup status
        const setupStatus = await engineClient.getSetupStatus();
        if (!setupStatus.configured) {
          navigate('setup');
          setIsBootstrapping(false);
          return;
        }

        if (setupStep && !setupStatus.configured) {
          navigate('setup');
          setIsBootstrapping(false);
          return;
        }

        // Step 4: If not authenticated, go to login page
        if (!authenticatedUser) {
          navigate('login');
          setIsBootstrapping(false);
          return;
        }

        // Step 5: Authenticated — Determine target screen (defaults to 'projects')
        let screenToOpen: ScreenId = 'projects';
        if (targetScreen) {
          screenToOpen = targetScreen;
        } else if (isInstalledRedirect) {
          screenToOpen = 'new';
        }

        // Step 6: Load projects & incidents
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
          setActiveApiKey(selectedProject.api_key_masked || '');

          // Load incidents
          const liveIncidents = await engineClient.getIncidents();
          if (liveIncidents && liveIncidents.length > 0) {
            const mapped = mapIncidents(liveIncidents);
            setIncidents(mapped);
            const targetIncidentId = params.get('incident') || params.get('incident_id');
            if (targetIncidentId && mapped.some((i) => i.id === targetIncidentId)) {
              setSelectedIncidentId(targetIncidentId);
              screenToOpen = 'incident_detail';
            } else if (targetProject) {
              screenToOpen = 'dashboard';
            }
          } else if (targetProject) {
            screenToOpen = 'dashboard';
          }
        }

        navigate(screenToOpen);
      } catch (e) {
        logger.warn('Bootstrap error:', e);
        navigate('login');
      } finally {
        setIsBootstrapping(false);
      }
    }
    bootstrap();
  }, []);

  // Subscribe to real-time telemetry events via Server-Sent Events (SSE)
  React.useEffect(() => {
    if (isBootstrapping || currentScreen === 'login' || currentScreen === 'setup') {
      return;
    }

    const streamUrl = engineClient.getEventsStreamUrl();
    let es: EventSource | null = null;
    try {
      es = new EventSource(streamUrl);
      es.onmessage = (event) => {
        try {
          if (!event.data) return;
          const raw = JSON.parse(event.data);
          if (raw.type === 'incident_created' && raw.data) {
            const mapped = mapIncidents([raw.data]);
            if (mapped.length > 0) {
              const newInc = mapped[0];
              setIncidents((prev) => {
                if (prev.some((i) => i.id === newInc.id)) {
                  return prev.map((i) => (i.id === newInc.id ? newInc : i));
                }
                return [newInc, ...prev];
              });
              showToast(`New Panic Ingested: ${newInc.triggeringFile}`, 'error');
            }
          } else if (
            (raw.type === 'incident_updated' || raw.type === 'incident_resolved') &&
            raw.data
          ) {
            const mapped = mapIncidents([raw.data]);
            if (mapped.length > 0) {
              const updated = mapped[0];
              setIncidents((prev) => prev.map((i) => (i.id === updated.id ? updated : i)));
              if (raw.type === 'incident_resolved' || updated.status === 'RESOLVED') {
                showToast(`Incident Resolved: ${updated.triggeringFile}`, 'success');
              }
            }
          }
        } catch {}
      };
    } catch {}

    return () => {
      if (es) es.close();
    };
  }, [isBootstrapping, currentScreen, activeRepo]);

  const selectedIncident = incidents.find((i) => i.id === selectedIncidentId);

  const showToast = (message: string, variant: ToastVariant = 'success') => {
    setToast({ message, variant });
    setTimeout(() => setToast(null), 4000);
  };

  const activeServiceIncidents = React.useMemo(() => {
    const cleanRepo = activeRepo.toLowerCase();
    const cleanRoot = (activeRootDir || '').toLowerCase();
    const activeProject =
      projects.find((p) => {
        const slug = `${p.owner}/${p.repo}`.toLowerCase();
        const pRoot = (p.root_dir || '').toLowerCase();
        return (slug === cleanRepo || p.repo.toLowerCase() === cleanRepo) && pRoot === cleanRoot;
      }) ||
      projects.find((p) => {
        const slug = `${p.owner}/${p.repo}`.toLowerCase();
        return slug === cleanRepo || p.repo.toLowerCase() === cleanRepo;
      });

    const cleanActiveRepo = activeRepo.toLowerCase();
    const activeRepoName = cleanActiveRepo.includes('/')
      ? cleanActiveRepo.split('/')[1]
      : cleanActiveRepo;
    const cleanRootDir = (activeRootDir || '').replace(/^\/+|\/+$/g, '').toLowerCase();

    return incidents.filter((incident) => {
      if (activeProject?.id && incident.repositoryId) {
        return incident.repositoryId === activeProject.id;
      }
      if (incident.repositoryId && projects.length > 0) {
        const matchedProj = projects.find((p) => p.id === incident.repositoryId);
        if (matchedProj) {
          const projSlug = `${matchedProj.owner}/${matchedProj.repo}`.toLowerCase();
          const projRoot = (matchedProj.root_dir || '').replace(/^\/+|\/+$/g, '').toLowerCase();
          const repoMatches =
            projSlug === cleanActiveRepo ||
            matchedProj.repo.toLowerCase() === cleanActiveRepo ||
            matchedProj.repo.toLowerCase() === activeRepoName;
          const rootMatches = projRoot === cleanRootDir;
          return repoMatches && rootMatches;
        }
      }
      if (incident.repositoryName) {
        const incRepo = incident.repositoryName.toLowerCase();
        if (incRepo !== cleanActiveRepo && incRepo !== activeRepoName) return false;
      }
      if (cleanRootDir) {
        const trigFile = (incident.triggeringFile || incident.astSnippet?.file || '').toLowerCase();
        if (trigFile && !trigFile.startsWith(cleanRootDir + '/') && trigFile !== cleanRootDir) {
          return false;
        }
      }
      return true;
    });
  }, [incidents, projects, activeRepo, activeRootDir]);

  const activeServiceOpenCount = React.useMemo(() => {
    return activeServiceIncidents.filter((i) => i.status === 'OPEN').length;
  }, [activeServiceIncidents]);

  const handleLoginSuccess = (user: any) => {
    setCurrentUser(user);
    showToast(`Authenticated as @${user.username} via GitHub`, 'success');
  };

  const handleLogout = () => {
    engineClient.logout().catch(() => {});
    localStorage.removeItem('triage_session');
    engineClient.setAuthToken(null);
    setCurrentUser(null);
    setActiveRepo('');
    setActiveRootDir('');
    setActiveApiKey('');
    setProjects([]);
    setIncidents([]);
    navigate('login');
    showToast('Logged out of Triage Console', 'success');
  };

  const handleSelectProject = (project: Project, targetScreen: ScreenId = 'dashboard') => {
    const owner = project.owner;
    const repo = project.repo;
    const rootDir = project.root_dir || '';
    setActiveRepo(`${owner}/${repo}`);
    setActiveRootDir(rootDir);

    const keyToUse = project.api_key_masked || '';
    setActiveApiKey(keyToUse);
    navigate(targetScreen);
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

  const handleProjectSetup = async (
    repo: string,
    apiKey: string,
    rootDir?: string,
    projectContext?: string,
  ) => {
    setActiveRepo(repo);
    setActiveRootDir(rootDir || '');
    let finalKey = apiKey;
    const cleanRoot = rootDir || '';

    // Only create project on backend if an API key was not already created during onboarding
    if (!finalKey || finalKey.includes('•') || finalKey.includes('...')) {
      try {
        const res = await engineClient.createProject(
          repo,
          rootDir,
          currentUser?.username,
          projectContext,
        );
        if (res && res.api_key) {
          finalKey = res.api_key;
        }
      } catch (e) {
        logger.warn('Failed to create project on backend during setup completion', e);
      }
    }

    if (finalKey) {
      setActiveApiKey(finalKey);
    }

    // Refresh projects list
    try {
      const updatedProjects = await engineClient.getProjects();
      if (updatedProjects && updatedProjects.length > 0) {
        setProjects(updatedProjects);
      }
    } catch (e) {
      logger.warn('Failed to reload projects list', e);
    }

    navigate('dashboard');

    showToast(
      `Project ${repo}${cleanRoot ? ` (${cleanRoot})` : ''} setup complete with API Key ${finalKey ? finalKey.substring(0, 12) + '...' : ''}`,
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
          onNavigate={navigate}
          openCount={activeServiceOpenCount}
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
            {currentScreen === 'setup' && <SetupWizardPage onNavigate={navigate} />}

            {currentScreen === 'login' && (
              <LoginPage
                onLoginSuccess={(user) => {
                  handleLoginSuccess(user);
                  navigate('projects');
                }}
                onNavigate={navigate}
              />
            )}

            {currentScreen === 'projects' && (
              <ProjectsPage
                projects={projects}
                incidents={incidents}
                onSelectProject={handleSelectProject}
                onNavigate={navigate}
                onRefresh={handleRefreshProjects}
                isRefreshing={isRefreshingProjects}
              />
            )}

            {currentScreen === 'new' && (
              <OnboardingPage
                currentUser={currentUser}
                onNavigate={navigate}
                onProjectSetup={(repo, key, rootDir, projectContext) => {
                  handleProjectSetup(repo, key, rootDir, projectContext);
                }}
              />
            )}

            {currentScreen === 'dashboard' && (
              <DashboardPage
                incidents={incidents}
                onNavigate={navigate}
                activeRepo={activeRepo}
                rootDir={activeRootDir}
                apiKey={activeApiKey}
                projects={projects}
                onSelectIncident={(id) => {
                  setSelectedIncidentId(id);
                  navigate('incident_detail');
                }}
              />
            )}

            {currentScreen === 'incident_detail' &&
              (selectedIncident ? (
                <IncidentDetailPage
                  incident={selectedIncident}
                  allIncidents={
                    activeServiceIncidents.length > 0 ? activeServiceIncidents : [selectedIncident]
                  }
                  onSelectIncident={(id) => setSelectedIncidentId(id)}
                  onIncidentUpdated={(updated) =>
                    setIncidents((prev) =>
                      prev.map((inc) => (inc.id === updated.id ? updated : inc)),
                    )
                  }
                  onNavigate={navigate}
                />
              ) : (
                <div className="text-center py-16 bg-white rounded-xl border border-slate-200 shadow-sm">
                  <h3 className="text-lg font-semibold text-slate-800">No incident selected</h3>
                  <p className="text-sm text-slate-500 mt-1">
                    Select an incident from the dashboard or simulate a panic to view details.
                  </p>
                </div>
              ))}

            {currentScreen === 'team' && (
              <TeamPage currentUser={currentUser} onNavigate={navigate} />
            )}
            {currentScreen === 'settings' && (
              <SettingsPage
                apiKeys={[]}
                activeApiKey={activeApiKey}
                onKeyUpdated={(newKey) => setActiveApiKey(newKey)}
                onNavigate={navigate}
                activeRepo={activeRepo}
                activeRootDir={activeRootDir}
              />
            )}
          </>
        )}
      </main>

      {!isBootstrapping && currentScreen !== 'setup' && currentScreen !== 'login' && (
        <footer className="border-t border-slate-200 bg-white py-4 font-mono text-xs text-slate-500">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-2.5">
              <div className="flex items-center space-x-2">
                <span className="font-semibold text-slate-800">Triage Engine</span>
                <span className="text-[11px] bg-slate-100 text-slate-700 px-1.5 py-0.5 rounded-sm border border-slate-200 font-bold">
                  {release.engineVersion || 'v0.1.0'}
                </span>
              </div>

              {release.hasUpdate && (
                <a
                  href={release.releaseUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 bg-amber-50 hover:bg-amber-100 text-amber-800 border border-amber-200 px-2 py-0.5 rounded-sm text-[11px] font-semibold transition-colors cursor-pointer"
                  title={`Update available: ${release.latestVersion}`}
                >
                  <Sparkles className="w-3 h-3 text-amber-600" />
                  <span>Update available: {release.latestVersion} ↗</span>
                </a>
              )}
            </div>

            <div className="flex items-center space-x-4">
              <a
                href="/docs/overview"
                className="text-slate-600 hover:text-slate-900 transition-colors"
              >
                Documentation
              </a>
              <a
                href="/docs/api-reference"
                className="text-slate-600 hover:text-slate-900 transition-colors"
              >
                API Reference
              </a>
              <a
                href="https://github.com/algotyrnt/triage"
                target="_blank"
                rel="noreferrer"
                className="text-slate-600 hover:text-slate-900 transition-colors"
              >
                GitHub ↗
              </a>
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}
