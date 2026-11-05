import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';

import { AuditEventList, useAuditEventPages } from '@/entities/audit-logs';
import { useIsMobile } from '@/shared/hooks';

interface Props {
  scrollContainerRef: RefObject<HTMLDivElement | null>;
}

export function InstanceAuditLogs({ scrollContainerRef }: Props) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const pages = useAuditEventPages({ pageSize: 50, scrollContainerRef });
  return (
    <section className="max-w-[1200px]">
      <h2 className="mb-4 text-xl font-bold">{t('auditLogs.title')}</h2>
      <AuditEventList
        failure={
          pages.error === undefined
            ? undefined
            : { error: pages.error, onRetry: () => void pages.reload() }
        }
        events={pages.items}
        isLoading={pages.isLoading}
        isLoadingMore={pages.isLoadingMore}
        hasMore={pages.hasMore}
        view={{
          shouldShowAuthor: true,
          shouldShowWorkspace: true,
          timeFormat: 'full',
          mode: isMobile ? 'cards' : 'table',
          emptyKey: 'auditLogs.empty',
        }}
      />
    </section>
  );
}
