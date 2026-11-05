import { useRef } from 'react';

import { AuditEventList, useAuditEventPages } from '@/entities/audit-logs';
import type { User } from '@/entities/users';

interface Props {
  user: User;
}

export function UserAuditLogsSidebar({ user }: Props) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const pages = useAuditEventPages({ authorUserId: user.id, pageSize: 50, scrollContainerRef });
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
          shouldShowAuthor: false,
          shouldShowWorkspace: true,
          timeFormat: 'full',
          mode: 'table',
          emptyKey: 'users.auditLogs.empty',
        }}
      />
    </div>
  );
}
