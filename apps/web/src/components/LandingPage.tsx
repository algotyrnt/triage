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

"use client";

import React, { useState } from "react";
import Link from "next/link";
import {
  ShieldAlert,
  Terminal,
  Cpu,
  Zap,
  Code2,
  Activity,
  ArrowRight,
  Copy,
  Check,
  Sparkles,
  Layers,
  CheckCircle2,
} from "lucide-react";

export function LandingPage() {
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState<"ast" | "llm" | "raw">("ast");

  const sdkCode = `import (
    "net/http"
    "triage/sdk"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/process", processData)

    // Wrap handler with Triage panic isolation middleware
    handler := triage.Middleware("tr_live_key_9042", "http://localhost:8080/api/v1/telemetry")(mux)
    http.ListenAndServe(":8081", handler)
}`;

  const copyCode = () => {
    navigator.clipboard.writeText(sdkCode);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="min-h-screen bg-slate-900 text-slate-100 font-sans antialiased">
      {/* Navigation Bar */}
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-black text-white px-2.5 py-1 rounded-sm font-mono text-xs tracking-wider font-bold">
              [TRIAGE]
            </div>
            <span className="font-bold text-slate-100 text-sm tracking-tight">
              Go Crash & AST Engine
            </span>
          </div>

          <nav className="flex items-center gap-6 text-xs font-mono text-slate-400">
            <a href="#features" className="hover:text-white transition-colors">
              Features
            </a>
            <a href="#engine" className="hover:text-white transition-colors">
              AST Engine
            </a>
            <a href="#quickstart" className="hover:text-white transition-colors">
              SDK Guide
            </a>
            <Link
              href="/dashboard"
              className="bg-black text-white hover:bg-slate-800 px-3.5 py-1.5 rounded-sm font-mono text-xs font-semibold flex items-center gap-1.5 border border-slate-700 transition-all shadow-sm"
            >
              <span>Open Dashboard</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero Section */}
      <section className="py-20 px-4 max-w-5xl mx-auto text-center space-y-6">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 font-mono text-xs font-semibold">
          <Sparkles className="w-3.5 h-3.5" /> GO CRASH ISOLATION & GEMINI AI DIAGNOSTICS
        </div>

        <h1 className="text-4xl sm:text-6xl font-extrabold text-white tracking-tight leading-tight">
          Zero-Latency Go Crash Isolation. <br />
          <span className="bg-gradient-to-r from-indigo-400 via-purple-400 to-cyan-400 bg-clip-text text-transparent">
            Instant Gemini AI Diagnosis.
          </span>
        </h1>

        <p className="text-slate-400 text-lg max-w-2xl mx-auto leading-relaxed">
          Intercepts Go panics non-blockingly using <code className="text-indigo-300 font-mono bg-slate-800 px-1.5 py-0.5 rounded">defer + recover</code>. Slices the exact surrounding <code className="text-indigo-300 font-mono bg-slate-800 px-1.5 py-0.5 rounded">*ast.FuncDecl</code> code node and streams instant root-cause analysis directly to your dashboard.
        </p>

        <div className="flex items-center justify-center gap-4 pt-4">
          <Link
            href="/dashboard"
            className="bg-indigo-600 hover:bg-indigo-500 text-white font-mono text-sm font-bold px-6 py-3 rounded-sm flex items-center gap-2 shadow-lg shadow-indigo-600/20 transition-all"
          >
            <span>Launch Studio Dashboard</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
          <a
            href="#quickstart"
            className="bg-slate-800 hover:bg-slate-700 text-slate-200 font-mono text-sm px-6 py-3 rounded-sm border border-slate-700 flex items-center gap-2 transition-all"
          >
            <Terminal className="w-4 h-4" />
            <span>Go SDK Guide</span>
          </a>
        </div>

        {/* Interactive Code Window */}
        <div className="pt-10 max-w-4xl mx-auto text-left">
          <div className="bg-slate-950 border border-slate-800 rounded-sm overflow-hidden shadow-2xl">
            <div className="bg-slate-900 border-b border-slate-800 px-4 py-2.5 flex items-center justify-between">
              <div className="flex items-center gap-2 font-mono text-xs text-slate-400">
                <span className="w-2.5 h-2.5 rounded-full bg-red-500 inline-block"></span>
                <span className="w-2.5 h-2.5 rounded-full bg-yellow-500 inline-block"></span>
                <span className="w-2.5 h-2.5 rounded-full bg-green-500 inline-block"></span>
                <span className="ml-2 text-slate-400">scripts/test-crash/main.go:21 [PANIC INTERCEPTED]</span>
              </div>

              <div className="flex gap-1">
                {(["ast", "llm", "raw"] as const).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setActiveTab(tab)}
                    className={`px-2.5 py-1 font-mono text-xs rounded-sm transition-colors ${
                      activeTab === tab
                        ? "bg-slate-800 text-white font-bold border border-slate-700"
                        : "text-slate-400 hover:text-white"
                    }`}
                  >
                    {tab === "ast" && "1. Func AST"}
                    {tab === "llm" && "2. Gemini AI"}
                    {tab === "raw" && "3. Raw Stack"}
                  </button>
                ))}
              </div>
            </div>

            <div className="p-4 font-mono text-xs text-slate-200">
              {activeTab === "ast" && (
                <div>
                  <div className="text-cyan-400 font-bold mb-2">
                    Extracted *ast.FuncDecl surrounding line 21:
                  </div>
                  <pre className="bg-slate-900 p-3 rounded border border-slate-800 leading-relaxed text-slate-300">
{`func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Triggering nil pointer dereference panic...")
		var ptr *int
		*ptr = 42 // PANIC: nil pointer dereference
	})
}`}
                  </pre>
                </div>
              )}

              {activeTab === "llm" && (
                <div>
                  <div className="text-emerald-400 font-bold mb-2">
                    Gemini 2.5 Structured Diagnosis:
                  </div>
                  <pre className="bg-slate-900 p-3 rounded border border-slate-800 leading-relaxed text-emerald-300">
{`{
  "root_cause": "Uninitialized pointer dereference (*ptr = 42) on line 21.",
  "suggested_fix": "Allocate memory before assignment: val := 42; ptr = &val"
}`}
                  </pre>
                </div>
              )}

              {activeTab === "raw" && (
                <div>
                  <div className="text-slate-400 font-bold mb-2">
                    Raw debug.Stack():
                  </div>
                  <pre className="bg-slate-900 p-3 rounded border border-slate-800 leading-relaxed text-slate-400 text-[11px]">
{`goroutine 21 [running]:
runtime/debug.Stack()
	/workspace/sdk/go/middleware.go:28 +0x68
main.main.func2({0x12995dae8, 0x102893268}, 0x0)
	/workspace/scripts/test-crash/main.go:21 +0x74`}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-16 px-4 max-w-6xl mx-auto border-t border-slate-800">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-slate-950 border border-slate-800 p-6 rounded-sm space-y-3">
            <Code2 className="w-8 h-8 text-indigo-400" />
            <h3 className="text-lg font-bold text-white">AST Node Isolation</h3>
            <p className="text-slate-400 text-sm leading-relaxed">
              Uses Go standard <code className="text-indigo-300">go/parser</code> to extract only the enclosing <code className="text-indigo-300">*ast.FuncDecl</code> around the crash line.
            </p>
          </div>

          <div className="bg-slate-950 border border-slate-800 p-6 rounded-sm space-y-3">
            <Zap className="w-8 h-8 text-emerald-400" />
            <h3 className="text-lg font-bold text-white">Non-Blocking SDK</h3>
            <p className="text-slate-400 text-sm leading-relaxed">
              Panic recovery launches an asynchronous, non-blocking goroutine payload so your HTTP handlers remain responsive.
            </p>
          </div>

          <div className="bg-slate-950 border border-slate-800 p-6 rounded-sm space-y-3">
            <Cpu className="w-8 h-8 text-purple-400" />
            <h3 className="text-lg font-bold text-white">Gemini 2.5 AI SDK</h3>
            <p className="text-slate-400 text-sm leading-relaxed">
              Official <code className="text-purple-300">google.golang.org/genai</code> SDK produces guaranteed JSON schema output with root cause and drop-in fixes.
            </p>
          </div>
        </div>
      </section>

      {/* Integration Guide */}
      <section id="quickstart" className="py-16 px-4 max-w-5xl mx-auto border-t border-slate-800">
        <div className="bg-slate-950 border border-slate-800 p-8 rounded-sm space-y-4">
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div>
              <h2 className="text-xl font-bold text-white">Integrate in 3 Lines of Go</h2>
              <p className="text-slate-400 text-sm">Add Triage middleware to any standard Go <code className="text-indigo-300">http.Handler</code> multiplexer.</p>
            </div>
            <button
              onClick={copyCode}
              className="bg-slate-800 hover:bg-slate-700 text-white font-mono text-xs px-4 py-2 rounded-sm border border-slate-700 flex items-center gap-1.5 transition-colors"
            >
              {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
              <span>{copied ? "Copied!" : "Copy Code"}</span>
            </button>
          </div>

          <pre className="bg-slate-900 p-4 rounded border border-slate-800 font-mono text-xs text-slate-200 overflow-x-auto">
            {sdkCode}
          </pre>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-slate-800 py-8 px-4 text-center text-xs font-mono text-slate-500">
        <p>
          Created by{" "}
          <a
            href="https://algotyrnt.com"
            target="_blank"
            rel="noopener noreferrer"
            className="text-slate-300 font-bold hover:underline"
          >
            Punjitha Bandara (algotyrnt)
          </a>
          . Licensed under Apache 2.0.
        </p>
      </footer>
    </div>
  );
}
