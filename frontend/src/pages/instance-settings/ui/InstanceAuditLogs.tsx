import { Button, Drawer } from '@mantine/core';
import { ScrollText } from 'lucide-react';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { AuditEventList, useAuditEventPages } from '@/entities/audit-logs';
import { useIsMobile } from '@/shared/hooks';
import { OverlayFocusReturn, closeOverlayOnEscape } from '@/shared/ui';

export function InstanceAuditLogs() {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [isOpen, setIsOpen] = useState(false);

  return (
    <section className="my-8 max-w-2xl">
      <h2 className="text-xl font-bold">{t('auditLogs.title')}</h2>
      <Button
        className="mt-3"
        size="xs"
        variant="default"
        leftSection={<ScrollText aria-hidden size={14} />}
        onClick={() => setIsOpen(true)}
      >
        {t('auditLogs.open')}
      </Button>

      <Drawer
        closeOnEscape={false}
        onKeyDown={(event) =>
          closeOverlayOnEscape(event, { isOpen, onClose: () => setIsOpen(false) })
        }
        title={<span className="text-lg font-semibold text-content">{t('auditLogs.title')}</span>}
        position="right"
        size={isMobile ? '100%' : 1000}
        opened={isOpen}
        onClose={() => setIsOpen(false)}
        closeButtonProps={{ 'aria-label': t('common.actions.close') }}
        // The list scrolls inside its own container, which pages the log as it nears the end.
        styles={{
          content: { display: 'flex', flexDirection: 'column' },
          body: { flex: 1, minHeight: 0 },
        }}
      >
        {isOpen && (
          <>
            <OverlayFocusReturn />
            <InstanceAuditEventPages isMobile={isMobile} />
          </>
        )}
      </Drawer>
    </section>
  );
}

function InstanceAuditEventPages({ isMobile }: { isMobile: boolean }) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const pages = useAuditEventPages({ pageSize: 50, scrollContainerRef });

  return (
    <div ref={scrollContainerRef} className="h-full min-w-0 overflow-y-auto">
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
    </div>
  );
}
