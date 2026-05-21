import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";
import { RefreshIcon } from "../icons";
import { accentForUptime, formatPercent, formatShortTime } from "../lib/format";
import type { TargetStatsResponse, TargetTimelinePoint } from "../types";
import { deriveIncidents, IncidentsList } from "./IncidentsList";
import { KpiCard } from "./ui/KpiCard";
import { UptimeStrip } from "./UptimeStrip";

type ChartPoint = {
  label: string;
  latency: number;
  success: boolean;
};

type DailyUptimePoint = {
  day: string;
  uptime: number;
};

function buildLatencySeries(timeline: TargetTimelinePoint[]): ChartPoint[] {
  return [...timeline]
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
    .map((point) => ({
      label: formatShortTime(point.timestamp),
      latency: point.response_time_ms,
      success: point.success
    }));
}

function buildDailyUptimeSeries(timeline: TargetTimelinePoint[]): DailyUptimePoint[] {
  const buckets = new Map<string, { success: number; total: number }>();

  for (const point of timeline) {
    const day = new Date(point.timestamp).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric"
    });
    const bucket = buckets.get(day) || { success: 0, total: 0 };
    bucket.total += 1;
    if (point.success) {
      bucket.success += 1;
    }
    buckets.set(day, bucket);
  }

  return [...buckets.entries()].map(([day, bucket]) => ({
    day,
    uptime: bucket.total > 0 ? (bucket.success / bucket.total) * 100 : 0
  }));
}

export function AnalyticsDashboard({
  stats,
  loading,
  range,
  onRangeChange,
  onRefresh
}: {
  stats: TargetStatsResponse;
  loading: boolean;
  range: "24h" | "7d";
  onRangeChange: (range: "24h" | "7d") => void;
  onRefresh: () => void;
}) {
  const latencySeries = buildLatencySeries(stats.timeline);
  const dailyUptime = buildDailyUptimeSeries(stats.timeline);
  const incidentCount = deriveIncidents(stats.timeline).length;

  return (
    <div className="analytics-dashboard">
      <div className="analytics-toolbar">
        <div className="range-switch" role="tablist" aria-label="Analytics range">
          <button
            type="button"
            className={range === "24h" ? "range-button active" : "range-button"}
            onClick={() => onRangeChange("24h")}
          >
            24h
          </button>
          <button
            type="button"
            className={range === "7d" ? "range-button active" : "range-button"}
            onClick={() => onRangeChange("7d")}
          >
            7d
          </button>
        </div>
        <button
          className="icon-button"
          type="button"
          title="Refresh analytics"
          aria-label="Refresh analytics"
          onClick={onRefresh}
          disabled={loading}
        >
          <RefreshIcon />
        </button>
      </div>

      <div className="kpi-grid">
        <KpiCard
          label="Uptime"
          value={`${formatPercent(stats.uptime_percent)}%`}
          tone={accentForUptime(stats.uptime_percent)}
        />
        <KpiCard label="Avg latency" value={`${Math.round(stats.avg_response_ms)} ms`} />
        <KpiCard label="Total checks" value={stats.total_checks} />
        <KpiCard
          label="Incidents"
          value={incidentCount}
          hint={`${stats.failed_checks} failed checks`}
          tone={stats.failed_checks > 0 ? "bad" : "good"}
        />
      </div>

      <div className="charts-grid">
        <section className="chart-card">
          <header>
            <h4>Response time (24h)</h4>
            <span>Milliseconds per check</span>
          </header>
          {latencySeries.length === 0 ? (
            <div className="chart-empty">No latency data for this range</div>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={latencySeries} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="latencyFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#22c55e" stopOpacity={0.35} />
                    <stop offset="100%" stopColor="#22c55e" stopOpacity={0.02} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 11 }} interval="preserveStartEnd" />
                <YAxis tick={{ fontSize: 11 }} width={42} />
                <Tooltip
                  formatter={(value) => [`${value} ms`, "Latency"]}
                  contentStyle={{ borderRadius: 8, border: "1px solid #e5e7eb" }}
                />
                <Area
                  type="monotone"
                  dataKey="latency"
                  stroke="#16a34a"
                  fill="url(#latencyFill)"
                  strokeWidth={2}
                  dot={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </section>

        <section className="chart-card">
          <header>
            <h4>Daily uptime (7d)</h4>
            <span>Percent successful checks per day</span>
          </header>
          {dailyUptime.length === 0 ? (
            <div className="chart-empty">No daily uptime data yet</div>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={dailyUptime} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                <XAxis dataKey="day" tick={{ fontSize: 11 }} />
                <YAxis domain={[0, 100]} tick={{ fontSize: 11 }} width={42} />
                <Tooltip
                  formatter={(value) => [`${formatPercent(Number(value))}%`, "Uptime"]}
                  contentStyle={{ borderRadius: 8, border: "1px solid #e5e7eb" }}
                />
                <Bar dataKey="uptime" fill="#2563eb" radius={[6, 6, 0, 0]} maxBarSize={48} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </section>
      </div>

      <UptimeStrip timeline={stats.timeline} status="unknown" title="Uptime history strip" />

      <section className="incidents-panel">
        <header>
          <h4>Recent incidents</h4>
          <span>Derived from failed checks in range</span>
        </header>
        <IncidentsList timeline={stats.timeline} />
      </section>
    </div>
  );
}
