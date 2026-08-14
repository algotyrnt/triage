/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { WebhookLog, ScreenId } from '@/types';
import {
  Webhook,
  CheckCircle2,
  AlertTriangle,
  Code2,
  Copy,
  Check,
  RefreshCw,
  Filter,
  ArrowRight,
  ExternalLink,
} from 'lucide-react';

interface WebhooksPageProps {
  logs: WebhookLog[];
  onNavigate: (screen: ScreenId) => void;
}

export const WebhooksPage: React.FC<WebhooksPageProps> = ({ logs, onNavigate }) => {
  const [selectedLog, setSelectedLog] = useState<WebhookLog | null>(logs[0] || null);
  const [filterType, setFilterType] = useState<string>('ALL');
  const [copiedPayload, setCopiedPayload] = useState(false);
  const [replaying, setReplaying] = useState(false);
  const [replaySuccess, setReplaySuccess] = useState(false);

  const filteredLogs = logs.filter((log) => {
    if (filterType === 'ALL') return true;
    if (filterType === 'SUCCESS') return log.status === 'SUCCESS';
    if (filterType === 'UNAUTHORIZED') return log.status === 'UNAUTHORIZED';
    if (filterType === 'ERROR') return log.status === 'ERROR';
    return true;
  });

  const visibleLog = filteredLogs.find((l) => l.id === selectedLog?.id) || filteredLogs[0] || null;

  const handleCopyPayload = () => {
    if (visibleLog) {
      navigator.clipboard.writeText(visibleLog.requestBody);
      setCopiedPayload(true);
      setTimeout(() => setCopiedPayload(false), 2000);
    }
  };

  const handleReplayWebhook = () => {
    setReplaying(true);
    setTimeout(() => {
      setReplaying(false);
      setReplaySuccess(true);
      setTimeout(() => setReplaySuccess(false), 3000);
    }, 500);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
            Webhooks & Ingress Delivery Logs
          </h1>
          <p className="text-xs text-slate-600 font-sans mt-0.5">
            Audit trail of HTTP webhook dispatches, signature verification status, and JSON payload
            inspection.
          </p>
        </div>

        <div className="flex items-center gap-2 font-mono text-xs">
          <div className="bg-emerald-50 text-emerald-700 border border-emerald-200 px-2.5 py-1 rounded-sm font-medium flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-600 animate-pulse"></span>
            <span>Ingress Endpoint Active</span>
          </div>
        </div>
      </div>

      {/* Filter bar */}
      <div className="flex items-center justify-between bg-white border border-slate-200 p-3 rounded-sm font-mono text-xs">
        <div className="flex items-center gap-2">
          <Filter className="w-3.5 h-3.5 text-slate-500" />
          <span className="text-slate-600">Filter Event Status:</span>
          {(['ALL', 'SUCCESS', 'UNAUTHORIZED', 'ERROR'] as const).map((type) => (
            <button
              key={type}
              onClick={() => setFilterType(type)}
              className={`px-2 py-0.5 rounded-sm transition-colors ${
                filterType === type
                  ? 'bg-black text-white font-bold'
                  : 'bg-slate-100 hover:bg-slate-200 text-slate-700'
              }`}
            >
              {type}
            </button>
          ))}
        </div>

        <span className="text-slate-500 text-[11px]">{filteredLogs.length} logs recorded</span>
      </div>

      {/* Main Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Audit Table */}
        <div className="lg:col-span-7 bg-white border border-slate-200 rounded-sm overflow-hidden space-y-0">
          <div className="bg-slate-100 border-b border-slate-200 p-3 font-mono text-xs font-bold text-slate-900 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Webhook className="w-4 h-4 text-slate-800" />
              <span>Webhook Audit Trail</span>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left font-mono text-xs">
              <thead className="bg-slate-50 border-b border-slate-200 text-slate-500 text-[11px] uppercase tracking-wider">
                <tr>
                  <th className="py-2.5 px-3 font-semibold">Status</th>
                  <th className="py-2.5 px-3 font-semibold">Event Type</th>
                  <th className="py-2.5 px-3 font-semibold">Source IP</th>
                  <th className="py-2.5 px-3 font-semibold">Timestamp</th>
                  <th className="py-2.5 px-3 font-semibold text-right">Latency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-slate-800">
                {filteredLogs.map((log) => {
                  const isSelected = visibleLog?.id === log.id;
                  const isSuccess = log.status === 'SUCCESS';
                  return (
                    <tr
                      key={log.id}
                      onClick={() => setSelectedLog(log)}
                      className={`hover:bg-slate-50 cursor-pointer select-none transition-colors ${
                        isSelected ? 'bg-slate-100/80 font-bold' : ''
                      }`}
                    >
                      <td className="py-2.5 px-3">
                        <span
                          className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm border ${
                            isSuccess
                              ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                              : log.status === 'UNAUTHORIZED'
                                ? 'bg-amber-50 text-amber-700 border-amber-200'
                                : 'bg-red-50 text-red-700 border-red-200'
                          }`}
                        >
                          {log.statusCode} {log.status}
                        </span>
                      </td>
                      <td className="py-2.5 px-3 font-bold text-slate-900">{log.eventType}</td>
                      <td className="py-2.5 px-3 text-slate-600">{log.sourceIp}</td>
                      <td className="py-2.5 px-3 text-slate-500 text-[11px]">
                        {log.timestamp.split(' ')[1]}
                      </td>
                      <td className="py-2.5 px-3 text-right text-slate-700 font-semibold">
                        {log.latencyMs}ms
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right Column: JSON Payload Inspector Drawer */}
        <div className="lg:col-span-5 bg-white border border-slate-200 rounded-sm overflow-hidden flex flex-col justify-between">
          <div>
            {/* Header */}
            <div className="bg-slate-100 border-b border-slate-200 p-3 flex items-center justify-between">
              <div className="font-mono text-xs font-bold text-slate-900 flex items-center gap-2">
                <Code2 className="w-4 h-4 text-slate-800" />
                <span>JSON Payload Inspector</span>
              </div>

              {visibleLog && (
                <button
                  onClick={handleReplayWebhook}
                  disabled={replaying}
                  className="bg-black hover:bg-slate-800 text-white font-mono text-[11px] px-2.5 py-1 rounded-sm border border-black flex items-center gap-1 transition-colors cursor-pointer"
                >
                  <RefreshCw className={`w-3 h-3 ${replaying ? 'animate-spin' : ''}`} />
                  <span>{replaying ? 'Replaying...' : 'Replay Delivery'}</span>
                </button>
              )}
            </div>

            {replaySuccess && (
              <div className="bg-emerald-50 border-b border-emerald-200 p-2 text-xs font-mono text-emerald-800 flex items-center gap-1.5">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
                <span>Webhook payload successfully replayed! Status: 200 OK</span>
              </div>
            )}

            {visibleLog ? (
              <div className="p-4 space-y-4 font-mono text-xs">
                {/* HTTP Headers */}
                <div className="space-y-1.5">
                  <div className="text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                    HTTP Request Headers:
                  </div>
                  <div className="bg-slate-50 p-2.5 border border-slate-200 rounded-sm space-y-1 text-[11px] text-slate-700">
                    {Object.entries(visibleLog.headers).map(([k, v]) => (
                      <div
                        key={k}
                        className="flex justify-between border-b border-slate-100 pb-0.5 last:border-0"
                      >
                        <span className="font-bold text-slate-900">{k}:</span>
                        <span className="text-slate-600 truncate ml-2">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* HTTP Request Body */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <span className="text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                      Request Body (JSON):
                    </span>
                    <button
                      onClick={handleCopyPayload}
                      className="text-slate-600 hover:text-black text-[11px] flex items-center gap-1 font-mono"
                    >
                      {copiedPayload ? (
                        <Check className="w-3 h-3 text-emerald-600" />
                      ) : (
                        <Copy className="w-3 h-3" />
                      )}
                      <span>{copiedPayload ? 'Copied' : 'Copy Body'}</span>
                    </button>
                  </div>
                  <pre className="bg-slate-900 text-slate-100 p-3 rounded-sm text-[11px] overflow-x-auto border border-slate-800 leading-relaxed font-mono">
                    {visibleLog.requestBody}
                  </pre>
                </div>

                {/* Response Body */}
                <div className="space-y-1.5">
                  <div className="text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                    Ingress Response ({visibleLog.statusCode}):
                  </div>
                  <pre className="bg-slate-100 text-slate-800 p-2.5 rounded-sm text-[11px] overflow-x-auto border border-slate-200 leading-relaxed font-mono">
                    {visibleLog.responseBody}
                  </pre>
                </div>
              </div>
            ) : (
              <div className="p-8 text-center font-mono text-xs text-slate-500">
                Select a webhook log from the left table to inspect request headers and JSON
                payloads.
              </div>
            )}
          </div>

          <div className="p-3 bg-slate-50 border-t border-slate-200 text-xs font-mono text-slate-600 flex justify-between">
            <span>HMAC SHA-256 Verified</span>
            <span>TLS 1.3 Ingress</span>
          </div>
        </div>
      </div>
    </div>
  );
};
