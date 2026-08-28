/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import {
  Server,
  ArrowRight,
  Database,
  Cpu,
  Layers,
  Sparkles,
  GitPullRequest,
  CheckCircle,
  Clock,
  Shield,
  Zap,
} from 'lucide-react';

export const ArchitectureSection: React.FC = () => {
  const steps = [
    {
      num: '01',
      title: 'Panic Interception',
      desc: 'Standard Go defer + recover() hooks intercept HTTP panics. Captures debug.Stack() and immediately releases the client response.',
      badge: '< 0.02ms',
      icon: <Server className="w-5 h-5 text-indigo-600" />,
    },
    {
      num: '02',
      title: 'Async Bounded Queue',
      desc: 'Telemetry is dispatched to an asynchronous 4-worker pool backed by a 1,000-job buffer. Zero impact on server throughput.',
      badge: 'Non-blocking',
      icon: <Zap className="w-5 h-5 text-emerald-600" />,
    },
    {
      num: '03',
      title: 'Multi-File AST Slicing',
      desc: 'The engine parses the package AST on the fly to isolate the crash site alongside cross-file receiver structs, constructors, and package helpers.',
      badge: '< 14ms',
      icon: <Layers className="w-5 h-5 text-cyan-600" />,
    },
    {
      num: '04',
      title: 'AI Incident Diagnostics',
      desc: 'AST code snippet + sanitized panic metadata are sent to the configured AI engine (Gemini, OpenAI, Claude, Ollama) for deterministic root-cause analysis and patch generation.',
      badge: '94% Token Savings',
      icon: <Cpu className="w-5 h-5 text-purple-600" />,
    },
    {
      num: '05',
      title: 'GitHub Issues & PRs',
      desc: 'Persists to embedded SQLite (WAL mode), streams live to Studio Dashboard, and automatically files GitHub issues or opens bugfix Pull Requests with verified patches.',
      badge: 'Automated PRs',
      icon: <GitPullRequest className="w-5 h-5 text-amber-600" />,
    },
  ];

  return (
    <section
      id="architecture"
      className="py-14 sm:py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-16"
    >
      {/* Title */}
      <div className="text-center max-w-3xl mx-auto space-y-2.5 sm:space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <Layers className="w-3.5 h-3.5 text-indigo-600" />
          <span>ZERO-OVERHEAD ARCHITECTURE</span>
        </div>
        <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-slate-900 tracking-tight">
          How Triage Delivers Sub-Millisecond Panic Isolation
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed max-w-2xl mx-auto">
          Designed from the ground up for high-throughput production Go microservices without
          background database pre-indexing.
        </p>
      </div>

      {/* Architecture Flow Visual Grid */}
      <div className="mt-10 sm:mt-12 grid grid-cols-1 md:grid-cols-5 gap-3.5 sm:gap-4 relative">
        {steps.map((step, idx) => (
          <div
            key={step.num}
            className="bg-white border border-slate-200 p-5 rounded-lg shadow-xs hover:shadow-md transition-all flex flex-col justify-between relative group hover:border-slate-300"
          >
            <div>
              <div className="flex items-center justify-between mb-3">
                <span className="font-mono text-xs font-bold text-slate-400 group-hover:text-slate-900 transition-colors">
                  {step.num}
                </span>
                <span className="bg-slate-100 text-slate-700 font-mono text-[10px] font-semibold px-2 py-0.5 rounded border border-slate-200">
                  {step.badge}
                </span>
              </div>

              <div className="mb-3 p-2.5 rounded-md bg-slate-50 w-fit border border-slate-100">
                {step.icon}
              </div>

              <h3 className="font-bold text-slate-900 text-sm mb-2">{step.title}</h3>
              <p className="text-slate-600 text-xs leading-relaxed">{step.desc}</p>
            </div>

            {idx < steps.length - 1 && (
              <div className="hidden md:block absolute -right-3 top-1/2 -translate-y-1/2 z-10 text-slate-300">
                <ArrowRight className="w-4 h-4" />
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Deep-dive 3-tier caching breakdown */}
      <div className="mt-12 bg-slate-900 text-slate-100 border border-slate-800 rounded-xl p-6 sm:p-8 shadow-xl">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 items-center">
          <div className="space-y-3 lg:col-span-1">
            <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-indigo-950 text-indigo-300 border border-indigo-800 font-mono text-xs font-bold">
              <Database className="w-3.5 h-3.5" />
              <span>3-TIER AST CACHE</span>
            </div>
            <h3 className="text-xl font-bold tracking-tight text-white">
              Multi-Layered AST Resolution
            </h3>
            <p className="text-slate-400 text-xs sm:text-sm leading-relaxed">
              Triage avoids heavy full-codebase pre-indexing by resolving AST nodes on demand using
              a layered caching architecture.
            </p>
          </div>

          <div className="lg:col-span-2 grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="bg-slate-950 border border-slate-800 p-4 rounded-lg">
              <div className="font-mono text-emerald-400 text-xs font-bold mb-1">
                Tier 1: In-Memory Range
              </div>
              <div className="text-2xl font-extrabold text-white mb-1">&lt; 1.0 ms</div>
              <p className="text-slate-400 text-xs leading-relaxed">
                Boundary-aware cache storing full [start..end] function spans in memory.
              </p>
            </div>

            <div className="bg-slate-950 border border-slate-800 p-4 rounded-lg">
              <div className="font-mono text-cyan-400 text-xs font-bold mb-1">
                Tier 2: Embedded SQLite
              </div>
              <div className="text-2xl font-extrabold text-white mb-1">&lt; 2.0 ms</div>
              <p className="text-slate-400 text-xs leading-relaxed">
                Persisted ast_nodes table lookup by repository, commit SHA, and line.
              </p>
            </div>

            <div className="bg-slate-950 border border-slate-800 p-4 rounded-lg">
              <div className="font-mono text-purple-400 text-xs font-bold mb-1">
                Tier 3: GitHub On-Demand
              </div>
              <div className="text-2xl font-extrabold text-white mb-1">&lt; 25 ms</div>
              <p className="text-slate-400 text-xs leading-relaxed">
                Fetches raw source via GitHub App Contents API and parses on the fly.
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
