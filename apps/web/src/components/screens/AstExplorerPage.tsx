/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { AstCommitIndex, AstFile, AstNode, ScreenId } from '../../types';
import {
  Code2,
  Folder,
  FileCode,
  ChevronRight,
  ChevronDown,
  GitCommit,
  Check,
  Search,
  Layers,
  Braces,
  Hash,
} from 'lucide-react';

interface AstExplorerPageProps {
  commitIndexes: AstCommitIndex[];
  astFiles: AstFile[];
  onNavigate: (screen: ScreenId) => void;
}

export const AstExplorerPage: React.FC<AstExplorerPageProps> = ({
  commitIndexes,
  astFiles,
  onNavigate,
}) => {
  const [selectedFilePath, setSelectedFilePath] = useState('pkg/handler/user.go');
  const [expandedFolders, setExpandedFolders] = useState<Record<string, boolean>>({
    pkg: true,
    'pkg/handler': true,
    'pkg/api': true,
  });
  const [viewMode, setViewMode] = useState<'nodes' | 'raw_json'>('nodes');
  const [searchQuery, setSearchQuery] = useState('');

  const toggleFolder = (path: string) => {
    setExpandedFolders((prev) => ({ ...prev, [path]: !prev[path] }));
  };

  // Find selected file
  const findFileByPath = (files: AstFile[], target: string): AstFile | null => {
    for (const f of files) {
      if (f.path === target) return f;
      if (f.children) {
        const found = findFileByPath(f.children, target);
        if (found) return found;
      }
    }
    return null;
  };

  const selectedFile = findFileByPath(astFiles, selectedFilePath);

  // Render tree recursively
  const renderTree = (files: AstFile[], depth = 0) => {
    return files.map((file) => {
      const isExpanded = expandedFolders[file.path];
      const isSelected = selectedFilePath === file.path;

      if (file.isDir) {
        return (
          <div key={file.path} className="font-mono text-xs">
            <div
              onClick={() => toggleFolder(file.path)}
              className="flex items-center gap-1.5 py-1 px-2 hover:bg-slate-100 rounded-sm cursor-pointer text-slate-700 select-none"
              style={{ paddingLeft: `${depth * 12 + 8}px` }}
            >
              {isExpanded ? (
                <ChevronDown className="w-3.5 h-3.5 text-slate-500" />
              ) : (
                <ChevronRight className="w-3.5 h-3.5 text-slate-500" />
              )}
              <Folder className="w-3.5 h-3.5 text-slate-600" />
              <span className="font-bold text-slate-800">{file.name}</span>
            </div>
            {isExpanded && file.children && renderTree(file.children, depth + 1)}
          </div>
        );
      }

      return (
        <div
          key={file.path}
          onClick={() => setSelectedFilePath(file.path)}
          className={`flex items-center justify-between py-1 px-2 text-xs font-mono rounded-sm cursor-pointer select-none ${
            isSelected
              ? 'bg-black text-white font-bold'
              : 'text-slate-700 hover:bg-slate-100 hover:text-black'
          }`}
          style={{ paddingLeft: `${depth * 12 + 20}px` }}
        >
          <div className="flex items-center gap-1.5 truncate">
            <FileCode className={`w-3.5 h-3.5 ${isSelected ? 'text-emerald-400' : 'text-slate-500'}`} />
            <span className="truncate">{file.name}</span>
          </div>
          {file.totalFuncs && (
            <span
              className={`text-[10px] px-1 rounded-sm ${
                isSelected ? 'bg-slate-800 text-slate-300' : 'bg-slate-100 text-slate-500'
              }`}
            >
              {file.totalFuncs} f
            </span>
          )}
        </div>
      );
    });
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
            Abstract Syntax Tree (AST) Repository Index
          </h1>
          <p className="text-xs text-slate-600 font-sans mt-0.5">
            Real-time Go AST node extraction, byte range maps, and package symbol definitions.
          </p>
        </div>

        <div className="flex items-center gap-2 font-mono text-xs">
          <span className="bg-emerald-50 text-emerald-700 border border-emerald-200 px-2.5 py-1 rounded-sm font-medium flex items-center gap-1">
            <Check className="w-3 h-3 text-emerald-600" />
            <span>AST Sync Complete (8f3a1b4)</span>
          </span>
        </div>
      </div>

      {/* Index History Table */}
      <div className="bg-white border border-slate-200 rounded-sm overflow-hidden">
        <div className="bg-slate-100 border-b border-slate-200 p-3 font-mono text-xs font-bold text-slate-900 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GitCommit className="w-4 h-4 text-slate-700" />
            <span>Commit Indexing Log</span>
          </div>
          <span className="text-slate-500 font-normal">3 Indexed Revisions</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left font-mono text-xs">
            <thead className="bg-slate-50 border-b border-slate-200 text-slate-500 text-[11px] uppercase tracking-wider">
              <tr>
                <th className="py-2 px-3 font-semibold">Commit</th>
                <th className="py-2 px-3 font-semibold">Branch</th>
                <th className="py-2 px-3 font-semibold">Parsed Files</th>
                <th className="py-2 px-3 font-semibold">Total Functions</th>
                <th className="py-2 px-3 font-semibold">Indexed At</th>
                <th className="py-2 px-3 font-semibold text-right">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-slate-800">
              {commitIndexes.map((idx) => (
                <tr key={idx.commitHash} className="hover:bg-slate-50">
                  <td className="py-2 px-3 font-bold text-slate-900 flex items-center gap-1.5">
                    <GitCommit className="w-3 h-3 text-slate-500" />
                    <span>{idx.commitHash}</span>
                  </td>
                  <td className="py-2 px-3 text-slate-600">{idx.branch}</td>
                  <td className="py-2 px-3">{idx.parsedFilesCount} files</td>
                  <td className="py-2 px-3 font-semibold">{idx.totalFunctionsCount} funcs</td>
                  <td className="py-2 px-3 text-slate-500">{idx.indexedAt}</td>
                  <td className="py-2 px-3 text-right">
                    <span className="bg-emerald-50 text-emerald-700 border border-emerald-200 text-[10px] font-bold px-1.5 py-0.2 rounded-sm">
                      {idx.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Two-Column File Tree Explorer: Folder tree (30%) + AST Node Breakdown (70%) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column (30% -> col-span-4) File Tree Explorer */}
        <div className="lg:col-span-4 bg-white border border-slate-200 rounded-sm p-3 space-y-3">
          <div className="flex items-center justify-between border-b border-slate-100 pb-2">
            <div className="font-mono text-xs font-bold text-slate-900 flex items-center gap-1.5">
              <Folder className="w-4 h-4 text-slate-700" />
              <span>Project Directory Tree</span>
            </div>
          </div>

          {/* Directory Tree */}
          <div className="space-y-0.5 max-h-[500px] overflow-y-auto">
            {renderTree(astFiles)}
          </div>
        </div>

        {/* Right Column (70% -> col-span-8) AST Node Breakdown */}
        <div className="lg:col-span-8 bg-white border border-slate-200 rounded-sm overflow-hidden flex flex-col justify-between">
          <div>
            {/* Header & Mode Switcher */}
            <div className="bg-slate-100 border-b border-slate-200 p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
              <div className="font-mono text-xs font-bold text-slate-900 flex items-center gap-2">
                <FileCode className="w-4 h-4 text-slate-800" />
                <span>
                  File: <span className="text-black font-mono font-bold underline">{selectedFilePath}</span>
                </span>
              </div>

              <div className="flex items-center gap-1 bg-white border border-slate-200 rounded-sm p-0.5 font-mono text-[11px]">
                <button
                  onClick={() => setViewMode('nodes')}
                  className={`px-2 py-0.5 rounded-sm transition-colors ${
                    viewMode === 'nodes' ? 'bg-black text-white font-bold' : 'text-slate-600 hover:text-black'
                  }`}
                >
                  AST Nodes
                </button>
                <button
                  onClick={() => setViewMode('raw_json')}
                  className={`px-2 py-0.5 rounded-sm transition-colors ${
                    viewMode === 'raw_json' ? 'bg-black text-white font-bold' : 'text-slate-600 hover:text-black'
                  }`}
                >
                  Raw AST JSON
                </button>
              </div>
            </div>

            {/* File Info stats bar */}
            {selectedFile && (
              <div className="bg-slate-50 border-b border-slate-200 p-2.5 px-3 flex flex-wrap items-center justify-between gap-3 text-xs font-mono text-slate-600">
                <div>
                  Lines: <strong className="text-slate-900">{selectedFile.totalLines || 142}</strong>
                </div>
                <div>
                  Functions: <strong className="text-slate-900">{selectedFile.totalFuncs || 6}</strong>
                </div>
                <div>
                  Size: <strong className="text-slate-900">{selectedFile.sizeBytes || 4210} bytes</strong>
                </div>
                <div>
                  Package: <strong className="text-slate-900">pkg/handler</strong>
                </div>
              </div>
            )}

            {/* Content view */}
            {viewMode === 'nodes' ? (
              <div className="p-4 space-y-3 font-mono text-xs min-h-[350px]">
                <div className="text-xs font-bold text-slate-800 flex items-center gap-1.5">
                  <Braces className="w-3.5 h-3.5 text-slate-700" />
                  <span>Extracted AST Node Declarations:</span>
                </div>

                {selectedFile?.nodes && selectedFile.nodes.length > 0 ? (
                  <div className="space-y-3">
                    {selectedFile.nodes.map((node) => (
                      <div
                        key={node.id}
                        className="border border-slate-200 bg-slate-50/50 rounded-sm p-3 space-y-2 hover:border-slate-300 transition-colors"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <span className="bg-black text-white text-[10px] font-bold px-1.5 py-0.2 rounded-sm uppercase tracking-wider">
                              {node.kind}
                            </span>
                            <span className="font-bold text-slate-900 text-xs">{node.name}</span>
                            {node.receiver && (
                              <span className="text-slate-500 text-[11px] font-mono">{node.receiver}</span>
                            )}
                          </div>

                          <div className="text-[10px] text-slate-500 font-mono">
                            Line {node.line} • Pos({node.pos}..{node.end})
                          </div>
                        </div>

                        <div className="bg-white p-2 rounded-sm border border-slate-200 text-[11.5px] text-slate-800 font-mono overflow-x-auto">
                          {node.signature}
                        </div>

                        {/* Child expressions */}
                        {node.children && node.children.length > 0 && (
                          <div className="pl-3 border-l-2 border-slate-300 space-y-1 text-[11px]">
                            <div className="text-slate-500 font-semibold text-[10px] uppercase">
                              Child Expression Statements:
                            </div>
                            {node.children.map((child) => (
                              <div key={child.id} className="flex items-center justify-between text-slate-700">
                                <span className="font-bold text-slate-900">{child.kind}:</span>
                                <code className="bg-slate-100 px-1.5 py-0.2 rounded-sm text-slate-800">
                                  {child.name}
                                </code>
                                <span className="text-slate-400 text-[10px]">L{child.line}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-slate-500 text-xs py-6 text-center">
                    Select a Go file from the tree to inspect AST node declarations.
                  </div>
                )}
              </div>
            ) : (
              <div className="p-4 bg-slate-900 text-emerald-400 font-mono text-xs overflow-x-auto min-h-[350px]">
                <pre className="text-[11px] leading-relaxed">
                  {JSON.stringify(selectedFile || { path: selectedFilePath, status: 'indexed' }, null, 2)}
                </pre>
              </div>
            )}
          </div>

          <div className="p-3 bg-slate-50 border-t border-slate-200 text-xs font-mono text-slate-600 flex justify-between">
            <span>Parser: Go `go/parser` AST package</span>
            <span>Offset indexing active</span>
          </div>
        </div>
      </div>
    </div>
  );
};
