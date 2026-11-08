import type { AuditEventPage } from '@/entities/audit-logs/model/AuditEventPage';
import type { ReadOptions } from '@/shared/api';
import { type ApiPaths, getApiClient, requireData } from '@/shared/api';

type AuditEventListQuery = NonNullable<ApiPaths['/audit-events']['get']['parameters']['query']>;

export const auditEventApi = {
  // The filters choose the view: none for the global one, authorUserId for a user's and
  // workspaceId for a workspace's.
  async listAuditEvents(
    listQuery: AuditEventListQuery,
    options: ReadOptions = {},
  ): Promise<AuditEventPage> {
    return requireData(
      await getApiClient(options).GET('/audit-events', { params: { query: listQuery } }),
    );
  },
};
