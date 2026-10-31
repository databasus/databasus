import { type RefObject, useState } from 'react';

import { usePagedList } from '../../../shared/hooks';
import { auditEventApi } from '../api/auditEventApi';
import type { AuditEvent } from './AuditEvent';

interface Params {
  // The filters choose the view: none for the global one, authorUserId for a user's and
  // workspaceId for a workspace's.
  workspaceId?: string;
  authorUserId?: string;
  pageSize: number;
  onError?: (error: unknown) => void;
  scrollContainerRef: RefObject<HTMLDivElement | null>;
}

// Pages the events of one audit log view as the user scrolls, the newest first.
export function useAuditEventPages({
  workspaceId,
  authorUserId,
  pageSize,
  scrollContainerRef,
  onError,
}: Params) {
  const [error, setError] = useState<unknown>();

  const pages = usePagedList<AuditEvent, string>({
    fetchPage: (after) => {
      setError(undefined);
      return auditEventApi.listAuditEvents({ workspaceId, authorUserId, limit: pageSize, after });
    },
    getItemKey: (event) => event.id,
    onError: (failure) => {
      setError(failure);
      onError?.(failure);
    },
    scrollContainerRef,
    resetKey: `${workspaceId ?? ''}|${authorUserId ?? ''}`,
  });

  return { ...pages, error };
}
