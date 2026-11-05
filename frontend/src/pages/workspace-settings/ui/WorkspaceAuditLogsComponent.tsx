import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';

import { AuditEventList, useAuditEventPages } from '@/entities/audit-logs';
import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile } from '@/shared/hooks';
import { LoadFailure } from '@/shared/ui';

interface Props {
  workspaceId: string;
  onUnavailable?: (error: unknown) => void;
  scrollContainerRef: RefObject<HTMLDivElement | null>;
}

export function WorkspaceAuditLogsComponent({
  workspaceId,
  scrollContainerRef,
  onUnavailable,
}: Props) {
  const handleFailure = (error: unknown) => {
    if (isCurrent() && error instanceof ApiError && [403, 404].includes(error.status ?? 0))
      onUnavailable?.(error);
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const pages = useAuditEventPages({
    workspaceId,
    pageSize: 10,
    scrollContainerRef,
    onError: handleFailure,
  });
  const hasLostAccess =
    pages.error instanceof ApiError && [403, 404].includes(pages.error.status ?? 0);

  return (
    <section className="max-w-300">
      <h2 className="mb-4 text-xl font-bold text-content">{t('auditLogs.title')}</h2>
      {pages.error !== undefined && (
        <LoadFailure error={pages.error} onRetry={() => void pages.reload()} />
      )}
      {!hasLostAccess && (pages.error === undefined || pages.items.length > 0) && (
        <AuditEventList
          events={pages.items}
          isLoading={pages.isLoading}
          isLoadingMore={pages.isLoadingMore}
          hasMore={pages.hasMore}
          view={{
            shouldShowAuthor: true,
            shouldShowWorkspace: false,
            timeFormat: 'short',
            mode: isMobile ? 'cards' : 'table',
            emptyKey: 'workspaces.auditLogs.empty',
          }}
        />
      )}
    </section>
  );
}
