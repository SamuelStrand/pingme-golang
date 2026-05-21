import type { ReactNode } from "react";

type KpiCardProps = {
  label: string;
  value: ReactNode;
  hint?: string;
  tone?: "default" | "good" | "bad" | "muted";
};

export function KpiCard({ label, value, hint, tone = "default" }: KpiCardProps) {
  return (
    <article className={`kpi-card ${tone}`}>
      <span className="kpi-label">{label}</span>
      <strong className="kpi-value">{value}</strong>
      {hint && <span className="kpi-hint">{hint}</span>}
    </article>
  );
}
