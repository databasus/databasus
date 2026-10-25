import { App } from 'antd';
import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';

import { usePagedList } from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { auditEventApi } from '../api/auditEventApi';
import type { AuditEvent } from './AuditEvent';

interface Params {
  // The filters choose the view: none for the global one, authorUserId for a user's and
  // workspaceId for a workspace's.
  workspaceId?: string;
  authorUserId?: string;
  pageSize: number;
  scrollContainerRef: RefObject<HTMLDivElement | null>;
}

// Pages the events of one audit log view as the user scrolls, the newest first.
export function useAuditEventPages({
  workspaceId,
  authorUserId,
  pageSize,
  scrollContainerRef,
}: Params) {
  const { t } = useTranslation();
  const { message } = App.useApp();

  return usePagedList<AuditEvent, string>({
    fetchPage: (after) =>
      auditEventApi.listAuditEvents({ workspaceId, authorUserId, limit: pageSize, after }),
    getItemKey: (event) => event.id,
    onError: (error) => message.error(translateApiError(error, t)),
    scrollContainerRef,
    resetKey: `${workspaceId ?? ''}|${authorUserId ?? ''}`,
  });
}
