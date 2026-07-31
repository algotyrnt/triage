/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { ScreenId } from '../types';
import {
  Terminal,
  Activity,
  Code2,
  Webhook,
  Users,
  Settings,
  PlusCircle,
  LogIn,
  AlertTriangle,
  CheckCircle2,
  RefreshCw,
  GitBranch,
} from 'lucide-react';

interface HeaderProps {
  currentScreen: ScreenId;
  onNavigate: (screen: ScreenId) => void;
  onSimulatePanic: () => void;
  criticalCount: number;
}

export const Header: React.FC<HeaderProps> = ({
  currentScreen,
  onNavigate,
  onSimulatePanic,
  criticalCount,
}) => {
  const navItems: { id: ScreenId; label: string; icon: React.ReactNode }[] = [
    { id: 'dashboard', label: 'Dashboard', icon: <Terminal className="w-3.5 h-3.5" /> },
    {
      id: 'incident_detail',
      label: 'Panic Inspector',
      icon: <AlertTriangle className="w-3.5 h-3.5" />,
    },
    { id: 'ast', label: 'AST Index', icon: <Code2 className="w-3.5 h-3.5" /> },
    { id: 'webhooks', label: 'Webhooks', icon: <Webhook className="w-3.5 h-3.5" /> },
    { id: 'team', label: 'Team', icon: <Users className="w-3.5 h-3.5" /> },
    { id: 'status', label: 'Engine Status', icon: <Activity className="w-3.5 h-3.5" /> },
    { id: 'settings', label: 'Settings', icon: <Settings className="w-3.5 h-3.5" /> },
  ];

  return (
    <header className="bg-white border-b border-slate-200 sticky top-0 z-40">
      {/* Top Bar Metadata */}
      <div className="max-w-7xl mx-auto px-4 py-2.5 flex items-center justify-between border-b border-slate-100 text-xs">
        {/* Brand Block */}
        <div className="flex items-center gap-3">
          <button
            onClick={() => onNavigate('dashboard')}
            className="flex items-center gap-2 text-black font-mono font-bold tracking-tight hover:opacity-80 transition-opacity"
          >
            <div className="bg-black text-white px-2 py-0.5 rounded-sm font-mono text-xs tracking-wider">
              [TRIAGE]
            </div>
            <span className="text-slate-900 font-semibold text-xs">Go Crash & AST Engine</span>
          </button>

          <span className="text-slate-300">/</span>

          {/* Org & Repo Selector Pill */}
          <div className="flex items-center gap-1.5 font-mono text-slate-600 bg-slate-100 border border-slate-200 px-2 py-0.5 rounded-sm text-[11px]">
            <GitBranch className="w-3 h-3 text-slate-500" />
            <span className="font-semibold text-slate-800">algotyrnt/beacon-app</span>
            <span className="text-slate-400">:main</span>
          </div>

          {/* Operational Badge */}
          <div className="hidden sm:flex items-center gap-1.5 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-sm text-[11px] font-mono">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-600 animate-pulse"></span>
            <span>Engine Operational</span>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2">
          <button
            onClick={onSimulatePanic}
            className="flex items-center gap-1.5 bg-red-50 text-red-700 hover:bg-red-100 border border-red-200 text-xs font-mono font-medium px-2.5 py-1 rounded-sm transition-colors cursor-pointer"
            title="Simulate dynamic Go runtime panic"
          >
            <RefreshCw className="w-3 h-3 text-red-600" />
            <span>Simulate Go Panic</span>
          </button>

          <button
            onClick={() => onNavigate('new')}
            className="flex items-center gap-1.5 bg-slate-100 text-slate-800 hover:bg-slate-200 border border-slate-200 text-xs font-mono px-2 py-1 rounded-sm transition-colors"
          >
            <PlusCircle className="w-3 h-3 text-slate-600" />
            <span className="hidden md:inline">New Repo</span>
          </button>

          <button
            onClick={() => onNavigate('login')}
            className="flex items-center gap-1.5 bg-black text-white hover:bg-slate-800 text-xs font-mono px-2.5 py-1 rounded-sm transition-colors"
          >
            <LogIn className="w-3 h-3" />
            <span className="hidden sm:inline">Auth</span>
          </button>
        </div>
      </div>

      {/* Screen Navigation Tabs */}
      <div className="max-w-7xl mx-auto px-4 flex items-center gap-1 overflow-x-auto scrollbar-none py-1">
        {navItems.map((item) => {
          const isActive = currentScreen === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onNavigate(item.id)}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium border-b-2 transition-all whitespace-nowrap ${
                isActive
                  ? 'border-black text-black font-semibold bg-slate-50'
                  : 'border-transparent text-slate-600 hover:text-slate-900 hover:bg-slate-50'
              }`}
            >
              <span className={isActive ? 'text-black' : 'text-slate-400'}>{item.icon}</span>
              <span>{item.label}</span>
              {item.id === 'incident_detail' && criticalCount > 0 && (
                <span className="bg-red-600 text-white text-[10px] font-mono px-1.5 py-0.2 rounded-full ml-1 font-bold">
                  {criticalCount}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </header>
  );
};
