/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useMemo } from 'react';
import { ScreenId, RepositoryItem } from '@/types';
import { GithubIcon as Github } from '@/components/GithubIcon';
import { engineClient } from '@/services/engineClient';
import { logger } from '@/services/logger';
import {
  Settings,
  CheckCircle2,
  ArrowRight,
  Shield,
  AlertTriangle,
  ExternalLink,
  RefreshCw,
  Loader2,
  Server,
  Key,
  Brain,
  PlusCircle,
  Eye,
  EyeOff,
  Building2,
  User,
  Layers,
  Lock,
  Globe,
  Sparkles,
  Zap,
  Cpu,
  Check,
  XCircle,
} from 'lucide-react';

interface SetupWizardPageProps {
  onNavigate: (screen: ScreenId) => void;
}

export const SetupWizardPage: React.FC<SetupWizardPageProps> = ({ onNavigate }) => {
  const [currentStep, setCurrentStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Step 1: App Manifest
  const [appCreated, setAppCreated] = useState(false);
  const [appSlug, setAppSlug] = useState('');

  // Step 2: App Install
  const [appInstalled, setAppInstalled] = useState(false);
  const [repos, setRepos] = useState<RepositoryItem[]>([]);
  const [selectedOwnerFilter, setSelectedOwnerFilter] = useState<string>('all');

  const orgCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const r of repos) {
      if (r.owner) {
        counts[r.owner] = (counts[r.owner] || 0) + 1;
      }
    }
    return counts;
  }, [repos]);

  const availableOrgs = useMemo(() => {
    return Object.keys(orgCounts).sort((a, b) => a.localeCompare(b));
  }, [orgCounts]);

  const filteredRepos = useMemo(() => {
    if (selectedOwnerFilter === 'all') return repos;
    return repos.filter(
      (r) => r.owner && r.owner.toLowerCase() === selectedOwnerFilter.toLowerCase(),
    );
  }, [repos, selectedOwnerFilter]);

  // Step 3: OAuth
  const [oauthConfigured, setOauthConfigured] = useState(false);
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');

  // Step 4: AI Configuration
  const [llmConfigured, setLlmConfigured] = useState(false);
  const [llmProvider, setLlmProvider] = useState<
    'gemini' | 'openai' | 'anthropic' | 'ollama' | 'custom'
  >('gemini');
  const [llmApiKey, setLlmApiKey] = useState('');
  const [llmModel, setLlmModel] = useState('');
  const [llmBaseUrl, setLlmBaseUrl] = useState('');
  const [showLlmKey, setShowLlmKey] = useState(false);
  const [testingLlm, setTestingLlm] = useState(false);
  const [llmTestResult, setLlmTestResult] = useState<{
    success: boolean;
    latency_ms?: number;
    error?: string;
  } | null>(null);

  // Step 5: Test
  const [connectionSuccess, setConnectionSuccess] = useState(false);
  const [appName, setAppName] = useState('');

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const step = params.get('setup_step');
    const isAppCreated = params.get('app_created');
    const isAppInstalled = params.get('app_installed');

    if (step) {
      setCurrentStep(parseInt(step, 10));
    }

    if (isAppCreated === 'true') {
      setAppCreated(true);
      if (!step) setCurrentStep(2);
    }

    if (isAppInstalled === 'true') {
      setAppInstalled(true);
      fetchRepos();
      if (!step) setCurrentStep(3);
    }

    checkStatus();
  }, []);

  const checkStatus = async () => {
    try {
      const status = await engineClient.getSetupStatus();
      if (status.github_app) setAppCreated(true);
      if ((status as any).app_slug) setAppSlug((status as any).app_slug);
      if (status.installation) {
        setAppInstalled(true);
        fetchRepos();
      }
      if (status.oauth) setOauthConfigured(true);
      if (status.llm) {
        setLlmConfigured(true);
        engineClient.getLlmConfig().then((cfg) => {
          if (cfg.provider) setLlmProvider(cfg.provider as any);
          if (cfg.api_key) setLlmApiKey(cfg.api_key);
          if (cfg.model) setLlmModel(cfg.model);
          if (cfg.base_url) setLlmBaseUrl(cfg.base_url);
        });
      }

      if (status.github_app && status.installation && status.oauth && status.llm) {
        setCurrentStep(5);
      } else if (status.github_app && status.installation && status.oauth) {
        setCurrentStep(4);
      } else if (status.github_app && status.installation) {
        setCurrentStep(3);
      } else if (status.github_app) {
        setCurrentStep(2);
      }
    } catch (err) {
      logger.error('Failed to check setup status', err);
    }
  };

  const fetchRepos = async () => {
    try {
      const installedRepos = await engineClient.getInstalledRepos();
      setRepos(installedRepos);
    } catch (err) {
      logger.error('Failed to fetch installed repositories', err);
    }
  };

  const handleCreateApp = async () => {
    setLoading(true);
    setError(null);
    try {
      const { manifest } = await engineClient.getSetupManifest(window.location.origin);

      // Create hidden form to POST manifest to GitHub
      const form = document.createElement('form');
      form.method = 'POST';
      form.action = 'https://github.com/settings/apps/new';

      const input = document.createElement('input');
      input.type = 'hidden';
      input.name = 'manifest';
      input.value = JSON.stringify(manifest);

      form.appendChild(input);
      document.body.appendChild(form);
      form.submit();
    } catch (err: any) {
      setError(err.message || 'Failed to generate manifest');
      setLoading(false);
    }
  };

  const handleInstallApp = async (customSlug?: string) => {
    setLoading(true);
    setError(null);
    try {
      const slugToUse = (typeof customSlug === 'string' && customSlug.trim()) || appSlug.trim();
      if (slugToUse) {
        window.location.href = `https://github.com/apps/${slugToUse}/installations/new`;
        return;
      }
      const { url } = await engineClient.getInstallUrl();
      if (url) {
        window.location.href = url;
      } else {
        throw new Error('Install URL was empty. Please provide the GitHub App slug below.');
      }
    } catch (err: any) {
      logger.error('Failed to get install URL', err);
      setError(
        err.message || 'Failed to get install URL. Make sure the GitHub App was created in Step 1.',
      );
      setLoading(false);
    }
  };

  const handleSaveOAuth = async () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      setError('Client ID and Secret are required');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await engineClient.saveOAuthConfig(clientId, clientSecret);
      setOauthConfigured(true);
      setCurrentStep(4);
    } catch (err: any) {
      setError(err.message || 'Failed to save OAuth config');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveLlmConfig = async () => {
    if (llmProvider !== 'ollama' && !llmApiKey.trim() && !llmModel.trim()) {
      setError('API Key or Model Name is required');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const success = await engineClient.saveLlmSetupConfig({
        provider: llmProvider,
        apiKey: llmApiKey,
        model: llmModel,
        baseUrl: llmBaseUrl,
      });
      if (success) {
        setLlmConfigured(true);
        setCurrentStep(5);
      } else {
        setError('Failed to save AI configuration');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to save AI configuration');
    } finally {
      setLoading(false);
    }
  };

  const handleTestLlmConnection = async () => {
    setTestingLlm(true);
    setLlmTestResult(null);
    try {
      const res = await engineClient.testLlmConfig({
        provider: llmProvider,
        apiKey: llmApiKey,
        model: llmModel,
        baseUrl: llmBaseUrl,
      });
      setLlmTestResult(res);
    } catch (err: any) {
      setLlmTestResult({ success: false, error: err?.message || 'Connection test failed' });
    } finally {
      setTestingLlm(false);
    }
  };

  const handleTestConnection = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await engineClient.testSetupConnection();
      if (res.success) {
        setConnectionSuccess(true);
        if (res.app_name) setAppName(res.app_name);
      } else {
        setError(res.error || 'Connection test failed');
      }
    } catch (err: any) {
      setError(err.message || 'Connection test failed');
    } finally {
      setLoading(false);
    }
  };

  const handleComplete = () => {
    // Clear URL params
    window.history.replaceState({}, document.title, window.location.pathname);
    onNavigate('login');
  };

  const renderStepIndicator = () => (
    <div className="flex items-center justify-between mb-8 w-full max-w-2xl">
      {[1, 2, 3, 4, 5].map((step) => {
        const isCompleted =
          currentStep > step ||
          (step === 1 && appCreated) ||
          (step === 2 && appInstalled) ||
          (step === 3 && oauthConfigured) ||
          (step === 4 && llmConfigured);

        return (
          <React.Fragment key={step}>
            <div className="flex flex-col items-center">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center font-mono text-xs font-bold ${
                  currentStep === step
                    ? 'bg-black text-white'
                    : isCompleted
                      ? 'bg-emerald-600 text-white'
                      : 'bg-slate-200 text-slate-500'
                }`}
              >
                {isCompleted ? <CheckCircle2 className="w-4 h-4" /> : step}
              </div>
              <span className="text-[10px] uppercase tracking-wider font-mono text-slate-500 mt-2">
                {step === 1
                  ? 'Create App'
                  : step === 2
                    ? 'Install'
                    : step === 3
                      ? 'OAuth'
                      : step === 4
                        ? 'AI Config'
                        : 'Verify'}
              </span>
            </div>
            {step < 5 && (
              <div
                className={`flex-1 h-px mx-4 ${isCompleted ? 'bg-emerald-600' : 'bg-slate-200'}`}
              />
            )}
          </React.Fragment>
        );
      })}
    </div>
  );

  return (
    <div className="min-h-[calc(100vh-100px)] bg-slate-50 flex flex-col items-center justify-center p-4">
      {renderStepIndicator()}

      <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-sm p-8 shadow-none space-y-8">
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-sm flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5" />
            <div className="text-sm font-mono">{error}</div>
          </div>
        )}

        {/* Step 1: Create GitHub App */}
        {currentStep === 1 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Github className="w-6 h-6" /> Create GitHub App
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Triage requires a GitHub App to access repositories, read ASTs, and create issues
                for detected crashes. We'll automatically generate the configuration manifest for
                you.
              </p>
            </div>

            <div className="bg-slate-50 border border-slate-200 rounded-sm p-4 text-xs font-mono text-slate-700 space-y-2">
              <div className="flex items-center gap-2 font-semibold text-slate-900">
                <Shield className="w-4 h-4" /> App Permissions
              </div>
              <ul className="list-disc pl-5 space-y-1">
                <li>Read access to code (for AST isolation)</li>
                <li>Read/Write access to issues (for bug reports)</li>
                <li>Read access to metadata</li>
              </ul>
            </div>

            {appCreated ? (
              <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center justify-between">
                <div className="flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  GitHub App created successfully
                </div>
                <button
                  onClick={() => setCurrentStep(2)}
                  className="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-sm text-xs font-mono font-semibold transition-colors flex items-center gap-2"
                >
                  Next Step <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={handleCreateApp}
                disabled={loading}
                className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
              >
                {loading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <ExternalLink className="w-4 h-4" />
                )}
                {loading ? 'Generating Manifest...' : 'Create GitHub App on GitHub'}
              </button>
            )}
          </div>
        )}

        {/* Step 2: Install App */}
        {currentStep === 2 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Server className="w-6 h-6" /> Install App into Organization
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Now that the App is created, it needs to be installed into your GitHub organization
                or personal account to grant access to specific repositories.
              </p>
            </div>

            {appInstalled ? (
              <div className="space-y-4">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center justify-between font-mono text-sm">
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="w-5 h-5" />
                    <span>App installed successfully</span>
                  </div>
                  <button
                    type="button"
                    onClick={fetchRepos}
                    className="flex items-center gap-1 text-xs text-emerald-800 hover:text-emerald-950 font-semibold cursor-pointer underline"
                  >
                    <RefreshCw className="w-3 h-3" /> Refresh
                  </button>
                </div>

                <div className="bg-slate-50 border border-slate-200 rounded-sm p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-xs font-mono font-semibold text-slate-900 uppercase tracking-wider">
                      Granted Repositories ({repos.length})
                    </h3>
                    <button
                      type="button"
                      onClick={() => handleInstallApp()}
                      className="text-xs font-mono text-slate-700 hover:text-black flex items-center gap-1 cursor-pointer font-medium"
                    >
                      <PlusCircle className="w-3.5 h-3.5" />
                      <span>Add / Configure Repos</span>
                    </button>
                  </div>

                  {availableOrgs.length > 1 && (
                    <div className="flex items-center gap-1 overflow-x-auto pb-1 text-xs font-mono scrollbar-thin">
                      <button
                        type="button"
                        onClick={() => setSelectedOwnerFilter('all')}
                        className={`flex items-center gap-1 px-2.5 py-1 rounded-sm border transition-all shrink-0 cursor-pointer ${
                          selectedOwnerFilter === 'all'
                            ? 'bg-black text-white border-black font-bold'
                            : 'bg-white hover:bg-slate-100 text-slate-700 border-slate-200'
                        }`}
                      >
                        <Layers className="w-3 h-3" />
                        <span>All ({repos.length})</span>
                      </button>
                      {availableOrgs.map((org) => (
                        <button
                          key={org}
                          type="button"
                          onClick={() => setSelectedOwnerFilter(org)}
                          className={`flex items-center gap-1 px-2.5 py-1 rounded-sm border transition-all shrink-0 cursor-pointer ${
                            selectedOwnerFilter === org
                              ? 'bg-black text-white border-black font-bold'
                              : 'bg-white hover:bg-slate-100 text-slate-700 border-slate-200'
                          }`}
                        >
                          <Building2 className="w-3 h-3" />
                          <span>
                            {org} ({orgCounts[org]})
                          </span>
                        </button>
                      ))}
                    </div>
                  )}

                  {filteredRepos.length > 0 ? (
                    <ul className="space-y-1 max-h-48 overflow-y-auto">
                      {filteredRepos.map((repo, idx) => (
                        <li
                          key={idx}
                          className="text-sm font-mono text-slate-700 flex items-center justify-between p-1.5 hover:bg-white rounded-sm"
                        >
                          <div className="flex items-center gap-2">
                            <Github className="w-3.5 h-3.5" />
                            <span>
                              {repo.owner}/{repo.repo}
                            </span>
                          </div>
                          <div className="flex items-center gap-1.5">
                            {repo.private ? (
                              <span className="flex items-center gap-1 bg-slate-900 text-slate-200 px-1.5 py-0.5 rounded-sm border border-slate-700 font-semibold text-[10px]">
                                <Lock className="w-2.5 h-2.5 text-amber-400" />
                                <span>Private</span>
                              </span>
                            ) : (
                              <span className="flex items-center gap-1 bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded-sm border border-slate-200 text-[10px]">
                                <Globe className="w-2.5 h-2.5 text-slate-400" />
                                <span>Public</span>
                              </span>
                            )}
                            {repo.lang && (
                              <span className="text-[10px] bg-slate-200 text-slate-700 px-1.5 py-0.5 rounded-sm">
                                {repo.lang}
                              </span>
                            )}
                          </div>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="text-xs font-mono text-slate-500">
                      No repositories granted yet. Click above to grant repository access.
                    </p>
                  )}

                  <p className="text-[11px] font-mono text-slate-500 pt-1">
                    Tip: You only need to install on a single repository to continue. You can grant
                    access to more repositories at any time.
                  </p>
                </div>

                <button
                  onClick={() => setCurrentStep(3)}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full cursor-pointer"
                >
                  Continue to OAuth Setup <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={() => handleInstallApp()}
                disabled={loading}
                className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full cursor-pointer"
              >
                {loading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <ExternalLink className="w-4 h-4" />
                )}
                {loading ? 'Preparing Install...' : 'Install GitHub App on GitHub'}
              </button>
            )}
          </div>
        )}

        {/* Step 3: OAuth */}
        {currentStep === 3 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Key className="w-6 h-6" /> Dashboard Login Configuration
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                To allow users to log in to the Triage dashboard using GitHub, we need the OAuth
                credentials from your GitHub App.
              </p>
            </div>

            {oauthConfigured ? (
              <div className="space-y-4">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  OAuth credentials auto-configured from GitHub App
                </div>
                <button
                  onClick={() => setCurrentStep(4)}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Continue to AI Configuration <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">
                    Client ID
                  </label>
                  <input
                    type="text"
                    value={clientId}
                    onChange={(e) => setClientId(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder="Iv1.xxxxxxxxxxxx"
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">
                    Client Secret
                  </label>
                  <input
                    type="password"
                    value={clientSecret}
                    onChange={(e) => setClientSecret(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                  />
                </div>
                <button
                  onClick={handleSaveOAuth}
                  disabled={loading}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  {loading ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <CheckCircle2 className="w-4 h-4" />
                  )}
                  Save OAuth Configuration
                </button>
              </div>
            )}
          </div>
        )}

        {/* Step 4: AI Configuration */}
        {currentStep === 4 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Brain className="w-6 h-6" /> AI Diagnostics & Model Setup
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Choose your preferred LLM provider for incident root-cause analysis, AST
                symbolication, and automated patch synthesis.
              </p>
            </div>

            {llmConfigured ? (
              <div className="space-y-4">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  AI configuration saved
                </div>
                <button
                  onClick={() => setCurrentStep(5)}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Continue to Verification <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <div className="space-y-5">
                {/* Provider Selector Cards */}
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">
                    Select AI Provider
                  </label>
                  <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2.5">
                    {[
                      {
                        id: 'gemini',
                        name: 'Google Gemini',
                        sub: 'Gemini 2.0 Flash / Pro',
                        icon: <Sparkles className="w-4 h-4 text-indigo-600" />,
                      },
                      {
                        id: 'openai',
                        name: 'OpenAI',
                        sub: 'GPT-4o, o3-mini',
                        icon: <Zap className="w-4 h-4 text-emerald-600" />,
                      },
                      {
                        id: 'anthropic',
                        name: 'Anthropic Claude',
                        sub: 'Claude 3.5 / 3.7 Sonnet',
                        icon: <Cpu className="w-4 h-4 text-amber-600" />,
                      },
                      {
                        id: 'ollama',
                        name: 'Local / Ollama',
                        sub: 'DeepSeek, Qwen',
                        icon: <Server className="w-4 h-4 text-purple-600" />,
                      },
                    ].map((prov) => {
                      const isSelected = llmProvider === prov.id;
                      return (
                        <button
                          key={prov.id}
                          type="button"
                          onClick={() => {
                            setLlmProvider(prov.id as any);
                            setLlmTestResult(null);
                          }}
                          className={`p-3 rounded-sm border text-left transition-all cursor-pointer flex flex-col justify-between gap-2 ${
                            isSelected
                              ? 'border-black bg-slate-900 text-white shadow-sm'
                              : 'border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50 text-slate-800'
                          }`}
                        >
                          <div className="flex items-center justify-between w-full">
                            <span className={isSelected ? 'text-white' : ''}>{prov.icon}</span>
                            {isSelected && <Check className="w-3.5 h-3.5 text-white" />}
                          </div>
                          <div>
                            <div className="font-bold text-xs">{prov.name}</div>
                            <div
                              className={`text-[10px] truncate ${isSelected ? 'text-slate-300' : 'text-slate-500'}`}
                            >
                              {prov.sub}
                            </div>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Base URL for Ollama / Custom */}
                {(llmProvider === 'ollama' || llmProvider === 'custom') && (
                  <div className="space-y-1.5 bg-purple-50/60 border border-purple-200 p-3.5 rounded-sm">
                    <label className="text-xs font-mono font-semibold text-purple-950 uppercase tracking-wider block">
                      Endpoint Base URL
                    </label>
                    <input
                      type="text"
                      value={llmBaseUrl}
                      onChange={(e) => setLlmBaseUrl(e.target.value)}
                      placeholder="http://localhost:11434/v1"
                      className="w-full bg-white border border-purple-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-purple-600"
                    />
                  </div>
                )}

                {/* API Key */}
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">
                    {llmProvider === 'gemini'
                      ? 'Gemini API Key'
                      : llmProvider === 'openai'
                        ? 'OpenAI API Key'
                        : llmProvider === 'anthropic'
                          ? 'Anthropic API Key'
                          : 'API Key (Optional for Local Ollama)'}
                  </label>
                  <div className="relative">
                    <input
                      type={showLlmKey ? 'text' : 'password'}
                      value={llmApiKey}
                      onChange={(e) => setLlmApiKey(e.target.value)}
                      className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 pr-10 text-sm font-mono focus:outline-none focus:border-black"
                      placeholder={
                        llmProvider === 'gemini'
                          ? 'AIzaSy...'
                          : llmProvider === 'openai'
                            ? 'sk-proj-...'
                            : llmProvider === 'anthropic'
                              ? 'sk-ant-api03-...'
                              : 'Optional'
                      }
                    />
                    <button
                      type="button"
                      onClick={() => setShowLlmKey(!showLlmKey)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-700 p-1 font-mono text-xs"
                      title={showLlmKey ? 'Hide Key' : 'Show Key'}
                    >
                      {showLlmKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                {/* Model Name */}
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">
                    Model Name
                  </label>
                  <input
                    type="text"
                    value={llmModel}
                    onChange={(e) => setLlmModel(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder={
                      llmProvider === 'gemini'
                        ? 'e.g. gemini-2.0-flash'
                        : llmProvider === 'openai'
                          ? 'e.g. gpt-4o'
                          : llmProvider === 'anthropic'
                            ? 'e.g. claude-3-5-sonnet-20241022'
                            : 'e.g. deepseek-coder-v2'
                    }
                  />
                </div>

                {/* Test Result Pill */}
                {llmTestResult && (
                  <div
                    className={`p-3 rounded-sm border flex items-start gap-2 text-xs font-mono ${
                      llmTestResult.success
                        ? 'bg-emerald-50 border-emerald-200 text-emerald-900'
                        : 'bg-red-50 border-red-200 text-red-900'
                    }`}
                  >
                    {llmTestResult.success ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-600 shrink-0 mt-0.5" />
                    )}
                    <div>
                      <div className="font-bold">
                        {llmTestResult.success
                          ? `Connection verified successfully (${llmTestResult.latency_ms}ms)`
                          : 'Connection test failed'}
                      </div>
                      {!llmTestResult.success && llmTestResult.error && (
                        <div className="text-[11px] break-all text-red-800">
                          {llmTestResult.error}
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Buttons */}
                <div className="flex flex-col sm:flex-row gap-3 pt-2">
                  <button
                    type="button"
                    onClick={handleTestLlmConnection}
                    disabled={testingLlm}
                    className="bg-slate-100 hover:bg-slate-200 text-slate-800 px-4 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 border border-slate-200 cursor-pointer disabled:opacity-50"
                  >
                    {testingLlm ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <RefreshCw className="w-4 h-4" />
                    )}
                    Test Connection
                  </button>

                  <button
                    type="button"
                    onClick={handleSaveLlmConfig}
                    disabled={loading}
                    className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 flex-1 cursor-pointer disabled:opacity-50"
                  >
                    {loading ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <CheckCircle2 className="w-4 h-4" />
                    )}
                    Save AI Configuration
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Step 5: Test Connection */}
        {currentStep === 5 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Settings className="w-6 h-6" /> Verify Setup
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Run a final test to ensure the dashboard can communicate with the Triage engine and
                the GitHub API successfully.
              </p>
            </div>

            {connectionSuccess ? (
              <div className="space-y-6">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-6 rounded-sm flex flex-col items-center justify-center gap-4 text-center">
                  <div className="w-12 h-12 bg-emerald-100 rounded-full flex items-center justify-center">
                    <CheckCircle2 className="w-6 h-6 text-emerald-600" />
                  </div>
                  <div>
                    <h3 className="font-bold font-sans text-lg text-emerald-900">
                      Setup Complete!
                    </h3>
                    <p className="font-mono text-sm mt-1">
                      GitHub App verified{appName ? `: ${appName}` : ''}
                    </p>
                  </div>
                </div>
                <button
                  onClick={handleComplete}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Complete Setup & Continue to Login <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={handleTestConnection}
                disabled={loading}
                className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
              >
                {loading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <RefreshCw className="w-4 h-4" />
                )}
                {loading ? 'Testing...' : 'Test Connection'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
