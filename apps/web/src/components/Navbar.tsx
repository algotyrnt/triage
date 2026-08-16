/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import {
  Terminal,
  Activity,
  Code2,
  ExternalLink,
  BookOpen,
  Menu,
  X,
  ArrowRight,
  Sparkles,
} from 'lucide-react';
import { GithubIcon } from '@/components/GithubIcon';

export const Navbar: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <header className="border-b border-slate-200 bg-white/90 backdrop-blur-md sticky top-0 z-50">
      {/* Main Navigation Bar */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3.5 flex items-center justify-between">
        {/* Brand Block */}
        <a href="/" className="flex items-center gap-3 group">
          <div className="bg-black text-white px-2.5 py-1 rounded-sm font-mono text-xs tracking-wider font-bold shadow-sm transition-transform group-hover:scale-105">
            [TRIAGE]
          </div>
          <div className="flex flex-col">
            <span className="font-bold text-slate-900 text-sm tracking-tight leading-none">
              triage
            </span>
            <span className="text-[11px] font-mono text-slate-500 tracking-tight leading-tight mt-0.5">
              Go Crash &amp; AST Engine
            </span>
          </div>
        </a>

        {/* Desktop Navigation Links */}
        <nav className="hidden md:flex items-center gap-6 text-xs font-mono text-slate-600 font-medium">
          <a href="#features" className="hover:text-black transition-colors">
            Features
          </a>
          <a href="#architecture" className="hover:text-black transition-colors">
            Architecture
          </a>
          <a
            href="#simulator"
            className="hover:text-black transition-colors flex items-center gap-1 text-indigo-600 font-semibold"
          >
            <Sparkles className="w-3 h-3" />
            <span>Panic Inspector</span>
          </a>
          <a href="#sdk" className="hover:text-black transition-colors">
            Go SDK
          </a>
          <a href="#benchmarks" className="hover:text-black transition-colors">
            Benchmarks
          </a>
          <a
            href="/docs/overview"
            className="flex items-center gap-1 hover:text-black transition-colors text-slate-900 font-semibold px-2 py-1 rounded hover:bg-slate-100"
          >
            <BookOpen className="w-3.5 h-3.5" />
            <span>Documentation</span>
          </a>
        </nav>

        {/* Action Buttons */}
        <div className="hidden sm:flex items-center gap-3">
          <a
            href="https://github.com/algotyrnt/triage"
            target="_blank"
            rel="noopener noreferrer"
            className="text-slate-600 hover:text-black p-2 rounded-sm border border-slate-300 hover:bg-slate-50 transition-colors flex items-center justify-center"
            title="GitHub Repository"
            aria-label="GitHub Repository"
          >
            <GithubIcon className="w-4 h-4" />
          </a>

          <a
            href="/docs/quickstart"
            className="text-xs font-mono text-slate-700 hover:text-black px-3 py-1.5 rounded-sm border border-slate-300 hover:bg-slate-50 transition-colors flex items-center gap-1.5"
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>Quickstart</span>
          </a>

          <a
            href="/docs/self-hosting"
            className="bg-black hover:bg-slate-800 text-white text-xs font-mono font-semibold px-3.5 py-1.5 rounded-sm flex items-center gap-1.5 transition-all shadow-sm group"
          >
            <span>Self-Hosting Guide</span>
            <ArrowRight className="w-3 h-3 group-hover:translate-x-0.5 transition-transform" />
          </a>
        </div>

        {/* Mobile menu trigger */}
        <div className="md:hidden flex items-center gap-2">
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="p-2 rounded-md text-slate-600 hover:text-black hover:bg-slate-100"
            aria-label="Toggle Navigation Menu"
          >
            {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>
      </div>

      {/* Mobile dropdown menu */}
      {mobileMenuOpen && (
        <div className="md:hidden border-t border-slate-200 bg-white px-4 py-4 space-y-3 font-mono text-sm shadow-xl">
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
            className="block text-indigo-600 font-semibold py-1 items-center gap-1.5"
          >
            <Sparkles className="w-4 h-4" />
            <span>Panic Inspector</span>
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
            className="block text-slate-900 font-bold py-1 border-t border-slate-100 pt-2 items-center gap-1.5"
          >
            <BookOpen className="w-4 h-4" />
            <span>Documentation</span>
          </a>
          <div className="pt-2 flex flex-col gap-2">
            <a
              href="https://github.com/algotyrnt/triage"
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)}
              className="w-full text-center border border-slate-300 text-slate-700 py-2 rounded-sm font-semibold flex items-center justify-center gap-2 hover:bg-slate-50"
            >
              <GithubIcon className="w-4 h-4" />
              <span>GitHub Repository</span>
            </a>
            <a
              href="/docs/self-hosting"
              onClick={() => setMobileMenuOpen(false)}
              className="w-full text-center bg-black text-white py-2 rounded-sm font-semibold flex items-center justify-center gap-2"
            >
              <span>Self-Hosting Guide</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </a>
          </div>
        </div>
      )}
    </header>
  );
};
