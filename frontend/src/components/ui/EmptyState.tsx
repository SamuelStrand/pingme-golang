import type { ReactNode } from "react";

export function EmptyState({
  title,
  description,
  icon
}: {
  title: string;
  description?: string;
  icon?: ReactNode;
}) {
  return (
    <div className="empty-state-rich">
      {icon && <div className="empty-state-icon">{icon}</div>}
      <h4>{title}</h4>
      {description && <p>{description}</p>}
    </div>
  );
}
