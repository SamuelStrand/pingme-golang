import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";
import { ApiError, fetchStatusPage } from "./api";
import { UptimeStrip } from "./components/UptimeStrip";
import { EmptyState } from "./components/ui/EmptyState";
import { KpiCard } from "./components/ui/KpiCard";
import { StatusBadge } from "./components/ui/StatusBadge";
import { formatPercent, formatShortTime } from "./lib/format";
import type { StatusPageResponse, TargetStatus, TargetTimelinePoint } from "./types";

type StatusPageProps = {
  slug: string;
};

function statusAsTargetStatus(status: string): TargetStatus {
  if (status === "up" || status === "down") {
    return status;
  }
  return "unknown";
}

function buildLatencySeries(timeline: TargetTimelinePoint[]) {
  return [...timeline]
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
    .map((point) => ({
      label: formatShortTime(point.timestamp),
      latency: point.response_time_ms,
      success: point.success
    }));
}

export function TimelineBars({
  timeline,
  failedChecks
}: {
  timeline: TargetTimelinePoint[];
  failedChecks?: number;
}) {
  if (timeline.length === 0) {
    return (
      <EmptyState
        title="No checks in the last 24 hours"
        description="Response time history will appear after monitoring runs."
      />
    );
  }

  const peakResponseTime = Math.max(
    ...timeline.map((point) => Math.max(point.response_time_ms, 1)),
    1
  );

  return (
    <div className="timeline-card">
      <header className="status-section-head">
        <h3>Response time</h3>
        <span>Per check over the last 24 hours</span>
      </header>
      <div className="timeline-grid" role="img" aria-label="Response timeline">
        {timeline.map((point) => {
          const height = Math.max(
            18,
            Math.round((Math.max(point.response_time_ms, 1) / peakResponseTime) * 92)
          );
          const label = `${formatShortTime(point.timestamp)} - ${point.success ? "Success" : "Failure"} - ${point.response_time_ms} ms`;
          return (
            <span
              key={`${point.timestamp}-${point.response_time_ms}-${point.success}`}
              className={point.success ? "timeline-bar success" : "timeline-bar failure"}
              style={{ height: `${height}px` }}
              title={label}
            />
          );
        })}
      </div>
      <div className="timeline-meta">
        <span>{timeline.length} checks shown</span>
        {failedChecks !== undefined && (
          <span>
            {failedChecks === 0 ? "No failures in range" : `${failedChecks} failed checks`}
          </span>
        )}
      </div>
    </div>
  );
}

export default function StatusPage({ slug }: StatusPageProps) {
  const [data, setData] = useState<StatusPageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setNotFound(false);
    setError(null);
    try {
      const response = await fetchStatusPage(slug);
      setData(response);
    } catch (err) {
      setData(null);
      if (err instanceof ApiError && err.status === 404) {
        setNotFound(true);
      } else if (err instanceof ApiError) {
        setError(err.payload?.message || err.message);
      } else if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to load status page");
      }
    } finally {
      setLoading(false);
    }
  }, [slug]);

  useEffect(() => {
    void load();
  }, [load]);

  const failedChecks = data?.timeline.filter((point) => !point.success).length ?? 0;
  const targetStatus = data ? statusAsTargetStatus(data.status) : "unknown";
  const latencySeries = useMemo(
    () => (data ? buildLatencySeries(data.timeline) : []),
    [data]
  );

  const uptimeTone =
    data && data.uptime_percent >= 99 ? "good" : data && data.uptime_percent < 95 ? "bad" : "muted";

  return (
    <main className="status-shell">
      <section className="status-panel status-public-page">
        <header className="status-header">
          <div className="brand-row compact">
            <div className="brand-mark">PM</div>
            <div>
              <h1>PingMe</h1>
              <p className="status-eyebrow">Public status page</p>
            </div>
          </div>
        </header>

        {loading ? (
          <EmptyState title="Loading status" description="Fetching the latest monitor checks." />
        ) : notFound ? (
          <div className="status-error">
            <h2>Status page not found</h2>
            <p>
              The monitor may be disabled, the slug is incorrect, or the public page is turned off.
            </p>
          </div>
        ) : error ? (
          <div className="status-error">
            <h2>Unable to load status page</h2>
            <p>{error}</p>
            <button className="primary-button" type="button" onClick={() => void load()}>
              Retry
            </button>
          </div>
        ) : data ? (
          <>
            <div className="status-hero-card">
              <div className="status-hero-main">
                <h2>{data.monitor_name}</h2>
                <a className="status-url" href={data.url} target="_blank" rel="noreferrer">
                  {data.url}
                </a>
                <p className="status-slug">/{slug}</p>
              </div>
              <StatusBadge status={targetStatus} enabled />
            </div>

            <div className="kpi-grid status-kpi-grid">
              <KpiCard
                label="Uptime (24h)"
                value={`${formatPercent(data.uptime_percent)}%`}
                tone={uptimeTone}
              />
              <KpiCard label="Avg latency" value={`${Math.round(data.avg_response_ms)} ms`} />
              <KpiCard label="Checks (24h)" value={data.timeline.length} />
              <KpiCard
                label="Failed checks"
                value={failedChecks}
                tone={failedChecks > 0 ? "bad" : "good"}
              />
            </div>

            <section className="status-section">
              <header className="status-section-head">
                <h3>Uptime history</h3>
                <span>Last 24 hours</span>
              </header>
              <UptimeStrip timeline={data.timeline} status={targetStatus} title="24h uptime strip" />
            </section>

            <section className="status-section chart-card">
              <header className="status-section-head">
                <h3>Latency trend</h3>
                <span>Milliseconds per check</span>
              </header>
              {latencySeries.length === 0 ? (
                <div className="chart-empty">No latency data yet</div>
              ) : (
                <ResponsiveContainer width="100%" height={220}>
                  <AreaChart data={latencySeries} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="publicLatencyFill" x1="0" y1="0" x2="0" y2="1">
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
                      fill="url(#publicLatencyFill)"
                      strokeWidth={2}
                      dot={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </section>

            <TimelineBars timeline={data.timeline} failedChecks={failedChecks} />

            <footer className="status-footer">
              <span>Powered by PingMe</span>
              <button className="ghost-button" type="button" onClick={() => void load()}>
                Refresh
              </button>
            </footer>
          </>
        ) : null}
      </section>
    </main>
  );
}
