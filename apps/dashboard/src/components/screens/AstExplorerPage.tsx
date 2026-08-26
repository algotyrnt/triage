/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useMemo } from 'react';
import { ScreenId } from '@/types';
import { engineClient } from '@/services/engineClient';
import {
  Code2,
  Folder,
  FileCode,
  Check,
  Search,
  Braces,
  Copy,
  RefreshCw,
  FolderGit2,
  ArrowRight,
  Terminal,
  Loader2,
  Zap,
} from 'lucide-react';

interface ASTFunctionSymbol {
  name: string;
  line: number;
  snippet?: string;
}

interface ASTFileRecord {
  path: string;
  name: string;
  package: string;
  functions: ASTFunctionSymbol[];
}

interface AstExplorerPageProps {
  onNavigate: (screen: ScreenId) => void;
  activeRepo?: string;
  activeRootDir?: string;
}

export const AstExplorerPage: React.FC<AstExplorerPageProps> = ({
  onNavigate,
  activeRepo = 'algotyrnt/triage',
  activeRootDir = '',
}) => {
  const [files, setFiles] = useState<ASTFileRecord[]>([]);
  const [selectedFilePath, setSelectedFilePath] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [indexing, setIndexing] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'nodes' | 'raw_json'>('nodes');
  const [copiedSnippet, setCopiedSnippet] = useState<string | null>(null);

  let owner = 'algotyrnt';
  let repo = activeRepo;
  if (activeRepo.includes('/')) {
    const parts = activeRepo.split('/');
    owner = parts[0];
    repo = parts[1];
  }

  const loadASTData = async () => {
    setLoading(true);
    try {
      const res = await engineClient.getASTTree(owner, repo, activeRootDir);
      if (res && res.files) {
        setFiles(res.files);
        if (res.files.length > 0 && !selectedFilePath) {
          setSelectedFilePath(res.files[0].path);
        }
      } else {
        setFiles([]);
      }
    } catch {
      setFiles([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadASTData();
  }, [activeRepo, activeRootDir]);

  const handleReindex = async () => {
    setIndexing(true);
    try {
      await engineClient.indexAST(owner, repo, 'main', activeRootDir);
      await loadASTData();
    } finally {
      setIndexing(false);
    }
  };

  const filteredFiles = useMemo(() => {
    if (!searchQuery.trim()) return files;
    const q = searchQuery.toLowerCase();
    return files.filter(
      (f) =>
        f.path.toLowerCase().includes(q) ||
        f.name.toLowerCase().includes(q) ||
        f.package.toLowerCase().includes(q) ||
        f.functions.some((fn) => fn.name.toLowerCase().includes(q)),
    );
  }, [files, searchQuery]);

  const selectedFile =
    filteredFiles.find((f) => f.path === selectedFilePath) ||
    files.find((f) => f.path === selectedFilePath) ||
    filteredFiles[0] ||
    null;

  const totalFunctionsCount = useMemo(() => {
    return files.reduce((acc, f) => acc + (f.functions ? f.functions.length : 0), 0);
  }, [files]);

  const handleCopy = (snippet: string, key: string) => {
    navigator.clipboard.writeText(snippet);
    setCopiedSnippet(key);
    setTimeout(() => setCopiedSnippet(null), 2000);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Breadcrumb Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500">
            <button
              onClick={() => onNavigate('projects')}
              className="text-slate-500 hover:text-black hover:underline cursor-pointer flex items-center gap-1"
            >
              <FolderGit2 className="w-3 h-3" />
              <span>Projects</span>
            </button>
            <span>/</span>
            <button
              onClick={() => onNavigate('dashboard')}
              className="text-slate-700 hover:text-black font-semibold hover:underline cursor-pointer"
            >
              {activeRepo}
            </button>
            {activeRootDir && (
              <>
                <span>/</span>
                <span className="text-indigo-600 font-semibold">{activeRootDir}</span>
              </>
            )}
          </div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans mt-1">
            Abstract Syntax Tree (AST) Explorer
          </h1>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => onNavigate('dashboard')}
            className="text-xs font-mono bg-white hover:bg-slate-50 text-slate-700 border border-slate-300 px-2.5 py-1 rounded-sm transition-colors cursor-pointer"
          >
            Dashboard
          </button>
          <button
            type="button"
            onClick={handleReindex}
            disabled={indexing}
            className="flex items-center gap-1.5 px-3 py-1 bg-black hover:bg-slate-800 disabled:bg-slate-300 text-white rounded-sm text-xs font-mono font-semibold transition-colors cursor-pointer"
          >
            <RefreshCw className={`w-3 h-3 ${indexing ? 'animate-spin' : ''}`} />
            <span>{indexing ? 'Re-indexing...' : 'Re-index AST'}</span>
          </button>
        </div>
      </div>

      {/* Overview Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1">
          <div className="text-xs font-mono text-slate-500 flex items-center gap-1">
            <Folder className="w-3.5 h-3.5 text-slate-700" />
            <span>Parsed Go Files</span>
          </div>
          <div className="text-lg font-bold text-slate-900 font-mono">
            {loading ? '—' : files.length} Files
          </div>
        </div>

        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1">
          <div className="text-xs font-mono text-slate-500 flex items-center gap-1">
            <Code2 className="w-3.5 h-3.5 text-slate-700" />
            <span>Indexed Functions & Methods</span>
          </div>
          <div className="text-lg font-bold text-slate-900 font-mono">
            {loading ? '—' : totalFunctionsCount} Functions
          </div>
        </div>

        <div className="bg-white border border-slate-200 p-3.5 rounded-sm space-y-1">
          <div className="text-xs font-mono text-slate-500 flex items-center gap-1">
            <Zap className="w-3.5 h-3.5 text-slate-700" />
            <span>AST Symbol Status</span>
          </div>
          <div className="text-xs font-bold text-emerald-600 font-mono flex items-center gap-1 mt-1">
            <Check className="w-3.5 h-3.5" />
            <span>Live AST Engine Synced</span>
          </div>
        </div>
      </div>

      {/* Main Two-Column AST Explorer View */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: File Tree */}
        <div className="lg:col-span-4 bg-white border border-slate-200 rounded-sm p-3 space-y-3 shadow-xs">
          <div className="flex items-center justify-between border-b border-slate-100 pb-2">
            <div className="font-mono text-xs font-bold text-slate-900 flex items-center gap-1.5">
              <Folder className="w-4 h-4 text-slate-700" />
              <span>Project Go Files</span>
            </div>
            <span className="text-[11px] font-mono text-slate-500">
              {filteredFiles.length} of {files.length}
            </span>
          </div>

          {/* Search Filter */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-2 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search files or functions..."
              className="w-full pl-8 pr-2.5 py-1 bg-slate-50 border border-slate-200 rounded-sm text-xs font-mono focus:bg-white focus:outline-none focus:border-black"
            />
          </div>

          {/* Files List */}
          {loading ? (
            <div className="py-12 text-center text-xs font-mono text-slate-500 flex items-center justify-center gap-2">
              <Loader2 className="w-4 h-4 animate-spin text-slate-700" />
              <span>Scanning AST symbols...</span>
            </div>
          ) : filteredFiles.length === 0 ? (
            <div className="p-6 text-center text-xs font-mono text-slate-500 space-y-2">
              <p>No Go files indexed for this repository scope.</p>
              <button
                type="button"
                onClick={handleReindex}
                className="text-black font-bold underline cursor-pointer"
              >
                Trigger AST Indexing
              </button>
            </div>
          ) : (
            <div className="space-y-1 max-h-[550px] overflow-y-auto pr-1">
              {filteredFiles.map((file) => {
                const isSelected = selectedFile?.path === file.path;
                return (
                  <button
                    key={file.path}
                    type="button"
                    onClick={() => setSelectedFilePath(file.path)}
                    className={`w-full text-left p-2 rounded-sm text-xs font-mono transition-colors flex items-center justify-between cursor-pointer ${
                      isSelected
                        ? 'bg-black text-white font-bold shadow-xs'
                        : 'bg-white hover:bg-slate-50 text-slate-800 border border-slate-100'
                    }`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <FileCode
                        className={`w-3.5 h-3.5 shrink-0 ${
                          isSelected ? 'text-emerald-400' : 'text-slate-500'
                        }`}
                      />
                      <div className="truncate">
                        <div className="truncate font-semibold">{file.name}</div>
                        <div
                          className={`text-[10px] truncate ${
                            isSelected ? 'text-slate-300' : 'text-slate-500'
                          }`}
                        >
                          {file.path}
                        </div>
                      </div>
                    </div>
                    {file.functions && file.functions.length > 0 && (
                      <span
                        className={`text-[10px] px-1.5 py-0.5 rounded-sm shrink-0 ml-1.5 ${
                          isSelected ? 'bg-slate-800 text-slate-200' : 'bg-slate-100 text-slate-600'
                        }`}
                      >
                        {file.functions.length} f
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Right Column: AST Functions & Code Inspection */}
        <div className="lg:col-span-8 bg-white border border-slate-200 rounded-sm overflow-hidden flex flex-col justify-between shadow-xs">
          <div>
            {/* Header & Mode Switcher */}
            <div className="bg-slate-100/80 border-b border-slate-200 p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2 font-mono">
              <div className="flex items-center gap-2 text-xs">
                <FileCode className="w-4 h-4 text-slate-800 shrink-0" />
                <span className="font-bold text-slate-900 truncate">
                  {selectedFile ? selectedFile.path : 'Select a Go file'}
                </span>
                {selectedFile?.package && (
                  <span className="text-[10px] bg-slate-200 text-slate-700 px-1.5 py-0.2 rounded-sm">
                    pkg: {selectedFile.package}
                  </span>
                )}
              </div>

              <div className="flex items-center gap-1 bg-white border border-slate-200 rounded-sm p-0.5 text-[11px]">
                <button
                  type="button"
                  onClick={() => setViewMode('nodes')}
                  className={`px-2.5 py-0.5 rounded-sm transition-colors cursor-pointer ${
                    viewMode === 'nodes'
                      ? 'bg-black text-white font-bold'
                      : 'text-slate-600 hover:text-black'
                  }`}
                >
                  Function ASTs ({selectedFile?.functions ? selectedFile.functions.length : 0})
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode('raw_json')}
                  className={`px-2.5 py-0.5 rounded-sm transition-colors cursor-pointer ${
                    viewMode === 'raw_json'
                      ? 'bg-black text-white font-bold'
                      : 'text-slate-600 hover:text-black'
                  }`}
                >
                  Raw JSON
                </button>
              </div>
            </div>

            {/* Content Area */}
            {!selectedFile ? (
              <div className="p-16 text-center space-y-2 font-mono text-xs text-slate-500">
                <Terminal className="w-8 h-8 mx-auto text-slate-400" />
                <p>Select a Go file from the tree to inspect AST symbols.</p>
              </div>
            ) : viewMode === 'nodes' ? (
              <div className="p-4 space-y-4 font-mono text-xs max-h-[600px] overflow-y-auto">
                {selectedFile.functions && selectedFile.functions.length > 0 ? (
                  selectedFile.functions.map((fn, idx) => {
                    const snippetKey = `${selectedFile.path}:${fn.name}:${fn.line}`;
                    return (
                      <div
                        key={`${fn.name}-${idx}`}
                        className="border border-slate-200 rounded-sm bg-slate-50/60 p-3.5 space-y-2.5 hover:border-slate-300 transition-colors"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <span className="bg-black text-white text-[10px] font-bold px-1.5 py-0.5 rounded-sm uppercase tracking-wider">
                              FUNC
                            </span>
                            <span className="font-bold text-slate-900 text-xs">{fn.name}</span>
                          </div>

                          <div className="flex items-center gap-2">
                            <span className="text-[11px] text-slate-500 font-mono">
                              Line {fn.line}
                            </span>
                            {fn.snippet && (
                              <button
                                type="button"
                                onClick={() => handleCopy(fn.snippet || '', snippetKey)}
                                className="text-[11px] text-slate-600 hover:text-black underline flex items-center gap-0.5 cursor-pointer ml-1"
                              >
                                {copiedSnippet === snippetKey ? (
                                  <Check className="w-3 h-3 text-emerald-600" />
                                ) : (
                                  <Copy className="w-3 h-3" />
                                )}
                                <span>{copiedSnippet === snippetKey ? 'Copied' : 'Copy'}</span>
                              </button>
                            )}
                          </div>
                        </div>

                        {fn.snippet ? (
                          <pre className="bg-slate-900 text-slate-100 p-3 rounded-sm text-[11px] overflow-x-auto leading-relaxed border border-slate-800 font-mono">
                            <code>{fn.snippet}</code>
                          </pre>
                        ) : (
                          <div className="text-[11px] text-slate-400 italic">
                            Function declared at line {fn.line}
                          </div>
                        )}
                      </div>
                    );
                  })
                ) : (
                  <div className="p-8 text-center text-xs text-slate-500 space-y-1">
                    <p>No function declarations extracted for this file.</p>
                    <p className="text-[11px] text-slate-400">
                      File may only contain package constants, types, or imports.
                    </p>
                  </div>
                )}
              </div>
            ) : (
              <div className="p-4 bg-slate-900 text-emerald-400 font-mono text-xs overflow-x-auto max-h-[600px]">
                <pre className="text-[11px] leading-relaxed select-all">
                  {JSON.stringify(selectedFile, null, 2)}
                </pre>
              </div>
            )}
          </div>

          <div className="p-3 bg-slate-50 border-t border-slate-200 text-xs font-mono text-slate-500 flex justify-between">
            <span>Parser: Go `go/ast` Symbol Table</span>
            <span>Symbolication Latency: &lt;1ms</span>
          </div>
        </div>
      </div>
    </div>
  );
};
