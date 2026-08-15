/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { HelpCircle, ChevronDown, ChevronUp } from 'lucide-react';

interface FaqItem {
  q: string;
  a: string;
}

const FAQS: FaqItem[] = [
  {
    q: 'How does Triage achieve zero latency overhead on HTTP requests?',
    a: 'When an HTTP handler panics, Triage catches the exception using defer + recover, writes the raw stack trace to an internal buffered ring buffer (1000 jobs), and immediately sends a sanitized 500 error response to the client. The telemetry payload is then dispatched asynchronously by a background 4-goroutine worker pool.',
  },
  {
    q: 'Do I need to pre-index all Go code in a database?',
    a: 'No! Triage features an on-demand AST slicing architecture. When a panic occurs, the engine uses the git commit SHA and file path to fetch only the required source file (via in-memory KV cache, local workspace, or GitHub Contents API) and parses the enclosing *ast.FuncDecl subtree synchronously in under 14 milliseconds.',
  },
  {
    q: 'Can I use my own Gemini API key or select different models?',
    a: 'Yes. You can supply your own Google AI Studio API key via the GEMINI_API_KEY environment variable or configure it via the Studio Dashboard setup wizard. You can configure any Gemini model of your choice via the GEMINI_MODEL_NAME setting.',
  },
  {
    q: 'How does GitHub App integration work?',
    a: 'Triage includes a 5-step automated setup wizard that creates and authenticates a GitHub App on your organization. This allows the engine to fetch exact commit trees on demand and automatically create triage issues with formatted AST snippets, stack traces, and suggested fixes.',
  },
  {
    q: 'Can Triage be self-hosted in a private VPC or air-gapped network?',
    a: 'Yes. The Triage Engine runs as a single Docker container (triage/engine:latest) backed by PostgreSQL. When running in offline or private networks, the engine can resolve AST nodes from local mounted source code repositories via the AST_WORKSPACE_ROOT configuration.',
  },
  {
    q: 'What happens if the Triage Engine is unreachable when a panic occurs?',
    a: 'The SDK gracefully fails open. If the telemetry worker pool fails to connect to the engine after 2 retries, the payload is safely dropped with zero impact on the host application and zero blocking of HTTP goroutines.',
  },
];

export const FaqSection: React.FC = () => {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  return (
    <section className="py-20 px-4 sm:px-6 lg:px-8 max-w-5xl mx-auto border-t border-slate-200">
      <div className="text-center max-w-3xl mx-auto space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <HelpCircle className="w-3.5 h-3.5 text-indigo-600" />
          <span>FREQUENTLY ASKED QUESTIONS</span>
        </div>
        <h2 className="text-3xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
          Everything You Need to Know
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed">
          Common architectural and operational questions regarding Triage panic isolation.
        </p>
      </div>

      <div className="mt-12 space-y-3">
        {FAQS.map((faq, idx) => {
          const isOpen = openIndex === idx;
          return (
            <div
              key={idx}
              className="border border-slate-200 rounded-lg bg-white overflow-hidden transition-all"
            >
              <button
                onClick={() => setOpenIndex(isOpen ? null : idx)}
                className="w-full px-6 py-4 text-left flex items-center justify-between gap-4 hover:bg-slate-50 transition-colors"
              >
                <span className="font-bold text-slate-900 text-sm sm:text-base">{faq.q}</span>
                {isOpen ? (
                  <ChevronUp className="w-4 h-4 text-slate-500 shrink-0" />
                ) : (
                  <ChevronDown className="w-4 h-4 text-slate-400 shrink-0" />
                )}
              </button>
              {isOpen && (
                <div className="px-6 pb-5 pt-1 text-slate-600 text-sm leading-relaxed border-t border-slate-100 font-normal">
                  {faq.a}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
};
