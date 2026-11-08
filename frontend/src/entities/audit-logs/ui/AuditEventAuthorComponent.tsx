import { useTranslation } from 'react-i18next';

import type { AuditEvent } from '@/entities/audit-logs/model/AuditEvent';

interface Props {
  author: AuditEvent['author'];
}

export function AuditEventAuthorComponent({ author }: Props) {
  const { t } = useTranslation();

  return (
    <span className="text-xs break-words text-content">
      {author ? `${author.name} (${author.email})` : t('auditLogs.systemUser')}
    </span>
  );
}
