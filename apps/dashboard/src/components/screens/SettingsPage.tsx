/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useRef } from 'react';
import { ApiKey, ScreenId } from '@/types';
import {
  Settings,
  Key,
  Webhook,
  AlertTriangle,
  Copy,
  Check,
  Plus,
  Trash2,
  Lock,
  X,
  Shield,
  Sliders,
  Brain,
  Eye,
  EyeOff,
  Loader2,
  RefreshCw,
  Sparkles,
  Zap,
  Cpu,
  Server,
  CheckCircle2,
  XCircle,
} from 'lucide-react';
import { engineClient } from '@/services/engineClient';
import { logger } from '@/services/logger';
import { GeminiIcon, OpenAIIcon, AnthropicIcon, OllamaIcon } from '@/components/ProviderIcons';

interface SettingsPageProps {
  apiKeys?: ApiKey[];
  activeApiKey?: string;
  onKeyUpdated?: (newKey: string) => void;
  onNavigate: (screen: ScreenId) => void;
  activeRepo?: string;
  activeRootDir?: string;
}

export const SettingsPage: React.FC<SettingsPageProps> = ({
  apiKeys = [],
  activeApiKey,
  onKeyUpdated,
  onNavigate,
  activeRepo = 'algotyrnt/beacon-app',
  activeRootDir = '',
}) => {
  const [activeTab, setActiveTab] = useState<'general' | 'keys' | 'webhooks' | 'ai' | 'danger'>(
    'ai',
  );
  const [keysList, setKeysList] = useState<ApiKey[]>(apiKeys);
  const [loadingKeys, setLoadingKeys] = useState(false);
  const [creatingKey, setCreatingKey] = useState(false);
  const [keyToast, setKeyToast] = useState<{
    message: string;
    variant: 'success' | 'error';
  } | null>(null);
  const [newlyCreatedKeys, setNewlyCreatedKeys] = useState<Record<string, string>>({});
  const [copiedKeyId, setCopiedKeyId] = useState<string | null>(null);

  // Webhook settings
  const [webhookUrl, setWebhookUrl] = useState('https://api.beacon-app.dev/v1/triage/webhook');
  const [webhookSecret, setWebhookSecret] = useState('whsec_demo_XXXXXXXXXXXX');
  const [showSecret, setShowSecret] = useState(false);
  const [webhookSaved, setWebhookSaved] = useState(false);

  // General settings
  const parts = activeRepo.split('/');
  const repoOwner = parts[0] || 'algotyrnt';
  const repoName = parts[1] || activeRepo;
  const [serviceRootDir, setServiceRootDir] = useState(activeRootDir);
  const [projectContext, setProjectContext] = useState('');
  const [generalSaved, setGeneralSaved] = useState(false);
  const [savingGeneral, setSavingGeneral] = useState(false);

  useEffect(() => {
    setServiceRootDir(activeRootDir);
  }, [activeRootDir]);

  useEffect(() => {
    engineClient.getProjects().then((projects) => {
      const match = projects.find(
        (p: any) =>
          p.owner === repoOwner &&
          p.repo === repoName &&
          (p.root_dir || '') === (serviceRootDir || ''),
      );
      if (match && match.context) {
        setProjectContext(match.context);
      }
    });
  }, [repoOwner, repoName, serviceRootDir]);

  // Danger Zone delete modal
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmationInput, setDeleteConfirmationInput] = useState('');
  const targetRepoName = activeRepo;

  // AI settings
  const [llmProvider, setLlmProvider] = useState<
    'gemini' | 'openai' | 'anthropic' | 'ollama' | 'custom'
  >('gemini');
  const [llmApiKey, setLlmApiKey] = useState('');
  const [llmModel, setLlmModel] = useState('');
  const [llmBaseUrl, setLlmBaseUrl] = useState('');
  const [showLlmSecret, setShowLlmSecret] = useState(false);
  const [aiSaved, setAiSaved] = useState(false);
  const [loadingAi, setLoadingAi] = useState(true);
  const [testingAi, setTestingAi] = useState(false);
  const [testResult, setTestResult] = useState<{
    success: boolean;
    latency_ms?: number;
    error?: string;
  } | null>(null);

  const loadProjectKeys = async () => {
    setLoadingKeys(true);
    try {
      const fetchedKeys = await engineClient.getProjectKeys(repoOwner, repoName, serviceRootDir);
      const storageKey = `triage_key_${repoOwner}_${repoName}_${serviceRootDir}`;
      const localStoredKey = localStorage.getItem(storageKey);
      const activeOrStoredKey = activeApiKey || localStoredKey;

      const merged = fetchedKeys.map((k) => {
        if (newlyCreatedKeys[k.id]) {
          return { ...k, fullKey: newlyCreatedKeys[k.id] };
        }
        if (
          activeOrStoredKey &&
          k.status === 'ACTIVE' &&
          activeOrStoredKey.endsWith(k.keyMasked.replace(/^tr_live_\.\.\./, ''))
        ) {
          return { ...k, fullKey: activeOrStoredKey };
        }
        return k;
      });
      setKeysList(merged);
    } catch (e) {
      logger.warn('Failed to load project keys', e);
    } finally {
      setLoadingKeys(false);
    }
  };

  useEffect(() => {
    loadProjectKeys();
  }, [repoOwner, repoName, serviceRootDir]);

  useEffect(() => {
    engineClient.getLlmConfig().then((cfg) => {
      if (cfg.provider) setLlmProvider(cfg.provider as any);
      if (cfg.api_key) setLlmApiKey(cfg.api_key);
      if (cfg.model) setLlmModel(cfg.model);
      if (cfg.base_url) setLlmBaseUrl(cfg.base_url);
      setLoadingAi(false);
    });
  }, []);

  const handleSaveAiConfig = async () => {
    const success = await engineClient.updateLlmConfig({
      provider: llmProvider,
      apiKey: llmApiKey,
      model: llmModel,
      baseUrl: llmBaseUrl,
    });
    if (success) {
      setAiSaved(true);
      setTimeout(() => setAiSaved(false), 2000);
    }
  };

  const handleTestAiConfig = async () => {
    setTestingAi(true);
    setTestResult(null);
    try {
      const res = await engineClient.testLlmConfig({
        provider: llmProvider,
        apiKey: llmApiKey,
        model: llmModel,
        baseUrl: llmBaseUrl,
      });
      setTestResult(res);
    } catch (err: any) {
      setTestResult({ success: false, error: err?.message || 'Connection test failed' });
    } finally {
      setTestingAi(false);
    }
  };

  const deleteTriggerRef = useRef<HTMLButtonElement | null>(null);
  const deleteInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (showDeleteModal) {
      deleteInputRef.current?.focus();
      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
          setShowDeleteModal(false);
        }
      };
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    } else {
      deleteTriggerRef.current?.focus();
    }
  }, [showDeleteModal]);

  const handleCopyKey = (key: ApiKey) => {
    const rawToCopy =
      key.fullKey ||
      newlyCreatedKeys[key.id] ||
      (activeApiKey && activeApiKey.endsWith(key.keyMasked.replace(/^tr_live_\.\.\./, ''))
        ? activeApiKey
        : key.keyMasked);
    navigator.clipboard.writeText(rawToCopy);
    setCopiedKeyId(key.id);
    setTimeout(() => setCopiedKeyId(null), 2000);
  };

  const handleRevokeKey = async (id: string) => {
    try {
      const ok = await engineClient.revokeApiKey(id);
      if (ok) {
        setKeysList((prev) =>
          prev.map((k) => (k.id === id ? { ...k, status: 'REVOKED' as const } : k)),
        );
        setKeyToast({ message: 'API key revoked successfully.', variant: 'success' });
        setTimeout(() => setKeyToast(null), 4000);
      } else {
        setKeyToast({ message: 'Failed to revoke API key.', variant: 'error' });
        setTimeout(() => setKeyToast(null), 4000);
      }
    } catch {
      setKeyToast({ message: 'Error while revoking API key.', variant: 'error' });
      setTimeout(() => setKeyToast(null), 4000);
    }
  };

  const handleCreateKey = async () => {
    setCreatingKey(true);
    try {
      const count = keysList.length + 1;
      const name = `Developer Ingestion Key ${count}`;
      const res = await engineClient.createApiKey(repoOwner, repoName, serviceRootDir, name);
      if (res.success && res.key) {
        const raw = res.key.raw_key || res.key.fullKey;
        if (raw) {
          const storageKey = `triage_key_${repoOwner}_${repoName}_${serviceRootDir}`;
          localStorage.setItem(storageKey, raw);
          setNewlyCreatedKeys((prev) => ({ ...prev, [res.key!.id]: raw }));
          if (onKeyUpdated) onKeyUpdated(raw);
        }
        setKeysList((prev) => [res.key!, ...prev]);
        setKeyToast({
          message: 'New API Ingestion Key generated. Copy the full key below!',
          variant: 'success',
        });
        setTimeout(() => setKeyToast(null), 5000);
      } else {
        setKeyToast({ message: 'Failed to generate new API key.', variant: 'error' });
        setTimeout(() => setKeyToast(null), 4000);
      }
    } catch {
      setKeyToast({ message: 'Failed to generate new API key.', variant: 'error' });
      setTimeout(() => setKeyToast(null), 4000);
    } finally {
      setCreatingKey(false);
    }
  };

  const handleSaveWebhook = (e: React.FormEvent) => {
    e.preventDefault();
    setWebhookSaved(true);
    setTimeout(() => setWebhookSaved(false), 2500);
  };

  const handleDeleteProject = () => {
    if (deleteConfirmationInput === targetRepoName) {
      alert(`Project ${targetRepoName} deleted.`);
      setShowDeleteModal(false);
      onNavigate('new');
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
          Project Settings & API Credentials
        </h1>
        <p className="text-xs text-slate-600 font-sans mt-0.5">
          Configure API ingestion keys, webhook signatures, and project lifecycle boundaries.
        </p>
      </div>

      {/* Settings Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Sub-Nav Sidebar */}
        <div className="lg:col-span-3 space-y-1 font-mono text-xs">
          {[
            {
              id: 'ai',
              label: 'AI Configuration',
              icon: <Brain className="w-3.5 h-3.5" />,
            },
            {
              id: 'keys',
              label: 'API Ingestion Keys',
              icon: <Key className="w-3.5 h-3.5" />,
            },
            {
              id: 'webhooks',
              label: 'Webhook Endpoint',
              icon: <Webhook className="w-3.5 h-3.5" />,
            },
            {
              id: 'general',
              label: 'General Project Info',
              icon: <Sliders className="w-3.5 h-3.5" />,
            },
            {
              id: 'danger',
              label: 'Danger Zone',
              icon: <AlertTriangle className="w-3.5 h-3.5" />,
            },
          ].map((tab) => {
            const isActive = activeTab === tab.id;
            const isDanger = tab.id === 'danger';
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`w-full text-left p-2.5 rounded-sm transition-all border flex items-center gap-2 cursor-pointer ${
                  isActive
                    ? isDanger
                      ? 'bg-red-600 text-white font-bold border-red-700'
                      : 'bg-black text-white font-bold border-black'
                    : isDanger
                      ? 'bg-red-50/50 text-red-700 border-red-200 hover:bg-red-100'
                      : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-50'
                }`}
              >
                {tab.icon}
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* Right Main Content Panel */}
        <div className="lg:col-span-9 space-y-6">
          {activeTab === 'ai' && (
            <div className="bg-white border border-slate-200 rounded-md overflow-hidden font-mono">
              <div className="p-4 border-b border-slate-200 bg-slate-50 flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                    <Brain className="w-4 h-4 text-indigo-600" />
                    Pluggable AI Diagnostics & Patch Engine
                  </h2>
                  <p className="text-[11px] text-slate-500 mt-0.5 font-sans">
                    Configure your preferred LLM provider for zero-token incident root cause
                    analysis, automated bugfix diff generation, and domain-aware triage.
                  </p>
                </div>
              </div>
              <div className="p-5 space-y-6 text-xs">
                {/* Provider Selector Cards */}
                <div className="space-y-2">
                  <label className="font-bold text-slate-800 block">Select AI Provider:</label>
                  <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2.5">
                    {[
                      {
                        id: 'gemini',
                        name: 'Google Gemini',
                        sub: 'Gemini 2.0 Flash / Pro',
                        icon: (selected: boolean) => (
                          <GeminiIcon
                            className={`w-4 h-4 ${selected ? 'text-white' : 'text-indigo-600'}`}
                          />
                        ),
                      },
                      {
                        id: 'openai',
                        name: 'OpenAI',
                        sub: 'GPT-4o, o3-mini, o1',
                        icon: (selected: boolean) => (
                          <OpenAIIcon
                            className={`w-4 h-4 ${selected ? 'text-white' : 'text-emerald-600'}`}
                          />
                        ),
                      },
                      {
                        id: 'anthropic',
                        name: 'Anthropic Claude',
                        sub: 'Claude 3.5 / 3.7 Sonnet',
                        icon: (selected: boolean) => (
                          <AnthropicIcon
                            className={`w-4 h-4 ${selected ? 'text-white' : 'text-amber-600'}`}
                          />
                        ),
                      },
                      {
                        id: 'ollama',
                        name: 'Local / Ollama',
                        sub: 'DeepSeek, Qwen (Air-Gapped)',
                        icon: (selected: boolean) => (
                          <OllamaIcon
                            className={`w-4 h-4 ${selected ? 'text-white' : 'text-purple-600'}`}
                          />
                        ),
                      },
                    ].map((prov) => {
                      const isSelected = llmProvider === prov.id;
                      return (
                        <button
                          key={prov.id}
                          type="button"
                          onClick={() => {
                            setLlmProvider(prov.id as any);
                            setTestResult(null);
                          }}
                          className={`p-3 rounded-sm border text-left transition-all cursor-pointer flex flex-col justify-between gap-2 ${
                            isSelected
                              ? 'border-black bg-slate-900 text-white shadow-sm'
                              : 'border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50 text-slate-800'
                          }`}
                        >
                          <div className="flex items-center justify-between w-full">
                            <span>{prov.icon(isSelected)}</span>
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

                {/* Provider-Specific Base URL (for Ollama / Custom) */}
                {(llmProvider === 'ollama' || llmProvider === 'custom') && (
                  <div className="space-y-1.5 bg-purple-50/60 border border-purple-200 p-3.5 rounded-sm">
                    <div className="flex items-center justify-between">
                      <label className="font-bold text-purple-950 block">
                        OpenAI-Compatible Endpoint URL (Base URL):
                      </label>
                      <span className="text-[10px] text-purple-700 font-sans">
                        Self-Hosted / Local
                      </span>
                    </div>
                    <input
                      type="text"
                      value={llmBaseUrl}
                      onChange={(e) => setLlmBaseUrl(e.target.value)}
                      placeholder="http://localhost:11434/v1 (e.g. Ollama, vLLM, LM Studio)"
                      className="w-full text-xs font-mono border border-purple-200 rounded-sm px-3 py-2 bg-white focus:outline-none focus:border-purple-600"
                    />
                    <p className="text-[10px] text-purple-800 font-sans">
                      Standard OpenAI chat completions compatible endpoint. Defaults to{' '}
                      <code className="bg-purple-100 px-1 py-0.5 rounded text-purple-900 font-mono">
                        http://localhost:11434/v1
                      </code>{' '}
                      for Ollama.
                    </p>
                  </div>
                )}

                {/* API Key Input */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="font-bold text-slate-800 block">
                      {llmProvider === 'gemini'
                        ? 'Google Gemini API Key'
                        : llmProvider === 'openai'
                          ? 'OpenAI API Key'
                          : llmProvider === 'anthropic'
                            ? 'Anthropic API Key'
                            : 'API Key (Optional for Local Ollama)'}
                    </label>
                    {llmProvider === 'ollama' && (
                      <span className="text-[10px] text-slate-500 font-sans">
                        Optional for local instances
                      </span>
                    )}
                  </div>
                  <div className="relative">
                    <input
                      type={showLlmSecret ? 'text' : 'password'}
                      value={llmApiKey}
                      onChange={(e) => setLlmApiKey(e.target.value)}
                      placeholder={
                        llmProvider === 'gemini'
                          ? 'AIzaSy...'
                          : llmProvider === 'openai'
                            ? 'sk-proj-...'
                            : llmProvider === 'anthropic'
                              ? 'sk-ant-api03-...'
                              : 'Leave blank if local endpoint does not require auth'
                      }
                      className="w-full text-xs font-mono border border-slate-200 rounded-sm px-3 py-2 pr-12 bg-white focus:outline-none focus:border-black"
                    />
                    <button
                      type="button"
                      onClick={() => setShowLlmSecret(!showLlmSecret)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-700 p-1 text-xs"
                      title={showLlmSecret ? 'Hide Key' : 'Show Key'}
                    >
                      {showLlmSecret ? (
                        <EyeOff className="w-3.5 h-3.5" />
                      ) : (
                        <Eye className="w-3.5 h-3.5" />
                      )}
                    </button>
                  </div>
                </div>

                {/* Model Name Input */}
                <div className="space-y-1.5">
                  <label className="font-bold text-slate-800 block">Model Name</label>
                  <input
                    type="text"
                    value={llmModel}
                    onChange={(e) => setLlmModel(e.target.value)}
                    placeholder={
                      llmProvider === 'gemini'
                        ? 'gemini-2.0-flash (default) or gemini-2.5-pro'
                        : llmProvider === 'openai'
                          ? 'gpt-4o (default) or o3-mini'
                          : llmProvider === 'anthropic'
                            ? 'claude-3-5-sonnet-20241022 (default) or claude-3-7-sonnet-20250219'
                            : 'deepseek-coder-v2 (default) or qwen2.5-coder:7b'
                    }
                    className="w-full text-xs font-mono border border-slate-200 rounded-sm px-3 py-2 bg-white focus:outline-none focus:border-black"
                  />
                  <p className="text-[10px] text-slate-500 font-sans">
                    Leave blank to use default model or environment variables (
                    <code className="bg-slate-100 px-1 py-0.5 rounded text-slate-700 font-mono">
                      LLM_MODEL
                    </code>
                    ).
                  </p>
                </div>

                {/* Test Connection Banner */}
                {testResult && (
                  <div
                    className={`p-3 rounded-sm border flex items-start gap-2 text-xs ${
                      testResult.success
                        ? 'bg-emerald-50 border-emerald-200 text-emerald-900'
                        : 'bg-red-50 border-red-200 text-red-900'
                    }`}
                  >
                    {testResult.success ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-600 shrink-0 mt-0.5" />
                    )}
                    <div className="space-y-0.5">
                      <div className="font-bold">
                        {testResult.success
                          ? `Connection verified successfully (${testResult.latency_ms}ms)`
                          : 'Connection test failed'}
                      </div>
                      {!testResult.success && testResult.error && (
                        <div className="text-[11px] font-mono break-all text-red-800">
                          {testResult.error}
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Action Buttons */}
                <div className="pt-2 flex flex-col sm:flex-row items-center justify-between gap-3 border-t border-slate-100">
                  <button
                    type="button"
                    onClick={handleTestAiConfig}
                    disabled={testingAi}
                    className="w-full sm:w-auto flex items-center justify-center gap-1.5 bg-slate-100 hover:bg-slate-200 text-slate-800 text-xs font-bold px-4 py-2 rounded-sm border border-slate-200 transition-colors cursor-pointer disabled:opacity-50"
                  >
                    {testingAi ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="w-3.5 h-3.5" />
                    )}
                    <span>{testingAi ? 'Testing Endpoint...' : 'Test Connection'}</span>
                  </button>

                  <button
                    type="button"
                    onClick={handleSaveAiConfig}
                    disabled={loadingAi}
                    className="w-full sm:w-auto flex items-center justify-center gap-1.5 bg-black hover:bg-slate-800 text-white text-xs font-bold px-5 py-2 rounded-sm transition-colors cursor-pointer disabled:opacity-50"
                  >
                    {aiSaved ? (
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                    ) : (
                      <Shield className="w-3.5 h-3.5" />
                    )}
                    <span>{aiSaved ? 'Configuration Saved' : 'Save AI Configuration'}</span>
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Tab 1: API Ingestion Keys */}
          {activeTab === 'keys' && (
            <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-4 font-mono text-xs">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-100 pb-3">
                <div>
                  <h2 className="text-sm font-bold text-slate-900 font-mono">
                    Project API Ingestion Keys
                  </h2>
                  <p className="text-[11px] text-slate-500 font-mono mt-0.5">
                    Used by Go{' '}
                    <code className="font-bold text-slate-700">triage.Middleware(apiKey, url)</code>{' '}
                    to authenticate crash telemetry.
                  </p>
                </div>
                <div className="flex items-center gap-2 self-start sm:self-auto">
                  <button
                    onClick={loadProjectKeys}
                    disabled={loadingKeys}
                    className="bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold px-2.5 py-1.5 rounded-sm flex items-center gap-1 border border-slate-200 transition-colors cursor-pointer"
                    title="Refresh keys list"
                  >
                    <RefreshCw className={`w-3.5 h-3.5 ${loadingKeys ? 'animate-spin' : ''}`} />
                    <span className="hidden sm:inline">Refresh</span>
                  </button>
                  <button
                    onClick={handleCreateKey}
                    disabled={creatingKey}
                    className="bg-black hover:bg-slate-800 text-white font-bold px-3 py-1.5 rounded-sm flex items-center gap-1 transition-colors cursor-pointer"
                  >
                    {creatingKey ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Plus className="w-3.5 h-3.5" />
                    )}
                    <span>{creatingKey ? 'Generating...' : 'Generate New Key'}</span>
                  </button>
                </div>
              </div>

              {keyToast && (
                <div
                  className={`p-3 rounded-sm text-xs font-mono flex items-center justify-between gap-2 border ${
                    keyToast.variant === 'success'
                      ? 'bg-emerald-50 text-emerald-800 border-emerald-200'
                      : 'bg-red-50 text-red-800 border-red-200'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {keyToast.variant === 'success' ? (
                      <Check className="w-4 h-4 text-emerald-600 shrink-0" />
                    ) : (
                      <AlertTriangle className="w-4 h-4 text-red-600 shrink-0" />
                    )}
                    <span>{keyToast.message}</span>
                  </div>
                  <button
                    onClick={() => setKeyToast(null)}
                    className="text-slate-500 hover:text-slate-800"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              )}

              {loadingKeys && keysList.length === 0 ? (
                <div className="text-center py-8 text-slate-400 space-y-2">
                  <Loader2 className="w-5 h-5 animate-spin mx-auto text-slate-600" />
                  <p className="text-xs font-mono">Loading project API keys...</p>
                </div>
              ) : keysList.length === 0 ? (
                <div className="text-center py-8 border border-dashed border-slate-200 rounded-sm bg-slate-50 space-y-2">
                  <Key className="w-6 h-6 text-slate-400 mx-auto" />
                  <p className="font-bold text-slate-700">No API Keys Generated</p>
                  <p className="text-[11px] text-slate-500 max-w-sm mx-auto font-sans">
                    Generate an Ingestion API Key to initialize the Go panic recovery middleware in
                    your application.
                  </p>
                  <button
                    onClick={handleCreateKey}
                    disabled={creatingKey}
                    className="mt-2 bg-black hover:bg-slate-800 text-white font-bold px-3 py-1.5 rounded-sm inline-flex items-center gap-1.5"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>Generate First Key</span>
                  </button>
                </div>
              ) : (
                <div className="space-y-3">
                  {keysList.map((key) => {
                    const isNewlyCreated = Boolean(newlyCreatedKeys[key.id]);
                    const rawKeyDisplay =
                      key.fullKey ||
                      newlyCreatedKeys[key.id] ||
                      (activeApiKey &&
                      activeApiKey.endsWith(key.keyMasked.replace(/^tr_live_\.\.\./, ''))
                        ? activeApiKey
                        : null);

                    return (
                      <div
                        key={key.id}
                        className={`border rounded-sm p-3 space-y-2 transition-colors ${
                          isNewlyCreated
                            ? 'border-emerald-300 bg-emerald-50/40'
                            : 'border-slate-200 bg-slate-50/50'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Key className="w-3.5 h-3.5 text-slate-700" />
                            <span className="font-bold text-slate-900 text-xs">{key.name}</span>
                            <span
                              className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm border ${
                                key.status === 'ACTIVE'
                                  ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                                  : 'bg-red-50 text-red-700 border-red-200'
                              }`}
                            >
                              {key.status}
                            </span>
                            {rawKeyDisplay && key.status === 'ACTIVE' && (
                              <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-sm bg-indigo-50 text-indigo-700 border border-indigo-200">
                                FULL KEY AVAILABLE
                              </span>
                            )}
                          </div>

                          <div className="text-[11px] text-slate-500">Created: {key.createdAt}</div>
                        </div>

                        <div className="flex items-center justify-between gap-2 bg-white p-2 rounded-sm border border-slate-200 text-xs">
                          <code className="text-slate-800 font-bold tracking-wide select-all font-mono break-all">
                            {rawKeyDisplay || key.keyMasked}
                          </code>
                          <div className="flex items-center gap-2 shrink-0">
                            <button
                              onClick={() => handleCopyKey(key)}
                              className="bg-slate-100 hover:bg-slate-200 text-slate-800 px-2 py-0.5 rounded-sm border border-slate-200 text-[11px] flex items-center gap-1 font-mono cursor-pointer"
                            >
                              {copiedKeyId === key.id ? (
                                <Check className="w-3 h-3 text-emerald-600" />
                              ) : (
                                <Copy className="w-3 h-3 text-slate-600" />
                              )}
                              <span>{copiedKeyId === key.id ? 'Copied' : 'Copy'}</span>
                            </button>

                            {key.status === 'ACTIVE' && (
                              <button
                                onClick={() => handleRevokeKey(key.id)}
                                className="text-red-600 hover:text-red-800 text-[11px] font-mono underline cursor-pointer"
                              >
                                Revoke Key
                              </button>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}

          {/* Tab 2: Webhooks */}
          {activeTab === 'webhooks' && (
            <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-4 font-mono text-xs">
              <div className="border-b border-slate-100 pb-3">
                <h2 className="text-sm font-bold text-slate-900 font-mono">
                  Outbound Webhook Dispatch Settings
                </h2>
                <p className="text-[11px] text-slate-500 font-mono mt-0.5">
                  Triage posts JSON HTTP payloads when crashes are symbolicated or AI patches
                  generated.
                </p>
              </div>

              {webhookSaved && (
                <div className="p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-sm flex items-center gap-2 text-xs font-bold">
                  <Check className="w-4 h-4 text-emerald-600" />
                  <span>Webhook endpoint settings saved successfully.</span>
                </div>
              )}

              <form onSubmit={handleSaveWebhook} className="space-y-4 text-xs">
                <div className="space-y-1">
                  <label className="font-bold text-slate-800 block">
                    Target Webhook Listener URL:
                  </label>
                  <input
                    type="url"
                    value={webhookUrl}
                    onChange={(e) => setWebhookUrl(e.target.value)}
                    required
                    className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-800 block">HMAC Signature Secret:</label>
                  <div className="relative">
                    <input
                      type={showSecret ? 'text' : 'password'}
                      value={webhookSecret}
                      onChange={(e) => setWebhookSecret(e.target.value)}
                      required
                      className="w-full px-3 py-2 pr-16 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                    />
                    <button
                      type="button"
                      onClick={() => setShowSecret(!showSecret)}
                      className="absolute right-2 top-2 text-[11px] font-mono text-slate-600 hover:text-black font-bold"
                    >
                      {showSecret ? 'Hide' : 'Show'}
                    </button>
                  </div>
                  <p className="text-[11px] text-slate-500">
                    Dispatches include header `X-Triage-Signature: sha256=&lt;hmac&gt;`.
                  </p>
                </div>

                <div className="pt-2">
                  <button
                    type="submit"
                    className="bg-black hover:bg-slate-800 text-white font-bold px-4 py-2 rounded-sm border border-black transition-colors cursor-pointer"
                  >
                    Save Webhook Settings
                  </button>
                </div>
              </form>
            </div>
          )}

          {/* Tab 3: General */}
          {activeTab === 'general' && (
            <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-4 font-mono text-xs">
              <div className="border-b border-slate-100 pb-3">
                <h2 className="text-sm font-bold text-slate-900 font-mono">
                  General Project Attributes & Monorepo Configuration
                </h2>
                <p className="text-[11px] text-slate-500 font-mono mt-0.5">
                  View and manage repository metadata and subdirectory service scoping.
                </p>
              </div>

              {generalSaved && (
                <div className="p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-sm flex items-center gap-2 text-xs font-bold">
                  <Check className="w-4 h-4 text-emerald-600" />
                  <span>Project configuration saved successfully.</span>
                </div>
              )}

              <div className="space-y-4">
                <div>
                  <label className="font-bold text-slate-800 block">Project Repository:</label>
                  <input
                    type="text"
                    value={activeRepo}
                    readOnly
                    className="w-full px-3 py-2 bg-slate-100 border border-slate-200 rounded-sm text-slate-700 font-mono cursor-not-allowed"
                  />
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="font-bold text-slate-800 block">Repository Name:</label>
                    <input
                      type="text"
                      value={repoName}
                      readOnly
                      className="w-full px-3 py-2 bg-slate-100 border border-slate-200 rounded-sm text-slate-700 font-mono cursor-not-allowed"
                    />
                  </div>

                  <div>
                    <label className="font-bold text-slate-800 block">
                      GitHub Organization / Owner:
                    </label>
                    <input
                      type="text"
                      value={repoOwner}
                      readOnly
                      className="w-full px-3 py-2 bg-slate-100 border border-slate-200 rounded-sm text-slate-700 font-mono cursor-not-allowed"
                    />
                  </div>
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-800 block">
                    Go Service Subdirectory (Monorepo Root Path):
                  </label>
                  <input
                    type="text"
                    value={serviceRootDir}
                    onChange={(e) => setServiceRootDir(e.target.value)}
                    placeholder="e.g. backend or apps/api (leave empty for repository root)"
                    className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                  />
                  <p className="text-[11px] text-slate-500 font-mono">
                    Scoping directory for AST symbolication and GitHub source fetching in monorepo
                    structures.
                  </p>
                </div>

                <div className="space-y-1.5 pt-1">
                  <label className="font-bold text-slate-800 block">
                    Project Context &amp; Architectural Description (Domain-Aware AI Triage):
                  </label>
                  <textarea
                    value={projectContext}
                    onChange={(e) => setProjectContext(e.target.value)}
                    rows={4}
                    placeholder="e.g. High-throughput payment gateway processing Stripe and crypto webhooks with strict idempotency and database transaction rollbacks."
                    className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-sans text-xs text-slate-800 placeholder:text-slate-400 focus:bg-white focus:outline-none focus:border-black"
                  />
                  <p className="text-[11px] text-slate-500 font-mono">
                    Domain rules, business invariants, and architecture passed to the AI engine to
                    enrich panic crash diagnosis and patch generation.
                  </p>
                </div>

                <div className="pt-2">
                  <button
                    type="button"
                    disabled={savingGeneral}
                    onClick={async () => {
                      setSavingGeneral(true);
                      try {
                        await engineClient.createProject(
                          activeRepo,
                          serviceRootDir,
                          repoOwner,
                          projectContext,
                        );
                        setGeneralSaved(true);
                        setTimeout(() => setGeneralSaved(false), 2500);
                      } catch (err) {
                        logger.error('Failed to save project attributes', err);
                      } finally {
                        setSavingGeneral(false);
                      }
                    }}
                    className="bg-black hover:bg-slate-800 text-white font-bold px-4 py-2 rounded-sm border border-black transition-colors cursor-pointer disabled:opacity-50"
                  >
                    {savingGeneral
                      ? 'Saving...'
                      : generalSaved
                        ? 'Saved'
                        : 'Save Project Attributes'}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Tab 4: Danger Zone ONLY */}
          {activeTab === 'danger' && (
            <div className="bg-red-50 border border-red-300 rounded-sm p-4 space-y-3 font-mono text-xs">
              <div className="flex items-center gap-2 font-bold text-red-900 border-b border-red-200 pb-2">
                <AlertTriangle className="w-4 h-4 text-red-600" />
                <span>Danger Zone — Irreversible Project Actions</span>
              </div>

              <p className="text-red-800 text-[11.5px] leading-relaxed">
                Deleting this project will permanently remove all AST symbolication caches, webhook
                audit logs, and API ingestion keys for{' '}
                <strong className="text-red-950 font-bold">{targetRepoName}</strong>. This action
                cannot be undone.
              </p>

              <div className="pt-1">
                <button
                  ref={deleteTriggerRef}
                  onClick={() => setShowDeleteModal(true)}
                  className="bg-red-600 hover:bg-red-700 text-white font-mono text-xs font-bold px-4 py-2 rounded-sm transition-colors cursor-pointer border border-red-700 flex items-center gap-1.5"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Delete Project ({targetRepoName})</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteModal && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-modal-title"
          className="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4 font-mono"
        >
          <div className="bg-white border border-red-300 rounded-sm p-6 w-full max-w-md space-y-4 shadow-none text-xs">
            <div className="flex items-center justify-between border-b border-red-100 pb-3">
              <div
                id="delete-modal-title"
                className="font-bold text-red-900 text-sm flex items-center gap-1.5"
              >
                <AlertTriangle className="w-4 h-4 text-red-600" />
                <span>Confirm Project Deletion</span>
              </div>
              <button
                onClick={() => setShowDeleteModal(false)}
                className="text-slate-400 hover:text-black"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <p className="text-slate-700 leading-relaxed">
              Please type <strong className="text-slate-900 font-bold">{targetRepoName}</strong>{' '}
              below to confirm deletion:
            </p>

            <input
              ref={deleteInputRef}
              type="text"
              value={deleteConfirmationInput}
              onChange={(e) => setDeleteConfirmationInput(e.target.value)}
              placeholder={targetRepoName}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-red-600"
            />

            <div className="pt-2 flex justify-end gap-2">
              <button
                onClick={() => setShowDeleteModal(false)}
                className="bg-slate-100 text-slate-700 px-3 py-1.5 rounded-sm border border-slate-200"
              >
                Cancel
              </button>
              <button
                disabled={deleteConfirmationInput !== targetRepoName}
                onClick={handleDeleteProject}
                className={`font-bold px-4 py-1.5 rounded-sm transition-colors text-white ${
                  deleteConfirmationInput === targetRepoName
                    ? 'bg-red-600 hover:bg-red-700 cursor-pointer'
                    : 'bg-red-300 cursor-not-allowed'
                }`}
              >
                Delete Project
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
