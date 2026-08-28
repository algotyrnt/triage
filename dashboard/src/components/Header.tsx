/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useRef, useEffect } from 'react';
import { ScreenId, Project } from '@/types';
import {
  Terminal,
  Settings,
  Users,
  PlusCircle,
  LogIn,
  GitBranch,
  FolderGit2,
  ChevronDown,
  Check,
  Search,
  Layers,
} from 'lucide-react';

interface HeaderProps {
  currentScreen: ScreenId;
  onNavigate: (screen: ScreenId) => void;
  openCount?: number;
  criticalCount?: number;
  activeRepo?: string;
  activeRootDir?: string;
  projects?: Project[];
  onSelectProject?: (project: Project, targetScreen?: ScreenId) => void;
  currentUser?: { username: string; avatarUrl?: string; role?: string } | null;
  onLogout?: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  currentScreen,
  onNavigate,
  openCount,
  criticalCount,
  activeRepo = 'algotyrnt/triage',
  activeRootDir = '',
  projects = [],
  onSelectProject,
  currentUser,
  onLogout,
}) => {
  const displayOpenCount = openCount !== undefined ? openCount : criticalCount || 0;
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [switcherSearch, setSwitcherSearch] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const navItems: { id: ScreenId; label: string; icon: React.ReactNode }[] = [
    {
      id: 'projects',
      label: 'Projects',
      icon: <FolderGit2 className="w-3.5 h-3.5" />,
    },
    {
      id: 'dashboard',
      label: 'Dashboard',
      icon: <Terminal className="w-3.5 h-3.5" />,
    },
    {
      id: 'team',
      label: 'Team',
      icon: <Users className="w-3.5 h-3.5" />,
    },
    {
      id: 'settings',
      label: 'Settings',
      icon: <Settings className="w-3.5 h-3.5" />,
    },
  ];

  const filteredProjects = projects.filter((p) => {
    const query = switcherSearch.toLowerCase().trim();
    if (!query) return true;
    const slug = `${p.owner}/${p.repo}`.toLowerCase();
    const root = (p.root_dir || '').toLowerCase();
    return slug.includes(query) || root.includes(query);
  });

  return (
    <header className="bg-white border-b border-slate-200 sticky top-0 z-40">
      {/* Top Bar Metadata */}
      <div className="max-w-7xl mx-auto px-4 py-2.5 flex items-center justify-between border-b border-slate-100 text-xs">
        {/* Brand Block & Project Switcher */}
        <div className="flex items-center gap-3">
          <button
            onClick={() => onNavigate('projects')}
            className="flex items-center gap-2 text-black font-mono font-bold tracking-tight hover:opacity-80 transition-opacity cursor-pointer"
            title="Go to All Projects"
          >
            <div className="bg-black text-white px-2 py-0.5 rounded-sm font-mono text-xs tracking-wider">
              [TRIAGE]
            </div>
            <span className="text-slate-900 font-semibold text-xs hidden sm:inline">
              Go Crash &amp; AST Engine
            </span>
          </button>

          <span className="text-slate-300">/</span>

          {/* Interactive Project Switcher Dropdown */}
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setDropdownOpen(!dropdownOpen)}
              className="flex items-center gap-1.5 font-mono text-slate-700 bg-slate-100 hover:bg-slate-200 border border-slate-200 px-2 py-1 rounded-sm text-[11px] transition-colors cursor-pointer"
              title="Switch Project"
            >
              <GitBranch className="w-3 h-3 text-slate-500 shrink-0" />
              <span className="font-bold text-slate-900 max-w-40 sm:max-w-55 truncate">
                {activeRepo || 'Select Project'}
              </span>
              {activeRootDir && (
                <span className="text-indigo-600 font-medium hidden md:inline truncate max-w-25">
                  ({activeRootDir})
                </span>
              )}
              <span className="text-slate-400 hidden sm:inline">:main</span>
              <ChevronDown className="w-3 h-3 text-slate-500 shrink-0 ml-0.5" />
            </button>

            {/* Dropdown Menu */}
            {dropdownOpen && (
              <div className="absolute left-0 mt-1.5 w-72 sm:w-80 bg-white border border-slate-200 rounded-sm shadow-xl z-50 animate-in fade-in slide-in-from-top-2 font-mono text-xs">
                {/* Search Box */}
                {projects.length > 3 && (
                  <div className="p-2 border-b border-slate-100">
                    <div className="relative">
                      <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2 top-1/2 -translate-y-1/2" />
                      <input
                        type="text"
                        placeholder="Search projects..."
                        value={switcherSearch}
                        onChange={(e) => setSwitcherSearch(e.target.value)}
                        className="w-full bg-slate-50 border border-slate-200 text-slate-900 pl-7 pr-2 py-1 rounded-sm text-[11px] focus:outline-none focus:ring-1 focus:ring-black"
                        autoFocus
                      />
                    </div>
                  </div>
                )}

                {/* Project List */}
                <div className="max-h-60 overflow-y-auto divide-y divide-slate-100">
                  <div className="px-3 py-1.5 text-[10px] uppercase tracking-wider text-slate-400 font-bold bg-slate-50">
                    Connected Projects ({projects.length})
                  </div>

                  {filteredProjects.length === 0 ? (
                    <div className="p-3 text-center text-slate-500 text-[11px]">
                      No projects found
                    </div>
                  ) : (
                    filteredProjects.map((p) => {
                      const slug = `${p.owner}/${p.repo}`;
                      const isCurrent =
                        activeRepo === slug && (p.root_dir || '') === (activeRootDir || '');

                      return (
                        <button
                          key={p.id || `${slug}_${p.root_dir || ''}`}
                          onClick={() => {
                            if (onSelectProject) {
                              onSelectProject(p, 'dashboard');
                            }
                            setDropdownOpen(false);
                          }}
                          className={`w-full text-left p-2.5 hover:bg-slate-50 flex items-center justify-between transition-colors cursor-pointer ${
                            isCurrent ? 'bg-slate-50 font-bold' : ''
                          }`}
                        >
                          <div className="min-w-0 pr-2">
                            <div className="flex items-center gap-1 text-slate-900 truncate font-semibold">
                              <span className="truncate">{slug}</span>
                            </div>
                            <div className="flex items-center gap-1 text-[10px] text-slate-500 mt-0.5">
                              <Layers className="w-2.5 h-2.5 text-slate-400 shrink-0" />
                              <span className="truncate">
                                {p.root_dir ? `Module: ${p.root_dir}/` : 'Root: /'}
                              </span>
                            </div>
                          </div>

                          {isCurrent && (
                            <Check className="w-3.5 h-3.5 text-emerald-600 shrink-0 ml-1" />
                          )}
                        </button>
                      );
                    })
                  )}
                </div>

                {/* Dropdown Actions */}
                <div className="p-2 border-t border-slate-100 bg-slate-50 flex items-center justify-between gap-2">
                  <button
                    onClick={() => {
                      onNavigate('projects');
                      setDropdownOpen(false);
                    }}
                    className="flex items-center gap-1 text-[11px] text-slate-700 hover:text-black font-semibold cursor-pointer"
                  >
                    <FolderGit2 className="w-3 h-3 text-slate-500" />
                    <span>All Projects</span>
                  </button>

                  <button
                    onClick={() => {
                      onNavigate('new');
                      setDropdownOpen(false);
                    }}
                    className="flex items-center gap-1 text-[11px] bg-black text-white hover:bg-slate-800 px-2 py-1 rounded-sm cursor-pointer"
                  >
                    <PlusCircle className="w-3 h-3" />
                    <span>New Project</span>
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => onNavigate('new')}
            className="flex items-center gap-1.5 bg-slate-100 text-slate-800 hover:bg-slate-200 border border-slate-200 text-xs font-mono px-2 py-1 rounded-sm transition-colors cursor-pointer"
          >
            <PlusCircle className="w-3 h-3 text-slate-600" />
            <span className="hidden md:inline">Setup Project</span>
          </button>

          {currentUser ? (
            <div className="flex items-center gap-2 bg-slate-100 border border-slate-200 px-2 py-1 rounded-sm font-mono text-xs text-slate-800">
              <span className="font-semibold">@{currentUser.username}</span>
              {currentUser.role && (
                <span
                  className={`text-[9px] uppercase font-bold px-1.5 py-0.5 rounded-sm ${
                    currentUser.role.toLowerCase() === 'owner'
                      ? 'bg-purple-100 text-purple-800 border border-purple-200'
                      : currentUser.role.toLowerCase() === 'admin'
                        ? 'bg-blue-100 text-blue-800 border border-blue-200'
                        : currentUser.role.toLowerCase() === 'developer'
                          ? 'bg-emerald-100 text-emerald-800 border border-emerald-200'
                          : 'bg-slate-200 text-slate-700'
                  }`}
                >
                  {currentUser.role}
                </span>
              )}
              {onLogout && (
                <button
                  onClick={onLogout}
                  className="text-slate-400 hover:text-red-600 transition-colors text-[10px] cursor-pointer"
                  title="Log out"
                >
                  Logout
                </button>
              )}
            </div>
          ) : (
            <button
              onClick={() => onNavigate('login')}
              className="flex items-center gap-1.5 bg-black text-white hover:bg-slate-800 text-xs font-mono px-2.5 py-1 rounded-sm transition-colors cursor-pointer"
            >
              <LogIn className="w-3 h-3" />
              <span className="hidden sm:inline">GitHub Login</span>
            </button>
          )}
        </div>
      </div>

      {/* Screen Navigation Tabs */}
      <div className="max-w-7xl mx-auto px-4 flex items-center gap-1 overflow-x-auto scrollbar-none py-1">
        {navItems.map((item) => {
          const isActive =
            currentScreen === item.id ||
            (item.id === 'dashboard' && currentScreen === 'incident_detail');
          return (
            <button
              key={item.id}
              onClick={() => onNavigate(item.id)}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium border-b-2 transition-all whitespace-nowrap cursor-pointer ${
                isActive
                  ? 'border-black text-black font-semibold bg-slate-50'
                  : 'border-transparent text-slate-600 hover:text-slate-900 hover:bg-slate-50'
              }`}
            >
              <span className={isActive ? 'text-black' : 'text-slate-400'}>{item.icon}</span>
              <span>{item.label}</span>
              {item.id === 'dashboard' && displayOpenCount > 0 && (
                <span className="bg-red-600 text-white text-[10px] font-mono px-1.5 py-0.2 rounded-full ml-1 font-bold">
                  {displayOpenCount}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </header>
  );
};
