/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import { BookOpen, ExternalLink, Terminal, Shield, Activity, Tag } from 'lucide-react';
import { GithubIcon } from '@/components/GithubIcon';
import { useLatestRelease } from '@/components/useLatestRelease';

export const Footer: React.FC = () => {
  const release = useLatestRelease();

  return (
    <footer className="border-t border-slate-200 bg-white text-slate-600 font-sans">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-14">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          {/* Column 1: Brand & Tagline */}
          <div className="space-y-4 md:col-span-1">
            <div className="flex items-center gap-2.5">
              <div className="bg-black text-white px-2.5 py-1 rounded-sm font-mono text-xs tracking-wider font-bold">
                [TRIAGE]
              </div>
              <span className="font-bold text-slate-900 text-sm tracking-tight">triage</span>
            </div>
            <p className="text-xs text-slate-500 leading-relaxed">
              Zero-overhead Go panic isolation, on-demand AST slicing, and automated AI incident
              triage engine.
            </p>
            <a
              href={release.releaseUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 text-xs font-mono text-slate-700 bg-slate-100 hover:bg-slate-200 border border-slate-200 px-2.5 py-1 rounded w-fit transition-colors"
            >
              <Tag className="w-3 h-3 text-slate-500" />
              <span>Release: {release.version}</span>
            </a>
          </div>

          {/* Column 2: Documentation */}
          <div className="space-y-3">
            <h4 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-900">
              Documentation
            </h4>
            <ul className="space-y-2 text-xs font-mono">
              <li>
                <a href="/docs/overview" className="hover:text-black transition-colors">
                  Overview &amp; Concepts
                </a>
              </li>
              <li>
                <a href="/docs/quickstart" className="hover:text-black transition-colors">
                  5-Min Quickstart
                </a>
              </li>
              <li>
                <a href="/docs/sdk" className="hover:text-black transition-colors">
                  Go SDK Guide
                </a>
              </li>
              <li>
                <a href="/docs/ast-engine" className="hover:text-black transition-colors">
                  AST Engine Internals
                </a>
              </li>
              <li>
                <a href="/docs/ai-diagnostics" className="hover:text-black transition-colors">
                  AI Incident Diagnostics
                </a>
              </li>
            </ul>
          </div>

          {/* Column 3: Deployment & Operations */}
          <div className="space-y-3">
            <h4 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-900">
              Operations
            </h4>
            <ul className="space-y-2 text-xs font-mono">
              <li>
                <a href="/docs/self-hosting" className="hover:text-black transition-colors">
                  Self-Hosting Guide
                </a>
              </li>
              <li>
                <a href="/docs/github-integration" className="hover:text-black transition-colors">
                  GitHub App Setup
                </a>
              </li>
              <li>
                <a href="/docs/configuration" className="hover:text-black transition-colors">
                  Environment Reference
                </a>
              </li>
              <li>
                <a href="/docs/api-reference" className="hover:text-black transition-colors">
                  Engine REST API
                </a>
              </li>
              <li>
                <a
                  href="/docs/quickstart"
                  className="hover:text-black transition-colors text-indigo-600 font-semibold"
                >
                  <span>5-Min Quickstart</span>
                </a>
              </li>
            </ul>
          </div>

          {/* Column 4: Open Source */}
          <div className="space-y-3">
            <h4 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-900">
              Community &amp; Open Source
            </h4>
            <ul className="space-y-2 text-xs font-mono">
              <li>
                <a
                  href="https://github.com/algotyrnt/triage"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-black transition-colors flex items-center gap-1"
                >
                  <GithubIcon className="w-3.5 h-3.5" />
                  <span>GitHub Repository</span>
                </a>
              </li>
              <li>
                <a
                  href="https://github.com/algotyrnt/triage/issues"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-black transition-colors"
                >
                  Issue Tracker
                </a>
              </li>
              <li>
                <a
                  href="https://pkg.go.dev/github.com/algotyrnt/triage/sdk/go"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-black transition-colors"
                >
                  pkg.go.dev / SDK
                </a>
              </li>
              <li>
                <span className="text-slate-400">Apache 2.0 Licensed</span>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom copyright */}
        <div className="mt-12 pt-6 border-t border-slate-100 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 font-mono gap-3">
          <div>
            Created by{' '}
            <a
              href="https://algotyrnt.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-900 font-bold hover:underline"
            >
              Punjitha Bandara (algotyrnt)
            </a>
            . Licensed under Apache 2.0.
          </div>

          <div className="flex items-center gap-4">
            <span className="text-slate-400">Powered by Multi-Provider AI &amp; Go AST Parser</span>
          </div>
        </div>
      </div>
    </footer>
  );
};
