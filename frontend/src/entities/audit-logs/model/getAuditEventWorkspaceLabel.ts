import type { AuditEvent } from '@/entities/audit-logs/model/AuditEvent';

// A deleted workspace has no name any more, so its event shows the workspace's ID.
export const getAuditEventWorkspaceLabel = (event: AuditEvent): string | undefined =>
  event.workspace ? (event.workspace.name ?? event.workspace.id) : undefined;
