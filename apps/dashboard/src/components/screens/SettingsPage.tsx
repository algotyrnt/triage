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
} from 'lucide-react';
import { engineClient } from '@/services/engineClient';

interface SettingsPageProps {
  apiKeys: ApiKey[];
  onNavigate: (screen: ScreenId) => void;
  activeRepo?: string;
  activeRootDir?: string;
}

export const SettingsPage: React.FC<SettingsPageProps> = ({
  apiKeys,
  onNavigate,
  activeRepo = 'algotyrnt/beacon-app',
  activeRootDir = '',
}) => {
  const [activeTab, setActiveTab] = useState<'general' | 'keys' | 'webhooks' | 'ai' | 'danger'>(
    'ai',
  );
  const [keysList, setKeysList] = useState<ApiKey[]>(apiKeys);
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
  const [generalSaved, setGeneralSaved] = useState(false);
  const [savingGeneral, setSavingGeneral] = useState(false);

  useEffect(() => {
    setServiceRootDir(activeRootDir);
  }, [activeRootDir]);

  // Danger Zone delete modal
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmationInput, setDeleteConfirmationInput] = useState('');
  const targetRepoName = activeRepo;

  // AI settings
  const [geminiApiKey, setGeminiApiKey] = useState('');
  const [geminiModel, setGeminiModel] = useState('');
  const [aiSaved, setAiSaved] = useState(false);
  const [loadingAi, setLoadingAi] = useState(true);

  useEffect(() => {
    engineClient.getLlmConfig().then((cfg) => {
      if (cfg.gemini_api_key) setGeminiApiKey(cfg.gemini_api_key);
      if (cfg.gemini_model) setGeminiModel(cfg.gemini_model);
      setLoadingAi(false);
    });
  }, []);

  const handleSaveAiConfig = async () => {
    const success = await engineClient.updateLlmConfig(geminiApiKey, geminiModel);
    if (success) {
      setAiSaved(true);
      setTimeout(() => setAiSaved(false), 2000);
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
    navigator.clipboard.writeText(key.fullKey);
    setCopiedKeyId(key.id);
    setTimeout(() => setCopiedKeyId(null), 2000);
  };

  const handleRevokeKey = (id: string) => {
    setKeysList((prev) =>
      prev.map((k) => (k.id === id ? { ...k, status: 'REVOKED' as const } : k)),
    );
  };

  const handleCreateKey = () => {
    const today = new Date().toISOString().split('T')[0];
    const newKey: ApiKey = {
      id: `key-${Date.now()}`,
      name: 'New Developer Key',
      keyMasked: 'trj_demo_XXXXXXXX...8f',
      fullKey: 'trj_demo_XXXXXXXXXXXX8f',
      createdAt: today,
      lastUsed: 'Never',
      status: 'ACTIVE',
    };
    setKeysList((prev) => [...prev, newKey]);
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
            <div className="bg-white border border-slate-200 rounded-md overflow-hidden">
              <div className="p-4 border-b border-slate-200 bg-slate-50 flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-bold text-slate-900 font-mono flex items-center gap-2">
                    <Brain className="w-4 h-4 text-indigo-600" />
                    AI Model Configuration
                  </h2>
                  <p className="text-[11px] text-slate-500 font-mono mt-0.5">
                    Configure the Gemini API key and model used for automated crash analysis.
                  </p>
                </div>
              </div>
              <div className="p-5 space-y-5">
                <div className="space-y-1.5">
                  <label className="text-xs font-bold text-slate-800 font-mono">
                    Gemini API Key
                  </label>
                  <input
                    type={showSecret ? 'text' : 'password'}
                    value={geminiApiKey}
                    onChange={(e) => setGeminiApiKey(e.target.value)}
                    placeholder="AIzaSy..."
                    className="w-full text-sm font-mono border border-slate-200 rounded-sm px-3 py-2 bg-white focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                  <p className="text-[10px] text-slate-500 font-mono">
                    Leave blank to fallback to the GEMINI_API_KEY environment variable.
                  </p>
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-bold text-slate-800 font-mono">Model Name</label>
                  <input
                    type="text"
                    value={geminiModel}
                    onChange={(e) => setGeminiModel(e.target.value)}
                    placeholder="e.g. gemini-1.5-flash or gemini-2.5-pro"
                    className="w-full text-sm font-mono border border-slate-200 rounded-sm px-3 py-2 bg-white focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                  <p className="text-[10px] text-slate-500 font-mono">
                    Leave blank to fallback to the GEMINI_MODEL_NAME environment variable.
                  </p>
                </div>

                <div className="pt-2 flex justify-end">
                  <button
                    onClick={handleSaveAiConfig}
                    disabled={loadingAi}
                    className="flex items-center gap-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-mono font-medium px-4 py-1.5 rounded-sm transition-colors"
                  >
                    {aiSaved ? (
                      <Check className="w-3.5 h-3.5" />
                    ) : (
                      <Shield className="w-3.5 h-3.5" />
                    )}
                    <span>{aiSaved ? 'Saved' : 'Save Configuration'}</span>
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
                    Used by Go `sdk.Init()` to send crash stack traces to the telemetry engine.
                  </p>
                </div>
                <button
                  onClick={handleCreateKey}
                  className="bg-black hover:bg-slate-800 text-white font-bold px-3 py-1.5 rounded-sm flex items-center gap-1 transition-colors cursor-pointer self-start sm:self-auto"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Generate New Key</span>
                </button>
              </div>

              <div className="space-y-3">
                {keysList.map((key) => (
                  <div
                    key={key.id}
                    className="border border-slate-200 rounded-sm p-3 bg-slate-50/50 space-y-2"
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
                      </div>

                      <div className="text-[11px] text-slate-500">Created: {key.createdAt}</div>
                    </div>

                    <div className="flex items-center justify-between gap-2 bg-white p-2 rounded-sm border border-slate-200 text-xs">
                      <code className="text-slate-800 font-bold tracking-wide select-all">
                        {key.keyMasked}
                      </code>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleCopyKey(key)}
                          className="bg-slate-100 hover:bg-slate-200 text-slate-800 px-2 py-0.5 rounded-sm border border-slate-200 text-[11px] flex items-center gap-1 font-mono"
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
                            className="text-red-600 hover:text-red-800 text-[11px] font-mono underline"
                          >
                            Revoke Key
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
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
                  Triage posts JSON HTTP payloads when crashes are symbolicated or Gemini patches
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
                    <label className="font-bold text-slate-800 block">GitHub Organization / Owner:</label>
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
                    Scoping directory for AST symbolication and GitHub source fetching in monorepo structures.
                  </p>
                </div>

                <div className="pt-2">
                  <button
                    type="button"
                    disabled={savingGeneral}
                    onClick={async () => {
                      setSavingGeneral(true);
                      try {
                        await engineClient.createProject(activeRepo, serviceRootDir, repoOwner);
                        setGeneralSaved(true);
                        setTimeout(() => setGeneralSaved(false), 2500);
                      } catch (err) {
                        console.error('Failed to save project attributes:', err);
                      } finally {
                        setSavingGeneral(false);
                      }
                    }}
                    className="bg-black hover:bg-slate-800 text-white font-bold px-4 py-2 rounded-sm border border-black transition-colors cursor-pointer disabled:opacity-50"
                  >
                    {savingGeneral ? 'Saving...' : generalSaved ? '✓ Saved' : 'Save Project Attributes'}
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
