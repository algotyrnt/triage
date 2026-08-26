/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import {
  Sparkles,
  ArrowRight,
  Terminal,
  Copy,
  Check,
  Zap,
  Cpu,
  Code2,
  ShieldCheck,
  Server,
  Tag,
} from 'lucide-react';
import { useLatestRelease } from '@/components/useLatestRelease';

export const Hero: React.FC = () => {
  const [copied, setCopied] = useState(false);
  const release = useLatestRelease();
  const installCmd = 'go get github.com/algotyrnt/triage/sdk/go';

  const copyInstall = () => {
    navigator.clipboard.writeText(installCmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section className="pt-10 pb-14 sm:pt-14 sm:pb-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto text-center relative overflow-hidden">
      {/* Background ambient accents */}
      <div className="absolute top-8 left-1/2 -translate-x-1/2 w-3/4 h-64 bg-linear-to-r from-indigo-100/50 via-purple-50/40 to-cyan-100/50 blur-3xl -z-10 rounded-full pointer-events-none" />

      {/* Pill Badge */}
      <a
        href={release.releaseUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-2 px-3.5 py-1 rounded-full bg-indigo-50 hover:bg-indigo-100/80 border border-indigo-200/80 text-indigo-700 font-mono text-xs font-semibold shadow-xs mb-6 transition-colors group"
      >
        <span className="bg-indigo-600 text-white text-[10px] px-2 py-0.5 rounded-full uppercase tracking-wider font-bold">
          {release.version}
        </span>
        <span>Go Crash Isolation &amp; Multi-Provider AI Diagnostics</span>
        <ArrowRight className="w-3 h-3 text-indigo-500 group-hover:translate-x-0.5 transition-transform" />
      </a>

      {/* Main Headline */}
      <h1 className="text-3xl sm:text-5xl lg:text-6xl font-extrabold text-slate-900 tracking-tight leading-[1.12] max-w-4xl mx-auto">
        Zero-Latency Go Crash Isolation.{' '}
        <span className="bg-linear-to-r from-indigo-600 via-purple-600 to-cyan-600 bg-clip-text text-transparent block sm:inline">
          Instant AI Diagnosis.
        </span>
      </h1>

      {/* Subtitle / Paragraph */}
      <p className="mt-5 text-slate-600 text-sm sm:text-base lg:text-lg max-w-3xl mx-auto leading-relaxed font-normal">
        Intercepts Go HTTP server panics non-blockingly using{' '}
        <code className="text-indigo-800 font-mono bg-indigo-50/70 px-1.5 py-0.5 rounded border border-indigo-100 font-semibold text-xs sm:text-sm">
          defer + recover
        </code>
        . Automatically isolates the crash site and surrounding multi-file package context (struct
        definitions, constructors, helpers), queries AI models for root causes, and opens automated
        bugfix Pull Requests with drop-in patches.
      </p>

      {/* Primary Action Buttons */}
      <div className="mt-8 flex flex-wrap items-center justify-center gap-3 sm:gap-4">
        <a
          href="/docs/quickstart"
          className="bg-black hover:bg-slate-800 text-white font-mono text-xs sm:text-sm font-semibold px-5 py-3 rounded-sm flex items-center gap-2 shadow-md hover:shadow-lg transition-all"
        >
          <span>Get Started in 5 Mins</span>
          <ArrowRight className="w-4 h-4" />
        </a>

        <a
          href="/docs/self-hosting"
          className="bg-white hover:bg-slate-50 text-slate-900 font-mono text-xs sm:text-sm font-semibold px-5 py-3 rounded-sm border border-slate-300 flex items-center gap-2 shadow-xs transition-all"
        >
          <Server className="w-4 h-4 text-slate-700" />
          <span>Self-Hosting Guide</span>
        </a>

        <a
          href="/docs/sdk"
          className="bg-slate-100 hover:bg-slate-200 text-slate-800 font-mono text-xs sm:text-sm font-semibold px-5 py-3 rounded-sm border border-slate-200 flex items-center gap-2 transition-all"
        >
          <Terminal className="w-4 h-4 text-slate-600" />
          <span>Go SDK Setup</span>
        </a>
      </div>

      {/* One-Click Copy Command Box */}
      <div className="mt-6 max-w-lg mx-auto">
        <div className="bg-slate-950 border border-slate-800 rounded-lg p-2.5 sm:p-3 flex items-center justify-between font-mono text-xs text-slate-300 shadow-xl">
          <div className="flex items-center gap-2.5 overflow-x-auto scrollbar-none pr-2">
            <span className="text-slate-500 select-none">$</span>
            <span className="text-emerald-400 font-medium whitespace-nowrap">{installCmd}</span>
          </div>
          <button
            onClick={copyInstall}
            className="shrink-0 bg-slate-800 hover:bg-slate-700 text-slate-200 hover:text-white px-3 py-1.5 rounded text-xs flex items-center gap-1.5 transition-colors border border-slate-700 font-sans font-medium"
            title="Copy command"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-emerald-400">Copied!</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5 text-slate-400" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Key Metric Highlights Grid */}
      <div className="mt-10 max-w-4xl mx-auto grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 text-left">
        <div className="bg-white border border-slate-200/80 p-4 rounded-lg shadow-xs hover:border-slate-300 transition-colors">
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500 uppercase font-semibold">
            <Zap className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
            <span>Overhead</span>
          </div>
          <div className="mt-1.5 text-2xl font-bold text-slate-900 tracking-tight">
            &lt; 0.02 ms
          </div>
          <div className="text-[11px] text-slate-500 mt-0.5">Async bounded worker pool</div>
        </div>

        <div className="bg-white border border-slate-200/80 p-4 rounded-lg shadow-xs hover:border-slate-300 transition-colors">
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500 uppercase font-semibold">
            <Code2 className="w-3.5 h-3.5 text-indigo-600 shrink-0" />
            <span>AST Isolation</span>
          </div>
          <div className="mt-1.5 text-2xl font-bold text-slate-900 tracking-tight">94% Savings</div>
          <div className="text-[11px] text-slate-500 mt-0.5">Exact *ast.FuncDecl node</div>
        </div>

        <div className="bg-white border border-slate-200/80 p-4 rounded-lg shadow-xs hover:border-slate-300 transition-colors">
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500 uppercase font-semibold">
            <Cpu className="w-3.5 h-3.5 text-purple-600 shrink-0" />
            <span>AI Inference</span>
          </div>
          <div className="mt-1.5 text-2xl font-bold text-slate-900 tracking-tight">
            Pluggable AI
          </div>
          <div className="text-[11px] text-slate-500 mt-0.5">Gemini, OpenAI, Claude, Ollama</div>
        </div>

        <div className="bg-white border border-slate-200/80 p-4 rounded-lg shadow-xs hover:border-slate-300 transition-colors">
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500 uppercase font-semibold">
            <ShieldCheck className="w-3.5 h-3.5 text-cyan-600 shrink-0" />
            <span>Self-Hosting</span>
          </div>
          <div className="mt-1.5 text-2xl font-bold text-slate-900 tracking-tight">1 Container</div>
          <div className="text-[11px] text-slate-500 mt-0.5">Single Docker container</div>
        </div>
      </div>
    </section>
  );
};
