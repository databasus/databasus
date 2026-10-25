import { useTranslation } from 'react-i18next';

import type { AuditEvent } from '../model/AuditEvent';

interface Props {
  author: AuditEvent['author'];
}

export function AuditEventAuthorComponent({ author }: Props) {
  const { t } = useTranslation();

  if (!author) {
    return (
      <span className="inline-block rounded-full bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">
        {t('auditLogs.systemUser')}
      </span>
    );
  }

  return (
    <span className="inline-block rounded-full bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
      {`${author.name} (${author.email})`}
    </span>
  );
}
