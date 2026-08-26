/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { ScreenId } from '@/types';
import { GithubIcon as Github } from '@/components/GithubIcon';
import { engineClient } from '@/services/engineClient';
import { ArrowRight } from 'lucide-react';

interface LoginPageProps {
  onNavigate: (screen: ScreenId) => void;
  onLoginSuccess: (user: { username: string; avatarUrl?: string }) => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onNavigate, onLoginSuccess }) => {
  const [loading, setLoading] = useState(false);

  const handleGitHubOAuthRedirect = () => {
    setLoading(true);
    // Redirect to Engine backend OAuth route
    window.location.href = `${engineClient.getBaseUrl()}/auth/github`;
  };

  return (
    <div className="min-h-[calc(100vh-100px)] bg-slate-50 flex flex-col items-center justify-center p-4">
      {/* Centered Card */}
      <div className="w-full max-w-md bg-white border border-slate-200 rounded-sm p-8 space-y-6 shadow-sm">
        {/* Header Block */}
        <div className="text-center space-y-2">
          <div className="inline-block bg-black text-white font-mono font-bold text-xs px-2.5 py-1 rounded-sm tracking-widest uppercase">
            [TRIAGE]
          </div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight">
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

        {/* Footer info */}
        <div className="text-center text-[11px] text-slate-400 font-mono">
          Triage Engine • Go 1.26+ Runtime Verified
        </div>
      </div>
    </div>
  );
};
