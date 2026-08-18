/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Project, Incident, ScreenId } from '@/types';
import {
  FolderGit2,
  GitBranch,
  Key,
  AlertTriangle,
  CheckCircle2,
  PlusCircle,
  Search,
  ArrowRight,
  Copy,
  Check,
  Activity,
  Settings,
  RefreshCw,
  Layers,
  Terminal,
  ShieldCheck,
} from 'lucide-react';

interface ProjectsPageProps {
  projects: Project[];
  incidents: Incident[];
  onSelectProject: (project: Project, targetScreen?: ScreenId) => void;
  onNavigate: (screen: ScreenId) => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

export const ProjectsPage: React.FC<ProjectsPageProps> = ({
  projects,
  incidents,
  onSelectProject,
  onNavigate,
  onRefresh,
  isRefreshing = false,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'critical' | 'healthy'>('all');
  const [copiedKeyId, setCopiedKeyId] = useState<string | null>(null);

  // Helper to get incidents for a specific project
  const getProjectIncidents = (project: Project): Incident[] => {
    const projectRepoSlug = `${project.owner}/${project.repo}`;
    return incidents.filter((inc) => {
      if (inc.repositoryId && project.id && inc.repositoryId === project.id) {
        return true;
      }
      if (
        inc.repositoryName &&
        (inc.repositoryName === projectRepoSlug || inc.repositoryName === project.repo)
      ) {
        return true;
      }
      return false;
    });
  };

  // Helper to get active API key for a project (local storage fallback if masked)
  const getProjectKey = (project: Project): string => {
    const storageKey = `triage_key_${project.owner}_${project.repo}_${project.root_dir || ''}`;
    const local = typeof window !== 'undefined' ? localStorage.getItem(storageKey) : null;
    return local || project.api_key_masked || 'tr_live_...key';
  };

  const handleCopyKey = (e: React.MouseEvent, project: Project) => {
    e.stopPropagation();
    const key = getProjectKey(project);
    navigator.clipboard.writeText(key);
    setCopiedKeyId(project.id || `${project.owner}/${project.repo}`);
    setTimeout(() => setCopiedKeyId(null), 2000);
  };

  // Filtered projects
  const filteredProjects = projects.filter((project) => {
    const projectSlug = `${project.owner}/${project.repo}`.toLowerCase();
    const rootDir = (project.root_dir || '').toLowerCase();
    const query = searchQuery.trim().toLowerCase();

    const matchesSearch =
      !query ||
      projectSlug.includes(query) ||
      project.repo.toLowerCase().includes(query) ||
      project.owner.toLowerCase().includes(query) ||
      rootDir.includes(query);

    if (!matchesSearch) return false;

    const projectIncs = getProjectIncidents(project);
    const criticalCount = projectIncs.filter((i) => i.status === 'CRITICAL').length;

    if (statusFilter === 'critical') {
      return criticalCount > 0;
    }
    if (statusFilter === 'healthy') {
      return criticalCount === 0;
    }

    return true;
  });

  // Global KPIs
  const totalProjects = projects.length;
  const totalIncidents = incidents.length;
  const criticalIncidents = incidents.filter((i) => i.status === 'CRITICAL').length;

  return (
    <div className="max-w-7xl mx-auto px-4 py-8 space-y-8">
      {/* Top Header & Action Controls */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-200 pb-6">
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-xs font-mono text-slate-500 uppercase tracking-wider">
            <FolderGit2 className="w-3.5 h-3.5 text-slate-700" />
            <span>Workspace Overview</span>
          </div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight font-sans">
            Go Projects &amp; Microservices
          </h1>
          <p className="text-sm text-slate-600 max-w-2xl">
            Select a project to inspect AST crash symbols, stream runtime panics, and dispatch
            Gemini AI automated patches.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {onRefresh && (
            <button
              onClick={onRefresh}
              disabled={isRefreshing}
              className="flex items-center gap-1.5 bg-white hover:bg-slate-50 text-slate-700 border border-slate-300 text-xs font-mono px-3 py-2 rounded-sm transition-colors shadow-sm disabled:opacity-50"
              title="Refresh project list"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          )}

          <button
            onClick={() => onNavigate('new')}
            className="flex items-center gap-2 bg-black hover:bg-slate-800 text-white text-xs font-mono font-semibold px-4 py-2 rounded-sm transition-all shadow-sm"
          >
            <PlusCircle className="w-4 h-4" />
            <span>Setup New Project</span>
          </button>
        </div>
      </div>

      {/* 4-Column Metric KPI Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Metric 1: Total Projects */}
        <div className="bg-white border border-slate-200 p-4 rounded-sm space-y-2 shadow-xs">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1.5">
              <FolderGit2 className="w-3.5 h-3.5 text-slate-700" />
              <span>Monitored Projects</span>
            </span>
            <span className="text-[10px] bg-slate-100 border border-slate-200 px-1.5 py-0.5 rounded-sm font-mono text-slate-600">
              Active
            </span>
          </div>
          <div className="text-2xl font-bold font-mono text-slate-900">{totalProjects}</div>
          <div className="text-[11px] font-mono text-slate-500">Go 1.22+ AST Instrumented</div>
        </div>

        {/* Metric 2: Panics Recorded */}
        <div className="bg-white border border-slate-200 p-4 rounded-sm space-y-2 shadow-xs">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1.5">
              <Terminal className="w-3.5 h-3.5 text-slate-700" />
              <span>Total Panics Intercepted</span>
            </span>
            <span className="text-[10px] bg-slate-100 border border-slate-200 px-1.5 py-0.5 rounded-sm font-mono text-slate-600">
              Live
            </span>
          </div>
          <div className="text-2xl font-bold font-mono text-slate-900">{totalIncidents}</div>
          <div className="text-[11px] font-mono text-slate-500">Across all active services</div>
        </div>

        {/* Metric 3: Critical Incidents */}
        <div className="bg-white border border-slate-200 p-4 rounded-sm space-y-2 shadow-xs">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1.5">
              <AlertTriangle className="w-3.5 h-3.5 text-red-600" />
              <span>Critical Incidents</span>
            </span>
            <span
              className={`text-[10px] border px-1.5 py-0.5 rounded-sm font-mono font-bold ${
                criticalIncidents > 0
                  ? 'bg-red-50 text-red-700 border-red-200'
                  : 'bg-emerald-50 text-emerald-700 border-emerald-200'
              }`}
            >
              {criticalIncidents > 0 ? 'Needs Attention' : 'All Clear'}
            </span>
          </div>
          <div className="text-2xl font-bold font-mono text-red-600">{criticalIncidents}</div>
          <div className="text-[11px] font-mono text-slate-500">Open unresolved crash events</div>
        </div>

        {/* Metric 4: Core Engine Status */}
        <div className="bg-white border border-slate-200 p-4 rounded-sm space-y-2 shadow-xs">
          <div className="flex items-center justify-between text-xs font-mono text-slate-500">
            <span className="flex items-center gap-1.5">
              <Activity className="w-3.5 h-3.5 text-emerald-600" />
              <span>Core Ingestion Engine</span>
            </span>
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
          </div>
          <div className="text-2xl font-bold font-mono text-slate-900">Operational</div>
          <div className="text-[11px] font-mono text-emerald-600 flex items-center gap-1">
            <ShieldCheck className="w-3 h-3" />
            <span>Zero-Overhead Symbolication</span>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-slate-100 p-3 rounded-sm border border-slate-200">
        <div className="relative flex-1 max-w-md">
          <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search projects by repository or Go module path..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-white border border-slate-200 text-xs font-mono pl-9 pr-3 py-2 rounded-sm focus:outline-none focus:ring-1 focus:ring-black placeholder:text-slate-400 text-slate-900"
          />
        </div>

        {/* Filter Pills */}
        <div className="flex items-center gap-1 bg-white border border-slate-200 rounded-sm p-0.5 self-start sm:self-auto">
          <button
            onClick={() => setStatusFilter('all')}
            className={`px-3 py-1 text-xs font-mono rounded-sm transition-colors ${
              statusFilter === 'all'
                ? 'bg-black text-white font-bold'
                : 'text-slate-600 hover:text-black'
            }`}
          >
            All ({projects.length})
          </button>
          <button
            onClick={() => setStatusFilter('critical')}
            className={`px-3 py-1 text-xs font-mono rounded-sm transition-colors ${
              statusFilter === 'critical'
                ? 'bg-black text-white font-bold'
                : 'text-slate-600 hover:text-black'
            }`}
          >
            With Panics (
            {
              projects.filter((p) => getProjectIncidents(p).some((i) => i.status === 'CRITICAL'))
                .length
            }
            )
          </button>
          <button
            onClick={() => setStatusFilter('healthy')}
            className={`px-3 py-1 text-xs font-mono rounded-sm transition-colors ${
              statusFilter === 'healthy'
                ? 'bg-black text-white font-bold'
                : 'text-slate-600 hover:text-black'
            }`}
          >
            Healthy (
            {
              projects.filter((p) => !getProjectIncidents(p).some((i) => i.status === 'CRITICAL'))
                .length
            }
            )
          </button>
        </div>
      </div>

      {/* Projects Grid */}
      {filteredProjects.length === 0 ? (
        <div className="bg-white border border-slate-200 rounded-sm p-12 text-center space-y-4">
          <div className="w-12 h-12 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center mx-auto text-slate-400">
            <FolderGit2 className="w-6 h-6" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-bold text-slate-900 font-sans">
              {searchQuery ? 'No matching projects found' : 'No Go Projects Configured'}
            </h3>
            <p className="text-xs text-slate-500 font-mono max-w-md mx-auto">
              {searchQuery
                ? `No projects matching "${searchQuery}". Try a different search query.`
                : 'Connect your GitHub repository and wrap your HTTP router with Triage Go middleware to start isolating panics.'}
            </p>
          </div>
          <button
            onClick={() => onNavigate('new')}
            className="inline-flex items-center gap-2 bg-black hover:bg-slate-800 text-white text-xs font-mono px-4 py-2 rounded-sm transition-all"
          >
            <PlusCircle className="w-3.5 h-3.5" />
            <span>Setup Your First Project</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {filteredProjects.map((project) => {
            const projectIncs = getProjectIncidents(project);
            const critCount = projectIncs.filter((i) => i.status === 'CRITICAL').length;
            const projectKey = getProjectKey(project);
            const isKeyCopied = copiedKeyId === (project.id || `${project.owner}/${project.repo}`);
            const repoSlug = `${project.owner}/${project.repo}`;

            return (
              <div
                key={project.id || `${project.owner}/${project.repo}/${project.root_dir || ''}`}
                onClick={() => onSelectProject(project, 'dashboard')}
                className="bg-white border border-slate-200 rounded-sm p-5 space-y-4 hover:border-slate-400 hover:shadow-md transition-all cursor-pointer group flex flex-col justify-between"
              >
                {/* Card Top: Repo & Branch */}
                <div className="space-y-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="space-y-1 min-w-0">
                      <div className="flex items-center gap-1.5 font-mono text-xs text-slate-500">
                        <GitBranch className="w-3 h-3 text-slate-400 shrink-0" />
                        <span className="truncate">{project.owner}</span>
                      </div>
                      <h2 className="text-base font-bold text-slate-900 group-hover:text-black group-hover:underline truncate font-mono">
                        {project.repo}
                      </h2>
                    </div>

                    {/* Status Badge */}
                    <span
                      className={`text-[10px] font-mono px-2 py-0.5 rounded-sm border shrink-0 font-bold ${
                        critCount > 0
                          ? 'bg-red-50 text-red-700 border-red-200 animate-pulse'
                          : 'bg-emerald-50 text-emerald-700 border-emerald-200'
                      }`}
                    >
                      {critCount > 0 ? `${critCount} CRITICAL` : 'HEALTHY'}
                    </span>
                  </div>

                  {/* Module / Subdirectory Badge */}
                  <div className="flex items-center gap-2 text-xs font-mono">
                    <div className="flex items-center gap-1 bg-slate-100 border border-slate-200 px-2 py-0.5 rounded-sm text-slate-700 text-[11px] truncate max-w-full">
                      <Layers className="w-3 h-3 text-slate-500 shrink-0" />
                      <span className="truncate">
                        {project.root_dir ? `Module: ${project.root_dir}/` : 'Root: / (Default)'}
                      </span>
                    </div>
                  </div>

                  {/* API Key Box */}
                  <div className="bg-slate-50 border border-slate-200 p-2.5 rounded-sm space-y-1 font-mono">
                    <div className="flex items-center justify-between text-[11px] text-slate-500">
                      <span className="flex items-center gap-1">
                        <Key className="w-3 h-3 text-slate-600" />
                        <span>Telemetry Key</span>
                      </span>
                      <button
                        onClick={(e) => handleCopyKey(e, project)}
                        className="text-slate-600 hover:text-black text-[10px] underline flex items-center gap-0.5 cursor-pointer"
                        title="Copy Ingestion API Key"
                      >
                        {isKeyCopied ? (
                          <Check className="w-3 h-3 text-emerald-600" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                        <span>{isKeyCopied ? 'Copied' : 'Copy'}</span>
                      </button>
                    </div>
                    <div className="text-xs font-bold text-slate-800 truncate select-all">
                      {projectKey}
                    </div>
                  </div>
                </div>

                {/* Card Bottom: Stats & Quick Actions */}
                <div className="pt-3 border-t border-slate-100 space-y-3">
                  <div className="flex items-center justify-between text-[11px] font-mono text-slate-500">
                    <span className="flex items-center gap-1">
                      <Terminal className="w-3 h-3 text-slate-400" />
                      <span>{projectIncs.length} Panics Ingested</span>
                    </span>
                    <span>Go 1.22+</span>
                  </div>

                  {/* Action Buttons */}
                  <div className="grid grid-cols-2 gap-2 pt-1">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectProject(project, 'dashboard');
                      }}
                      className="bg-black hover:bg-slate-800 text-white font-mono text-xs py-1.5 px-2 rounded-sm transition-colors flex items-center justify-center gap-1 shadow-xs cursor-pointer"
                    >
                      <span>Dashboard</span>
                      <ArrowRight className="w-3 h-3" />
                    </button>

                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectProject(project, 'settings');
                      }}
                      className="bg-slate-100 hover:bg-slate-200 text-slate-700 font-mono text-xs py-1.5 px-2 rounded-sm border border-slate-200 transition-colors flex items-center justify-center gap-1 cursor-pointer"
                      title="Project Settings & Keys"
                    >
                      <Settings className="w-3 h-3" />
                      <span>Settings</span>
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
