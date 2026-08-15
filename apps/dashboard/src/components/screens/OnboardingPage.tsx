/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useRef } from 'react';
import { ScreenId } from '@/types';
import {
  GitBranch,
  CheckCircle2,
  Copy,
  Check,
  Search,
  ArrowRight,
  Key,
  Shield,
  Layers,
  Sparkles,
} from 'lucide-react';
import { engineClient } from '@/services/engineClient';

interface OnboardingPageProps {
  onNavigate: (screen: ScreenId) => void;
  onProjectSetup?: (repo: string, apiKey: string) => void;
  currentUser?: { username: string; avatarUrl?: string } | null;
}

export const OnboardingPage: React.FC<OnboardingPageProps> = ({
  onNavigate,
  onProjectSetup,
  currentUser,
}) => {
  const [currentStep, setCurrentStep] = useState<1 | 2 | 3>(1);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRepo, setSelectedRepo] = useState('');
  const [customRepoInput, setCustomRepoInput] = useState('');
  const [generatedKey, setGeneratedKey] = useState('');
  const [copiedKey, setCopiedKey] = useState(false);
  const [loadingRepos, setLoadingRepos] = useState(false);
  const [repos, setRepos] = useState<
    { name: string; visibility: string; branch: string; lang: string }[]
  >([]);
  const repoRefs = useRef<(HTMLDivElement | null)[]>([]);

  const username = currentUser?.username || 'algotyrnt';

  React.useEffect(() => {
    async function fetchUserGitHubRepos() {
      setLoadingRepos(true);
      try {
        const installedRepos = await engineClient.getSetupRepos();
        if (Array.isArray(installedRepos) && installedRepos.length > 0) {
          const mapped = installedRepos.map((r) => ({
            name: `${r.owner}/${r.repo}`,
            visibility: 'Installed',
            branch: 'main',
            lang: 'Go / TS',
          }));
          setRepos(mapped);
          if (mapped[0]) setSelectedRepo(mapped[0].name);
        }
      } catch (e) {
        console.warn('Failed to fetch installed repos:', e);
      } finally {
        setLoadingRepos(false);
      }
    }
    fetchUserGitHubRepos();
  }, [username]);

  const generateNewKey = () => {
    const hex = Math.random().toString(36).substring(2, 12);
    const key = `tr_live_${hex}`;
    setGeneratedKey(key);
    return key;
  };

  const filteredRepos = repos.filter((r) =>
    r.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const activeKey = generatedKey || 'tr_live_demo_key_9042';

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(activeKey);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    } catch (e) {
      console.error('Failed to copy API key', e);
    }
  };

  const handleCompleteSetup = () => {
    const key = generatedKey || generateNewKey();
    const repo = customRepoInput.trim() || selectedRepo;
    if (onProjectSetup) {
      onProjectSetup(repo, key);
    } else {
      onNavigate('dashboard');
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8 space-y-6">
      {/* Title */}
      <div className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
          Project Onboarding & Go AST Setup
        </h1>
        <p className="text-xs text-slate-600 font-sans mt-1">
          Connect your GitHub repository to index Go AST syntax trees and enable live panic crash
          symbolication.
        </p>
      </div>

      {/* 3-Step Indicator Bar */}
      <div className="grid grid-cols-3 gap-2 bg-white border border-slate-200 p-2 rounded-sm font-mono text-xs">
        {[
          { num: 1, title: 'Select Repo', desc: 'Choose target Go project' },
          {
            num: 2,
            title: 'GitHub App Setup',
            desc: 'Read-only tree permissions',
          },
          {
            num: 3,
            title: 'SDK Integration Key',
            desc: 'Generate telemetry token',
          },
        ].map((step) => {
          const isActive = currentStep === step.num;
          const isDone = currentStep > step.num;
          return (
            <button
              key={step.num}
              onClick={() => setCurrentStep(step.num as any)}
              className={`text-left p-2.5 rounded-sm transition-all border ${
                isActive
                  ? 'border-black bg-black text-white'
                  : isDone
                    ? 'border-emerald-200 bg-emerald-50/50 text-slate-900'
                    : 'border-slate-100 bg-slate-50 text-slate-500'
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span
                  className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm ${
                    isActive
                      ? 'bg-white text-black'
                      : isDone
                        ? 'bg-emerald-600 text-white'
                        : 'bg-slate-200 text-slate-700'
                  }`}
                >
                  STEP 0{step.num}
                </span>
                {isDone && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />}
              </div>
              <div className="font-bold text-xs">{step.title}</div>
              <div
                className={`text-[10px] truncate ${isActive ? 'text-slate-300' : 'text-slate-500'}`}
              >
                {step.desc}
              </div>
            </button>
          );
        })}
      </div>

      {/* Step Content */}
      <div className="bg-white border border-slate-200 rounded-sm p-6 space-y-6">
        {currentStep === 1 && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-bold text-slate-900 font-sans">
                  Select a Go Repository from <span className="font-mono">@{username}</span>
                </h2>
                <p className="text-xs text-slate-500 font-sans">
                  Select one of your real GitHub repositories or enter a custom repository.
                </p>
              </div>
              <span className="text-xs font-mono text-slate-500">
                {filteredRepos.length} repositories found
              </span>
            </div>

            {/* Custom Repo Manual Input Box */}
            <div className="space-y-1 bg-slate-50 border border-slate-200 p-3 rounded-sm font-mono text-xs">
              <label className="font-bold text-slate-800 text-[11px] uppercase tracking-wider block">
                Track Custom GitHub Repository:
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={customRepoInput}
                  onChange={(e) => {
                    setCustomRepoInput(e.target.value);
                    if (e.target.value) setSelectedRepo(e.target.value);
                  }}
                  placeholder={`e.g. ${username}/my-go-backend`}
                  className="flex-1 bg-white border border-slate-300 rounded-sm px-3 py-1.5 text-xs font-mono focus:outline-none focus:border-black"
                />
              </div>
            </div>

            {/* Search Box */}
            <div className="relative">
              <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-slate-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={`Search @${username}'s repositories...`}
                className="w-full pl-9 pr-3 py-1.5 bg-slate-50 border border-slate-200 rounded-sm text-xs font-mono focus:bg-white focus:outline-none focus:border-black"
              />
            </div>

            {/* Repo List */}
            {loadingRepos ? (
              <div className="py-8 text-center text-xs font-mono text-slate-500 animate-pulse">
                Fetching live repositories for @{username} from GitHub API...
              </div>
            ) : (
              <div
                role="radiogroup"
                aria-label="Select Repository"
                className="space-y-2 max-h-60 overflow-y-auto"
              >
                {filteredRepos.map((repo, index) => {
                  const isSelected = selectedRepo === repo.name;
                  const isSelectedInFiltered = filteredRepos.some((r) => r.name === selectedRepo);
                  const hasFocusEntry =
                    isSelected || (index === 0 && (!selectedRepo || !isSelectedInFiltered));
                  return (
                    <div
                      key={repo.name}
                      ref={(el) => {
                        repoRefs.current[index] = el;
                      }}
                      role="radio"
                      aria-checked={isSelected}
                      tabIndex={hasFocusEntry ? 0 : -1}
                      onClick={() => setSelectedRepo(repo.name)}
                      onKeyDown={(e) => {
                        if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
                          e.preventDefault();
                          const nextIdx = (index + 1) % filteredRepos.length;
                          setSelectedRepo(filteredRepos[nextIdx].name);
                          repoRefs.current[nextIdx]?.focus();
                        } else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
                          e.preventDefault();
                          const prevIdx = (index - 1 + filteredRepos.length) % filteredRepos.length;
                          setSelectedRepo(filteredRepos[prevIdx].name);
                          repoRefs.current[prevIdx]?.focus();
                        } else if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          setSelectedRepo(repo.name);
                        }
                      }}
                      className={`p-3 rounded-sm border cursor-pointer transition-all flex items-center justify-between font-mono text-xs focus:outline-none focus:ring-1 focus:ring-black ${
                        isSelected
                          ? 'border-black bg-slate-50'
                          : 'border-slate-200 hover:border-slate-300 bg-white'
                      }`}
                    >
                      <div className="flex items-center gap-2.5">
                        <div
                          className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center ${
                            isSelected ? 'border-black bg-black' : 'border-slate-300'
                          }`}
                        >
                          {isSelected && <div className="w-1.5 h-1.5 bg-white rounded-full"></div>}
                        </div>
                        <div>
                          <span className="font-bold text-slate-900">{repo.name}</span>
                          <span className="text-[11px] text-slate-500 ml-2">({repo.branch})</span>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 text-[11px]">
                        <span className="bg-slate-100 text-slate-700 px-2 py-0.5 rounded-sm border border-slate-200">
                          {repo.lang}
                        </span>
                        <span
                          className={`px-2 py-0.5 rounded-sm border ${
                            repo.visibility === 'Public'
                              ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                              : 'bg-slate-100 text-slate-600 border-slate-200'
                          }`}
                        >
                          {repo.visibility}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setCurrentStep(2)}
                className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2 px-4 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                <span>Continue to Step 2 (GitHub App Setup)</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}

        {currentStep === 2 && (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-bold text-slate-900 font-sans">
                GitHub App Authorization & AST Webhook Ingress
              </h2>
              <p className="text-xs text-slate-500 font-sans mt-0.5">
                Targeting <span className="font-mono text-slate-900 font-bold">{selectedRepo}</span>
              </p>
            </div>

            {/* Authorization Banner */}
            <div className="bg-emerald-50 border border-emerald-200 p-4 rounded-sm space-y-2">
              <div className="flex items-center gap-2 text-emerald-900 font-mono text-xs font-bold">
                <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                <span>GitHub App Authorized for org: algotyrnt</span>
              </div>
              <p className="text-xs font-mono text-emerald-800 leading-relaxed">
                Triage GitHub App installed with read-only tree permissions. Webhook listener
                configured at:
                <code className="block mt-1 p-1.5 bg-white border border-emerald-200 rounded-sm text-[11px] text-slate-900 font-mono">
                  https://api.triage.dev/v1/github/webhook/wh_algotyrnt_beacon
                </code>
              </p>
            </div>

            {/* Scope details */}
            <div className="bg-slate-50 border border-slate-200 p-3 rounded-sm space-y-1.5 font-mono text-xs">
              <div className="font-bold text-slate-800">Granted Scopes & Webhook Triggers:</div>
              <div className="text-slate-600 text-[11px] space-y-1">
                <div>
                  • <span className="font-semibold text-slate-900">push:</span> Automatically
                  re-indexes Go AST nodes on git push
                </div>
                <div>
                  • <span className="font-semibold text-slate-900">issues:write:</span>{' '}
                  Automatically links symbolicated panic crashes to GitHub Issues
                </div>
                <div>
                  • <span className="font-semibold text-slate-900">pull_requests:write:</span>{' '}
                  Enables Gemini automated patch generation comments
                </div>
              </div>
            </div>

            <div className="flex justify-between items-center pt-2">
              <button
                onClick={() => setCurrentStep(1)}
                className="bg-slate-100 text-slate-700 hover:bg-slate-200 font-mono text-xs py-2 px-3 rounded-sm border border-slate-200"
              >
                Back to Repositories
              </button>
              <button
                onClick={() => setCurrentStep(3)}
                className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2 px-4 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                <span>Generate Ingestion API Key</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}

        {currentStep === 3 && (
          <div className="space-y-4">
            <div>
              <h2 className="text-sm font-bold text-slate-900 font-sans">
                SDK Telemetry Key Generated
              </h2>
              <p className="text-xs text-slate-500 font-sans mt-0.5">
                Use this API key in your Go application initialization script (`triage.Middleware`).
              </p>
            </div>

            {/* Key Box */}
            <div className="bg-slate-900 text-slate-100 p-4 rounded-sm font-mono space-y-2 border border-slate-800">
              <div className="flex items-center justify-between text-xs text-slate-400">
                <span className="flex items-center gap-1.5">
                  <Key className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Production Telemetry Key (Repo: {selectedRepo})</span>
                </span>
                <span className="text-[10px] text-emerald-400 font-bold">STATUS: ACTIVE</span>
              </div>

              <div className="flex items-center justify-between gap-2 bg-black p-2.5 rounded-sm border border-slate-800">
                <code className="text-xs text-emerald-400 font-bold tracking-wide select-all break-all">
                  {activeKey}
                </code>
                <button
                  onClick={handleCopy}
                  className="bg-slate-800 hover:bg-slate-700 text-white text-xs px-2.5 py-1 rounded-sm border border-slate-700 flex items-center gap-1 shrink-0 font-mono"
                >
                  {copiedKey ? (
                    <Check className="w-3 h-3 text-emerald-400" />
                  ) : (
                    <Copy className="w-3 h-3" />
                  )}
                  <span>{copiedKey ? 'Copied!' : 'Copy'}</span>
                </button>
              </div>
            </div>

            {/* Go Code snippet */}
            <div className="space-y-1.5 font-mono">
              <div className="text-xs font-bold text-slate-800">
                Go SDK QuickStart Initialization:
              </div>
              <pre className="bg-slate-900 text-slate-100 p-3 rounded-sm text-[11px] overflow-x-auto border border-slate-800 leading-relaxed">
                {`package main

import (
	"net/http"
	triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
	mux := http.NewServeMux()

	// Wrap HTTP multiplexer with triage panic recovery middleware
	telemetryURL := os.Getenv("TRIAGE_ENGINE_URL")
	handler := triage.Middleware("${activeKey}", triage.WithGatewayURL(telemetryURL))(mux)

	http.ListenAndServe(":8081", handler)
}`}
              </pre>
            </div>

            <div className="flex justify-between items-center pt-2">
              <button
                onClick={() => setCurrentStep(2)}
                className="bg-slate-100 text-slate-700 hover:bg-slate-200 font-mono text-xs py-2 px-3 rounded-sm border border-slate-200"
              >
                Back
              </button>
              <button
                onClick={handleCompleteSetup}
                className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2.5 px-5 rounded-sm transition-colors flex items-center gap-2 cursor-pointer"
              >
                <span>Complete Setup & Open Dashboard</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
