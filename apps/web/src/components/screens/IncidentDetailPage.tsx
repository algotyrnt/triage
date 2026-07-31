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
import { Incident, ScreenId } from '../../types';
import { GithubIcon as Github } from '../GithubIcon';
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

  // Trigger Gemini AI Root Cause Analysis
  const handleRunAiAnalysis = async () => {
    setAnalyzing(true);
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
      const data = await res.json();
      setAiAnalysis({
        rootCause: data.rootCause || 'Nil Pointer Dereference in Receiver Struct',
        explanation: data.explanation || 'Uninitialized receiver struct dereference.',
        severity: data.severity || 'CRITICAL',
        recommendedFix: data.recommendedFix || 'Add defensive nil check before struct field access.',
      });
    } catch (e) {
      console.error(e);
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
            <span className="bg-red-50 text-red-700 border border-red-200 text-xs font-mono font-bold px-2 py-0.5 rounded-sm">
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

        {/* Incident Switcher Dropdown */}
        <div className="flex items-center gap-2 font-mono text-xs">
          <span className="text-slate-500 hidden sm:inline">Select Incident:</span>
          <select
            value={incident.id}
            onChange={(e) => onSelectIncident(e.target.value)}
            className="bg-white border border-slate-200 rounded-sm px-2.5 py-1 text-xs font-mono focus:outline-none focus:border-black"
          >
            {allIncidents.map((inc) => (
              <option key={inc.id} value={inc.id}>
                {inc.id}: {inc.title.substring(0, 32)}...
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Top Metadata Grid */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3 bg-white border border-slate-200 p-3.5 rounded-sm font-mono text-xs">
        <div>
          <div className="text-[11px] text-slate-500">Triggering File</div>
          <div className="font-bold text-slate-900 truncate mt-0.5">{incident.triggeringFile}</div>
        </div>
        <div>
          <div className="text-[11px] text-slate-500">Goroutine ID</div>
          <div className="font-bold text-slate-800 mt-0.5">{incident.goroutineId}</div>
        </div>
        <div>
          <div className="text-[11px] text-slate-500">Symbolication Latency</div>
          <div className="font-bold text-slate-800 mt-0.5">{incident.latencyMs}ms</div>
        </div>
        <div>
          <div className="text-[11px] text-slate-500">Commit Hash</div>
          <div className="font-bold text-slate-800 mt-0.5 flex items-center gap-1">
            <GitCommit className="w-3 h-3 text-slate-500" />
            <span>{incident.commitHash}</span>
          </div>
        </div>
        <div>
          <div className="text-[11px] text-slate-500">Ingested At</div>
          <div className="font-bold text-slate-800 mt-0.5">{incident.timestamp.split(' ')[1]}</div>
        </div>
      </div>

      {/* Split View: Left Raw Stack Trace (50%) vs Right AST Snippet & Gemini Analysis (50%) */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Raw Stack Trace */}
        <div className="bg-white border border-slate-200 rounded-sm overflow-hidden flex flex-col justify-between">
          <div>
            <div className="bg-slate-100 border-b border-slate-200 p-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Code2 className="w-4 h-4 text-slate-800" />
                <span className="text-xs font-mono font-bold text-slate-900">
                  Raw `runtime/debug.Stack()` Trace
                </span>
              </div>
              <button
                onClick={handleCopyStack}
                className="text-slate-600 hover:text-black font-mono text-[11px] flex items-center gap-1"
              >
                {copiedStack ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                <span>{copiedStack ? 'Copied' : 'Copy Trace'}</span>
              </button>
            </div>

            {/* Panic banner */}
            <div className="bg-red-50 border-b border-red-200 p-3 text-xs font-mono text-red-900 space-y-1">
              <div className="font-bold flex items-center gap-1.5 text-red-700">
                <AlertTriangle className="w-3.5 h-3.5" />
                <span>Runtime Panic Message:</span>
              </div>
              <code className="block bg-white p-1.5 rounded-sm border border-red-200 text-red-900 font-bold select-all">
                {incident.panicMessage}
              </code>
            </div>

            {/* Code Block in Charcoal (#111827) */}
            <div className="bg-slate-900 text-slate-100 p-4 font-mono text-xs overflow-x-auto min-h-[320px]">
              <pre className="leading-relaxed text-[11.5px] text-slate-200 whitespace-pre-wrap">
                {incident.rawStackTrace}
              </pre>
            </div>
          </div>

          <div className="p-3 bg-slate-50 border-t border-slate-200 text-xs font-mono text-slate-600">
            Symbolicated via Go ELF DWARF debug section in 14ms.
          </div>
        </div>

        {/* Right Column: Isolated FuncDecl AST Snippet (Line 42 Highlighted in #FEF2F2) + Gemini Analysis */}
        <div className="space-y-6">
          {/* Isolated AST Code Block */}
          <div className="bg-white border border-slate-200 rounded-sm overflow-hidden">
            <div className="bg-slate-100 border-b border-slate-200 p-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <FileCode className="w-4 h-4 text-slate-800" />
                <span className="text-xs font-mono font-bold text-slate-900">
                  Isolated `FuncDecl` AST Snippet: <span className="text-slate-600">{incident.astSnippet.functionName}()</span>
                </span>
              </div>
              <span className="text-[11px] font-mono text-slate-500 bg-white border border-slate-200 px-2 py-0.5 rounded-sm">
                Line {incident.triggeringLine} Highlighted
              </span>
            </div>

            {/* AST Code Viewer with Red Line Highlight (#FEF2F2) */}
            <div className="bg-white font-mono text-xs divide-y divide-slate-100 overflow-x-auto">
              {incident.astSnippet.lines.map((line) => {
                const isHighlighted = line.isTriggerLine || line.lineNum === incident.triggeringLine;
                return (
                  <div
                    key={line.lineNum}
                    className={`flex items-center px-3 py-1 font-mono transition-colors ${
                      isHighlighted
                        ? 'bg-red-50 text-red-900 border-l-2 border-red-600 font-semibold'
                        : 'text-slate-800 hover:bg-slate-50'
                    }`}
                  >
                    <span
                      className={`w-10 text-right pr-3 select-none text-[11px] ${
                        isHighlighted ? 'text-red-600 font-bold' : 'text-slate-400'
                      }`}
                    >
                      {line.lineNum}
                    </span>
                    <pre className="text-[11.5px] font-mono whitespace-pre flex-1">{line.content}</pre>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Gemini AI Root-Cause Diagnostic Box */}
          <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3">
            <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
              <div className="flex items-center gap-2 font-mono text-xs font-bold text-slate-900">
                <Sparkles className="w-4 h-4 text-slate-900" />
                <span>Gemini 3.6 Diagnostic Root-Cause Analysis</span>
              </div>

              <button
                onClick={handleRunAiAnalysis}
                disabled={analyzing}
                className="bg-slate-100 hover:bg-slate-200 text-slate-800 font-mono text-[11px] px-2.5 py-1 rounded-sm border border-slate-200 flex items-center gap-1 transition-colors"
              >
                <RefreshCw className={`w-3 h-3 ${analyzing ? 'animate-spin' : ''}`} />
                <span>{analyzing ? 'Analyzing...' : 'Re-run Diagnostic'}</span>
              </button>
            </div>

            {aiAnalysis ? (
              <div className="space-y-2.5 font-mono text-xs">
                <div>
                  <span className="text-[11px] text-slate-500 uppercase tracking-wider block">Root Cause Title:</span>
                  <span className="font-bold text-slate-900 text-xs block mt-0.5">{aiAnalysis.rootCause}</span>
                </div>

                <div className="bg-slate-50 p-3 rounded-sm border border-slate-200 text-slate-700 leading-relaxed text-[11.5px]">
                  {aiAnalysis.explanation}
                </div>

                <div>
                  <span className="text-[11px] text-slate-500 uppercase tracking-wider block">Recommended Fix:</span>
                  <span className="text-slate-800 font-medium text-[11.5px] block mt-0.5">{aiAnalysis.recommendedFix}</span>
                </div>
              </div>
            ) : (
              <div className="text-xs font-mono text-slate-500 py-2">
                Click "Re-run Diagnostic" to perform live Gemini AST analysis.
              </div>
            )}
          </div>

          {/* Automated Fix PR Patch Generator */}
          <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-3">
            <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
              <div className="flex items-center gap-2 font-mono text-xs font-bold text-slate-900">
                <Github className="w-4 h-4 text-slate-900" />
                <span>Automated Fix Pull Request Generator</span>
              </div>

              <button
                onClick={handleGenerateFixPatch}
                disabled={generatingPatch}
                className="bg-black hover:bg-slate-800 text-white font-mono text-xs px-3 py-1.5 rounded-sm flex items-center gap-1.5 transition-colors cursor-pointer"
              >
                <Sparkles className={`w-3.5 h-3.5 ${generatingPatch ? 'animate-spin' : ''}`} />
                <span>{generatingPatch ? 'Generating Patch...' : 'Generate Go Patch'}</span>
              </button>
            </div>

            {patchCode && (
              <div className="space-y-2 font-mono">
                <div className="flex items-center justify-between text-xs text-slate-600">
                  <span className="font-bold text-slate-800">Generated Go AST Patch:</span>
                  <button
                    onClick={handleCopyPatch}
                    className="text-slate-600 hover:text-black text-[11px] flex items-center gap-1 font-mono"
                  >
                    {copiedPatch ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                    <span>{copiedPatch ? 'Copied' : 'Copy Code'}</span>
                  </button>
                </div>

                <pre className="bg-slate-900 text-emerald-400 p-3 rounded-sm text-[11px] overflow-x-auto border border-slate-800 leading-relaxed font-mono">
                  {patchCode}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
