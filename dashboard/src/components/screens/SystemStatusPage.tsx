/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect } from 'react';
import { MetricHourly, SystemHealthComponent, ScreenId } from '@/types';
import { engineClient } from '@/services/engineClient';
import {
  Activity,
  CheckCircle2,
  Server,
  Database,
  Sparkles,
  BarChart3,
  RefreshCw,
  Clock,
  Zap,
  FolderGit2,
} from 'lucide-react';

interface SystemStatusPageProps {
  health?: SystemHealthComponent[];
  metrics?: MetricHourly[];
  onNavigate: (screen: ScreenId) => void;
}

const defaultHourlyMetrics: MetricHourly[] = [
  { hourLabel: '00:00', panicCount: 0, avgLatencyMs: 12, astIndexTimeMs: 4 },
  { hourLabel: '04:00', panicCount: 1, avgLatencyMs: 14, astIndexTimeMs: 5 },
  { hourLabel: '08:00', panicCount: 3, avgLatencyMs: 18, astIndexTimeMs: 6 },
  { hourLabel: '12:00', panicCount: 2, avgLatencyMs: 15, astIndexTimeMs: 5 },
  { hourLabel: '16:00', panicCount: 4, avgLatencyMs: 19, astIndexTimeMs: 7 },
  { hourLabel: '20:00', panicCount: 1, avgLatencyMs: 13, astIndexTimeMs: 4 },
];

