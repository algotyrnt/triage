/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useRef, useEffect, useMemo } from 'react';
import { ScreenId, DetectedModule, RepositoryItem } from '@/types';
import {
  GitBranch,
  CheckCircle2,
  Copy,
  Check,
  Search,
  ArrowRight,
  Key,
  Shield,
  Layers,
  Sparkles,
  FolderTree,
  Folder,
  ExternalLink,
  RefreshCw,
  AlertTriangle,
  Loader2,
  PlusCircle,
  User,
  Building2,
  Lock,
  Globe,
} from 'lucide-react';
import { engineClient } from '@/services/engineClient';
import { logger } from '@/services/logger';

interface OnboardingPageProps {
  onNavigate: (screen: ScreenId) => void;
  onProjectSetup?: (repo: string, apiKey: string, rootDir?: string) => void;
  currentUser?: { username: string; avatarUrl?: string } | null;
}

export const OnboardingPage: React.FC<OnboardingPageProps> = ({
  onNavigate,
  onProjectSetup,
  currentUser,
}) => {
  const [currentStep, setCurrentStep] = useState<1 | 2 | 3>(1);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRepo, setSelectedRepo] = useState('');
  const [customRepoInput, setCustomRepoInput] = useState('');
  const [rootDir, setRootDir] = useState('');
  const [detectedModules, setDetectedModules] = useState<DetectedModule[]>([]);
  const [loadingModules, setLoadingModules] = useState(false);
  const [generatedKey, setGeneratedKey] = useState('');
  const [copiedKey, setCopiedKey] = useState(false);
  const [selectedOwnerFilter, setSelectedOwnerFilter] = useState<string>('all');
  const [loadingRepos, setLoadingRepos] = useState(false);
  const [verifyingInstall, setVerifyingInstall] = useState(false);
  const [repos, setRepos] = useState<RepositoryItem[]>([]);
  const [installUrl, setInstallUrl] = useState<string>('');
  const repoRefs = useRef<(HTMLDivElement | null)[]>([]);

  const username = currentUser?.username || 'algotyrnt';

  const personalReposCount = useMemo(() => {
    return repos.filter((r) => r.owner && r.owner.toLowerCase() === username.toLowerCase()).length;
  }, [repos, username]);

  const orgCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const r of repos) {
      if (r.owner && r.owner.toLowerCase() !== username.toLowerCase()) {
        counts[r.owner] = (counts[r.owner] || 0) + 1;
      }
    }
    return counts;
  }, [repos, username]);

  const availableOrgs = useMemo(() => {
    return Object.keys(orgCounts).sort((a, b) => a.localeCompare(b));
  }, [orgCounts]);

  const filteredRepos = useMemo(() => {
    return repos.filter((r) => {
      // 1. Account / Org Filter (default: personal user repos)
      if (selectedOwnerFilter === 'personal') {
        if (!r.owner || r.owner.toLowerCase() !== username.toLowerCase()) {
          return false;
        }
      } else if (selectedOwnerFilter !== 'all') {
        if (!r.owner || r.owner.toLowerCase() !== selectedOwnerFilter.toLowerCase()) {
          return false;
        }
      }

      // 2. Search Query Filter
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        const matchName = r.name && r.name.toLowerCase().includes(q);
        const matchFull = `${r.owner}/${r.repo}`.toLowerCase().includes(q);
        return matchName || matchFull;
      }

      return true;
    });
  }, [repos, selectedOwnerFilter, searchQuery, username]);

  const loadRepos = async () => {
    setLoadingRepos(true);
    try {
      // 1. Fetch which repos have the GitHub App installed from the engine DB
      const installedRepos = await engineClient.getInstalledRepos();

      // 2. Map engine format to UI format
      const mergedRepos: RepositoryItem[] = installedRepos.map((r: any) => ({
        owner: r.owner,
        repo: r.repo,
        name: r.name,
        branch: r.branch || 'main',
        lang: r.lang || 'Unknown',
        visibility: r.visibility || 'Private',
        private: r.private || false,
        installed: true,
      }));

      // Sort alphabetically
      mergedRepos.sort((a, b) => {
        return (a.name || '').localeCompare(b.name || '');
      });

      setRepos(mergedRepos);

      if (!selectedRepo && mergedRepos.length > 0) {
        setSelectedRepo(mergedRepos[0].name || `${mergedRepos[0].owner}/${mergedRepos[0].repo}`);
      }
    } catch (e) {
      console.error('Failed to load repositories', e);
    } finally {
      setLoadingRepos(false);
    }
  };

  useEffect(() => {
    loadRepos();
    engineClient
      .getInstallUrl()
      .then((res) => {
        if (res && res.url) setInstallUrl(res.url);
      })
      .catch(() => {});
  }, [username]);

  // Sync selected repo when filtered list changes
  useEffect(() => {
    if (!customRepoInput && filteredRepos.length > 0) {
      const isSelectedPresent = filteredRepos.some(
        (r) => (r.name || `${r.owner}/${r.repo}`) === selectedRepo,
      );
      if (!isSelectedPresent) {
        setSelectedRepo(
          filteredRepos[0].name || `${filteredRepos[0].owner}/${filteredRepos[0].repo}`,
        );
      }
    }
  }, [filteredRepos, customRepoInput, selectedRepo]);

  // Auto-detect Go modules whenever active repo changes
  useEffect(() => {
    const activeTarget = customRepoInput.trim() || selectedRepo;
    if (!activeTarget) return;

    let owner = username;
    let repo = activeTarget;
    if (activeTarget.includes('/')) {
      const parts = activeTarget.split('/');
      owner = parts[0];
      repo = parts[1];
    }

    let cancelled = false;
    const timer = setTimeout(() => {
      setLoadingModules(true);
      engineClient
        .detectGoModules(owner, repo)
        .then((modules) => {
          if (cancelled) return;
          setDetectedModules(modules);
          // If there's a non-root module detected and current rootDir is empty, suggest first detected
          setRootDir((prev) => {
            if (!prev && modules.length > 1) {
              const nonRoot = modules.find((m) => !m.is_root);
              return nonRoot ? nonRoot.path : prev;
            }
            return prev;
          });
        })
        .catch((e) => {
          if (!cancelled) logger.warn('Failed to detect Go modules:', e);
        })
        .finally(() => {
          if (!cancelled) setLoadingModules(false);
        });
    }, 300);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [selectedRepo, customRepoInput, username]);

  const activeTarget = customRepoInput.trim() || selectedRepo;
  let activeOwner = username;
  let activeRepoName = activeTarget;
  if (activeTarget.includes('/')) {
    const parts = activeTarget.split('/');
    activeOwner = parts[0];
    activeRepoName = parts[1];
  }

  // Determine if active target repository is installed
  const matchingRepoItem = repos.find(
    (r) =>
      r.name.toLowerCase() === activeTarget.toLowerCase() ||
      `${r.owner}/${r.repo}`.toLowerCase() === activeTarget.toLowerCase(),
  );
  const isTargetInstalled = matchingRepoItem ? matchingRepoItem.installed : false;

  const handleVerifyInstallation = async () => {
    setVerifyingInstall(true);
    try {
      const res = await engineClient.checkRepoInstalled(activeOwner, activeRepoName);
      if (res && res.installed) {
        await loadRepos();
      } else {
        await loadRepos();
      }
    } catch (e) {
      logger.warn('Failed to verify install', e);
    } finally {
      setVerifyingInstall(false);
    }
  };

  const handleOpenInstallApp = () => {
    if (installUrl) {
      window.open(installUrl, '_blank', 'noopener,noreferrer');
    }
  };

  const [generatingKey, setGeneratingKey] = useState(false);

  const generateNewKey = () => {
    const hex = Math.random().toString(36).substring(2, 12);
    const key = `tr_live_${hex}`;
    setGeneratedKey(key);
    return key;
  };

  const handleProceedToStep3 = async () => {
    const repo = customRepoInput.trim() || selectedRepo;
    if (!repo) {
      setCurrentStep(3);
      return;
    }
    setGeneratingKey(true);
    try {
      const storageKey = `triage_key_${activeOwner}_${activeRepoName}_${rootDir}`;
      const res = await engineClient.createProject(repo, rootDir, username);
      if (res && res.api_key) {
        setGeneratedKey(res.api_key);
        localStorage.setItem(storageKey, res.api_key);
      } else {
        const existingStoredKey = localStorage.getItem(storageKey);
        if (existingStoredKey) {
          setGeneratedKey(existingStoredKey);
        } else {
          const fallback = generateNewKey();
          localStorage.setItem(storageKey, fallback);
        }
      }
    } catch (e) {
      logger.warn('Project registration warning:', e);
      const storageKey = `triage_key_${activeOwner}_${activeRepoName}_${rootDir}`;
      const existingStoredKey = localStorage.getItem(storageKey);
      if (existingStoredKey) {
        setGeneratedKey(existingStoredKey);
      } else if (!generatedKey) {
        const fallback = generateNewKey();
        localStorage.setItem(storageKey, fallback);
      }
    } finally {
      setGeneratingKey(false);
      setCurrentStep(3);
    }
  };

  const installedCount = filteredRepos.filter((r) => r.installed).length;
  const activeKey = generatedKey || 'tr_live_fetching_key';

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(activeKey);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    } catch (e) {
      logger.error('Failed to copy API key to clipboard', e);
    }
  };

  const handleCompleteSetup = () => {
    const key = generatedKey || activeKey;
    const repo = customRepoInput.trim() || selectedRepo;
    if (onProjectSetup) {
      onProjectSetup(repo, key, rootDir);
    } else {
      onNavigate('dashboard');
    }
  };

  const appSlugMatch = installUrl?.match(/\/apps\/([^/]+)\/installations/);
  const appSlug = appSlugMatch ? appSlugMatch[1] : '';
  const newOrgInstallUrl = appSlug
    ? `https://github.com/settings/apps/${appSlug}/installations`
    : installUrl || '';

  return (
    <div className="max-w-4xl mx-auto px-4 py-8 space-y-6">
      {/* Title */}
      <div className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
          Project Onboarding & Go AST Setup
        </h1>
        <p className="text-xs text-slate-600 font-sans mt-1">
          Connect your GitHub repository to index Go AST syntax trees and enable live panic crash
          symbolication.
        </p>
      </div>

      {/* 3-Step Indicator Bar */}
      <div className="grid grid-cols-3 gap-2 bg-white border border-slate-200 p-2 rounded-sm font-mono text-xs">
        {[
          { num: 1, title: 'Select Repo', desc: 'Choose target Go project' },
          {
            num: 2,
            title: 'GitHub App Setup',
            desc: 'Read-only tree permissions',
          },
          {
            num: 3,
            title: 'SDK Integration Key',
            desc: 'Generate telemetry token',
          },
        ].map((step) => {
          const isActive = currentStep === step.num;
          const isDone = currentStep > step.num;
          return (
            <button
              key={step.num}
              onClick={() => {
                if (step.num === 3 && !generatedKey) {
                  handleProceedToStep3();
                } else {
                  setCurrentStep(step.num as any);
                }
              }}
              className={`text-left p-2.5 rounded-sm transition-all border ${
                isActive
                  ? 'border-black bg-black text-white'
                  : isDone
                    ? 'border-emerald-200 bg-emerald-50/50 text-slate-900'
                    : 'border-slate-100 bg-slate-50 text-slate-500'
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span
                  className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm ${
                    isActive
                      ? 'bg-white text-black'
                      : isDone
                        ? 'bg-emerald-600 text-white'
                        : 'bg-slate-200 text-slate-700'
                  }`}
                >
                  STEP 0{step.num}
                </span>
                {isDone && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />}
              </div>
              <div className="font-bold text-xs">{step.title}</div>
              <div
                className={`text-[10px] truncate ${isActive ? 'text-slate-300' : 'text-slate-500'}`}
              >
                {step.desc}
              </div>
            </button>
          );
        })}
      </div>

      {/* Step Content */}
      <div className="bg-white border border-slate-200 rounded-sm p-6 space-y-6">
        {currentStep === 1 && (
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <h2 className="text-sm font-bold text-slate-900 font-sans">
                  Select a Repository from <span className="font-mono">@{username}</span>
                </h2>
                <p className="text-xs text-slate-500 font-sans">
                  Choose an installed repository or select any repository to install the GitHub App.
                </p>
              </div>

              {/* Action Buttons: Refresh & Install GitHub App */}
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => loadRepos()}
                  disabled={loadingRepos}
                  className="flex items-center gap-1.5 px-2.5 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-sm text-xs font-mono border border-slate-200 transition-colors"
                  title="Sync repositories from GitHub"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${loadingRepos ? 'animate-spin' : ''}`} />
                  <span className="hidden sm:inline">Refresh</span>
                </button>

                {installUrl && (
                  <button
                    type="button"
                    onClick={handleOpenInstallApp}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-black hover:bg-slate-800 text-white rounded-sm text-xs font-mono font-semibold transition-colors"
                    title="Install GitHub App on more repositories or organizations"
                  >
                    <PlusCircle className="w-3.5 h-3.5" />
                    <span>Install on More Repos</span>
                    <ExternalLink className="w-3 h-3 ml-0.5 opacity-70" />
                  </button>
                )}
              </div>
            </div>

            {/* Custom Repo Manual Input Box */}
            <div className="space-y-1.5 bg-slate-50 border border-slate-200 p-3.5 rounded-sm font-mono text-xs">
              <div className="flex items-center justify-between">
                <label className="font-bold text-slate-800 text-[11px] uppercase tracking-wider block">
                  Track Custom GitHub Repository:
                </label>
                {customRepoInput.trim() && (
                  <span
                    className={`text-[10px] px-1.5 py-0.5 rounded-sm font-bold border ${
                      isTargetInstalled
                        ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                        : 'bg-amber-50 text-amber-700 border-amber-200'
                    }`}
                  >
                    {isTargetInstalled ? 'App Installed' : 'App Setup Required in Step 2'}
                  </span>
                )}
              </div>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={customRepoInput}
                  onChange={(e) => {
                    setCustomRepoInput(e.target.value);
                    if (e.target.value) setSelectedRepo(e.target.value);
                  }}
                  placeholder={`e.g. ${username}/my-go-backend`}
                  className="flex-1 bg-white border border-slate-300 rounded-sm px-3 py-1.5 text-xs font-mono focus:outline-none focus:border-black"
                />
              </div>
            </div>

            {/* Account & Organization Filter Tabs */}
            <div className="space-y-2">
              <div className="flex items-center justify-between text-[11px] font-mono text-slate-500 px-0.5">
                <span className="font-bold uppercase tracking-wider text-slate-700">
                  Filter by Account / Organization:
                </span>
                <span>
                  {filteredRepos.length} shown ({installedCount} installed)
                </span>
              </div>
              <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs font-mono scrollbar-thin">
                {/* All Repos */}
                <button
                  type="button"
                  onClick={() => setSelectedOwnerFilter('all')}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-sm border transition-all shrink-0 cursor-pointer ${
                    selectedOwnerFilter === 'all'
                      ? 'bg-black text-white border-black font-bold shadow-xs'
                      : 'bg-slate-50 hover:bg-slate-100 text-slate-700 border-slate-200'
                  }`}
                >
                  <Layers className="w-3.5 h-3.5" />
                  <span>All ({repos.length})</span>
                </button>

                {/* Default: User's personal repos */}
                <button
                  type="button"
                  onClick={() => setSelectedOwnerFilter('personal')}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-sm border transition-all shrink-0 cursor-pointer ${
                    selectedOwnerFilter === 'personal'
                      ? 'bg-black text-white border-black font-bold shadow-xs'
                      : 'bg-slate-50 hover:bg-slate-100 text-slate-700 border-slate-200'
                  }`}
                >
                  <User className="w-3.5 h-3.5" />
                  <span>@{username} (Personal)</span>
                  <span
                    className={`text-[10px] px-1.5 py-0.2 rounded-sm ${
                      selectedOwnerFilter === 'personal'
                        ? 'bg-slate-800 text-white'
                        : 'bg-slate-200 text-slate-700'
                    }`}
                  >
                    {personalReposCount}
                  </span>
                </button>

                {/* Dynamic Orgs */}
                {availableOrgs.map((org) => (
                  <button
                    key={org}
                    type="button"
                    onClick={() => setSelectedOwnerFilter(org)}
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-sm border transition-all shrink-0 cursor-pointer ${
                      selectedOwnerFilter === org
                        ? 'bg-black text-white border-black font-bold shadow-xs'
                        : 'bg-slate-50 hover:bg-slate-100 text-slate-700 border-slate-200'
                    }`}
                  >
                    <Building2 className="w-3.5 h-3.5" />
                    <span>{org}</span>
                    <span
                      className={`text-[10px] px-1.5 py-0.2 rounded-sm ${
                        selectedOwnerFilter === org
                          ? 'bg-slate-800 text-white'
                          : 'bg-slate-200 text-slate-700'
                      }`}
                    >
                      {orgCounts[org]}
                    </span>
                  </button>
                ))}
              </div>

              {/* Search Box */}
              <div className="relative pt-1">
                <Search className="w-3.5 h-3.5 absolute left-3 top-3.5 text-slate-400" />
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder={
                    selectedOwnerFilter === 'personal'
                      ? `Search @${username}'s repositories...`
                      : selectedOwnerFilter === 'all'
                        ? 'Search all repositories...'
                        : `Search ${selectedOwnerFilter} repositories...`
                  }
                  className="w-full pl-9 pr-3 py-1.5 bg-slate-50 border border-slate-200 rounded-sm text-xs font-mono focus:bg-white focus:outline-none focus:border-black"
                />
              </div>
            </div>

            {/* Repo List */}
            {loadingRepos ? (
              <div className="py-8 text-center text-xs font-mono text-slate-500 flex items-center justify-center gap-2">
                <Loader2 className="w-4 h-4 animate-spin text-slate-700" />
                <span>Fetching live repositories for @{username} from GitHub API...</span>
              </div>
            ) : filteredRepos.length === 0 ? (
              <div className="py-8 text-center bg-slate-50 border border-dashed border-slate-200 rounded-sm p-6 space-y-3 font-mono text-xs text-slate-600">
                <p>No repositories matched your search.</p>
                {installUrl && (
                  <div className="flex items-center justify-center gap-3">
                    <button
                      type="button"
                      onClick={handleOpenInstallApp}
                      className="inline-flex items-center gap-1.5 bg-black text-white px-3 py-1.5 rounded-sm font-semibold hover:bg-slate-800"
                    >
                      <PlusCircle className="w-3.5 h-3.5" />
                      <span>Grant Access</span>
                    </button>
                    <a
                      href={newOrgInstallUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="text-xs text-slate-500 hover:text-black font-semibold hover:underline"
                    >
                      Install on New Org
                    </a>
                  </div>
                )}
              </div>
            ) : (
              <div
                role="radiogroup"
                aria-label="Select Repository"
                className="space-y-2 max-h-64 overflow-y-auto"
              >
                {filteredRepos.map((repo, index) => {
                  const repoName = repo.name || `${repo.owner}/${repo.repo}`;
                  const isSelected = selectedRepo === repoName && !customRepoInput;
                  const isSelectedInFiltered = filteredRepos.some(
                    (r) => (r.name || `${r.owner}/${r.repo}`) === selectedRepo,
                  );
                  const hasFocusEntry =
                    isSelected || (index === 0 && (!selectedRepo || !isSelectedInFiltered));
                  return (
                    <div
                      key={repoName}
                      ref={(el) => {
                        repoRefs.current[index] = el;
                      }}
                      role="radio"
                      aria-checked={isSelected}
                      tabIndex={hasFocusEntry ? 0 : -1}
                      onClick={() => {
                        setSelectedRepo(repoName);
                        setCustomRepoInput('');
                      }}
                      onKeyDown={(e) => {
                        if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
                          e.preventDefault();
                          const nextIdx = (index + 1) % filteredRepos.length;
                          const nextRepo =
                            filteredRepos[nextIdx].name ||
                            `${filteredRepos[nextIdx].owner}/${filteredRepos[nextIdx].repo}`;
                          setSelectedRepo(nextRepo);
                          setCustomRepoInput('');
                          repoRefs.current[nextIdx]?.focus();
                        } else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
                          e.preventDefault();
                          const prevIdx = (index - 1 + filteredRepos.length) % filteredRepos.length;
                          const prevRepo =
                            filteredRepos[prevIdx].name ||
                            `${filteredRepos[prevIdx].owner}/${filteredRepos[prevIdx].repo}`;
                          setSelectedRepo(prevRepo);
                          setCustomRepoInput('');
                          repoRefs.current[prevIdx]?.focus();
                        } else if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          setSelectedRepo(repoName);
                          setCustomRepoInput('');
                        }
                      }}
                      className={`p-3 rounded-sm border cursor-pointer transition-all flex items-center justify-between font-mono text-xs focus:outline-none focus:ring-1 focus:ring-black ${
                        isSelected
                          ? 'border-black bg-slate-50'
                          : 'border-slate-200 hover:border-slate-300 bg-white'
                      }`}
                    >
                      <div className="flex items-center gap-2.5">
                        <div
                          className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center ${
                            isSelected ? 'border-black bg-black' : 'border-slate-300'
                          }`}
                        >
                          {isSelected && <div className="w-1.5 h-1.5 bg-white rounded-full"></div>}
                        </div>
                        <div>
                          <span className="font-bold text-slate-900">{repoName}</span>
                          <span className="text-[11px] text-slate-500 ml-2">
                            ({repo.branch || 'main'})
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 text-[11px]">
                        {repo.private ? (
                          <span className="flex items-center gap-1 bg-slate-900 text-slate-200 px-2 py-0.5 rounded-sm border border-slate-700 font-semibold text-[10px]">
                            <Lock className="w-2.5 h-2.5 text-amber-400" />
                            <span>Private</span>
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 bg-slate-100 text-slate-600 px-2 py-0.5 rounded-sm border border-slate-200 text-[10px]">
                            <Globe className="w-2.5 h-2.5 text-slate-400" />
                            <span>Public</span>
                          </span>
                        )}
                        {repo.lang && (
                          <span className="bg-slate-100 text-slate-700 px-2 py-0.5 rounded-sm border border-slate-200">
                            {repo.lang}
                          </span>
                        )}
                        <span
                          className={`px-2 py-0.5 rounded-sm border font-semibold flex items-center gap-1 ${
                            repo.installed
                              ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                              : 'bg-amber-50 text-amber-700 border-amber-200'
                          }`}
                        >
                          {repo.installed ? (
                            <>
                              <CheckCircle2 className="w-3 h-3 text-emerald-600" />
                              <span>Installed</span>
                            </>
                          ) : (
                            <>
                              <AlertTriangle className="w-3 h-3 text-amber-600" />
                              <span>App Required</span>
                            </>
                          )}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {installUrl && filteredRepos.length > 0 && (
              <div className="mt-2 border border-slate-200 flex justify-between items-center bg-white p-2 px-3 rounded-sm shadow-sm">
                <span className="text-[11px] text-slate-500 font-mono">
                  Don't see your repository?
                </span>
                <div className="flex items-center gap-4">
                  <a
                    href={newOrgInstallUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[11px] text-slate-500 hover:text-black font-semibold hover:underline"
                  >
                    Install on New Org
                  </a>
                  <button
                    type="button"
                    onClick={handleOpenInstallApp}
                    className="inline-flex items-center gap-1.5 text-[11px] text-black font-semibold hover:underline"
                  >
                    <PlusCircle className="w-3 h-3" />
                    <span>Grant Access</span>
                  </button>
                </div>
              </div>
            )}

            {/* Selected Repo Status Tip */}
            {activeTarget && !isTargetInstalled && (
              <div className="bg-amber-50 border border-amber-200 p-3 rounded-sm text-xs font-mono text-amber-800 flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600 mt-0.5" />
                <div>
                  <span className="font-bold">
                    GitHub App is not yet installed on {activeTarget}.
                  </span>{' '}
                  You can authorize and install it in Step 2 with 1 click.
                </div>
              </div>
            )}

            {/* Monorepo Go Backend Subdirectory Selector */}
            <div className="bg-slate-50 border border-slate-200 p-3.5 rounded-sm space-y-2.5 font-mono text-xs">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 font-bold text-slate-900">
                  <FolderTree className="w-3.5 h-3.5 text-slate-700" />
                  <span>Go Backend Subdirectory (Monorepo Support)</span>
                </div>
                {loadingModules && (
                  <span className="text-[11px] text-slate-500 animate-pulse">
                    Scanning for go.mod...
                  </span>
                )}
              </div>
              <p className="text-[11px] text-slate-600 font-sans">
                If your Go backend lives in a subfolder (e.g.{' '}
                <code className="font-mono text-slate-800">backend</code>,{' '}
                <code className="font-mono text-slate-800">apps/api</code>, or{' '}
                <code className="font-mono text-slate-800">services/engine</code>), specify it below
                for precise AST symbolication.
              </p>

              {/* Detected Go Module Quick Selector */}
              {detectedModules.length > 0 && (
                <div className="space-y-1">
                  <label className="text-[10px] font-bold text-slate-500 uppercase tracking-wider block">
                    Detected Go Modules:
                  </label>
                  <div className="flex flex-wrap gap-1.5">
                    {detectedModules.map((mod) => {
                      const isSelected = rootDir === mod.path;
                      return (
                        <button
                          key={mod.path}
                          type="button"
                          onClick={() => setRootDir(mod.path)}
                          className={`px-2 py-1 rounded-sm border text-[11px] font-mono flex items-center gap-1 cursor-pointer transition-all ${
                            isSelected
                              ? 'bg-black text-white border-black font-bold'
                              : 'bg-white text-slate-700 border-slate-300 hover:border-slate-400'
                          }`}
                        >
                          <Folder className="w-3 h-3" />
                          <span>{mod.path ? `${mod.path}/` : 'Root (/)'}</span>
                          {mod.is_root && <span className="text-[9px] opacity-75">(Root)</span>}
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Manual Input for Monorepo Subdirectory */}
              <div className="flex items-center gap-2">
                <input
                  id="monorepo-root-dir-input"
                  aria-label="Go backend subdirectory"
                  type="text"
                  value={rootDir}
                  onChange={(e) => setRootDir(e.target.value)}
                  placeholder="e.g. backend or apps/api (leave empty for repository root)"
                  className="flex-1 px-3 py-1.5 bg-white border border-slate-300 rounded-sm font-mono text-xs focus:outline-none focus:border-black"
                />
                {rootDir && (
                  <button
                    type="button"
                    onClick={() => setRootDir('')}
                    className="text-[11px] font-mono text-slate-500 hover:text-black underline px-1"
                  >
                    Reset to Root
                  </button>
                )}
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setCurrentStep(2)}
                disabled={!activeTarget}
                className="bg-black hover:bg-slate-800 disabled:bg-slate-300 text-white font-mono text-xs font-semibold py-2 px-4 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                <span>Continue to Step 2 (GitHub App Setup)</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}

        {currentStep === 2 && (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-bold text-slate-900 font-sans">
                GitHub App Authorization & AST Webhook Ingress
              </h2>
              <p className="text-xs text-slate-500 font-sans mt-0.5">
                Targeting <span className="font-mono text-slate-900 font-bold">{activeTarget}</span>
                {rootDir && (
                  <span className="ml-2 font-mono text-slate-700 bg-slate-100 px-1.5 py-0.5 rounded-sm border border-slate-200 text-[11px]">
                    Subdirectory: {rootDir}/
                  </span>
                )}
              </p>
            </div>

            {/* Authorization Banner or Installation Prompt */}
            {isTargetInstalled ? (
              <div className="bg-emerald-50 border border-emerald-200 p-4 rounded-sm space-y-2">
                <div className="flex items-center gap-2 text-emerald-900 font-mono text-xs font-bold">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                  <span>GitHub App Authorized for repository: {activeTarget}</span>
                </div>
                <p className="text-xs font-mono text-emerald-800 leading-relaxed">
                  Triage GitHub App is active with read-only tree and issue permissions. Webhook
                  listener ready to receive AST triggers.
                </p>
              </div>
            ) : (
              <div className="bg-amber-50 border border-amber-200 p-5 rounded-sm space-y-3 font-mono">
                <div className="flex items-center gap-2 text-amber-900 text-xs font-bold">
                  <AlertTriangle className="w-4 h-4 text-amber-600" />
                  <span>GitHub App Installation Required for {activeTarget}</span>
                </div>
                <p className="text-xs text-amber-800 leading-relaxed font-sans">
                  The Triage GitHub App is not yet installed on this repository or organization.
                  Click below to grant repository permissions on GitHub, then click Verify.
                </p>
                <div className="flex flex-wrap items-center gap-3 pt-1">
                  <button
                    type="button"
                    onClick={handleOpenInstallApp}
                    className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2 px-4 rounded-sm flex items-center gap-2 transition-colors cursor-pointer"
                  >
                    <PlusCircle className="w-3.5 h-3.5" />
                    <span>Install GitHub App on GitHub</span>
                    <ExternalLink className="w-3 h-3 opacity-70" />
                  </button>

                  <button
                    type="button"
                    onClick={handleVerifyInstallation}
                    disabled={verifyingInstall}
                    className="bg-white hover:bg-slate-100 text-slate-800 font-mono text-xs font-semibold py-2 px-3.5 rounded-sm border border-slate-300 flex items-center gap-1.5 transition-colors cursor-pointer"
                  >
                    <RefreshCw
                      className={`w-3.5 h-3.5 ${verifyingInstall ? 'animate-spin' : ''}`}
                    />
                    <span>{verifyingInstall ? 'Verifying...' : 'Verify Installation'}</span>
                  </button>
                </div>
              </div>
            )}

            {/* Scope details */}
            <div className="bg-slate-50 border border-slate-200 p-3.5 rounded-sm space-y-1.5 font-mono text-xs">
              <div className="font-bold text-slate-800">Granted Scopes & Webhook Triggers:</div>
              <div className="text-slate-600 text-[11px] space-y-1">
                <div>
                  • <span className="font-semibold text-slate-900">push:</span> Automatically
                  re-indexes Go AST nodes on git push
                </div>
                <div>
                  • <span className="font-semibold text-slate-900">issues:write:</span>{' '}
                  Automatically links symbolicated panic crashes to GitHub Issues
                </div>
                <div>
                  • <span className="font-semibold text-slate-900">pull_requests:write:</span>{' '}
                  Enables Gemini automated patch generation comments
                </div>
              </div>
            </div>

            <div className="flex justify-between items-center pt-2">
              <button
                onClick={() => setCurrentStep(1)}
                className="bg-slate-100 text-slate-700 hover:bg-slate-200 font-mono text-xs py-2 px-3 rounded-sm border border-slate-200"
              >
                Back to Repositories
              </button>
              <button
                onClick={handleProceedToStep3}
                disabled={generatingKey}
                className="bg-black hover:bg-slate-800 disabled:bg-slate-700 text-white font-mono text-xs font-semibold py-2 px-4 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                {generatingKey ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    <span>Generating Ingestion Key...</span>
                  </>
                ) : (
                  <>
                    <span>Generate Ingestion API Key</span>
                    <ArrowRight className="w-3.5 h-3.5" />
                  </>
                )}
              </button>
            </div>
          </div>
        )}

        {currentStep === 3 && (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-bold text-slate-900 font-sans">
                SDK Telemetry Key Generated
              </h2>
              <p className="text-xs text-slate-500 font-sans mt-0.5">
                Use this API key in your Go application initialization script (`triage.Middleware`).
              </p>
            </div>

            {/* Key Box */}
            <div className="bg-slate-900 text-slate-100 p-4 rounded-sm font-mono space-y-2 border border-slate-800">
              <div className="flex items-center justify-between text-xs text-slate-400">
                <span className="flex items-center gap-1.5">
                  <Key className="w-3.5 h-3.5 text-emerald-400" />
                  <span>
                    Production Telemetry Key (Repo: {customRepoInput.trim() || selectedRepo}
                    {rootDir ? ` • ${rootDir}/` : ''})
                  </span>
                </span>
                <span className="text-[10px] text-emerald-400 font-bold">STATUS: ACTIVE</span>
              </div>

              <div className="flex items-center justify-between gap-2 bg-black p-2.5 rounded-sm border border-slate-800">
                <code className="text-xs text-emerald-400 font-bold tracking-wide select-all break-all">
                  {activeKey}
                </code>
                <button
                  onClick={handleCopy}
                  className="bg-slate-800 hover:bg-slate-700 text-white text-xs px-2.5 py-1 rounded-sm border border-slate-700 flex items-center gap-1 shrink-0 font-mono"
                >
                  {copiedKey ? (
                    <Check className="w-3 h-3 text-emerald-400" />
                  ) : (
                    <Copy className="w-3 h-3" />
                  )}
                  <span>{copiedKey ? 'Copied!' : 'Copy'}</span>
                </button>
              </div>
            </div>

            {/* Go Code snippet */}
            <div className="space-y-1.5 font-mono">
              <div className="text-xs font-bold text-slate-800">
                Go SDK QuickStart Initialization:
              </div>
              <pre className="bg-slate-900 text-slate-100 p-3 rounded-sm text-[11px] overflow-x-auto border border-slate-800 leading-relaxed">
                {`package main

import (
	"net/http"
	"os"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()

	// Wrap HTTP multiplexer with triage panic recovery middleware
	telemetryURL := os.Getenv("TRIAGE_ENGINE_URL")
	handler := triage.Middleware("${activeKey}", telemetryURL)(mux)

	http.ListenAndServe(":8081", handler)
}`}
              </pre>
            </div>

            <div className="flex justify-between items-center pt-2">
              <button
                onClick={() => setCurrentStep(2)}
                className="bg-slate-100 text-slate-700 hover:bg-slate-200 font-mono text-xs py-2 px-3 rounded-sm border border-slate-200"
              >
                Back
              </button>
              <button
                onClick={handleCompleteSetup}
                className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2.5 px-5 rounded-sm transition-colors flex items-center gap-2 cursor-pointer"
              >
                <span>Complete Setup & Open Dashboard</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
