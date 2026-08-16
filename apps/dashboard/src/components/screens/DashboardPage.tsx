/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect } from 'react';
import { Incident, ScreenId } from '@/types';
import { engineClient } from '@/services/engineClient';
import {
  Key,
  GitBranch,
  Code2,
  Activity,
  Copy,
  Check,
  AlertTriangle,
  ArrowRight,
  ExternalLink,
  Zap,
  Terminal,
  RefreshCw,
} from 'lucide-react';

interface DashboardPageProps {
  incidents: Incident[];
  onSelectIncident: (id: string) => void;
  onNavigate: (screen: ScreenId) => void;
  activeRepo?: string;
  rootDir?: string;
  apiKey?: string;
}

export const DashboardPage: React.FC<DashboardPageProps> = ({
  incidents,
  onSelectIncident,
  onNavigate,
  activeRepo = 'algotyrnt/triage',
  rootDir = '',
  apiKey = 'tr_live_demo_key_9042',
}) => {
  const [activeCodeTab, setActiveCodeTab] = useState<'main.go' | 'middleware.go' | 'go.mod'>(
    'main.go',
  );
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedSnippet, setCopiedSnippet] = useState(false);
  const [stats, setStats] = useState<{
    total_incidents: number;
    funcs_indexed: number;
    total_projects: number;
  } | null>(null);

  useEffect(() => {
    engineClient.getStats().then((data) => {
      if (data) setStats(data);
    });
  }, []);

  const codeSnippets = {
    'main.go': `package main

import (
	"log"
	"net/http"
	"os"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()

	// Initialize Triage Panic Symbolication Engine
	telemetryURL := os.Getenv("TRIAGE_ENGINE_URL")
	handler := triage.Middleware("${apiKey}", telemetryURL)(mux)

	log.Println("[INFO] Server listening on :8081...")
	http.ListenAndServe(":8081", handler)
}`,
    'middleware.go': `package middleware

import (
	"net/http"
	"os"
	triage "github.com/algotyrnt/triage/sdk/go"
)

// RecoveryMiddleware wraps HTTP handlers to isolate panics at route boundaries
func RecoveryMiddleware(next http.Handler) http.Handler {
	telemetryURL := os.Getenv("TRIAGE_ENGINE_URL")
	return triage.Middleware("${apiKey}", telemetryURL)(next)
}`,
    'go.mod': `module ${activeRepo}

go 1.22

require (
	github.com/algotyrnt/triage/sdk/go v1.0.0
)`,
  };

  const handleCopyKey = () => {
    navigator.clipboard.writeText(apiKey);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  const handleCopySnippet = () => {
    navigator.clipboard.writeText(codeSnippets[activeCodeTab]);
    setCopiedSnippet(true);
    setTimeout(() => setCopiedSnippet(false), 2000);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Breadcrumb Header + Status */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500">
            <span>Projects</span>
            <span>/</span>
            <span className="font-bold text-slate-900">{activeRepo}</span>
            {rootDir && (
              <>
                <span>/</span>
                <span className="text-indigo-600 font-semibold">{rootDir}</span>
              </>
            )}
          </div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans mt-1">
            System Overview & Ingestion Dashboard
          </h1>
        </div>

        <div className="flex items-center gap-2">
          <div className="bg-emerald-50 text-emerald-700 border border-emerald-200 text-xs font-mono px-2.5 py-1 rounded-sm flex items-center gap-2 font-medium">
            <span className="w-2 h-2 rounded-full bg-emerald-600 animate-pulse"></span>
            <span>Engine Operational</span>
          </div>
          <button
            onClick={() => onNavigate('status')}
            className="text-xs font-mono bg-slate-100 hover:bg-slate-200 text-slate-800 border border-slate-200 px-2.5 py-1 rounded-sm transition-colors"
          >
            Metrics
          </button>
        </div>
      </div>

      {/* 4-Column Metric Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Metric 1: API Key */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <Key className="w-3.5 h-3.5 text-slate-700" />
              <span>Ingestion Key</span>
            </span>
            <button
              onClick={handleCopyKey}
              className="text-slate-600 hover:text-black font-mono text-[11px] underline flex items-center gap-0.5"
            >
              {copiedKey ? (
                <Check className="w-3 h-3 text-emerald-600" />
              ) : (
                <Copy className="w-3 h-3" />
              )}
              <span>{copiedKey ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900 truncate">{apiKey}</div>
          <div className="text-[11px] font-mono text-slate-500 flex items-center justify-between">
            <span>Env: Production</span>
            <span className="text-emerald-600 font-semibold">Active</span>
          </div>
        </div>

        {/* Metric 2: Monitored Repo */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <GitBranch className="w-3.5 h-3.5 text-slate-700" />
              <span>Monitored Repo</span>
            </span>
            <span className="text-[10px] bg-slate-100 border border-slate-200 px-1.5 py-0.2 rounded-sm text-slate-700 font-mono">
              {rootDir ? 'Monorepo' : 'Root AST'}
            </span>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900 truncate">
            {activeRepo}
          </div>
          <div className="text-[11px] font-mono text-slate-500 flex items-center justify-between">
            <span>{rootDir ? `Subdir: ${rootDir}/` : 'Root: /'}</span>
            <span className="text-slate-800 font-mono">Go 1.22+</span>
          </div>
        </div>

        {/* Metric 3: AST Index Status */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <Code2 className="w-3.5 h-3.5 text-slate-700" />
              <span>AST Index Status</span>
            </span>
            <button
              onClick={() => onNavigate('ast')}
              className="text-slate-600 hover:text-black text-[11px] underline"
            >
              Tree
            </button>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900">
            {stats ? `${stats.funcs_indexed.toLocaleString()} Funcs Indexed` : '— Loading...'}
          </div>
          <div className="text-[11px] font-mono text-emerald-600 flex items-center gap-1">
            <Check className="w-3 h-3" />
            <span>100% Up-to-Date (Commit 8f3a1b4)</span>
          </div>
        </div>

        {/* Metric 4: Total Dispatches */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <Activity className="w-3.5 h-3.5 text-slate-700" />
              <span>Total Dispatches</span>
            </span>
            <span className="text-[10px] bg-red-50 text-red-700 border border-red-200 px-1.5 py-0.2 rounded-sm font-mono font-bold">
              {incidents.filter((i) => i.status === 'CRITICAL').length} Critical
            </span>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900">
            {stats ? `${stats.total_incidents.toLocaleString()} Telemetry Events` : '— Loading...'}
          </div>
          <div className="text-[11px] font-mono text-slate-500">
            Avg Latency: <span className="text-slate-800 font-semibold">14ms</span>
          </div>
        </div>
      </div>

      {/* 2-Column Split View: Left 60% Code Snippet, Right 40% Live Dispatch Table */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column (60% -> col-span-7) */}
        <div className="lg:col-span-7 bg-white border border-slate-200 rounded-sm overflow-hidden space-y-0">
          {/* Header & Tabs */}
          <div className="bg-slate-100 border-b border-slate-200 p-3 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Terminal className="w-4 h-4 text-slate-700" />
              <span className="text-xs font-mono font-bold text-slate-900">
                Go SDK Integration & AST Recovery Hook
              </span>
            </div>

            {/* Tab Switcher */}
            <div className="flex items-center gap-1 bg-white border border-slate-200 rounded-sm p-0.5">
              {(['main.go', 'middleware.go', 'go.mod'] as const).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveCodeTab(tab)}
                  className={`px-2 py-0.5 text-[11px] font-mono rounded-sm transition-colors ${
                    activeCodeTab === tab
                      ? 'bg-black text-white font-bold'
                      : 'text-slate-600 hover:text-black'
                  }`}
                >
                  {tab}
                </button>
              ))}
            </div>
          </div>

          {/* Code Viewer in Charcoal (#111827) */}
          <div className="bg-slate-900 text-slate-100 p-4 font-mono text-xs relative group border-b border-slate-800 min-h-[300px]">
            <button
              onClick={handleCopySnippet}
              className="absolute top-3 right-3 bg-slate-800 hover:bg-slate-700 text-slate-200 hover:text-white px-2.5 py-1 rounded-sm text-[11px] border border-slate-700 flex items-center gap-1 transition-colors font-mono"
            >
              {copiedSnippet ? (
                <Check className="w-3 h-3 text-emerald-400" />
              ) : (
                <Copy className="w-3 h-3" />
              )}
              <span>{copiedSnippet ? 'Copied' : 'Copy'}</span>
            </button>

            <pre className="overflow-x-auto leading-relaxed text-[11.5px] text-slate-200">
              {codeSnippets[activeCodeTab]}
            </pre>
          </div>

          {/* Integration Notes */}
          <div className="p-3 bg-slate-50 text-xs font-mono text-slate-600 space-y-1">
            <div className="font-bold text-slate-900 flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5 text-slate-700" />
              <span>How `defer triage.Recovery()` works under the hood:</span>
            </div>
            <p className="text-[11px] text-slate-600 leading-relaxed">
              When a Go goroutine panics, `Recovery()` inspects `runtime/debug.Stack()`, extracts
              the exact source file and byte line offset (e.g. `user.go:42`), and queries Triage AST
              repository index to isolate the crashing `FuncDecl` block in under 15ms.
            </p>
          </div>
        </div>

        {/* Right Column (40% -> col-span-5) */}
        <div className="lg:col-span-5 bg-white border border-slate-200 rounded-sm flex flex-col justify-between">
          <div className="space-y-0">
            {/* Header */}
            <div className="p-3 bg-slate-100 border-b border-slate-200 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-red-600" />
                <span className="text-xs font-mono font-bold text-slate-900">
                  Live Dispatch Event Log
                </span>
              </div>
            </div>

            {/* Event List */}
            <div className="divide-y divide-slate-100">
              {incidents.length === 0 ? (
                <div className="p-8 text-center space-y-3 font-mono">
                  <div className="w-10 h-10 mx-auto rounded-full bg-slate-100 flex items-center justify-center border border-slate-200">
                    <Terminal className="w-5 h-5 text-slate-400" />
                  </div>
                  <div className="space-y-1">
                    <h3 className="text-xs font-bold text-slate-800">
                      No Crash Incidents Recorded
                    </h3>
                    <p className="text-[11px] text-slate-500 max-w-xs mx-auto leading-normal">
                      Integrate the Go SDK into your app to ingest live crash telemetry.
                    </p>
                  </div>
                </div>
              ) : (
                incidents.map((incident) => {
                  const isCritical = incident.status === 'CRITICAL';
                  return (
                    <div
                      key={incident.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => onSelectIncident(incident.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          onSelectIncident(incident.id);
                        }
                      }}
                      className="p-3 hover:bg-slate-50 transition-colors cursor-pointer space-y-1.5 group focus:outline-none focus:ring-1 focus:ring-black"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2 font-mono text-xs">
                          <span className="font-bold text-slate-900 group-hover:underline">
                            {incident.id}
                          </span>
                          <span
                            className={`text-[10px] font-bold px-1.5 py-0.2 rounded-sm border ${
                              isCritical
                                ? 'bg-red-50 text-red-700 border-red-200'
                                : incident.status === 'INVESTIGATING'
                                  ? 'bg-amber-50 text-amber-700 border-amber-200'
                                  : 'bg-emerald-50 text-emerald-700 border-emerald-200'
                            }`}
                          >
                            {incident.status}
                          </span>
                        </div>
                        <span className="text-[11px] font-mono text-slate-500">
                          {incident.latencyMs}ms
                        </span>
                      </div>

                      <div className="text-xs font-mono text-slate-800 line-clamp-1 font-medium">
                        {incident.title}
                      </div>

                      <div className="flex items-center justify-between text-[11px] font-mono text-slate-500">
                        <span className="text-slate-600">{incident.triggeringFile}</span>
                        <span>{incident.timestamp.split(' ')[1]}</span>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          {/* Footer Action */}
          <div className="p-3 bg-slate-50 border-t border-slate-200 text-center">
            <button
              onClick={() => onNavigate('incident_detail')}
              className="w-full bg-slate-900 hover:bg-black text-white font-mono text-xs py-2 px-3 rounded-sm transition-colors flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <span>Inspect All Incidents in AST Inspector</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
