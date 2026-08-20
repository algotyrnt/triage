/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { BookOpen, Menu, X, ArrowRight, Sparkles } from 'lucide-react';
import { GithubIcon } from '@/components/GithubIcon';
import { useLatestRelease } from '@/components/useLatestRelease';

export const Navbar: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const release = useLatestRelease();

  return (
    <header className="border-b border-slate-200 bg-white/90 backdrop-blur-md sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-14 flex items-center justify-between">
        {/* Brand Block */}
        <div className="flex items-center gap-3">
          <a href="/" className="flex items-center gap-2 group">
            <div className="bg-black text-white px-2 py-0.5 rounded-sm font-mono text-xs tracking-wider font-bold shadow-xs">
              [TRIAGE]
            </div>
            <span className="font-bold text-slate-900 text-sm tracking-tight">
              Go Crash &amp; AST Engine
            </span>
          </a>

          {/* Version pill */}
          <a
            href={release.releaseUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="hidden sm:inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-slate-100 hover:bg-slate-200 text-slate-600 hover:text-slate-900 font-mono text-[10px] transition-colors border border-slate-200"
            title="View latest GitHub release"
          >
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
            <span>{release.version}</span>
          </a>
        </div>

        {/* Desktop Navigation Links */}
        <nav className="hidden md:flex items-center gap-6 text-xs font-mono text-slate-600 font-medium">
          <a href="#features" className="hover:text-black transition-colors">
            Features
          </a>
          <a href="#architecture" className="hover:text-black transition-colors">
            Architecture
          </a>
          <a href="#simulator" className="hover:text-black transition-colors">
            Simulator
          </a>
          <a href="#sdk" className="hover:text-black transition-colors">
            Go SDK
          </a>
          <a href="#benchmarks" className="hover:text-black transition-colors">
            Benchmarks
          </a>
          <a href="/docs/overview" className="hover:text-black transition-colors">
            Docs
          </a>
        </nav>

        {/* Action Controls */}
        <div className="hidden sm:flex items-center gap-3">
          <a
            href="https://github.com/algotyrnt/triage"
            target="_blank"
            rel="noopener noreferrer"
            className="text-slate-600 hover:text-black p-1.5 rounded-sm border border-slate-200 hover:bg-slate-50 transition-colors flex items-center justify-center"
            title="GitHub Repository"
            aria-label="GitHub Repository"
          >
            <GithubIcon className="w-4 h-4" />
          </a>

          <a
            href="/docs/quickstart"
            className="bg-black hover:bg-slate-800 text-white text-xs font-mono font-semibold px-3.5 py-1.5 rounded-sm flex items-center gap-1.5 transition-all shadow-xs"
          >
            <span>Quickstart</span>
            <ArrowRight className="w-3 h-3" />
          </a>
        </div>

        {/* Mobile menu trigger */}
        <div className="md:hidden flex items-center gap-2">
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="p-1.5 rounded-md text-slate-600 hover:text-black hover:bg-slate-100"
            aria-label="Toggle Navigation Menu"
          >
            {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>
      </div>

      {/* Mobile dropdown menu */}
      {mobileMenuOpen && (
        <div className="md:hidden border-t border-slate-200 bg-white px-4 py-4 space-y-3 font-mono text-xs shadow-xl">
          <a
            href="#features"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-slate-700 hover:text-black py-1"
          >
            Features
          </a>
          <a
            href="#architecture"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-slate-700 hover:text-black py-1"
          >
            Architecture
          </a>
          <a
            href="#simulator"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-slate-700 hover:text-black py-1"
          >
            Simulator
          </a>
          <a
            href="#sdk"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-slate-700 hover:text-black py-1"
          >
            Go SDK
          </a>
          <a
            href="#benchmarks"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-slate-700 hover:text-black py-1"
          >
            Benchmarks
          </a>
          <a
            href="/docs/overview"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-slate-900 font-bold py-1 border-t border-slate-100 pt-2"
          >
            Documentation
          </a>

          <div className="pt-2 flex items-center gap-2">
            <a
              href="https://github.com/algotyrnt/triage"
              target="_blank"
              rel="noopener noreferrer"
              className="flex-1 text-center border border-slate-200 text-slate-700 py-2 rounded-sm font-semibold flex items-center justify-center gap-1.5 hover:bg-slate-50"
            >
              <GithubIcon className="w-4 h-4" />
              <span>GitHub ({release.version})</span>
            </a>
            <a
              href="/docs/quickstart"
              onClick={() => setMobileMenuOpen(false)}
              className="flex-1 text-center bg-black text-white py-2 rounded-sm font-semibold flex items-center justify-center gap-1.5"
            >
              <span>Quickstart</span>
              <ArrowRight className="w-3 h-3" />
            </a>
          </div>
        </div>
      )}
    </header>
  );
};
