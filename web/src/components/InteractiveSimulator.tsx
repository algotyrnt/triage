/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import {
  Terminal,
  Cpu,
  Zap,
  Code2,
  Sparkles,
  Play,
  CheckCircle2,
  AlertTriangle,
  FileCode,
  Layers,
  Activity,
  Copy,
  Check,
} from 'lucide-react';

interface Scenario {
  id: string;
  name: string;
  badge: string;
  file: string;
  line: number;
  funcName: string;
  panicMsg: string;
  astCode: string;
  suggestedPatch: string;
  rootCause: string;
  severity: 'CRITICAL' | 'HIGH';
  rawStack: string;
  telemetryJson: string;
}

const SCENARIOS: Scenario[] = [
  {
    id: 'nil-ptr',
    name: 'Nil Pointer Dereference',
    badge: 'runtime.panic',
    file: 'test-service/handlers/payment.go',
    line: 28,
    funcName: 'ProcessTransaction',
    panicMsg: 'runtime error: invalid memory address or nil pointer dereference',
    astCode: `func ProcessTransaction(w http.ResponseWriter, r *http.Request) {
    var req *PaymentPayload
    // Attempting to access struct field without initialization:
    if req.Amount <= 0 {  // <--- PANIC TRIGGER [LINE 28]
        http.Error(w, "invalid amount", http.StatusBadRequest)
        return
    }
    executeTransfer(req)
}`,
    suggestedPatch: `@@ -26,3 +26,4 @@ func ProcessTransaction(w http.ResponseWriter, r *http.Request) {
-    var req *PaymentPayload
-    if req.Amount <= 0 {
+    req := &PaymentPayload{}
+    if err := json.NewDecoder(r.Body).Decode(req); err != nil || req.Amount <= 0 {`,
    rootCause:
      'Attempted to evaluate req.Amount on an uninitialized nil pointer (*PaymentPayload) on line 28.',
    severity: 'CRITICAL',
    rawStack: `goroutine 42 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x68
github.com/algotyrnt/triage/sdk/go.Middleware.func1.1()
	/workspace/sdk/go/middleware.go:48 +0xa4
panic({0x104b20e40?, 0x104d553b0?})
	/usr/local/go/src/runtime/panic.go:785 +0x124
main.ProcessTransaction({0x104c55980, 0x140001a4000}, 0x1400018a200)
	/workspace/test-service/handlers/payment.go:28 +0x88
net/http.HandlerFunc.ServeHTTP(0x104be9f38?, {0x104c55980?, 0x140001a4000?}, 0x1400005a058?)
	/usr/local/go/src/net/http/server.go:2168 +0x38`,
    telemetryJson: `{
  "trace_id": "tr_7f9c2d1e8a4b0c3d9a1f",
  "commit_sha": "7f8b9e1a2c3d4e5f60718293",
  "file": "test-service/handlers/payment.go",
  "line": 28,
  "function_name": "ProcessTransaction",
  "panic_message": "runtime error: invalid memory address or nil pointer dereference",
  "goroutine_id": "goroutine 42 [running]",
  "latency_ms": 1.4
}`,
  },
  {
    id: 'slice-bounds',
    name: 'Slice Bounds Out of Range',
    badge: 'index.out_of_bounds',
    file: 'test-service/services/worker.go',
    line: 64,
    funcName: 'BatchAggregate',
    panicMsg: 'runtime error: index out of range [4] with length 4',
    astCode: `func BatchAggregate(records []MetricRecord) AggregationResult {
    total := 0.0
    // Incorrect loop termination condition causing index overflow:
    for i := 0; i <= len(records); i++ {  // <--- PANIC TRIGGER [LINE 64]
        total += records[i].Value
    }
    return AggregationResult{Sum: total}
}`,
    suggestedPatch: `@@ -62,3 +62,3 @@ func BatchAggregate(records []MetricRecord) AggregationResult {
-    for i := 0; i <= len(records); i++ {
+    for i := 0; i < len(records); i++ {`,
    rootCause:
      'Off-by-one condition in for loop (i <= len(records)) accesses index equal to length, exceeding slice bounds on line 64.',
    severity: 'HIGH',
    rawStack: `goroutine 18 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x68
main.BatchAggregate({0x140001a2000, 0x4, 0x4})
	/workspace/test-service/services/worker.go:64 +0x5c
main.main.func1({0x104c55980, 0x140001a4000}, 0x1400018a200)
	/workspace/test-service/main.go:42 +0x30`,
    telemetryJson: `{
  "trace_id": "tr_3e1a8f9c2d1b40c7e5a0",
  "commit_sha": "a1b2c3d4e5f60718293a4b5c",
  "file": "test-service/services/worker.go",
  "line": 64,
  "function_name": "BatchAggregate",
  "panic_message": "runtime error: index out of range [4] with length 4",
  "goroutine_id": "goroutine 18 [running]",
  "latency_ms": 1.2
}`,
  },
  {
    id: 'concurrent-map',
    name: 'Concurrent Map Write',
    badge: 'sync.race_panic',
    file: 'test-service/cache/session.go',
    line: 91,
    funcName: 'RecordHeartbeat',
    panicMsg: 'fatal error: concurrent map writes',
    astCode: `func RecordHeartbeat(sessionID string, ts int64) {
    // Unsynchronized write to shared map across parallel goroutines:
    activeSessions[sessionID] = ts  // <--- PANIC TRIGGER [LINE 91]
}`,
    suggestedPatch: `@@ -89,3 +89,5 @@ func RecordHeartbeat(sessionID string, ts int64) {
+    sessionMutex.Lock()
+    defer sessionMutex.Unlock()
     activeSessions[sessionID] = ts`,
    rootCause:
      'Concurrent unsynchronized write operation to activeSessions map without a sync.RWMutex lock on line 91.',
    severity: 'CRITICAL',
    rawStack: `goroutine 88 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x68
main.RecordHeartbeat({0x104a8b792, 0x10}, 0x18e8f20b)
	/workspace/test-service/cache/session.go:91 +0x40
main.main.func3()
	/workspace/test-service/main.go:88 +0x34`,
    telemetryJson: `{
  "trace_id": "tr_9a0b1c2d3e4f5a6b7c8d",
  "commit_sha": "f1e2d3c4b5a6978012345678",
  "file": "test-service/cache/session.go",
  "line": 91,
  "function_name": "RecordHeartbeat",
  "panic_message": "fatal error: concurrent map writes",
  "goroutine_id": "goroutine 88 [running]",
  "latency_ms": 1.8
}`,
  },
];

