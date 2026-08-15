/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import { Activity, Zap, TrendingDown, Clock, CheckCircle2, Shield } from 'lucide-react';

export const BenchmarksSection: React.FC = () => {
  return (
    <section id="benchmarks" className="py-20 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-20">
      {/* Header */}
      <div className="text-center max-w-3xl mx-auto space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <Activity className="w-3.5 h-3.5 text-emerald-600" />
          <span>PERFORMANCE &amp; COST BENCHMARKS</span>
        </div>
        <h2 className="text-3xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
          Engineered for Extreme Efficiency
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed">
          Triage isolates crashes with negligible server overhead while reducing LLM diagnostic token consumption by over 93%.
        </p>
      </div>

      {/* Benchmark Metric Cards */}
      <div className="mt-14 grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Metric 1: Request Latency Overhead */}
        <div className="bg-white border border-slate-200 p-6 rounded-xl shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <span className="font-mono text-xs text-slate-500 uppercase font-semibold">
              Latency Overhead
            </span>
            <span className="bg-emerald-50 text-emerald-700 font-mono text-[10px] font-bold px-2 py-0.5 rounded border border-emerald-200">
              47x Faster
            </span>
          </div>

          <div className="text-3xl font-extrabold text-slate-900 mb-1">0.018 ms</div>
          <p className="text-slate-500 text-xs mb-6">Average HTTP handler latency overhead per request</p>

          <div className="space-y-3 font-mono text-xs">
            <div>
              <div className="flex justify-between text-slate-700 mb-1">
                <span className="font-bold text-slate-900">triage SDK</span>
                <span className="text-emerald-600 font-bold">0.018 ms</span>
              </div>
              <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
                <div className="bg-emerald-500 h-full rounded-full" style={{ width: '4%' }} />
              </div>
            </div>

            <div>
              <div className="flex justify-between text-slate-500 mb-1">
                <span>Traditional APM A</span>
                <span>0.840 ms</span>
              </div>
              <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
                <div className="bg-slate-400 h-full rounded-full" style={{ width: '65%' }} />
              </div>
            </div>

            <div>
              <div className="flex justify-between text-slate-500 mb-1">
                <span>Traditional APM B</span>
                <span>1.220 ms</span>
              </div>
              <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
                <div className="bg-slate-300 h-full rounded-full" style={{ width: '95%' }} />
              </div>
            </div>
          </div>
        </div>

        {/* Metric 2: LLM Token Savings */}
        <div className="bg-white border border-slate-200 p-6 rounded-xl shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <span className="font-mono text-xs text-slate-500 uppercase font-semibold">
              LLM Token Efficiency
            </span>
            <span className="bg-indigo-50 text-indigo-700 font-mono text-[10px] font-bold px-2 py-0.5 rounded border border-indigo-200">
              93.6% Saved
            </span>
          </div>

          <div className="text-3xl font-extrabold text-slate-900 mb-1">180 Tokens</div>
          <p className="text-slate-500 text-xs mb-6">Tokens sent per incident with AST FuncDecl isolation</p>

          <div className="space-y-3 font-mono text-xs">
            <div>
              <div className="flex justify-between text-slate-700 mb-1">
                <span className="font-bold text-slate-900">Triage AST Node</span>
                <span className="text-indigo-600 font-bold">180 tokens</span>
              </div>
              <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
                <div className="bg-indigo-600 h-full rounded-full" style={{ width: '6.4%' }} />
              </div>
            </div>

            <div>
              <div className="flex justify-between text-slate-500 mb-1">
                <span>Full Source File Ingest</span>
                <span>2,800 tokens</span>
              </div>
              <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
                <div className="bg-slate-400 h-full rounded-full" style={{ width: '100%' }} />
              </div>
            </div>

            <div className="pt-2 text-[11px] text-slate-500 font-sans leading-relaxed">
              AST slicing cuts noise and prevents prompt saturation while reducing Gemini API costs to &lt; $0.0001 per incident.
            </div>
          </div>
        </div>

        {/* Metric 3: Memory Footprint */}
        <div className="bg-white border border-slate-200 p-6 rounded-xl shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <span className="font-mono text-xs text-slate-500 uppercase font-semibold">
              Client SDK Memory
            </span>
            <span className="bg-cyan-50 text-cyan-700 font-mono text-[10px] font-bold px-2 py-0.5 rounded border border-cyan-200">
              Zero Allocation
            </span>
          </div>

          <div className="text-3xl font-extrabold text-slate-900 mb-1">&lt; 1.2 MB</div>
          <p className="text-slate-500 text-xs mb-6">Peak resident set size for 10,000 requests/sec</p>

          <div className="space-y-2.5 text-xs text-slate-600">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
              <span>Bounded 1,000-job channel queue</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
              <span>No background daemon subprocesses</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
              <span>Garbage collection neutral payload recycling</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
              <span>Native Go net/http connection reuse</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
