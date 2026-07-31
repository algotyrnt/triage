import React, { useState } from 'react';
import { MetricHourly, SystemHealthComponent, ScreenId } from '../../types';
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
} from 'lucide-react';

interface SystemStatusPageProps {
  health: SystemHealthComponent[];
  metrics: MetricHourly[];
  onNavigate: (screen: ScreenId) => void;
}

export const SystemStatusPage: React.FC<SystemStatusPageProps> = ({
  health,
  metrics,
  onNavigate,
}) => {
  const [metricType, setMetricType] = useState<'panics' | 'latency' | 'astTime'>('panics');

  const maxPanic = Math.max(...metrics.map((m) => m.panicCount), 1);
  const maxLatency = Math.max(...metrics.map((m) => m.avgLatencyMs), 1);

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
            System Status & Engine Metrics
          </h1>
          <p className="text-xs text-slate-600 font-sans mt-0.5">
            Real-time telemetry and service health monitoring across Cloud Run workers, GCS AST Storage, and Gemini.
          </p>
        </div>

        <div className="text-xs font-mono text-slate-500">
          Uptime: <strong className="text-emerald-700">99.99%</strong> (Last 90 Days)
        </div>
      </div>

      {/* Green Operational Banner */}
      <div className="bg-emerald-50 border border-emerald-200 p-4 rounded-sm flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2.5 text-emerald-900 font-mono text-xs font-bold">
          <div className="w-2.5 h-2.5 rounded-full bg-emerald-600 animate-pulse"></div>
          <span className="text-sm">All Triage Services Operational</span>
        </div>

        <div className="text-xs font-mono text-emerald-800">
          Symbolication Latency: <strong className="text-emerald-950 font-bold">14ms</strong>
        </div>
      </div>

      {/* 3-Component Health Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {health.map((item, idx) => {
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
              className="bg-white border border-slate-200 p-4 rounded-sm space-y-2.5 font-mono text-xs"
            >
              <div className="flex items-center justify-between border-b border-slate-100 pb-2">
                <div className="flex items-center gap-2 font-bold text-slate-900">
                  {icon}
                  <span>{item.name}</span>
                </div>
                <span className="bg-emerald-50 text-emerald-700 border border-emerald-200 text-[10px] font-bold px-1.5 py-0.2 rounded-sm">
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

      {/* Minimalist Black Bar Graph (#000000) Showing Hourly Panic Volume */}
      <div className="bg-white border border-slate-200 rounded-sm p-5 space-y-4 font-mono">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 pb-3">
          <div className="flex items-center gap-2 text-xs font-bold text-slate-900">
            <BarChart3 className="w-4 h-4 text-slate-800" />
            <span>24-Hour Ingestion Telemetry Graph</span>
          </div>

          <div className="flex items-center gap-1 bg-slate-100 p-0.5 rounded-sm border border-slate-200 text-[11px]">
            <button
              onClick={() => setMetricType('panics')}
              className={`px-2.5 py-1 rounded-sm transition-colors ${
                metricType === 'panics' ? 'bg-black text-white font-bold' : 'text-slate-600 hover:text-black'
              }`}
            >
              Panic Ingestion Volume
            </button>
            <button
              onClick={() => setMetricType('latency')}
              className={`px-2.5 py-1 rounded-sm transition-colors ${
                metricType === 'latency' ? 'bg-black text-white font-bold' : 'text-slate-600 hover:text-black'
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
              const heightPercent = maxVal > 0 ? Math.max((val / maxVal) * 100, 6) : 6;

              return (
                <div
                  key={m.hourLabel}
                  className="flex-1 flex flex-col items-center justify-end h-full group relative"
                >
                  {/* Tooltip on hover */}
                  <div className="opacity-0 group-hover:opacity-100 transition-opacity absolute -top-8 bg-black text-white text-[10px] py-0.5 px-1.5 rounded-sm font-mono whitespace-nowrap z-10">
                    {m.hourLabel}: {val} {metricType === 'panics' ? 'panics' : 'ms'}
                  </div>

                  {/* Solid Pitch Black Bar (#000000) */}
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
          <span>Peak Panic Ingestion Rate: 4 events/hr</span>
          <span>Sample Rate: 100% (Zero-drop telemetry)</span>
        </div>
      </div>
    </div>
  );
};
