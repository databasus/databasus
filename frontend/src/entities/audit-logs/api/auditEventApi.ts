import type { AuditEventPage } from '@/entities/audit-logs/model/AuditEventPage';
import { type ApiPaths, getApiClient, requireData } from '@/shared/api';

type AuditEventListQuery = NonNullable<ApiPaths['/audit-events']['get']['parameters']['query']>;

export const auditEventApi = {
  // The filters choose the view: none for the global one, authorUserId for a user's and
  // workspaceId for a workspace's.
  async listAuditEvents(listQuery: AuditEventListQuery): Promise<AuditEventPage> {
    return requireData(await getApiClient().GET('/audit-events', { params: { query: listQuery } }));
  },
};
