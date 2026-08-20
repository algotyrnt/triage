/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import {
  Code2,
  Zap,
  Cpu,
  GitPullRequest,
  ShieldCheck,
  Server,
  Sparkles,
  Terminal,
  Activity,
  Layers,
} from 'lucide-react';

export const FeaturesGrid: React.FC = () => {
  const features = [
    {
      icon: <Code2 className="w-6 h-6 text-indigo-600" />,
      title: 'Multi-File Package AST Slicing',
      desc: 'Extracts the crashing function alongside cross-file receiver structs, referenced types, constructors, and package helpers using Go’s standard go/parser. Eliminates >90% of irrelevant code while preserving full type context.',
      badge: 'go/ast & go/parser',
      tagColor: 'bg-indigo-50 text-indigo-700 border-indigo-200',
    },
    {
      icon: <Zap className="w-6 h-6 text-emerald-600" />,
      title: 'Non-Blocking Async Dispatch',
      desc: 'Middleware captures panics with defer + recover and dispatches telemetry across a bounded 4-goroutine worker pool. Zero latency added to HTTP client responses.',
      badge: '< 0.02ms Overhead',
      tagColor: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    },
    {
      icon: <Cpu className="w-6 h-6 text-purple-600" />,
      title: 'Gemini AI Diagnostics & Patches',
      desc: 'Leverages Google Gemini AI with deterministic structured JSON schemas to deliver precise root-cause analysis and unified git diff patches with configurable models.',
      badge: 'google.golang.org/genai',
      tagColor: 'bg-purple-50 text-purple-700 border-purple-200',
    },
    {
      icon: <GitPullRequest className="w-6 h-6 text-amber-600" />,
      title: 'Automated Fix PRs & Issue Filing',
      desc: 'Generates GitHub issues with AST snippets and opens automated bugfix Pull Requests by creating dedicated fix branches and applying AI patches cleanly.',
      badge: 'GitHub App & PRs',
      tagColor: 'bg-amber-50 text-amber-700 border-amber-200',
    },
    {
      icon: <Layers className="w-6 h-6 text-cyan-600" />,
      title: 'Multi-Project & Monorepo Support',
      desc: 'Manage all your Go microservices from a single dashboard with an instant project switcher, automatic go.mod submodule discovery, and project-scoped API keys.',
      badge: 'Monorepo Discovery',
      tagColor: 'bg-cyan-50 text-cyan-700 border-cyan-200',
    },
    {
      icon: <ShieldCheck className="w-6 h-6 text-slate-800" />,
      title: 'Zero-Trust Secret Scrubbing',
      desc: 'Sanitizes HTTP headers, query parameters, Authorization tokens, cookies, and sensitive connection strings before transmitting panic telemetry payloads.',
      badge: 'Enterprise Security',
      tagColor: 'bg-slate-100 text-slate-800 border-slate-200',
    },
  ];

  return (
    <section
      id="features"
      className="py-20 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-20"
    >
      {/* Section Header */}
      <div className="text-center max-w-3xl mx-auto space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <Sparkles className="w-3.5 h-3.5 text-indigo-600" />
          <span>ENGINE CAPABILITIES</span>
        </div>
        <h2 className="text-3xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
          Engineered for High-Reliability Go Systems
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed">
          Everything you need to intercept, symbolicate, diagnose, and resolve Go panics without
          adding latency or managing heavyweight agent sidecars.
        </p>
      </div>

      {/* Feature Grid */}
      <div className="mt-14 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {features.map((f, i) => (
          <div
            key={i}
            className="bg-white border border-slate-200/90 p-6 rounded-lg shadow-xs hover:shadow-md transition-all hover:border-slate-300 flex flex-col justify-between group"
          >
            <div>
              <div className="flex items-center justify-between mb-4">
                <div className="p-2.5 rounded-md bg-slate-50 border border-slate-100 group-hover:bg-slate-100 transition-colors">
                  {f.icon}
                </div>
                <span
                  className={`font-mono text-[10px] font-semibold px-2 py-0.5 rounded border ${f.tagColor}`}
                >
                  {f.badge}
                </span>
              </div>

              <h3 className="text-base font-bold text-slate-900 mb-2">{f.title}</h3>
              <p className="text-slate-600 text-xs sm:text-sm leading-relaxed">{f.desc}</p>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
};
