import { formatDate, formatDurationMs, formatShortTime } from "../lib/format";
import type { TargetTimelinePoint } from "../types";
import { EmptyState } from "./ui/EmptyState";

export type DerivedIncident = {
  start: string;
  end: string | null;
  cause: string;
};

export function deriveIncidents(timeline: TargetTimelinePoint[]): DerivedIncident[] {
  const sorted = [...timeline].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  );
  const incidents: DerivedIncident[] = [];
  let open: DerivedIncident | null = null;

  for (const point of sorted) {
    if (!point.success) {
      if (!open) {
        open = {
          start: point.timestamp,
          end: null,
          cause: point.response_time_ms > 0 ? `Check failed (${point.response_time_ms} ms)` : "Check failed"
        };
      }
      continue;
    }

    if (open) {
      open.end = point.timestamp;
      incidents.push(open);
      open = null;
    }
  }

  if (open) {
    incidents.push(open);
  }

  return incidents.reverse();
}

export function IncidentsList({ timeline }: { timeline: TargetTimelinePoint[] }) {
  const incidents = deriveIncidents(timeline);

  if (incidents.length === 0) {
    return (
      <EmptyState
        title="No incidents in this range"
        description="Downtime periods are inferred from failed checks in the selected analytics window."
      />
    );
  }

  return (
    <ul className="incidents-list">
      {incidents.map((incident) => (
        <li key={`${incident.start}-${incident.end || "open"}`} className="incident-item">
          <div className="incident-marker" aria-hidden="true" />
          <div className="incident-body">
            <div className="incident-top">
              <strong>{formatShortTime(incident.start)}</strong>
              <span className="incident-duration">{formatDurationMs(incident.start, incident.end)}</span>
            </div>
            <p>{incident.cause}</p>
            <small>
              {incident.end ? `Resolved ${formatDate(incident.end)}` : "Ongoing incident"}
            </small>
          </div>
        </li>
      ))}
    </ul>
  );
}
