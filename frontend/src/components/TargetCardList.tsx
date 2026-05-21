import type { Target, TargetStatsResponse } from "../types";
import { formatPercent } from "../lib/format";
import {
  EditIcon,
  ExternalIcon,
  LogsIcon,
  TrashIcon
} from "../icons";
import { UptimeStrip } from "./UptimeStrip";
import { StatusBadge } from "./ui/StatusBadge";
import { Tooltip } from "./ui/Tooltip";
import { EmptyState } from "./ui/EmptyState";

function displayTargetName(target: Target) {
  return target.name || target.url;
}

export function TargetCardList({
  targets,
  selectedId,
  statsForSelected,
  onLogs,
  onEdit,
  onDelete
}: {
  targets: Target[];
  selectedId?: string | null;
  statsForSelected?: TargetStatsResponse | null;
  onLogs: (target: Target) => void;
  onEdit: (target: Target) => void;
  onDelete: (target: Target) => void;
}) {
  if (targets.length === 0) {
    return (
      <EmptyState
        title="No targets yet"
        description="Create your first monitor to start tracking uptime and response times."
      />
    );
  }

  return (
    <div className="target-card-list">
      {targets.map((target) => {
        const isSelected = target.id === selectedId;
        const stats = isSelected ? statsForSelected : null;
        const uptimeLabel =
          stats && stats.total_checks > 0 ? `${formatPercent(stats.uptime_percent)}%` : "—";
        const latencyLabel =
          stats && stats.total_checks > 0 ? `${Math.round(stats.avg_response_ms)} ms` : "—";

        return (
          <article
            key={target.id}
            className={`target-card ${isSelected ? "selected" : ""}`}
            onClick={() => onLogs(target)}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onLogs(target);
              }
            }}
            role="button"
            tabIndex={0}
          >
            <div className="target-card-main">
              <div className="target-card-head">
                <div className="target-card-title">
                  <strong>{displayTargetName(target)}</strong>
                  <div className="target-card-meta">
                    <a
                      href={target.url}
                      target="_blank"
                      rel="noreferrer"
                      onClick={(event) => event.stopPropagation()}
                    >
                      {target.url}
                    </a>
                    {target.slug && target.status_page_enabled && (
                      <a
                        className="status-page-link"
                        href={`/status/${target.slug}`}
                        target="_blank"
                        rel="noreferrer"
                        title="Open public status page"
                        onClick={(event) => event.stopPropagation()}
                      >
                        <ExternalIcon size={14} />
                      </a>
                    )}
                  </div>
                </div>
                <StatusBadge status={target.status} enabled={target.enabled} />
              </div>

              <div className="target-card-metrics">
                <div>
                  <span>Uptime</span>
                  <strong>{uptimeLabel}</strong>
                </div>
                <div>
                  <span>Latency</span>
                  <strong>{latencyLabel}</strong>
                </div>
                <div>
                  <span>Interval</span>
                  <strong>{target.interval}s</strong>
                </div>
                <div>
                  <span>Last check</span>
                  <strong>{target.last_checked_at ? new Date(target.last_checked_at).toLocaleString() : "Never"}</strong>
                </div>
              </div>

              <UptimeStrip
                timeline={stats?.timeline}
                status={target.status}
                title={stats ? "Uptime from loaded analytics" : "Status preview (select for history)"}
              />
            </div>

            <div className="target-card-actions" onClick={(event) => event.stopPropagation()}>
              <Tooltip label="View logs">
                <button
                  className="icon-button"
                  type="button"
                  aria-label="View logs"
                  onClick={() => onLogs(target)}
                >
                  <LogsIcon />
                </button>
              </Tooltip>
              <Tooltip label="Edit target">
                <button
                  className="icon-button"
                  type="button"
                  aria-label="Edit target"
                  onClick={() => onEdit(target)}
                >
                  <EditIcon />
                </button>
              </Tooltip>
              <Tooltip label="Delete target">
                <button
                  className="icon-button danger"
                  type="button"
                  aria-label="Delete target"
                  onClick={() => onDelete(target)}
                >
                  <TrashIcon />
                </button>
              </Tooltip>
            </div>
          </article>
        );
      })}
    </div>
  );
}
