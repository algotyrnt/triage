/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useMemo } from 'react';
import { Incident, Project, ScreenId } from '@/types';
import { engineClient } from '@/services/engineClient';
import {
  Key,
  GitBranch,
  Code2,
  Activity,
  Check,
  AlertTriangle,
  ArrowRight,
  Terminal,
  FolderGit2,
  Search,
  ExternalLink,
  Settings,
  Sparkles,
} from 'lucide-react';

interface DashboardPageProps {
  incidents: Incident[];
  onSelectIncident: (id: string) => void;
  onNavigate: (screen: ScreenId) => void;
  activeRepo?: string;
  rootDir?: string;
  apiKey?: string;
  projects?: Project[];
}

export const DashboardPage: React.FC<DashboardPageProps> = ({
  incidents,
  onSelectIncident,
  onNavigate,
  activeRepo = 'algotyrnt/triage',
  rootDir = '',
  apiKey: rawApiKey,
  projects = [],
}) => {
  const [searchFilter, setSearchFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<'ALL' | 'OPEN' | 'RESOLVED'>('OPEN');
  const [severityFilter, setSeverityFilter] = useState<'ALL' | 'CRITICAL' | 'HIGH' | 'MEDIUM'>(
    'ALL',
  );
  const [stats, setStats] = useState<{
    total_incidents: number;
    funcs_indexed: number;
    total_projects: number;
  } | null>(null);

  const maskedKeyDisplay = rawApiKey
    ? rawApiKey.startsWith('...')
      ? `••••••••••••${rawApiKey.replace('...', '')}`
      : `••••••••••••${rawApiKey.slice(-4)}`
    : '••••••••••••••••';

  useEffect(() => {
    engineClient.getStats().then((data) => {
      if (data) setStats(data);
    });
  }, [incidents.length]);

  // Find active project metadata
  const activeProject = useMemo(() => {
    if (!projects || projects.length === 0) return undefined;
    const cleanRepo = activeRepo.toLowerCase();
    const cleanRoot = (rootDir || '').toLowerCase();
    return (
      projects.find((p) => {
        const slug = `${p.owner}/${p.repo}`.toLowerCase();
        const pRoot = (p.root_dir || '').toLowerCase();
        return (slug === cleanRepo || p.repo.toLowerCase() === cleanRepo) && pRoot === cleanRoot;
      }) ||
      projects.find((p) => {
        const slug = `${p.owner}/${p.repo}`.toLowerCase();
        return slug === cleanRepo || p.repo.toLowerCase() === cleanRepo;
      })
    );
  }, [projects, activeRepo, rootDir]);

  // Scope incidents to only those belonging to the selected service/project
  const serviceIncidents = useMemo(() => {
    const cleanActiveRepo = activeRepo.toLowerCase();
    const activeRepoName = cleanActiveRepo.includes('/')
      ? cleanActiveRepo.split('/')[1]
      : cleanActiveRepo;
    const cleanRootDir = (rootDir || '').replace(/^\/+|\/+$/g, '').toLowerCase();

    return incidents.filter((incident) => {
      // 1. Direct ID match if project ID is available
      if (activeProject?.id && incident.repositoryId) {
        return incident.repositoryId === activeProject.id;
      }

      // 2. If incident has repositoryId, check if it matches another known project
      if (incident.repositoryId && projects && projects.length > 0) {
        const matchedProj = projects.find((p) => p.id === incident.repositoryId);
        if (matchedProj) {
          const projSlug = `${matchedProj.owner}/${matchedProj.repo}`.toLowerCase();
          const projRoot = (matchedProj.root_dir || '').replace(/^\/+|\/+$/g, '').toLowerCase();
          const repoMatches =
            projSlug === cleanActiveRepo ||
            matchedProj.repo.toLowerCase() === cleanActiveRepo ||
            matchedProj.repo.toLowerCase() === activeRepoName;
          const rootMatches = projRoot === cleanRootDir;
          return repoMatches && rootMatches;
        }
      }

      // 3. Match by repositoryName
      if (incident.repositoryName) {
        const incRepo = incident.repositoryName.toLowerCase();
        const repoMatches = incRepo === cleanActiveRepo || incRepo === activeRepoName;
        if (!repoMatches) return false;
      }

      // 4. For monorepo services with rootDir, check triggering file path prefix
      if (cleanRootDir) {
        const trigFile = (incident.triggeringFile || incident.astSnippet?.file || '').toLowerCase();
        if (trigFile && !trigFile.startsWith(cleanRootDir + '/') && trigFile !== cleanRootDir) {
          return false;
        }
      }

      return true;
    });
  }, [incidents, activeProject, activeRepo, rootDir, projects]);

  const filteredIncidents = useMemo(() => {
    return serviceIncidents.filter((incident) => {
      if (statusFilter !== 'ALL' && incident.status !== statusFilter) {
        return false;
      }
      if (severityFilter !== 'ALL' && incident.severity !== severityFilter) {
        return false;
      }
      if (searchFilter.trim()) {
        const q = searchFilter.toLowerCase();
        return (
          incident.id.toLowerCase().includes(q) ||
          incident.title.toLowerCase().includes(q) ||
          incident.triggeringFile.toLowerCase().includes(q) ||
          (incident.panicMessage && incident.panicMessage.toLowerCase().includes(q))
        );
      }
      return true;
    });
  }, [serviceIncidents, statusFilter, severityFilter, searchFilter]);

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Breadcrumb Header + Status */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500">
            <button
              onClick={() => onNavigate('projects')}
              className="text-slate-500 hover:text-black hover:underline cursor-pointer flex items-center gap-1"
              title="Back to All Projects"
            >
              <FolderGit2 className="w-3 h-3" />
              <span>Projects</span>
            </button>
            <span>/</span>
            <span className="font-bold text-slate-900">{activeRepo}</span>
            {rootDir && (
              <>
                <span>/</span>
                <span className="text-indigo-600 font-semibold">{rootDir}</span>
              </>
            )}
          </div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans mt-1">
            System Overview & Ingestion Dashboard
          </h1>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => onNavigate('projects')}
            className="text-xs font-mono bg-white hover:bg-slate-50 text-slate-700 border border-slate-300 px-2.5 py-1 rounded-sm transition-colors flex items-center gap-1.5 shadow-xs cursor-pointer"
            title="View all monitored projects"
          >
            <FolderGit2 className="w-3 h-3 text-slate-500" />
            <span>All Projects</span>
          </button>
          <button
            onClick={() => onNavigate('settings')}
            className="text-xs font-mono bg-slate-100 hover:bg-slate-200 text-slate-800 border border-slate-200 px-2.5 py-1 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer"
            title="Project Settings"
          >
            <Settings className="w-3 h-3 text-slate-600" />
            <span>Settings</span>
          </button>
        </div>
      </div>

      {/* 4-Column Metric Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Metric 1: API Key */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <Key className="w-3.5 h-3.5 text-slate-700" />
              <span>Ingestion Key</span>
            </span>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900 truncate">
            {maskedKeyDisplay}
          </div>
          <div className="text-[11px] font-mono text-slate-500 flex items-center justify-between">
            <span>Env: Production</span>
            <span className="text-emerald-600 font-semibold">Active</span>
          </div>
        </div>

        {/* Metric 2: Monitored Repo */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <GitBranch className="w-3.5 h-3.5 text-slate-700" />
              <span>Monitored Repo</span>
            </span>
            <span className="text-[10px] bg-slate-100 border border-slate-200 px-1.5 py-0.2 rounded-sm text-slate-700 font-mono">
              {rootDir ? 'Monorepo' : 'Root AST'}
            </span>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900 truncate">{activeRepo}</div>
          <div className="text-[11px] font-mono text-slate-500 flex items-center justify-between">
            <span>{rootDir ? `Subdir: ${rootDir}/` : 'Root: /'}</span>
            <span className="text-slate-800 font-mono">Go 1.26+</span>
          </div>
        </div>

        {/* Metric 3: Indexed Functions */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <Code2 className="w-3.5 h-3.5 text-slate-700" />
              <span>Monitored Functions</span>
            </span>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900">
            {stats ? `${stats.funcs_indexed.toLocaleString()} Funcs Indexed` : '— Loading...'}
          </div>
          <div className="text-[11px] font-mono text-emerald-600 flex items-center gap-1">
            <Check className="w-3 h-3" />
            <span>100% Up-to-Date (Commit 8f3a1b4)</span>
          </div>
        </div>

        {/* Metric 4: Total Dispatches */}
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1.5">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1">
              <Activity className="w-3.5 h-3.5 text-slate-700" />
              <span>Service Dispatches</span>
            </span>
            <span className="text-[10px] bg-red-50 text-red-700 border border-red-200 px-1.5 py-0.2 rounded-sm font-mono font-bold">
              {serviceIncidents.filter((i) => i.status === 'OPEN').length} Open
            </span>
          </div>
          <div className="font-mono text-xs font-bold text-slate-900">
            {`${serviceIncidents.length} Telemetry Events`}
          </div>
          <div className="text-[11px] font-mono text-slate-500 truncate">
            Scope:{' '}
            <span className="text-slate-800 font-semibold">
              {activeRepo}
              {rootDir ? ` (${rootDir})` : ''}
            </span>
          </div>
        </div>
      </div>

      {/* Full-Width Incidents & Live Telemetry Dispatch Table */}
      <div className="bg-white border border-slate-200 rounded-sm overflow-hidden shadow-xs">
        {/* Header & Controls */}
        <div className="bg-slate-100/80 border-b border-slate-200 p-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-3 font-mono">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-red-600 shrink-0" />
            <span className="text-xs font-bold text-slate-900">
              Live Panic Telemetry & Incident Dispatches
            </span>
            <span className="text-[11px] text-slate-500">
              ({filteredIncidents.length} of {serviceIncidents.length})
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {/* Status Filter Tabs */}
            <div className="flex items-center bg-white border border-slate-200 rounded-sm p-0.5 text-xs">
              {(['ALL', 'OPEN', 'RESOLVED'] as const).map((status) => (
                <button
                  key={status}
                  type="button"
                  onClick={() => setStatusFilter(status)}
                  className={`px-2 py-0.5 rounded-sm transition-colors text-[10px] font-mono font-medium cursor-pointer ${
                    statusFilter === status
                      ? 'bg-black text-white font-bold'
                      : 'text-slate-600 hover:text-black'
                  }`}
                >
                  {status}
                </button>
              ))}
            </div>

            {/* Severity Filter Tabs */}
            <div className="flex items-center bg-white border border-slate-200 rounded-sm p-0.5 text-xs">
              {(['ALL', 'CRITICAL', 'HIGH', 'MEDIUM'] as const).map((sev) => (
                <button
                  key={sev}
                  type="button"
                  onClick={() => setSeverityFilter(sev)}
                  className={`px-2 py-0.5 rounded-sm transition-colors text-[10px] font-mono font-medium cursor-pointer ${
                    severityFilter === sev
                      ? 'bg-black text-white font-bold'
                      : 'text-slate-600 hover:text-black'
                  }`}
                >
                  {sev}
                </button>
              ))}
            </div>

            {/* Search Filter */}
            <div className="relative">
              <Search className="w-3 h-3 absolute left-2.5 top-2 text-slate-400" />
              <input
                type="text"
                value={searchFilter}
                onChange={(e) => setSearchFilter(e.target.value)}
                placeholder="Filter incidents..."
                className="pl-7 pr-2.5 py-1 bg-white border border-slate-200 rounded-sm text-xs font-mono focus:outline-none focus:border-black"
              />
            </div>
          </div>
        </div>

        {/* Table Content */}
        {filteredIncidents.length === 0 ? (
          <div className="p-12 text-center space-y-3 font-mono">
            <div className="w-12 h-12 mx-auto rounded-full bg-slate-100 flex items-center justify-center border border-slate-200">
              <Terminal className="w-6 h-6 text-slate-400" />
            </div>
            <div className="space-y-1">
              <h3 className="text-sm font-bold text-slate-800">
                {serviceIncidents.length === 0 ? 'No Incidents Recorded' : 'No Matching Incidents'}
              </h3>
              <p className="text-xs text-slate-500 max-w-md mx-auto font-sans leading-normal">
                {serviceIncidents.length === 0 ? (
                  <>
                    No panics recorded for this service. When runtime panics occur, real-time crash
                    diagnostics and AST symbolication traces will appear here.
                  </>
                ) : (
                  'Try adjusting your status or search filters above.'
                )}
              </p>
            </div>
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {filteredIncidents.map((incident) => {
              return (
                <div
                  key={incident.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => onSelectIncident(incident.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      onSelectIncident(incident.id);
                    }
                  }}
                  className="p-4 hover:bg-slate-50 transition-colors cursor-pointer group flex flex-col sm:flex-row sm:items-start justify-between gap-4 font-mono text-xs focus:outline-none focus:bg-slate-50"
                >
                  <div className="space-y-2 flex-1 min-w-0">
                    {/* Header: Title + Badges */}
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-bold text-slate-900 group-hover:underline text-sm font-sans line-clamp-1">
                        {incident.title}
                      </span>
                      {(incident.occurrenceCount ?? 1) > 1 && (
                        <span className="text-[10px] font-bold px-1.5 py-0.2 rounded-sm bg-purple-50 text-purple-700 border border-purple-200">
                          {incident.occurrenceCount}x
                        </span>
                      )}
                      {incident.status === 'RESOLVED' ? (
                        <span className="text-[10px] font-bold px-2 py-0.5 rounded-sm border bg-emerald-50 text-emerald-700 border-emerald-200">
                          RESOLVED
                        </span>
                      ) : (
                        <span
                          className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm border ${
                            incident.severity === 'CRITICAL'
                              ? 'bg-rose-50 text-rose-700 border-rose-200'
                              : incident.severity === 'HIGH'
                                ? 'bg-orange-50 text-orange-700 border-orange-200'
                                : incident.severity === 'MEDIUM'
                                  ? 'bg-blue-50 text-blue-700 border-blue-200'
                                  : 'bg-red-50 text-red-700 border-red-200'
                          }`}
                        >
                          {incident.severity || 'OPEN'}
                        </span>
                      )}
                      {incident.githubIssueNumber && (
                        <span className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded-sm bg-emerald-50 text-emerald-700 border border-emerald-200 flex items-center gap-1">
                          Issue #{incident.githubIssueNumber}
                        </span>
                      )}
                      {incident.githubPrNumber && (
                        <span className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded-sm bg-blue-50 text-blue-700 border border-blue-200 flex items-center gap-1">
                          PR #{incident.githubPrNumber}
                        </span>
                      )}
                    </div>

                    {/* Context Row: Function / Crash Site + Root Cause Preview */}
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-600 font-sans">
                      <div className="flex items-center gap-1 font-mono text-[11px] text-slate-700 bg-slate-100 border border-slate-200 px-1.5 py-0.5 rounded-sm">
                        <Code2 className="w-3 h-3 text-slate-500" />
                        <span>{incident.triggeringFile}</span>
                      </div>

                      {incident.astSnippet?.functionName &&
                        incident.astSnippet.functionName !== 'main' && (
                          <div className="font-mono text-[11px] text-slate-600 bg-slate-50 border border-slate-200 px-1.5 py-0.5 rounded-sm">
                            fn:{' '}
                            <span className="font-semibold text-slate-800">
                              {incident.astSnippet.functionName}()
                            </span>
                          </div>
                        )}

                      {incident.aiAnalysis?.rootCause ? (
                        <div className="text-[11px] text-slate-600 flex items-center gap-1 line-clamp-1">
                          <Sparkles className="w-3 h-3 text-purple-600 shrink-0" />
                          <span className="font-medium text-slate-700">Root Cause:</span>
                          <span className="text-slate-500">{incident.aiAnalysis.rootCause}</span>
                        </div>
                      ) : incident.panicMessage && incident.panicMessage !== incident.title ? (
                        <div className="text-[11px] text-slate-500 line-clamp-1 font-mono">
                          {incident.panicMessage}
                        </div>
                      ) : null}
                    </div>
                  </div>

                  {/* Right Meta Column */}
                  <div className="flex items-center gap-4 text-xs text-slate-500 shrink-0 sm:pt-1">
                    <span className="font-mono text-[11px]">{incident.latencyMs}ms</span>
                    <span className="text-slate-400 font-mono text-[11px]">
                      {incident.timestamp}
                    </span>
                    <ArrowRight className="w-3.5 h-3.5 text-slate-400 group-hover:text-black group-hover:translate-x-0.5 transition-all" />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
