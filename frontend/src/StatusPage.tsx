import { useCallback, useEffect, useState } from "react";
import { ApiError, fetchStatusPage } from "./api";
import type { StatusPageResponse, TargetTimelinePoint } from "./types";

type StatusPageProps = {
  slug: string;
};

function formatPercent(value: number) {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: value % 1 === 0 ? 0 : 1,
    maximumFractionDigits: 1
  }).format(value);
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

function statusLabel(status: string) {
  switch (status) {
    case "up":
      return "Operational";
    case "down":
      return "Down";
    default:
      return "Unknown";
  }
}

function statusTone(status: string): "good" | "bad" | "muted" {
  switch (status) {
    case "up":
      return "good";
    case "down":
      return "bad";
    default:
      return "muted";
  }
}

export function TimelineBars({
  timeline,
  failedChecks
}: {
  timeline: TargetTimelinePoint[];
  failedChecks?: number;
}) {
  if (timeline.length === 0) {
    return <div className="empty-state">No checks recorded for this range</div>;
  }

  const peakResponseTime = Math.max(
    ...timeline.map((point) => Math.max(point.response_time_ms, 1)),
    1
  );

  return (
    <TimelineCard>
      <div className="timeline-grid" role="img" aria-label="Response timeline">
        {timeline.map((point) => {
          const height = Math.max(18, Math.round((Math.max(point.response_time_ms, 1) / peakResponseTime) * 92));
          const label = `${formatDate(point.timestamp)} - ${point.success ? "Success" : "Failure"} - ${point.response_time_ms} ms`;
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
    </TimelineCard>
  );
}

function TimelineCard({ children }: { children: import("react").ReactNode }) {
  return <div className="timeline-card">{children}</div>;
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
  const tone = data ? statusTone(data.status) : "muted";

  return (
    <main className="status-shell">
      <section className="status-panel">
        <header className="status-header">
          <div className="brand-row compact">
            <div className="brand-mark">PM</div>
            <div>
              <h1>PingMe</h1>
            </div>
          </div>
          <p className="status-eyebrow">Public status</p>
        </header>

        {loading ? (
          <LoadingBlock />
        ) : notFound ? (
          <NotFoundBlock />
        ) : error ? (
          <LoadErrorBlock error={error} onRetry={() => void load()} />
        ) : data ? (
          <>
            <StatusHero data={data} tone={tone} />
            <StatusMetrics data={data} tone={tone} />
            <p className="status-range-note">Last 24 hours</p>
            <TimelineBars timeline={data.timeline} failedChecks={failedChecks} />
          </>
        ) : null}
      </section>
    </main>
  );
}

function LoadingBlock() {
  return <div className="empty-state">Loading status page</div>;
}

function NotFoundBlock() {
  return (
    <div className="status-error">
      <h2>Status page not found</h2>
      <p>The page may be disabled or the address is incorrect.</p>
    </div>
  );
}

function LoadErrorBlock({ error, onRetry }: { error: string; onRetry: () => void }) {
  return (
    <div className="status-error">
      <h2>Unable to load status page</h2>
      <p>{error}</p>
      <button className="primary-button" type="button" onClick={onRetry}>
        Retry
      </button>
    </div>
  );
}

function StatusHero({
  data,
  tone
}: {
  data: StatusPageResponse;
  tone: "good" | "bad" | "muted";
}) {
  return (
    <div className="status-hero">
      <div>
        <h2>{data.monitor_name}</h2>
        <a href={data.url} target="_blank" rel="noreferrer">
          {data.url}
        </a>
      </div>
      <span className={`status-pill ${tone}`}>{statusLabel(data.status)}</span>
    </div>
  );
}

function StatusMetrics({
  data,
  tone
}: {
  data: StatusPageResponse;
  tone: "good" | "bad" | "muted";
}) {
  return (
    <div className="status-metrics">
      <div className={`metric ${tone}`}>
        <span>Uptime (24h)</span>
        <strong>{formatPercent(data.uptime_percent)}%</strong>
      </div>
      <div className="metric">
        <span>Avg latency</span>
        <strong>{Math.round(data.avg_response_ms)} ms</strong>
      </div>
    </div>
  );
}
