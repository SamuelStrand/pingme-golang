import type { TargetStatus } from "../../types";

export function StatusBadge({ status, enabled }: { status: TargetStatus; enabled: boolean }) {
  if (!enabled) {
    return (
      <span className="status-pill status-disabled">
        <span className="status-dot-static" />
        Disabled
      </span>
    );
  }

  const label = status === "up" ? "UP" : status === "down" ? "DOWN" : "UNKNOWN";
  return (
    <span className={`status-pill status-${status}`}>
      <span className={`status-dot-static ${status === "up" ? "pulse" : ""}`} />
      {label}
    </span>
  );
}
