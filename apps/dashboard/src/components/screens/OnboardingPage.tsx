/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useMemo } from 'react';
import { ScreenId, DetectedModule, RepositoryItem } from '@/types';
import {
  CheckCircle2,
  Copy,
  Check,
  Search,
  ArrowRight,
  Key,
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
  Eye,
  EyeOff,
  BookOpen,
} from 'lucide-react';
import { engineClient } from '@/services/engineClient';
import { logger } from '@/services/logger';

interface OnboardingPageProps {
  onNavigate: (screen: ScreenId) => void;
  onProjectSetup?: (
    repo: string,
    apiKey: string,
    rootDir?: string,
    projectContext?: string,
  ) => void;
  currentUser?: { username: string; avatarUrl?: string } | null;
}

export const OnboardingPage: React.FC<OnboardingPageProps> = ({
  onNavigate,
  onProjectSetup,
  currentUser,
}) => {
  const [currentStep, setCurrentStep] = useState<1 | 2>(1);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRepo, setSelectedRepo] = useState('');
  const [customRepoInput, setCustomRepoInput] = useState('');
  const [useManualRepo, setUseManualRepo] = useState(false);
  const [rootDir, setRootDir] = useState('');
  const [projectContext, setProjectContext] = useState('');
  const [detectedModules, setDetectedModules] = useState<DetectedModule[]>([]);
  const [loadingModules, setLoadingModules] = useState(false);
  const [generatedKey, setGeneratedKey] = useState('');
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedInstall, setCopiedInstall] = useState(false);
  const [selectedOwnerFilter, setSelectedOwnerFilter] = useState<string>('all');
  const [loadingRepos, setLoadingRepos] = useState(false);
  const [repos, setRepos] = useState<RepositoryItem[]>([]);
  const [installUrl, setInstallUrl] = useState<string>('');
  const [generatingKey, setGeneratingKey] = useState(false);
  const [onboardingError, setOnboardingError] = useState<string | null>(null);
  const [showKey, setShowKey] = useState(true);

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
      if (selectedOwnerFilter === 'personal') {
        if (!r.owner || r.owner.toLowerCase() !== username.toLowerCase()) return false;
      } else if (selectedOwnerFilter !== 'all') {
        if (!r.owner || r.owner.toLowerCase() !== selectedOwnerFilter.toLowerCase()) return false;
      }

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
      const installedRepos = await engineClient.getInstalledRepos();
      const mergedRepos: RepositoryItem[] = installedRepos.map((r: any) => ({
        owner: r.owner,
        repo: r.repo,
        name: r.name,
        branch: r.branch || 'main',
        lang: r.lang || 'Go',
        visibility: r.visibility || 'Private',
        private: r.private || false,
        installed: true,
      }));

      mergedRepos.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
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
    if (!useManualRepo && filteredRepos.length > 0) {
      const isSelectedPresent = filteredRepos.some(
        (r) => (r.name || `${r.owner}/${r.repo}`) === selectedRepo,
      );
      if (!isSelectedPresent && !selectedRepo) {
        setSelectedRepo(
          filteredRepos[0].name || `${filteredRepos[0].owner}/${filteredRepos[0].repo}`,
        );
      }
    }
  }, [filteredRepos, useManualRepo, selectedRepo]);

  // Auto-detect Go modules whenever active repo changes
  useEffect(() => {
    const activeTarget = useManualRepo ? customRepoInput.trim() : selectedRepo;
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
  }, [selectedRepo, customRepoInput, useManualRepo, username]);

  const activeTarget = useManualRepo ? customRepoInput.trim() : selectedRepo;

  // Determine if active target repository is installed
  const matchingRepoItem = repos.find(
    (r) =>
      r.name.toLowerCase() === activeTarget.toLowerCase() ||
      `${r.owner}/${r.repo}`.toLowerCase() === activeTarget.toLowerCase(),
  );
  const isTargetInstalled = matchingRepoItem ? matchingRepoItem.installed : false;

  const handleOpenInstallApp = () => {
    if (installUrl) {
      window.open(installUrl, '_blank', 'noopener,noreferrer');
    }
  };

  const handleCreateProject = async () => {
    const repo = useManualRepo ? customRepoInput.trim() : selectedRepo;
    if (!repo) return;

    setGeneratingKey(true);
    setOnboardingError(null);
    try {
      const res = await engineClient.createProject(repo, rootDir, username, projectContext);
      if (res && res.api_key) {
        setGeneratedKey(res.api_key);
      } else if (res && res.key_masked) {
        setGeneratedKey(res.key_masked);
      }
      setCurrentStep(2);
    } catch (e: any) {
      logger.warn('Project registration warning:', e);
      setOnboardingError(e?.message || 'Failed to connect to Triage engine backend.');
    } finally {
      setGeneratingKey(false);
    }
  };

  const activeKey = generatedKey || '••••••••••••••••••••••••••••••••';

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(generatedKey || activeKey);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    } catch (e) {
      logger.error('Failed to copy API key to clipboard', e);
    }
  };

  const handleCompleteSetup = () => {
    const key = generatedKey || activeKey;
    const repo = useManualRepo ? customRepoInput.trim() : selectedRepo;
    if (onProjectSetup) {
      onProjectSetup(repo, key, rootDir, projectContext);
    } else {
      onNavigate('dashboard');
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8 space-y-6">
      {/* Header */}
      <div className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
          Setup New Go Project
        </h1>
        <p className="text-xs text-slate-600 font-sans mt-1">
          Connect your GitHub repository to index Go AST syntax trees and receive real-time panic
          diagnostics.
        </p>
      </div>

      {/* 2-Step Navigation Indicator */}
      <div className="grid grid-cols-2 gap-3">
        {[
          { num: 1, title: 'Configure Repository', desc: 'Select Go project & path' },
          { num: 2, title: 'API Key & SDK Setup', desc: 'Telemetry middleware token' },
        ].map((step) => {
          const isActive = currentStep === step.num;
          const isDone = currentStep > step.num;
          return (
            <button
              key={step.num}
              type="button"
              onClick={() => {
                if (step.num === 2 && !generatedKey) {
                  handleCreateProject();
                } else if (step.num <= currentStep || isDone) {
                  setCurrentStep(step.num as any);
                }
              }}
              className={`text-left p-3 rounded-sm transition-all border font-mono ${
                isActive
                  ? 'border-black bg-black text-white shadow-xs'
                  : isDone
                    ? 'border-emerald-200 bg-emerald-50/50 text-slate-900 hover:border-emerald-300'
                    : 'border-slate-200 bg-white text-slate-400 cursor-not-allowed opacity-75'
              }`}
            >
              <div className="flex items-center justify-between mb-1.5">
                <span
                  className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm ${
                    isActive
                      ? 'bg-white text-black'
                      : isDone
                        ? 'bg-emerald-600 text-white'
                        : 'bg-slate-100 text-slate-500'
                  }`}
                >
                  STEP 0{step.num}
                </span>
                {isDone && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />}
              </div>
              <div className="font-bold text-xs">{step.title}</div>
              <div
                className={`text-[10px] truncate mt-0.5 ${
                  isActive ? 'text-slate-300' : 'text-slate-500'
                }`}
              >
                {step.desc}
              </div>
            </button>
          );
        })}
      </div>

      {/* Step 1: Repository & Scope Configuration */}
      {currentStep === 1 && (
        <div className="bg-white border border-slate-200 rounded-sm p-6 space-y-6">
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <h2 className="text-sm font-bold text-slate-900 font-sans">
                  Choose a Repository from <span className="font-mono">@{username}</span>
                </h2>
                <p className="text-xs text-slate-500 font-sans mt-0.5">
                  Select a repository with the Triage GitHub App installed.
                </p>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => loadRepos()}
                  disabled={loadingRepos}
                  className="flex items-center gap-1 px-2.5 py-1 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-sm text-xs font-mono border border-slate-200 transition-colors cursor-pointer"
                  title="Refresh repository list"
                >
                  <RefreshCw className={`w-3 h-3 ${loadingRepos ? 'animate-spin' : ''}`} />
                  <span>Refresh</span>
                </button>

                {installUrl && (
                  <button
                    type="button"
                    onClick={handleOpenInstallApp}
                    className="flex items-center gap-1.5 px-3 py-1 bg-black hover:bg-slate-800 text-white rounded-sm text-xs font-mono font-semibold transition-colors cursor-pointer"
                    title="Install GitHub App on more repositories"
                  >
                    <PlusCircle className="w-3 h-3" />
                    <span>Install on More Repos</span>
                    <ExternalLink className="w-2.5 h-2.5 opacity-70" />
                  </button>
                )}
              </div>
            </div>

            {!useManualRepo ? (
              <div className="space-y-3">
                {/* Account / Org Filter Pills */}
                <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs font-mono">
                  <button
                    type="button"
                    onClick={() => setSelectedOwnerFilter('all')}
                    className={`flex items-center gap-1 px-2.5 py-1 rounded-sm border transition-all shrink-0 cursor-pointer ${
                      selectedOwnerFilter === 'all'
                        ? 'bg-black text-white border-black font-bold'
                        : 'bg-slate-50 hover:bg-slate-100 text-slate-700 border-slate-200'
                    }`}
                  >
                    <Layers className="w-3 h-3" />
                    <span>All ({repos.length})</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => setSelectedOwnerFilter('personal')}
                    className={`flex items-center gap-1 px-2.5 py-1 rounded-sm border transition-all shrink-0 cursor-pointer ${
                      selectedOwnerFilter === 'personal'
                        ? 'bg-black text-white border-black font-bold'
                        : 'bg-slate-50 hover:bg-slate-100 text-slate-700 border-slate-200'
                    }`}
                  >
                    <User className="w-3 h-3" />
                    <span>@{username}</span>
                    <span className="text-[10px] opacity-75">({personalReposCount})</span>
                  </button>

                  {availableOrgs.map((org) => (
                    <button
                      key={org}
                      type="button"
                      onClick={() => setSelectedOwnerFilter(org)}
                      className={`flex items-center gap-1 px-2.5 py-1 rounded-sm border transition-all shrink-0 cursor-pointer ${
                        selectedOwnerFilter === org
                          ? 'bg-black text-white border-black font-bold'
                          : 'bg-slate-50 hover:bg-slate-100 text-slate-700 border-slate-200'
                      }`}
                    >
                      <Building2 className="w-3 h-3" />
                      <span>{org}</span>
                      <span className="text-[10px] opacity-75">({orgCounts[org]})</span>
                    </button>
                  ))}
                </div>

                {/* Search Bar */}
                <div className="relative">
                  <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-slate-400" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search repositories..."
                    className="w-full pl-9 pr-3 py-1.5 bg-slate-50 border border-slate-200 rounded-sm text-xs font-mono focus:bg-white focus:outline-none focus:border-black"
                  />
                </div>

                {/* Repository Selection List */}
                {loadingRepos ? (
                  <div className="py-8 text-center text-xs font-mono text-slate-500 flex items-center justify-center gap-2">
                    <Loader2 className="w-4 h-4 animate-spin text-slate-700" />
                    <span>Loading repositories...</span>
                  </div>
                ) : filteredRepos.length === 0 ? (
                  <div className="py-6 text-center bg-slate-50 border border-dashed border-slate-200 rounded-sm p-4 space-y-2 font-mono text-xs text-slate-600">
                    <p>No installed repositories found matching your filter.</p>
                    <button
                      type="button"
                      onClick={() => setUseManualRepo(true)}
                      className="text-xs text-black font-bold underline cursor-pointer"
                    >
                      Enter repository manually instead
                    </button>
                  </div>
                ) : (
                  <div className="space-y-1.5 max-h-56 overflow-y-auto pr-1">
                    {filteredRepos.map((repo) => {
                      const repoName = repo.name || `${repo.owner}/${repo.repo}`;
                      const isSelected = selectedRepo === repoName;

                      return (
                        <div
                          key={repoName}
                          onClick={() => setSelectedRepo(repoName)}
                          className={`p-2.5 rounded-sm border cursor-pointer transition-all flex items-center justify-between font-mono text-xs ${
                            isSelected
                              ? 'border-black bg-slate-900 text-white shadow-xs'
                              : 'border-slate-200 hover:border-slate-300 bg-white text-slate-900'
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <div
                              className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center ${
                                isSelected
                                  ? 'border-emerald-400 bg-emerald-500'
                                  : 'border-slate-300'
                              }`}
                            >
                              {isSelected && <Check className="w-2.5 h-2.5 text-black stroke-3" />}
                            </div>
                            <div>
                              <span className="font-bold">{repoName}</span>
                              <span
                                className={`text-[11px] ml-2 ${
                                  isSelected ? 'text-slate-400' : 'text-slate-500'
                                }`}
                              >
                                ({repo.branch || 'main'})
                              </span>
                            </div>
                          </div>

                          <div className="flex items-center gap-2 text-[11px]">
                            {repo.private ? (
                              <span
                                className={`flex items-center gap-1 px-1.5 py-0.5 rounded-sm text-[10px] font-semibold ${
                                  isSelected
                                    ? 'bg-slate-800 text-slate-300'
                                    : 'bg-slate-100 text-slate-600'
                                }`}
                              >
                                <Lock className="w-2.5 h-2.5 text-amber-400" />
                                <span>Private</span>
                              </span>
                            ) : (
                              <span
                                className={`flex items-center gap-1 px-1.5 py-0.5 rounded-sm text-[10px] ${
                                  isSelected
                                    ? 'bg-slate-800 text-slate-300'
                                    : 'bg-slate-100 text-slate-600'
                                }`}
                              >
                                <Globe className="w-2.5 h-2.5 text-slate-400" />
                                <span>Public</span>
                              </span>
                            )}
                            <span
                              className={`px-2 py-0.5 rounded-sm text-[10px] font-semibold ${
                                isSelected
                                  ? 'bg-emerald-950 text-emerald-300 border border-emerald-800'
                                  : 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                              }`}
                            >
                              Connected
                            </span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                <div className="flex items-center justify-between text-[11px] font-mono text-slate-500 pt-1">
                  <span>
                    Selected: <strong className="text-slate-900">{selectedRepo || 'None'}</strong>
                  </span>
                  <button
                    type="button"
                    onClick={() => setUseManualRepo(true)}
                    className="text-slate-600 hover:text-black underline cursor-pointer"
                  >
                    Enter custom repository path instead
                  </button>
                </div>
              </div>
            ) : (
              /* Custom / Manual Repository Input */
              <div className="bg-slate-50 border border-slate-200 p-4 rounded-sm space-y-3 font-mono text-xs">
                <div className="flex items-center justify-between">
                  <label className="font-bold text-slate-800 text-[11px] uppercase tracking-wider block">
                    Custom GitHub Repository:
                  </label>
                  <button
                    type="button"
                    onClick={() => setUseManualRepo(false)}
                    className="text-[11px] text-slate-500 hover:text-black underline cursor-pointer"
                  >
                    ← Select from installed list
                  </button>
                </div>
                <input
                  type="text"
                  value={customRepoInput}
                  onChange={(e) => setCustomRepoInput(e.target.value)}
                  placeholder={`e.g. ${username}/my-microservice`}
                  className="w-full bg-white border border-slate-300 rounded-sm px-3 py-2 text-xs font-mono focus:outline-none focus:border-black"
                />
              </div>
            )}

            {/* Uninstalled Repository Notice */}
            {activeTarget && !isTargetInstalled && installUrl && (
              <div className="bg-amber-50 border border-amber-200 p-3 rounded-sm text-xs font-mono text-amber-900 flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0" />
                  <span>
                    GitHub App access required for <strong>{activeTarget}</strong>
                  </span>
                </div>
                <button
                  type="button"
                  onClick={handleOpenInstallApp}
                  className="text-xs bg-amber-600 hover:bg-amber-700 text-white px-2.5 py-1 rounded-sm font-semibold flex items-center gap-1 shrink-0 cursor-pointer"
                >
                  <span>Grant Access</span>
                  <ExternalLink className="w-3 h-3" />
                </button>
              </div>
            )}

            {/* Service Scope & Monorepo Configuration Card */}
            <div className="bg-slate-50 border border-slate-200 p-4 rounded-sm space-y-3 font-mono text-xs">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 font-bold text-slate-900">
                  <FolderTree className="w-3.5 h-3.5 text-slate-700" />
                  <span>Service Directory (Monorepo Subfolder)</span>
                </div>
                {loadingModules && (
                  <span className="text-[11px] text-slate-500 animate-pulse">
                    Scanning for go.mod...
                  </span>
                )}
              </div>

              {detectedModules.length > 0 && (
                <div className="flex flex-wrap gap-1.5 items-center">
                  <span className="text-[10px] text-slate-500 uppercase tracking-wider font-bold">
                    Detected Modules:
                  </span>
                  {detectedModules.map((mod) => {
                    const isSelected = rootDir === mod.path;
                    return (
                      <button
                        key={mod.path}
                        type="button"
                        onClick={() => setRootDir(mod.path)}
                        className={`px-2 py-0.5 rounded-sm border text-[11px] font-mono flex items-center gap-1 cursor-pointer transition-all ${
                          isSelected
                            ? 'bg-black text-white border-black font-bold'
                            : 'bg-white text-slate-700 border-slate-300 hover:border-slate-400'
                        }`}
                      >
                        <Folder className="w-2.5 h-2.5" />
                        <span>{mod.path ? `${mod.path}/` : '/ (Root)'}</span>
                      </button>
                    );
                  })}
                </div>
              )}

              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={rootDir}
                  onChange={(e) => setRootDir(e.target.value)}
                  placeholder="Leave empty for root (/), or specify subfolder (e.g. backend, apps/api)"
                  className="flex-1 px-3 py-1.5 bg-white border border-slate-300 rounded-sm font-mono text-xs focus:outline-none focus:border-black"
                />
                {rootDir && (
                  <button
                    type="button"
                    onClick={() => setRootDir('')}
                    className="text-[11px] text-slate-500 hover:text-black underline px-1 cursor-pointer"
                  >
                    Reset to Root
                  </button>
                )}
              </div>

              {/* Domain Context Field */}
              <div className="pt-2 border-t border-slate-200/80 space-y-1.5">
                <div className="flex items-center justify-between">
                  <label className="flex items-center gap-1.5 font-bold text-slate-800 text-[11px]">
                    <Sparkles className="w-3 h-3 text-indigo-600" />
                    <span>Domain Context & Architectural Notes (Optional)</span>
                  </label>
                  <span className="text-[10px] text-slate-400 uppercase tracking-wider font-semibold">
                    AI Diagnostics Context
                  </span>
                </div>
                <input
                  type="text"
                  value={projectContext}
                  onChange={(e) => setProjectContext(e.target.value)}
                  placeholder="e.g. High-throughput payment gateway processing Stripe webhooks with database rollbacks."
                  className="w-full px-3 py-1.5 bg-white border border-slate-300 rounded-sm font-sans text-xs text-slate-800 placeholder:text-slate-400 focus:outline-none focus:border-black"
                />
              </div>
            </div>

            {onboardingError && (
              <div className="bg-red-50 border border-red-200 text-red-700 text-xs font-mono p-3 rounded-sm flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0 text-red-600" />
                <span>{onboardingError}</span>
              </div>
            )}

            {/* Bottom Actions */}
            <div className="flex justify-between items-center pt-2">
              <button
                type="button"
                onClick={() => onNavigate('projects')}
                className="text-xs font-mono text-slate-600 hover:text-black underline cursor-pointer"
              >
                Cancel & Back to Projects
              </button>
              <button
                type="button"
                onClick={handleCreateProject}
                disabled={!activeTarget || generatingKey}
                className="bg-black hover:bg-slate-800 disabled:bg-slate-300 text-white font-mono text-xs font-semibold py-2.5 px-5 rounded-sm transition-colors flex items-center gap-2 cursor-pointer shadow-xs"
              >
                {generatingKey ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    <span>Creating Project & Key...</span>
                  </>
                ) : (
                  <>
                    <span>Create Project & Get API Key</span>
                    <ArrowRight className="w-3.5 h-3.5" />
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Step 2: API Key & SDK Instructions */}
      {currentStep === 2 && (
        <div className="bg-white border border-slate-200 rounded-sm p-6 space-y-6">
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-bold text-slate-900 font-sans">
                Project Ingestion Key Ready
              </h2>
              <p className="text-xs text-slate-500 font-sans mt-0.5">
                Pass this API key to your Go telemetry middleware to begin symbolication.
              </p>
            </div>

            {/* Key Box */}
            <div className="bg-slate-900 text-slate-100 p-4 rounded-sm font-mono space-y-2 border border-slate-800 shadow-sm">
              <div className="flex items-center justify-between text-xs text-slate-400">
                <span className="flex items-center gap-1.5">
                  <Key className="w-3.5 h-3.5 text-emerald-400" />
                  <span>
                    Project API Key: {activeTarget}
                    {rootDir ? ` (${rootDir}/)` : ''}
                  </span>
                </span>
                <span className="text-[10px] text-emerald-400 font-bold px-1.5 py-0.5 bg-emerald-950/80 border border-emerald-800 rounded-sm">
                  ACTIVE
                </span>
              </div>

              <div className="flex items-center justify-between gap-2 bg-black p-2.5 rounded-sm border border-slate-800">
                <code className="text-xs text-emerald-400 font-bold tracking-wide select-all break-all">
                  {showKey
                    ? generatedKey || activeKey
                    : (generatedKey || activeKey).replace(/./g, '•')}
                </code>
                <div className="flex items-center gap-1.5 shrink-0">
                  <button
                    type="button"
                    onClick={() => setShowKey(!showKey)}
                    className="bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white text-xs px-2 py-1 rounded-sm border border-slate-700 flex items-center gap-1 font-mono transition-colors cursor-pointer"
                    title={showKey ? 'Hide API key' : 'Reveal API key'}
                    aria-label={showKey ? 'Hide API key' : 'Reveal API key'}
                  >
                    {showKey ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                    <span>{showKey ? 'Hide' : 'Reveal'}</span>
                  </button>
                  <button
                    type="button"
                    onClick={handleCopy}
                    className="bg-slate-800 hover:bg-slate-700 text-white text-xs px-2.5 py-1 rounded-sm border border-slate-700 flex items-center gap-1 shrink-0 font-mono cursor-pointer transition-colors"
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
            </div>

            {/* Quick Install SDK Block */}
            <div className="bg-slate-50 border border-slate-200 rounded-sm p-4 space-y-3 font-mono text-xs">
              <div className="flex items-center justify-between">
                <span className="font-bold text-slate-900">Install Go Telemetry SDK:</span>
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText('go get github.com/algotyrnt/triage/sdk/go');
                    setCopiedInstall(true);
                    setTimeout(() => setCopiedInstall(false), 2000);
                  }}
                  className="text-[11px] text-slate-600 hover:text-black underline flex items-center gap-1 cursor-pointer"
                >
                  {copiedInstall ? (
                    <Check className="w-3 h-3 text-emerald-600" />
                  ) : (
                    <Copy className="w-3 h-3 text-slate-400" />
                  )}
                  <span>{copiedInstall ? 'Copied' : 'Copy'}</span>
                </button>
              </div>

              <div className="bg-slate-900 text-slate-100 p-2.5 rounded-sm select-all">
                <code>go get github.com/algotyrnt/triage/sdk/go</code>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pt-1 text-[11px] text-slate-600 font-sans border-t border-slate-200/80">
                <span>Middleware examples for net/http, Chi, Gin, Echo & Fiber.</span>
                <a
                  href="/docs/sdk"
                  target="_blank"
                  rel="noreferrer"
                  className="font-mono text-black font-semibold underline flex items-center gap-1 hover:text-indigo-600"
                >
                  <BookOpen className="w-3 h-3" />
                  <span>View Documentation</span>
                  <ExternalLink className="w-2.5 h-2.5" />
                </a>
              </div>
            </div>

            {/* Bottom Actions */}
            <div className="flex justify-between items-center pt-3 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setCurrentStep(1)}
                className="bg-slate-100 text-slate-700 hover:bg-slate-200 font-mono text-xs py-2 px-3.5 rounded-sm border border-slate-200 cursor-pointer"
              >
                Back to Repository Selection
              </button>
              <button
                type="button"
                onClick={handleCompleteSetup}
                className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2.5 px-6 rounded-sm transition-colors flex items-center gap-2 cursor-pointer shadow-xs"
              >
                <span>Complete Setup & Open Dashboard</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
