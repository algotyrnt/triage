/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect } from 'react';
import { Incident, ScreenId, GEMINI_MODEL_NAME } from '@/types';
import { GithubIcon as Github } from '@/components/GithubIcon';
import {
  AlertTriangle,
  CheckCircle2,
  GitCommit,
  Clock,
  Code2,
  Sparkles,
  Copy,
  Check,
  RefreshCw,
  FileCode,
  ArrowLeft,
  ChevronRight,
  ExternalLink,
} from 'lucide-react';

interface IncidentDetailPageProps {
  incident: Incident;
  allIncidents: Incident[];
  onSelectIncident: (id: string) => void;
  onNavigate: (screen: ScreenId) => void;
}

export const IncidentDetailPage: React.FC<IncidentDetailPageProps> = ({
  incident,
  allIncidents,
  onSelectIncident,
  onNavigate,
}) => {
  const [analyzing, setAnalyzing] = useState(false);
  const [aiAnalysis, setAiAnalysis] = useState(incident.geminiAnalysis);
  const [patchCode, setPatchCode] = useState<string | null>(incident.suggestedPatch || null);
  const [generatingPatch, setGeneratingPatch] = useState(false);
  const [copiedPatch, setCopiedPatch] = useState(false);
  const [copiedStack, setCopiedStack] = useState(false);
  const [analysisError, setAnalysisError] = useState<string | null>(null);

  // Reset analysis & patch state when selected incident changes
  useEffect(() => {
    setAiAnalysis(incident.geminiAnalysis);
    setPatchCode(incident.suggestedPatch || null);
    setAnalysisError(null);
  }, [incident.id, incident.geminiAnalysis, incident.suggestedPatch]);

  // Trigger Gemini AI Root Cause Analysis
  const handleRunAiAnalysis = async () => {
    setAnalyzing(true);
    setAnalysisError(null);
    try {
      const res = await fetch('/api/gemini/analyze-panic', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          panicMessage: incident.panicMessage,
          rawStackTrace: incident.rawStackTrace,
          triggeringFile: incident.triggeringFile,
          astCode: incident.astSnippet.lines.map((l) => l.content).join('\n'),
        }),
      });
      if (!res.ok) {
        throw new Error(`AI Analysis API returned status ${res.status}`);
      }
      const data = await res.json();
      setAiAnalysis({
        rootCause: data.rootCause,
        explanation: data.explanation,
        severity: data.severity,
        recommendedFix: data.recommendedFix,
      });
    } catch (e) {
      console.error(e);
      setAnalysisError(e instanceof Error ? e.message : 'Failed to run AI analysis');
    } finally {
      setAnalyzing(false);
    }
  };

  // Trigger Gemini AI Fix Patch Generator
  const handleGenerateFixPatch = async () => {
    setGeneratingPatch(true);
    try {
      const res = await fetch('/api/gemini/generate-fix-pr', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          triggeringFile: incident.triggeringFile,
          panicMessage: incident.panicMessage,
          astCode: incident.astSnippet.lines.map((l) => l.content).join('\n'),
        }),
      });
      if (!res.ok) {
        throw new Error(`Fix patch generation returned status ${res.status}`);
      }
      const data = await res.json();
      if (data.patch) {
        setPatchCode(data.patch);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setGeneratingPatch(false);
    }
  };

  const handleCopyPatch = () => {
    if (patchCode) {
      navigator.clipboard.writeText(patchCode);
      setCopiedPatch(true);
      setTimeout(() => setCopiedPatch(false), 2000);
    }
  };

  const handleCopyStack = () => {
    navigator.clipboard.writeText(incident.rawStackTrace);
    setCopiedStack(true);
    setTimeout(() => setCopiedStack(false), 2000);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Top Header & Breadcrumb */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div className="space-y-1">
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500">
            <button
              onClick={() => onNavigate('dashboard')}
              className="hover:text-black flex items-center gap-1"
            >
              <ArrowLeft className="w-3 h-3" />
              <span>Incidents</span>
            </button>
            <span>/</span>
            <span className="font-bold text-slate-900">{incident.id}</span>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-lg font-bold text-slate-900 tracking-tight font-sans">
              {incident.title}
            </h1>
            <span
              className={`text-xs font-mono font-bold px-2 py-0.5 rounded-sm border ${
                incident.status === 'CRITICAL'
                  ? 'bg-red-50 text-red-700 border-red-200'
                  : incident.status === 'INVESTIGATING'
                    ? 'bg-amber-50 text-amber-700 border-amber-200'
                    : 'bg-emerald-50 text-emerald-700 border-emerald-200'
              }`}
            >
              {incident.status}
            </span>
            {incident.githubIssueNumber && (
              <a
                href={incident.githubIssueUrl || '#'}
                target="_blank"
                rel="noreferrer"
                className="bg-emerald-50 text-emerald-800 hover:bg-emerald-100 border border-emerald-200 text-xs font-mono px-2 py-0.5 rounded-sm flex items-center gap-1 font-medium transition-colors"
              >
                <Github className="w-3 h-3 text-emerald-700" />
                <span>GitHub Issue #{incident.githubIssueNumber} Created</span>
                <ExternalLink className="w-2.5 h-2.5 ml-0.5 text-emerald-600" />
              </a>
            )}
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2 font-mono text-xs">
          <button
            onClick={handleRunAiAnalysis}
            disabled={analyzing}
            className="bg-black hover:bg-slate-800 text-white font-bold px-3 py-1.5 rounded-sm flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            <Sparkles className={`w-3.5 h-3.5 ${analyzing ? 'animate-spin' : 'text-amber-400'}`} />
            <span>
              {analyzing ? 'Running Analysis...' : `Re-analyze with ${GEMINI_MODEL_NAME}`}
            </span>
          </button>
        </div>
      </div>

      {analysisError && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 font-mono text-xs rounded-sm">
          {analysisError}
        </div>
      )}

      {/* Main Grid: Left Column Details (65%) vs Right Column Incident Selector (35%) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column (65% -> col-span-8) */}
        <div className="lg:col-span-8 space-y-6">
          {/* Section 1: Gemini AI Root Cause Analysis */}
          {aiAnalysis && (
            <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3 font-mono text-xs">
              <div className="flex items-center justify-between border-b border-slate-100 pb-2">
                <div className="flex items-center gap-2">
                  <Sparkles className="w-4 h-4 text-amber-500" />
                  <span className="font-bold text-slate-900 text-sm">
                    Gemini AI Structured Root Cause Analysis ({GEMINI_MODEL_NAME})
                  </span>
                </div>
                <span className="text-[10px] bg-black text-white px-2 py-0.5 rounded-sm font-bold">
                  SEVERITY: {aiAnalysis.severity}
                </span>
              </div>

              <div className="space-y-2">
                <div>
                  <span className="text-slate-500 font-semibold block text-[11px]">
                    Primary Root Cause:
                  </span>
                  <p className="font-bold text-slate-900 text-xs mt-0.5">{aiAnalysis.rootCause}</p>
                </div>

                <div>
                  <span className="text-slate-500 font-semibold block text-[11px]">
                    Detailed Diagnostic Explanation:
                  </span>
                  <p className="text-slate-700 leading-relaxed mt-0.5 text-[11.5px] bg-slate-50 p-2.5 rounded-sm border border-slate-200">
                    {aiAnalysis.explanation}
                  </p>
                </div>

                <div>
                  <span className="text-slate-500 font-semibold block text-[11px]">
                    Recommended Engineering Fix:
                  </span>
                  <p className="text-emerald-900 font-medium mt-0.5 text-[11.5px] bg-emerald-50/60 p-2.5 rounded-sm border border-emerald-200">
                    {aiAnalysis.recommendedFix}
                  </p>
                </div>
              </div>

              <div className="pt-2 border-t border-slate-100 flex items-center justify-between">
                <button
                  onClick={handleGenerateFixPatch}
                  disabled={generatingPatch}
                  className="bg-slate-900 hover:bg-black text-white px-3 py-1.5 rounded-sm text-xs font-bold flex items-center gap-1.5 transition-colors cursor-pointer"
                >
                  <Code2 className="w-3.5 h-3.5 text-emerald-400" />
                  <span>
                    {generatingPatch ? 'Generating Fix Patch...' : 'Generate Code Fix Patch Diff'}
                  </span>
                </button>
              </div>
            </div>
          )}

          {/* Section 2: Code Fix Patch Diff */}
          {patchCode && (
            <div className="bg-slate-950 text-slate-100 border border-slate-800 rounded-sm overflow-hidden font-mono text-xs space-y-0 shadow-sm">
              <div className="bg-slate-900 border-b border-slate-800 p-3 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Code2 className="w-4 h-4 text-emerald-400" />
                  <span className="font-bold text-white">
                    Suggested Code Fix Diff (Git Unified Format)
                  </span>
                </div>

                <button
                  onClick={handleCopyPatch}
                  className="bg-slate-800 hover:bg-slate-700 text-slate-200 px-2.5 py-1 rounded-sm text-[11px] border border-slate-700 flex items-center gap-1 font-mono transition-colors"
                >
                  {copiedPatch ? (
                    <Check className="w-3 h-3 text-emerald-400" />
                  ) : (
                    <Copy className="w-3 h-3" />
                  )}
                  <span>{copiedPatch ? 'Copied Patch!' : 'Copy Patch'}</span>
                </button>
              </div>

              <div className="p-4 bg-slate-950 text-slate-200 text-[11px] leading-relaxed overflow-x-auto">
                <pre>{patchCode}</pre>
              </div>
            </div>
          )}

          {/* Section 3: Isolated AST Code Node Viewer */}
          <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3 font-mono text-xs">
            <div className="flex items-center justify-between border-b border-slate-100 pb-2">
              <div className="flex items-center gap-2">
                <FileCode className="w-4 h-4 text-slate-800" />
                <span className="font-bold text-slate-900 text-sm">
                  Isolated Function AST Node (
                  <code className="text-black">{incident.astSnippet.functionName}</code>)
                </span>
              </div>
              <span className="text-[11px] text-slate-500 font-mono">
                Line {incident.astSnippet.startLine} • {incident.astSnippet.lines.length} Lines
                Isolated
              </span>
            </div>

            <div className="bg-slate-900 text-slate-100 p-3 rounded-sm text-[11.5px] font-mono overflow-x-auto border border-slate-800 leading-relaxed">
              <div className="space-y-0.5">
                {incident.astSnippet.lines.map((l) => (
                  <div
                    key={l.lineNum}
                    className={`flex items-center gap-3 px-2 py-0.5 rounded-sm ${
                      l.isTriggerLine
                        ? 'bg-red-950/80 text-red-200 border-l-2 border-red-500 font-bold'
                        : ''
                    }`}
                  >
                    <span className="text-slate-500 w-8 select-none text-right shrink-0">
                      {l.lineNum}
                    </span>
                    <span className="whitespace-pre">{l.content}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Section 4: Raw Runtime Stack Trace */}
          <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3 font-mono text-xs">
            <div className="flex items-center justify-between border-b border-slate-100 pb-2">
              <div className="flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-red-600" />
                <span className="font-bold text-slate-900 text-sm">Raw Go Runtime Stack Trace</span>
              </div>

              <button
                onClick={handleCopyStack}
                className="bg-slate-100 hover:bg-slate-200 text-slate-700 px-2.5 py-1 rounded-sm text-[11px] border border-slate-200 flex items-center gap-1 font-mono transition-colors"
              >
                {copiedStack ? (
                  <Check className="w-3 h-3 text-emerald-600" />
                ) : (
                  <Copy className="w-3 h-3" />
                )}
                <span>{copiedStack ? 'Copied Trace!' : 'Copy Raw Trace'}</span>
              </button>
            </div>

            <div className="bg-slate-950 text-slate-300 p-3 rounded-sm text-[11px] font-mono overflow-x-auto border border-slate-800 leading-relaxed">
              <pre>{incident.rawStackTrace}</pre>
            </div>
          </div>
        </div>

        {/* Right Column (35% -> col-span-4) Incident Selector & Metadata */}
        <div className="lg:col-span-4 space-y-6 font-mono text-xs">
          {/* Metadata Card */}
          <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3">
            <div className="font-bold text-slate-900 border-b border-slate-100 pb-2 flex items-center justify-between">
              <span>Incident Metadata</span>
              <span className="text-[11px] text-slate-500 font-normal">Repo Scope</span>
            </div>

            <div className="space-y-2 text-slate-700">
              <div className="flex justify-between">
                <span className="text-slate-500">Incident ID:</span>
                <strong className="text-slate-900">{incident.id}</strong>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Triggering File:</span>
                <span className="text-slate-900 font-semibold truncate max-w-[180px]">
                  {incident.triggeringFile}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Commit Hash:</span>
                <span className="text-slate-900 flex items-center gap-1 font-bold">
                  <GitCommit className="w-3 h-3 text-slate-500" />
                  {incident.commitHash}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Branch:</span>
                <span className="text-slate-800">{incident.branch}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Symbolication Latency:</span>
                <span className="text-emerald-700 font-bold">{incident.latencyMs}ms</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Timestamp:</span>
                <span className="text-slate-600 text-[11px]">{incident.timestamp}</span>
              </div>
            </div>
          </div>

          {/* Related Incidents Navigator */}
          <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3">
            <div className="font-bold text-slate-900 border-b border-slate-100 pb-2">
              All Ingested Repository Crashes
            </div>

            <div className="space-y-1.5">
              {allIncidents.map((inc) => {
                const isSelected = inc.id === incident.id;
                return (
                  <button
                    key={inc.id}
                    onClick={() => onSelectIncident(inc.id)}
                    className={`w-full text-left p-2.5 rounded-sm border transition-all ${
                      isSelected
                        ? 'border-black bg-slate-900 text-white font-bold'
                        : 'border-slate-200 hover:border-slate-300 bg-white text-slate-800'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-bold text-xs">{inc.id}</span>
                      <span
                        className={`text-[9px] px-1.5 py-0.5 font-bold rounded-sm border ${
                          inc.status === 'CRITICAL'
                            ? 'bg-red-50 text-red-700 border-red-200'
                            : inc.status === 'INVESTIGATING'
                              ? 'bg-amber-50 text-amber-700 border-amber-200'
                              : 'bg-emerald-50 text-emerald-700 border-emerald-200'
                        }`}
                      >
                        {inc.status}
                      </span>
                    </div>
                    <div className="text-[11px] truncate">{inc.title}</div>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
