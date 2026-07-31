/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react';
import { ApiKey, ScreenId } from '../../types';
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
} from 'lucide-react';

interface SettingsPageProps {
  apiKeys: ApiKey[];
  onNavigate: (screen: ScreenId) => void;
}

export const SettingsPage: React.FC<SettingsPageProps> = ({ apiKeys, onNavigate }) => {
  const [activeTab, setActiveTab] = useState<'general' | 'keys' | 'webhooks' | 'danger'>('keys');
  const [keysList, setKeysList] = useState<ApiKey[]>(apiKeys);
  const [copiedKeyId, setCopiedKeyId] = useState<string | null>(null);

  // Webhook settings
  const [webhookUrl, setWebhookUrl] = useState('https://api.beacon-app.dev/v1/triage/webhook');
  const [webhookSecret, setWebhookSecret] = useState('whsec_9f8a3c2b1e4d7f6a');
  const [webhookSaved, setWebhookSaved] = useState(false);

  // Danger Zone delete modal
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmationInput, setDeleteConfirmationInput] = useState('');
  const targetRepoName = 'algotyrnt/beacon-app';

  const handleCopyKey = (key: ApiKey) => {
    navigator.clipboard.writeText(key.fullKey);
    setCopiedKeyId(key.id);
    setTimeout(() => setCopiedKeyId(null), 2000);
  };

  const handleRevokeKey = (id: string) => {
    setKeysList((prev) =>
      prev.map((k) => (k.id === id ? { ...k, status: 'REVOKED' as const } : k))
    );
  };

  const handleCreateKey = () => {
    const newKey: ApiKey = {
      id: `key-${Date.now()}`,
      name: 'New Developer Key',
      keyMasked: `trj_dev_${Math.random().toString(36).substring(2, 8)}...8f`,
      fullKey: `trj_dev_${Math.random().toString(36).substring(2, 18)}8f`,
      createdAt: '2026-07-28',
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

      {/* Settings Grid: Left Sidebar Sub-Nav vs Right Main Content Panel */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Sub-Nav Sidebar (col-span-3) */}
        <div className="lg:col-span-3 space-y-1 font-mono text-xs">
          {[
            { id: 'keys', label: 'API Ingestion Keys', icon: <Key className="w-3.5 h-3.5" /> },
            { id: 'webhooks', label: 'Webhook Endpoint', icon: <Webhook className="w-3.5 h-3.5" /> },
            { id: 'general', label: 'General Configuration', icon: <Sliders className="w-3.5 h-3.5" /> },
            { id: 'danger', label: 'Danger Zone', icon: <AlertTriangle className="w-3.5 h-3.5 text-red-600" /> },
          ].map((tab) => {
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`w-full text-left px-3 py-2 rounded-sm border transition-all flex items-center gap-2 cursor-pointer ${
                  isActive
                    ? tab.id === 'danger'
                      ? 'bg-red-50 text-red-900 border-red-300 font-bold'
                      : 'bg-black text-white border-black font-bold'
                    : 'bg-white border-slate-200 text-slate-700 hover:bg-slate-50'
                }`}
              >
                <span>{tab.icon}</span>
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* Right Main Content Panel (col-span-9) */}
        <div className="lg:col-span-9 space-y-6">
          {/* Tab 1: API Keys */}
          {activeTab === 'keys' && (
            <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-4">
              <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                <div>
                  <h2 className="text-sm font-bold text-slate-900 font-mono">
                    API Ingestion Keys & Telemetry Tokens
                  </h2>
                  <p className="text-xs text-slate-500 font-sans">
                    Keys used by Go microservices to transmit symbolicated stack traces to Triage.
                  </p>
                </div>

                <button
                  onClick={handleCreateKey}
                  className="bg-black hover:bg-slate-800 text-white font-mono text-xs px-3 py-1.5 rounded-sm flex items-center gap-1.5 transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Generate New Key</span>
                </button>
              </div>

              {/* API Key Table */}
              <div className="overflow-x-auto border border-slate-200 rounded-sm">
                <table className="w-full text-left font-mono text-xs">
                  <thead className="bg-slate-50 border-b border-slate-200 text-slate-500 text-[11px] uppercase tracking-wider">
                    <tr>
                      <th className="py-2.5 px-3 font-semibold">Key Name</th>
                      <th className="py-2.5 px-3 font-semibold">Key Token</th>
                      <th className="py-2.5 px-3 font-semibold">Created</th>
                      <th className="py-2.5 px-3 font-semibold">Last Used</th>
                      <th className="py-2.5 px-3 font-semibold">Status</th>
                      <th className="py-2.5 px-3 font-semibold text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 text-slate-800">
                    {keysList.map((key) => {
                      const isActive = key.status === 'ACTIVE';
                      return (
                        <tr key={key.id} className="hover:bg-slate-50">
                          <td className="py-2.5 px-3 font-bold text-slate-900">{key.name}</td>
                          <td className="py-2.5 px-3 text-slate-700 font-bold">{key.keyMasked}</td>
                          <td className="py-2.5 px-3 text-slate-500">{key.createdAt}</td>
                          <td className="py-2.5 px-3 text-slate-500">{key.lastUsed}</td>
                          <td className="py-2.5 px-3">
                            <span
                              className={`text-[10px] font-bold px-1.5 py-0.2 rounded-sm border ${
                                isActive
                                  ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                                  : 'bg-slate-100 text-slate-500 border-slate-200'
                              }`}
                            >
                              {key.status}
                            </span>
                          </td>
                          <td className="py-2.5 px-3 text-right space-x-2">
                            {isActive && (
                              <>
                                <button
                                  onClick={() => handleCopyKey(key)}
                                  className="text-slate-700 hover:text-black text-xs underline font-mono"
                                >
                                  {copiedKeyId === key.id ? 'Copied' : 'Copy'}
                                </button>
                                <button
                                  onClick={() => handleRevokeKey(key.id)}
                                  className="text-red-600 hover:text-red-800 text-xs underline font-mono"
                                >
                                  Revoke
                                </button>
                              </>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Tab 2: Webhooks */}
          {activeTab === 'webhooks' && (
            <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-4">
              <div className="border-b border-slate-100 pb-3">
                <h2 className="text-sm font-bold text-slate-900 font-mono">
                  Webhook Endpoint & Event Signing
                </h2>
                <p className="text-xs text-slate-500 font-sans">
                  Configure outgoing webhook notifications when critical Go runtime panics occur.
                </p>
              </div>

              {webhookSaved && (
                <div className="bg-emerald-50 border border-emerald-200 p-2.5 rounded-sm text-xs font-mono text-emerald-800 flex items-center gap-1.5">
                  <Check className="w-3.5 h-3.5 text-emerald-600" />
                  <span>Webhook configuration updated and HMAC secret saved!</span>
                </div>
              )}

              <form onSubmit={handleSaveWebhook} className="space-y-4 font-mono text-xs">
                <div className="space-y-1">
                  <label className="font-bold text-slate-800 block">Target Webhook URL:</label>
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
                  <input
                    type="text"
                    value={webhookSecret}
                    onChange={(e) => setWebhookSecret(e.target.value)}
                    required
                    className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                  />
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
                  General Project Attributes
                </h2>
              </div>

              <div className="space-y-3">
                <div>
                  <label className="font-bold text-slate-800 block">Project Name:</label>
                  <input
                    type="text"
                    value="beacon-app"
                    readOnly
                    className="w-full px-3 py-2 bg-slate-100 border border-slate-200 rounded-sm text-slate-700 font-mono cursor-not-allowed"
                  />
                </div>

                <div>
                  <label className="font-bold text-slate-800 block">GitHub Organization:</label>
                  <input
                    type="text"
                    value="algotyrnt"
                    readOnly
                    className="w-full px-3 py-2 bg-slate-100 border border-slate-200 rounded-sm text-slate-700 font-mono cursor-not-allowed"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Tab 4 / Danger Zone: Red Container (#FEF2F2 with #FCA5A5 border) and Solid Red Button (#DC2626) */}
          {(activeTab === 'danger' || activeTab === 'keys' || activeTab === 'webhooks' || activeTab === 'general') && (
            <div className="bg-red-50 border border-red-300 rounded-sm p-4 space-y-3 font-mono text-xs">
              <div className="flex items-center gap-2 font-bold text-red-900 border-b border-red-200 pb-2">
                <AlertTriangle className="w-4 h-4 text-red-600" />
                <span>Danger Zone — Irreversible Project Actions</span>
              </div>

              <p className="text-red-800 text-[11.5px] leading-relaxed">
                Deleting this project will permanently remove all AST symbolication caches, webhook audit logs, and API ingestion keys for <strong className="text-red-950 font-bold">{targetRepoName}</strong>. This action cannot be undone.
              </p>

              <div className="pt-1">
                <button
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
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4 font-mono">
          <div className="bg-white border border-red-300 rounded-sm p-6 w-full max-w-md space-y-4 shadow-none text-xs">
            <div className="flex items-center justify-between border-b border-red-100 pb-3">
              <div className="font-bold text-red-900 text-sm flex items-center gap-1.5">
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
              Please type <strong className="text-slate-900 font-bold">{targetRepoName}</strong> below to confirm deletion:
            </p>

            <input
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
