import type { AuditEvent } from './AuditEvent';

// A deleted workspace has no name any more, so its event shows the workspace's ID.
export const getAuditEventWorkspaceLabel = (event: AuditEvent): string | undefined =>
  event.workspace ? (event.workspace.name ?? event.workspace.id) : undefined;
