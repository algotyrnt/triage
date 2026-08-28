/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import { Activity, Clock, CheckCircle2 } from 'lucide-react';

export const BenchmarksSection: React.FC = () => {
  return (
    <section
      id="benchmarks"
      className="py-14 sm:py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-16"
    >
      {/* Section Header */}
      <div className="text-center max-w-3xl mx-auto space-y-2.5 sm:space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <Activity className="w-3.5 h-3.5 text-emerald-600" />
          <span>PERFORMANCE &amp; BENCHMARKS</span>
        </div>
        <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-slate-900 tracking-tight">
          Engineered for Extreme Efficiency
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed max-w-2xl mx-auto">
          Triage isolates crashes with negligible server overhead while reducing LLM diagnostic
          token consumption by over 93% compared to industry-standard APM and error tracking
          platforms.
        </p>
      </div>

      {/* Triage Core Architecture Highlights Grid */}
      <div className="mt-10 sm:mt-12 grid grid-cols-1 md:grid-cols-3 gap-5 sm:gap-6 items-stretch">
        {/* Highlight 1: Request Interception Overhead */}
        <div className="bg-white border border-slate-200 p-6 rounded-xl shadow-xs flex flex-col justify-between hover:border-slate-300 transition-colors">
          <div>
            <div className="flex items-center justify-between h-6 mb-3">
              <span className="font-mono text-xs text-slate-500 uppercase font-semibold">
                Interception Overhead
              </span>
              <span className="bg-emerald-50 text-emerald-700 font-mono text-[10px] font-bold px-2 py-0.5 rounded border border-emerald-200">
                &lt; 0.02 ms
              </span>
            </div>

            <div className="text-3xl font-extrabold text-slate-900 tracking-tight mb-1 h-9 flex items-center">
              &lt; 0.02 ms
            </div>
            <p className="text-slate-500 text-xs mb-5 min-h-9 flex items-center leading-relaxed">
              Non-blocking defer/recover with async worker queue
            </p>

            <div className="space-y-3.5 text-xs text-slate-600">
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">Immediate Client Return:</strong>{' '}
                  HTTP client responses finish without waiting for telemetry transmission.
                </span>
              </div>
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">Bounded 1,000-Job Queue:</strong>{' '}
                  Asynchronous 4-worker goroutine pool prevents memory growth during traffic surges.
                </span>
              </div>
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">Zero Pipeline Lockup:</strong>{' '}
                  Eliminates synchronous stack walks on active request handling threads.
                </span>
              </div>
            </div>
          </div>

          <div className="mt-5 pt-3.5 border-t border-slate-100 text-[11px] text-slate-500 font-sans leading-relaxed min-h-9 flex items-center">
            Sub-millisecond panic capture with zero blocking on active request goroutines.
          </div>
        </div>

        {/* Highlight 2: Multi-File AST Slicing */}
        <div className="bg-white border border-slate-200 p-6 rounded-xl shadow-xs flex flex-col justify-between hover:border-slate-300 transition-colors">
          <div>
            <div className="flex items-center justify-between h-6 mb-3">
              <span className="font-mono text-xs text-slate-500 uppercase font-semibold">
                AST Code Isolation
              </span>
              <span className="bg-indigo-50 text-indigo-700 font-mono text-[10px] font-bold px-2 py-0.5 rounded border border-indigo-200">
                93.8% Saved
              </span>
            </div>

            <div className="text-3xl font-extrabold text-slate-900 tracking-tight mb-1 h-9 flex items-center">
              ~240 Tokens
            </div>
            <p className="text-slate-500 text-xs mb-5 min-h-9 flex items-center leading-relaxed">
              Exact function and receiver type context extracted on the fly
            </p>

            <div className="space-y-3.5 text-xs text-slate-600">
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-indigo-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">Enclosing AST Node:</strong>{' '}
                  Extracts the exact{' '}
                  <code className="font-mono text-indigo-700">*ast.FuncDecl</code> subtree
                  containing the panic site.
                </span>
              </div>
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-indigo-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">Cross-File Slicing:</strong>{' '}
                  Pulls receiver struct definitions, constructors, and package helper declarations.
                </span>
              </div>
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-indigo-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">93.8% Token Savings:</strong>{' '}
                  Replaces 4,800-token whole-file dumps with a concise ~240-token semantic context.
                </span>
              </div>
            </div>
          </div>

          <div className="mt-5 pt-3.5 border-t border-slate-100 text-[11px] text-slate-500 font-sans leading-relaxed min-h-9 flex items-center">
            Eliminates prompt noise while giving AI models exact semantic type context.
          </div>
        </div>

        {/* Highlight 3: Pure Standard Library SDK */}
        <div className="bg-white border border-slate-200 p-6 rounded-xl shadow-xs flex flex-col justify-between hover:border-slate-300 transition-colors">
          <div>
            <div className="flex items-center justify-between h-6 mb-3">
              <span className="font-mono text-xs text-slate-500 uppercase font-semibold">
                SDK Dependencies
              </span>
              <span className="bg-cyan-50 text-cyan-700 font-mono text-[10px] font-bold px-2 py-0.5 rounded border border-cyan-200">
                0 Dependencies
              </span>
            </div>

            <div className="text-3xl font-extrabold text-slate-900 tracking-tight mb-1 h-9 flex items-center">
              0 Third-Party
            </div>
            <p className="text-slate-500 text-xs mb-5 min-h-9 flex items-center leading-relaxed">
              Pure Go standard library implementation with zero external bloat
            </p>

            <div className="space-y-3.5 text-xs text-slate-600">
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-cyan-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">0 External Dependencies:</strong>{' '}
                  Pure Go stdlib (<code className="font-mono text-slate-700">net/http</code>,{' '}
                  <code className="font-mono text-slate-700">runtime/debug</code>,{' '}
                  <code className="font-mono text-slate-700">sync</code>).
                </span>
              </div>
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-cyan-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">&lt; 1.2 MB Memory:</strong> Low
                  resident set size footprint with zero runtime reflection allocations.
                </span>
              </div>
              <div className="flex items-start gap-2.5 min-h-11">
                <CheckCircle2 className="w-4 h-4 text-cyan-600 shrink-0 mt-0.5" />
                <span>
                  <strong className="text-slate-900 font-semibold">No Sidecar Daemons:</strong> Zero
                  external agents, CGO bindings, or background telemetry daemons.
                </span>
              </div>
            </div>
          </div>

          <div className="mt-5 pt-3.5 border-t border-slate-100 text-[11px] text-slate-500 font-sans leading-relaxed min-h-9 flex items-center">
            Zero supply chain risk and effortless drop-in middleware integration.
          </div>
        </div>
      </div>

      {/* Head-to-Head Comparison Table - Same 4 Products with Fixed Columns */}
      <div className="mt-10 sm:mt-12 bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200 bg-slate-50/70 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
          <div>
            <h3 className="text-sm sm:text-base font-bold text-slate-900">Technical Comparison</h3>
            <p className="text-xs text-slate-500">
              Architecture and feature matrix for Go HTTP microservices
            </p>
          </div>
          <span className="inline-flex items-center gap-1 text-[11px] font-mono text-slate-500 bg-white px-2.5 py-1 rounded border border-slate-200">
            <Clock className="w-3 h-3 text-indigo-500" />
            Tested on Go 1.26
          </span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs table-fixed min-w-190">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50/50 font-mono text-slate-600">
                <th className="py-3.5 px-4 sm:px-5 font-semibold w-[22%]">Capability</th>
                <th className="py-3.5 px-4 sm:px-5 font-bold text-indigo-700 bg-indigo-50/50 border-x border-indigo-100 w-[26%]">
                  Triage
                </th>
                <th className="py-3.5 px-4 sm:px-5 font-semibold w-[18%]">Sentry (sentry-go)</th>
                <th className="py-3.5 px-4 sm:px-5 font-semibold w-[17%]">Datadog (dd-trace-go)</th>
                <th className="py-3.5 px-4 sm:px-5 font-semibold w-[17%]">Bugsnag (bugsnag-go)</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-slate-700 font-sans">
              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Panic Interception
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-emerald-700 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  Non-blocking (4-worker queue)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Synchronous panic hook
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Synchronous tracer span
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Synchronous panic notifier
                </td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Code Context Slicing
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-indigo-700 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  Multi-File AST (*ast.FuncDecl)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  5-line stack window
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Trace spans &amp; stacks only
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  7-line raw text snippet
                </td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Diagnostic Tokens
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-emerald-700 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  ~240 tokens (&gt;93% saved)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  3,000+ tokens (Seer AI)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-500 align-middle">
                  N/A (No AI diagnosis)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-500 align-middle">
                  N/A (No AI diagnosis)
                </td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Automated Bugfix PRs
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-emerald-700 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  <span className="inline-flex items-center gap-1.5">
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
                    <span>1-Click Verified PR</span>
                  </span>
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-500 align-middle">
                  Manual review only
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-500 align-middle">
                  Alert notification only
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-500 align-middle">
                  Error dashboard only
                </td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Crash Deduplication
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-slate-900 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  SHA-256 Hash + Count Badge
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Heuristic issue grouping
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">Tag aggregation</td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Error class hashing
                </td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Client Dependencies
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-emerald-700 bg-indigo-50/20 border-x border-indigo-100 font-mono align-middle">
                  0 (Pure Go Standard Library)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  8+ external packages
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  14+ external packages
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  6+ external packages
                </td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  AI Model Flexibility
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-slate-900 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  Gemini, OpenAI, Claude, Local
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Proprietary SaaS (Seer)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Proprietary SaaS (Bits)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-500 align-middle">None</td>
              </tr>

              <tr>
                <td className="py-3.5 px-4 sm:px-5 font-medium text-slate-900 font-mono align-middle">
                  Deployment Model
                </td>
                <td className="py-3.5 px-4 sm:px-5 font-semibold text-emerald-700 bg-indigo-50/20 border-x border-indigo-100 align-middle">
                  1 Docker Container (Self-Hostable)
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  30+ Helm pods or SaaS
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Proprietary Cloud SaaS only
                </td>
                <td className="py-3.5 px-4 sm:px-5 text-slate-600 align-middle">
                  Enterprise On-Prem / SaaS
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
};
