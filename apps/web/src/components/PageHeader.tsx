import type { ReactNode } from "react";

export function PageHeader({
  label,
  title,
  lead,
  actions,
}: {
  label?: string;
  title: string;
  lead?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="page-head">
      <div>
        {label && <p className="page-label">{label}</p>}
        <h1 className="page-title">{title}</h1>
        {lead && <p className="page-lead">{lead}</p>}
      </div>
      {actions}
    </div>
  );
}
