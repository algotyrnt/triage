/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { ScreenId } from '@/types';
import { GithubIcon as Github } from '@/components/GithubIcon';
import { engineClient } from '@/services/engineClient';
import { ShieldCheck, Lock, Check, ArrowRight } from 'lucide-react';

interface LoginPageProps {
  onNavigate: (screen: ScreenId) => void;
  onLoginSuccess: (user: { username: string; avatarUrl?: string }) => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onNavigate, onLoginSuccess }) => {
  const [loading, setLoading] = useState(false);
  const handleGitHubOAuthRedirect = () => {
    setLoading(true);
    // Redirect to backend OAuth route
    window.location.href = engineClient.getAuthGitHubUrl();
  };

  return (
    <div className="min-h-[calc(100vh-100px)] bg-slate-50 flex flex-col items-center justify-center p-4">
      {/* 420px Centered Card */}
      <div className="w-full max-w-[420px] bg-white border border-slate-200 rounded-sm p-6 shadow-none space-y-6">
        {/* Header Block */}
        <div className="text-center space-y-2">
          <div className="inline-block bg-black text-white font-mono font-bold text-sm px-3 py-1 rounded-sm tracking-widest uppercase">
            [TRIAGE]
          </div>
          <h1 className="text-lg font-bold text-slate-900 tracking-tight">
            Go Backend Crash Detection & AST Isolation
          </h1>
          <p className="text-xs text-slate-600 font-sans">
            Sign in with your GitHub account to connect repositories and view live panic telemetry.
          </p>
        </div>

        {/* Auth Action */}
        <div className="space-y-4 font-mono">
          <button
            onClick={handleGitHubOAuthRedirect}
            disabled={loading}
            className="w-full bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2.5 px-4 rounded-sm border border-black transition-all flex items-center justify-center gap-2 cursor-pointer"
          >
            <Github className="w-4 h-4" />
            <span>{loading ? 'Redirecting to GitHub OAuth...' : 'Sign in via GitHub OAuth'}</span>
            {!loading && <ArrowRight className="w-3.5 h-3.5 ml-1 opacity-70" />}
          </button>
        </div>

        {/* Security Disclaimer Box (#F1F5F9) */}
        <div className="bg-slate-100 border border-slate-200 rounded-sm p-3.5 space-y-2 text-xs font-mono text-slate-600">
          <div className="flex items-center gap-1.5 text-slate-900 font-semibold text-[11px] uppercase tracking-wider">
            <ShieldCheck className="w-3.5 h-3.5 text-slate-700" />
            <span>Security & Data Compliance Scope</span>
          </div>

          <ul className="space-y-1.5 text-[11px] text-slate-600 leading-relaxed">
            <li className="flex items-start gap-1.5">
              <Check className="w-3 h-3 text-emerald-600 mt-0.5 shrink-0" />
              <span>
                <strong className="text-slate-800">Read-Only AST Scope:</strong> Parses exported
                package ASTs, function signatures & byte offsets. Zero write permissions to source.
              </span>
            </li>
            <li className="flex items-start gap-1.5">
              <Check className="w-3 h-3 text-emerald-600 mt-0.5 shrink-0" />
              <span>
                <strong className="text-slate-800">Zero-Log Policy:</strong> Runtime stack traces
                are symbolicated on-the-fly and never stored in unencrypted persistent logs.
              </span>
            </li>
            <li className="flex items-start gap-1.5">
              <Lock className="w-3 h-3 text-slate-700 mt-0.5 shrink-0" />
              <span>
                <strong className="text-slate-800">TLS 1.3 Encryption:</strong> Webhooks and
                telemetry ingress protected in transit.
              </span>
            </li>
          </ul>
        </div>

        {/* Footer info */}
        <div className="text-center text-[11px] text-slate-400 font-mono">
          Triage Engine v1.4.2 • Go 1.22+ Runtime Verified
        </div>
      </div>
    </div>
  );
};