export const InteractiveSimulator: React.FC = () => {
  const [selectedId, setSelectedId] = useState<string>('nil-ptr');
  const [activeTab, setActiveTab] = useState<'ast' | 'llm' | 'stack' | 'telemetry'>('ast');
  const [isSimulating, setIsSimulating] = useState<boolean>(false);
  const [copiedPatch, setCopiedPatch] = useState(false);

  const scenario = SCENARIOS.find((s) => s.id === selectedId) || SCENARIOS[0];

  const handleSimulate = () => {
    setIsSimulating(true);
    setTimeout(() => {
      setIsSimulating(false);
    }, 450);
  };

  const copyPatch = () => {
    navigator.clipboard.writeText(scenario.suggestedPatch);
    setCopiedPatch(true);
    setTimeout(() => setCopiedPatch(false), 2000);
  };

  return (
    <section
      id="simulator"
      className="py-14 sm:py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-16"
    >
      {/* Header section */}
      <div className="text-center max-w-3xl mx-auto space-y-2.5 sm:space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <Activity className="w-3.5 h-3.5 text-indigo-600" />
          <span>INTERACTIVE CRASH &amp; AST INSPECTOR</span>
        </div>
        <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-slate-900 tracking-tight">
          See How Triage Isolates Crashes in Real Time
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed max-w-2xl mx-auto">
          Select a real Go panic scenario below and see how the engine extracts the enclosing AST
          node, sends it to the AI diagnostics engine, and generates actionable fixes.
        </p>
      </div>

      {/* Scenario Selector Chips */}
      <div className="mt-6 sm:mt-8 flex flex-wrap items-center justify-center gap-2">
        {SCENARIOS.map((s) => (
          <button
            key={s.id}
            onClick={() => {
              setSelectedId(s.id);
              handleSimulate();
            }}
            className={`flex items-center gap-2 px-3.5 py-1.5 rounded-md font-mono text-xs transition-all border ${
              selectedId === s.id
                ? 'bg-black text-white border-black shadow-md font-bold'
                : 'bg-white text-slate-700 border-slate-300 hover:bg-slate-50 hover:border-slate-400'
            }`}
          >
            <span
              className={`w-2 h-2 rounded-full ${
                selectedId === s.id ? 'bg-emerald-400 animate-pulse' : 'bg-slate-400'
              }`}
            />
            <span>{s.name}</span>
            <span
              className={`text-[10px] px-1.5 py-0.2 rounded ${
                selectedId === s.id ? 'bg-slate-800 text-slate-300' : 'bg-slate-100 text-slate-500'
              }`}
            >
              {s.badge}
            </span>
          </button>
        ))}
      </div>

      {/* Main Terminal & Inspector Panel */}
      <div className="mt-6 sm:mt-8 max-w-5xl mx-auto">
        <div className="bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-2xl">
          {/* Top Window Bar */}
          <div className="bg-slate-900/90 border-b border-slate-800 px-4 py-3 flex flex-wrap items-center justify-between gap-3">
            {/* Window controls & file path */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1.5">
                <span className="w-3 h-3 rounded-full bg-red-500/90 inline-block" />
                <span className="w-3 h-3 rounded-full bg-yellow-500/90 inline-block" />
                <span className="w-3 h-3 rounded-full bg-green-500/90 inline-block" />
              </div>
              <div className="font-mono text-xs text-slate-300 flex items-center gap-1.5">
                <FileCode className="w-3.5 h-3.5 text-indigo-400" />
                <span className="font-semibold text-white">{scenario.file}</span>
                <span className="text-red-400 font-bold">:{scenario.line}</span>
                <span className="text-slate-500">({scenario.funcName})</span>
              </div>
            </div>

            {/* Interactive Tab Switchers */}
            <div className="flex items-center gap-1 bg-slate-950 p-1 rounded-md border border-slate-800">
              <button
                onClick={() => setActiveTab('ast')}
                className={`px-3 py-1 font-mono text-xs rounded transition-all flex items-center gap-1.5 ${
                  activeTab === 'ast'
                    ? 'bg-indigo-600 text-white font-bold shadow-xs'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Code2 className="w-3 h-3" />
                <span>1. AST Node</span>
              </button>

              <button
                onClick={() => setActiveTab('llm')}
                className={`px-3 py-1 font-mono text-xs rounded transition-all flex items-center gap-1.5 ${
                  activeTab === 'llm'
                    ? 'bg-purple-600 text-white font-bold shadow-xs'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Sparkles className="w-3 h-3" />
                <span>2. AI Diagnostics</span>
              </button>

              <button
                onClick={() => setActiveTab('stack')}
                className={`px-3 py-1 font-mono text-xs rounded transition-all flex items-center gap-1.5 ${
                  activeTab === 'stack'
                    ? 'bg-slate-800 text-white font-bold shadow-xs'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Layers className="w-3 h-3" />
                <span>3. Raw Stack</span>
              </button>

              <button
                onClick={() => setActiveTab('telemetry')}
                className={`px-3 py-1 font-mono text-xs rounded transition-all flex items-center gap-1.5 ${
                  activeTab === 'telemetry'
                    ? 'bg-emerald-600 text-white font-bold shadow-xs'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Zap className="w-3 h-3" />
                <span>4. Telemetry</span>
              </button>
            </div>
          </div>

          {/* Panic Status Banner */}
          <div className="bg-red-950/40 border-b border-red-900/40 px-4 py-2 flex items-center justify-between font-mono text-xs">
            <div className="flex items-center gap-2 text-red-300">
              <AlertTriangle className="w-4 h-4 text-red-400 shrink-0 animate-bounce" />
              <span className="font-bold uppercase text-red-400">PANIC INTERCEPTED:</span>
              <span className="text-slate-200 truncate">{scenario.panicMsg}</span>
            </div>
            <button
              onClick={handleSimulate}
              disabled={isSimulating}
              className="bg-red-900/40 hover:bg-red-800/60 border border-red-700 text-red-200 px-2.5 py-1 rounded text-[11px] flex items-center gap-1 transition-colors"
            >
              <Play className="w-3 h-3" />
              <span>{isSimulating ? 'Analyzing...' : 'Re-run'}</span>
            </button>
          </div>

          {/* Content Body */}
          <div className="p-6 font-mono text-xs text-slate-200 min-h-80">
            {isSimulating ? (
              <div className="flex flex-col items-center justify-center h-64 space-y-3">
                <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
                <p className="text-slate-400 font-mono text-xs">
                  Extracting *ast.FuncDecl and running AI diagnostics inference...
                </p>
              </div>
            ) : (
              <>
                {/* TAB 1: AST Node */}
                {activeTab === 'ast' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="text-cyan-400 font-bold flex items-center gap-2">
                        <Code2 className="w-4 h-4" />
                        <span>Isolated *ast.FuncDecl (Go AST Subtree):</span>
                      </div>
                      <span className="text-slate-500 text-[11px]">
                        Zero pre-indexing • On-demand AST slice
                      </span>
                    </div>

                    <pre className="bg-slate-900 p-4 rounded-lg border border-slate-800 leading-relaxed text-slate-200 overflow-x-auto">
                      {scenario.astCode}
                    </pre>

                    <div className="bg-slate-900/60 border border-slate-800 p-3 rounded-md text-slate-400 text-[11px] flex items-start gap-2">
                      <Zap className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                      <div>
                        <strong className="text-slate-200">AST Node Isolation:</strong> The Go
                        engine only sent the 8 lines enclosing{' '}
                        <code className="text-indigo-300 font-bold">{scenario.funcName}</code>{' '}
                        instead of all 450 lines in the source file. This saves{' '}
                        <span className="text-emerald-400 font-bold">94% in LLM token cost</span>{' '}
                        while preserving complete semantic context.
                      </div>
                    </div>
                  </div>
                )}

                {/* TAB 2: AI Diagnostics */}
                {activeTab === 'llm' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="text-purple-400 font-bold flex items-center gap-2">
                        <Sparkles className="w-4 h-4" />
                        <span>AI Structured Diagnosis:</span>
                      </div>
                      <span className="bg-purple-950 text-purple-300 border border-purple-800 px-2 py-0.5 rounded text-[10px] font-bold">
                        LATENCY: ~180ms
                      </span>
                    </div>

                    {/* AI Explanation Card */}
                    <div className="bg-slate-900 p-4 rounded-lg border border-purple-900/40 space-y-3">
                      <div>
                        <div className="text-slate-400 text-[11px] uppercase tracking-wider font-bold mb-1">
                          Root Cause Analysis:
                        </div>
                        <p className="text-emerald-300 font-sans text-sm leading-relaxed">
                          {scenario.rootCause}
                        </p>
                      </div>

                      <div className="border-t border-slate-800 pt-3">
                        <div className="flex items-center justify-between mb-2">
                          <div className="text-slate-400 text-[11px] uppercase tracking-wider font-bold">
                            Suggested Git Patch:
                          </div>
                          <button
                            onClick={copyPatch}
                            className="text-slate-400 hover:text-white flex items-center gap-1 text-[11px] transition-colors"
                          >
                            {copiedPatch ? (
                              <>
                                <Check className="w-3 h-3 text-emerald-400" />
                                <span className="text-emerald-400">Copied</span>
                              </>
                            ) : (
                              <>
                                <Copy className="w-3 h-3" />
                                <span>Copy Diff</span>
                              </>
                            )}
                          </button>
                        </div>
                        <pre className="bg-slate-950 p-3 rounded border border-slate-800 text-xs text-slate-300 overflow-x-auto leading-relaxed">
                          {scenario.suggestedPatch}
                        </pre>
                      </div>
                    </div>
                  </div>
                )}

                {/* TAB 3: Raw Stack */}
                {activeTab === 'stack' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="text-slate-400 font-bold flex items-center gap-2">
                        <Layers className="w-4 h-4" />
                        <span>Raw runtime/debug.Stack() Telemetry:</span>
                      </div>
                      <span className="text-slate-500 text-[11px]">Unsanitized Runtime Output</span>
                    </div>

                    <pre className="bg-slate-900 p-4 rounded-lg border border-slate-800 leading-relaxed text-slate-400 text-[11px] overflow-x-auto">
                      {scenario.rawStack}
                    </pre>

                    <div className="bg-slate-900/60 border border-slate-800 p-3 rounded-md text-slate-400 text-[11px] flex items-start gap-2">
                      <CheckCircle2 className="w-4 h-4 text-cyan-400 shrink-0 mt-0.5" />
                      <div>
                        <strong className="text-slate-200">Frame Pruning:</strong> Triage
                        automatically filters internal stdlib and middleware frames so your
                        engineering team jumps directly to the failing application line.
                      </div>
                    </div>
                  </div>
                )}

                {/* TAB 4: Telemetry */}
                {activeTab === 'telemetry' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="text-emerald-400 font-bold flex items-center gap-2">
                        <Zap className="w-4 h-4" />
                        <span>Non-Blocking OpenTelemetry Dispatch Payload:</span>
                      </div>
                      <span className="text-emerald-500 font-mono text-[11px]">
                        POST /api/v1/telemetry
                      </span>
                    </div>

                    <pre className="bg-slate-900 p-4 rounded-lg border border-slate-800 leading-relaxed text-emerald-300 overflow-x-auto">
                      {scenario.telemetryJson}
                    </pre>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </section>
  );
};