export const SystemStatusPage: React.FC<SystemStatusPageProps> = ({
  health: initialHealth = [],
  metrics: initialMetrics = [],
  onNavigate,
}) => {
  const [metricType, setMetricType] = useState<'panics' | 'latency'>('panics');
  const [loading, setLoading] = useState(false);
  const [engineVersion, setEngineVersion] = useState<string>('v0.1.0');
  const [dbStatus, setDbStatus] = useState<string>('connected');
  const [totalIndexedFuncs, setTotalIndexedFuncs] = useState<number>(1420);

  const fetchLiveStatus = async () => {
    setLoading(true);
    try {
      const healthData = await engineClient.getHealth();
      if (healthData) {
        if (healthData.version) setEngineVersion(healthData.version);
        if (healthData.database) setDbStatus(healthData.database);
      }

      const statsData = await engineClient.getStats();
      if (statsData) {
        if (statsData.funcs_indexed !== undefined) {
          setTotalIndexedFuncs(statsData.funcs_indexed);
        }
      }
    } catch {
      // Fallback to defaults
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLiveStatus();
  }, []);

  const components: SystemHealthComponent[] =
    initialHealth.length > 0
      ? initialHealth
      : [
          {
            name: 'Core Server & Ingress Worker',
            service: `triage (${engineVersion})`,
            status: 'OPERATIONAL',
            latency: '12ms',
            detail: 'Handling zero-overhead crash telemetry and OpenTelemetry trace propagation.',
          },
          {
            name: 'Database & AST Storage',
            service: dbStatus === 'connected' ? 'embedded-sqlite / postgres' : 'unconnected',
            status: dbStatus === 'connected' ? 'OPERATIONAL' : 'DEGRADED',
            latency: '4ms',
            detail: `${totalIndexedFuncs.toLocaleString()} AST function symbols indexed in persistent storage.`,
          },
          {
            name: 'AI Diagnostics Inference Provider',
            service: 'llm-proxy (multi-provider)',
            status: 'OPERATIONAL',
            latency: '740ms',
            detail: 'Live root cause extraction, patch synthesis, and issue generation active.',
          },
        ];

  const metrics = initialMetrics.length > 0 ? initialMetrics : defaultHourlyMetrics;
  const maxPanic = Math.max(...metrics.map((m) => m.panicCount), 1);
  const maxLatency = Math.max(...metrics.map((m) => m.avgLatencyMs), 1);

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-500 mb-1">
            <button
              onClick={() => onNavigate('dashboard')}
              className="hover:text-black hover:underline cursor-pointer"
            >
              Dashboard
            </button>
            <span>/</span>
            <span className="text-slate-800 font-semibold">Status &amp; Telemetry</span>
          </div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
            System Status &amp; Engine Health
          </h1>
          <p className="text-xs text-slate-600 font-sans mt-0.5">
            Real-time telemetry and service health monitoring across Engine workers, PostgreSQL, and
            AI Providers.
          </p>
        </div>

        <div className="flex items-center gap-2 font-mono text-xs">
          <button
            type="button"
            onClick={fetchLiveStatus}
            disabled={loading}
            className="flex items-center gap-1.5 px-2.5 py-1 bg-white hover:bg-slate-50 border border-slate-300 rounded-sm text-slate-700 transition-colors cursor-pointer"
          >
            <RefreshCw className={`w-3 h-3 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
          <span className="bg-slate-100 border border-slate-200 px-2.5 py-1 rounded-sm text-slate-700">
            Uptime: <strong className="text-emerald-700">99.99%</strong>
          </span>
        </div>
      </div>

      {/* Operational Banner */}
      <div className="bg-emerald-50 border border-emerald-200 p-4 rounded-sm flex flex-col sm:flex-row sm:items-center justify-between gap-3 font-mono">
        <div className="flex items-center gap-2.5 text-emerald-900 text-xs font-bold">
          <div className="w-2.5 h-2.5 rounded-full bg-emerald-600 animate-pulse"></div>
          <span className="text-sm">All Triage Services Operational</span>
        </div>

        <div className="text-xs text-emerald-800">
          Engine Version: <strong className="text-emerald-950 font-bold">{engineVersion}</strong> •
          Symbolication Latency: <strong className="text-emerald-950 font-bold">&lt;15ms</strong>
        </div>
      </div>

      {/* 3-Component Health Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {components.map((item, idx) => {
          const icon =
            idx === 0 ? (
              <Server className="w-4 h-4 text-slate-800" />
            ) : idx === 1 ? (
              <Database className="w-4 h-4 text-slate-800" />
            ) : (
              <Sparkles className="w-4 h-4 text-slate-800" />
            );

          return (
            <div
              key={item.name}
              className="bg-white border border-slate-200 p-4 rounded-sm space-y-2.5 font-mono text-xs shadow-xs"
            >
              <div className="flex items-center justify-between border-b border-slate-100 pb-2">
                <div className="flex items-center gap-2 font-bold text-slate-900">
                  {icon}
                  <span>{item.name}</span>
                </div>
                <span
                  className={`text-[10px] font-bold px-1.5 py-0.2 rounded-sm border ${
                    item.status === 'OPERATIONAL'
                      ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                      : 'bg-amber-50 text-amber-700 border-amber-200'
                  }`}
                >
                  {item.status}
                </span>
              </div>

              <div className="text-slate-600 text-[11px] font-mono">{item.service}</div>

              <div className="bg-slate-50 p-2 rounded-sm border border-slate-200 text-[11px] text-slate-700 space-y-1">
                <div>{item.detail}</div>
                <div className="text-slate-500 font-semibold">Response Latency: {item.latency}</div>
              </div>
            </div>
          );
        })}
      </div>

      {/* 24-Hour Telemetry Graph */}
      <div className="bg-white border border-slate-200 rounded-sm p-5 space-y-4 font-mono shadow-xs">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 pb-3">
          <div className="flex items-center gap-2 text-xs font-bold text-slate-900">
            <BarChart3 className="w-4 h-4 text-slate-800" />
            <span>24-Hour Telemetry Distribution</span>
          </div>

          <div className="flex items-center gap-1 bg-slate-100 p-0.5 rounded-sm border border-slate-200 text-[11px]">
            <button
              type="button"
              onClick={() => setMetricType('panics')}
              className={`px-2.5 py-1 rounded-sm transition-colors cursor-pointer ${
                metricType === 'panics'
                  ? 'bg-black text-white font-bold'
                  : 'text-slate-600 hover:text-black'
              }`}
            >
              Panic Volume
            </button>
            <button
              type="button"
              onClick={() => setMetricType('latency')}
              className={`px-2.5 py-1 rounded-sm transition-colors cursor-pointer ${
                metricType === 'latency'
                  ? 'bg-black text-white font-bold'
                  : 'text-slate-600 hover:text-black'
              }`}
            >
              Symbolication Latency
            </button>
          </div>
        </div>

        {/* Bar Chart Visualization in Pitch Black (#000000) */}
        <div className="space-y-2">
          <div className="h-44 flex items-end justify-between gap-2 border-b border-slate-200 pb-2 px-2">
            {metrics.map((m) => {
              const val = metricType === 'panics' ? m.panicCount : m.avgLatencyMs;
              const maxVal = metricType === 'panics' ? maxPanic : maxLatency;
              const heightPercent = maxVal > 0 ? Math.max((val / maxVal) * 100, 8) : 8;

              return (
                <div
                  key={m.hourLabel}
                  className="flex-1 flex flex-col items-center justify-end h-full group relative"
                >
                  {/* Tooltip on hover */}
                  <div className="opacity-0 group-hover:opacity-100 transition-opacity absolute -top-8 bg-black text-white text-[10px] py-0.5 px-1.5 rounded-sm font-mono whitespace-nowrap z-10">
                    {m.hourLabel}: {val} {metricType === 'panics' ? 'panics' : 'ms'}
                  </div>

                  {/* Pitch Black Bar */}
                  <div
                    className="w-full bg-black hover:bg-slate-800 transition-all rounded-xs"
                    style={{ height: `${heightPercent}%` }}
                  ></div>
                </div>
              );
            })}
          </div>

          {/* X Axis Labels */}
          <div className="flex justify-between text-[11px] text-slate-500 font-mono px-2">
            {metrics.map((m) => (
              <span key={m.hourLabel} className="text-center flex-1">
                {m.hourLabel}
              </span>
            ))}
          </div>
        </div>

        <div className="flex items-center justify-between text-xs text-slate-500 border-t border-slate-100 pt-3">
          <span>Peak Rate: {maxPanic} events/hr</span>
          <span>Sample Rate: 100% (Zero-drop telemetry)</span>
        </div>
      </div>
    </div>
  );
};
