import { useTranslation } from 'react-i18next';

import type { AuditEvent } from '@/entities/audit-logs/model/AuditEvent';

interface Props {
  author: AuditEvent['author'];
}

export function AuditEventAuthorComponent({ author }: Props) {
  const { t } = useTranslation();

  if (!author) {
    return (
      <span className="inline-block rounded-full bg-surface px-1.5 py-0.5 text-xs font-medium text-muted">
        {t('auditLogs.systemUser')}
      </span>
    );
  }

  return (
    <span className="inline-block rounded-full bg-hover px-1.5 py-0.5 text-xs font-medium text-accent">
      {`${author.name} (${author.email})`}
    </span>
  );
}
