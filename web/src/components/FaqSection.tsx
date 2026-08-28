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
    q: 'How does automated Bugfix Pull Request (PR) generation work?',
    a: 'When an incident is diagnosed, you can click "Generate Fix (PR)" in the Studio Dashboard. The Triage Engine uses the configured AI model to apply the suggested patch cleanly to the target file, creates a new Git branch (e.g. triage/fix-inc...), commits the fix, and opens a Pull Request on GitHub linked to the incident and closing any related GitHub issue.',
  },
  {
    q: 'Do I need to pre-index all Go code in a database?',
    a: 'No! Triage features an on-demand AST slicing architecture. When a panic occurs, the engine uses the git commit SHA and file path to fetch only the required source file (via in-memory KV cache, local workspace, or GitHub Contents API) and parses the enclosing *ast.FuncDecl subtree and package context synchronously in under 14 milliseconds.',
  },
  {
    q: 'How does Triage handle Go monorepos and multi-module projects?',
    a: 'Triage automatically detects nested go.mod files across your repository subdirectories (/api/v1/repos/detect-modules). It normalizes file paths between module-relative and repo-relative paths, allowing accurate AST symbolication and GitHub file patching regardless of directory structure.',
  },
  {
    q: 'Can I manage multiple Go projects and API keys from a single deployment?',
    a: 'Yes. The Studio Dashboard includes a Workspace Projects page and Header Project Switcher. You can track multiple microservices, generate project-specific API keys, revoke keys, and filter incidents by repository ID or name.',
  },
  {
    q: 'Can I use my own AI API keys or select different providers?',
    a: 'Yes. Triage supports Google Gemini, OpenAI (GPT-4o, o3-mini), Anthropic Claude (Claude 3.5/3.7), and local/self-hosted models via Ollama and vLLM. You can configure credentials and run live connection latency tests directly via the Studio Dashboard setup wizard or Settings page.',
  },
  {
    q: 'What happens if the Triage Engine is unreachable when a panic occurs?',
    a: 'The SDK gracefully fails open. If the telemetry worker pool fails to connect to the engine after retries, the payload is safely dropped with zero impact on the host application and zero blocking of HTTP goroutines.',
  },
];

export const FaqSection: React.FC = () => {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  return (
    <section
      id="faq"
      className="py-14 sm:py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-16"
    >
      <div className="max-w-4xl mx-auto">
        <div className="text-center space-y-2.5 sm:space-y-3">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
            <HelpCircle className="w-3.5 h-3.5 text-indigo-600" />
            <span>FREQUENTLY ASKED QUESTIONS</span>
          </div>
          <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-slate-900 tracking-tight">
            Everything You Need to Know
          </h2>
          <p className="text-slate-600 text-sm sm:text-base leading-relaxed max-w-2xl mx-auto">
            Common architectural and operational questions regarding Triage panic isolation.
          </p>
        </div>

        <div className="mt-10 sm:mt-12 space-y-3">
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
      </div>
    </section>
  );
};
