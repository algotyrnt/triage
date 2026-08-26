/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Terminal, Copy, Check, Code2, ArrowRight, ExternalLink } from 'lucide-react';

export const SdkShowcase: React.FC = () => {
  const [activeRouter, setActiveRouter] = useState<'net/http' | 'chi' | 'gin' | 'echo' | 'fiber'>(
    'net/http',
  );
  const [copied, setCopied] = useState(false);

  const snippets: Record<string, string> = {
    'net/http': `package main

import (
    "net/http"
    triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/process", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Process completed successfully"))
    })

    // Wrap your mux with triage panic isolation middleware
    handler := triage.Middleware(
        "tr_live_key_9042",
        "https://triage.yourcompany.com/api/v1/telemetry",
    )(mux)

    http.ListenAndServe(":8080", handler)
}`,
    chi: `package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
    r := chi.NewRouter()

    // Add triage middleware to chi router stack
    r.Use(triage.Middleware(
        "tr_live_key_9042",
        "https://triage.yourcompany.com/api/v1/telemetry",
    ))

    r.Get("/orders", handleGetOrders)
    http.ListenAndServe(":8080", r)
}`,
    gin: `package main

import (
    "github.com/gin-gonic/gin"
    triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
    r := gin.New()

    // Wrap gin router with triage panic recovery adapter
    r.Use(gin.WrapH(triage.Middleware(
        "tr_live_key_9042",
        "https://triage.yourcompany.com/api/v1/telemetry",
    )(r)))

    r.GET("/api/users", handleGetUsers)
    r.Run(":8080")
}`,
    echo: `package main

import (
    "github.com/labstack/echo/v4"
    triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
    e := echo.New()

    // Use triage standard HTTP middleware adapter with Echo
    e.Use(echo.WrapMiddleware(triage.Middleware(
        "tr_live_key_9042",
        "https://triage.yourcompany.com/api/v1/telemetry",
    )))

    e.GET("/health", handleHealth)
    e.Start(":8080")
}`,
    fiber: `package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/adaptor/v2"
    triage "github.com/algotyrnt/triage/sdk/go"
)

func main() {
    app := fiber.New()

    // Adapt triage middleware for Fiber fast HTTP stack
    app.Use(adaptor.HTTPMiddleware(triage.Middleware(
        "tr_live_key_9042",
        "https://triage.yourcompany.com/api/v1/telemetry",
    )))

    app.Get("/metrics", handleMetrics)
    app.Listen(":8080")
}`,
  };

  const copyCode = () => {
    navigator.clipboard.writeText(snippets[activeRouter]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section
      id="sdk"
      className="py-14 sm:py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto border-t border-slate-200 scroll-mt-16"
    >
      {/* Header */}
      <div className="text-center max-w-3xl mx-auto space-y-2.5 sm:space-y-3">
        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 text-slate-800 font-mono text-xs font-semibold">
          <Terminal className="w-3.5 h-3.5 text-indigo-600" />
          <span>GO SDK INTEGRATION</span>
        </div>
        <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-slate-900 tracking-tight">
          Integrate with Any Go Router in Under 60 Seconds
        </h2>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed max-w-2xl mx-auto">
          Triage works as standard Go{' '}
          <code className="text-indigo-700 font-mono">http.Handler</code> middleware. Drop it into
          your existing HTTP routers without refactoring.
        </p>
      </div>

      {/* Code Showcase Window */}
      <div className="mt-10 sm:mt-12 max-w-5xl mx-auto bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-2xl">
        {/* Router tabs */}
        <div className="bg-slate-900 border-b border-slate-800 px-4 py-3 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2 overflow-x-auto scrollbar-none">
            {(['net/http', 'chi', 'gin', 'echo', 'fiber'] as const).map((router) => (
              <button
                key={router}
                onClick={() => setActiveRouter(router)}
                className={`px-3 py-1.5 font-mono text-xs rounded transition-all whitespace-nowrap ${
                  activeRouter === router
                    ? 'bg-black text-white font-bold border border-slate-700 shadow-xs'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800'
                }`}
              >
                {router}
              </button>
            ))}
          </div>

          <button
            onClick={copyCode}
            className="bg-slate-800 hover:bg-slate-700 text-slate-200 px-3 py-1.5 rounded text-xs font-mono flex items-center gap-1.5 border border-slate-700 transition-colors"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-emerald-400">Copied!</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5 text-slate-400" />
                <span>Copy Snippet</span>
              </>
            )}
          </button>
        </div>

        {/* Code Content */}
        <div className="p-6 font-mono text-xs text-slate-200">
          <pre className="overflow-x-auto leading-relaxed text-slate-300">
            {snippets[activeRouter]}
          </pre>
        </div>

        {/* Bottom features bar */}
        <div className="bg-slate-900/80 border-t border-slate-800 px-6 py-4 font-mono text-xs">
          <div className="text-slate-400 font-bold mb-2 flex items-center justify-between">
            <span>SDK Runtime Architecture & Guarantees:</span>
            <a
              href="/docs/sdk"
              className="text-indigo-400 hover:text-indigo-300 flex items-center gap-1 text-[11px] font-sans font-medium"
            >
              <span>Full SDK Documentation</span>
              <ArrowRight className="w-3 h-3" />
            </a>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-[11px] text-slate-300">
            <div className="bg-slate-950 p-2.5 rounded border border-slate-800">
              <span className="text-emerald-400 font-bold">Zero-Boilerplate Config</span>
              <p className="text-slate-400 mt-1 font-sans">
                Local VCS commit detection with API-key-based repository and root-path resolution.
              </p>
            </div>
            <div className="bg-slate-950 p-2.5 rounded border border-slate-800">
              <span className="text-cyan-400 font-bold">Non-Blocking Dispatch</span>
              <p className="text-slate-400 mt-1 font-sans">
                Bounded 4-worker pool with 1,000-job buffer protects memory.
              </p>
            </div>
            <div className="bg-slate-950 p-2.5 rounded border border-slate-800">
              <span className="text-amber-400 font-bold">OpenTelemetry Native</span>
              <p className="text-slate-400 mt-1 font-sans">
                Auto-injects W3C traceparent and X-Triage-Trace-ID headers.
              </p>
            </div>
            <div className="bg-slate-950 p-2.5 rounded border border-slate-800">
              <span className="text-purple-400 font-bold">Panic Crash Isolation</span>
              <p className="text-slate-400 mt-1 font-sans">
                Standard library defer and recover boundary for non-blocking crash recovery.
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
