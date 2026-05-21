export function formatPercent(value: number) {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: value % 1 === 0 ? 0 : 1,
    maximumFractionDigits: 1
  }).format(value);
}

export function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

export function formatShortTime(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

export function formatDurationMs(start: string, end: string | null) {
  const startMs = new Date(start).getTime();
  const endMs = end ? new Date(end).getTime() : Date.now();
  const diff = Math.max(0, endMs - startMs);
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) {
    return `${days}d ${hours % 24}h`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return "<1m";
}

export function accentForUptime(value: number): "good" | "bad" | "muted" {
  if (value >= 99) {
    return "good";
  }
  if (value >= 95) {
    return "muted";
  }
  return "bad";
}
