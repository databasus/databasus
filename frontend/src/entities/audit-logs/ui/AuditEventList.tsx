import { Loader } from '@mantine/core';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { type AuditEvent } from '@/entities/audit-logs/model/AuditEvent';
import { getAuditEventWorkspaceLabel } from '@/entities/audit-logs/model/getAuditEventWorkspaceLabel';
import { AuditEventAuthorComponent } from '@/entities/audit-logs/ui/AuditEventAuthorComponent';
import { type TranslationKey, useLocale } from '@/shared/i18n';
import { getUserTimeFormat, getUserShortTimeFormat } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';
import tableStyles from '@/shared/ui/nativeTable.module.css';
interface Props {
  events: AuditEvent[];
  failure?: {
    error: unknown;
    onRetry: () => void;
  };
  isLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  view: {
    shouldShowAuthor: boolean;
    shouldShowWorkspace: boolean;
    timeFormat: 'short' | 'full';
    mode: 'table' | 'cards';
    emptyKey: TranslationKey;
  };
}

export function AuditEventList({
  events,
  isLoading,
  isLoadingMore,
  hasMore,
  view,
  failure,
}: Props) {
  const formatTimestamp = (createdAt: string) =>
    `${dayjs(createdAt).format((view.timeFormat === 'short' ? getUserShortTimeFormat() : getUserTimeFormat()).format)} (${formatRelativeTime(dayjs(createdAt))})`;
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  if (isLoading)
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader size="md" role="status" />
      </div>
    );

  if (failure && !events.length)
    return <LoadFailure error={failure.error} onRetry={failure.onRetry} />;

  if (!events.length)
    return <div className="py-8 text-center text-sm text-muted">{t(view.emptyKey)}</div>;

  return (
    <>
      {failure && <LoadFailure error={failure.error} onRetry={failure.onRetry} />}
      {view.mode === 'cards' ? (
        events.map((event) => (
          <article
            key={event.id}
            className="mb-3 min-w-0 rounded border border-border bg-surface p-3 break-words text-content"
          >
            {view.shouldShowAuthor && <AuditEventAuthorComponent author={event.author} />}
            <div className="text-xs text-muted">{formatTimestamp(event.createdAt)}</div>
            <div className="my-2 text-sm">{event.message}</div>
            {view.shouldShowWorkspace && (
              <div className="text-xs text-content">
                {getAuditEventWorkspaceLabel(event) ?? '-'}
              </div>
            )}
          </article>
        ))
      ) : (
        <div
          className="overflow-x-auto"
          role="region"
          tabIndex={0}
          aria-label={t('auditLogs.title')}
        >
          <table className={tableStyles.table}>
            <thead>
              <tr>
                {view.shouldShowAuthor && (
                  <th scope="col" className="w-75">
                    {t('auditLogs.columns.user')}
                  </th>
                )}
                <th scope="col">{t('auditLogs.columns.message')}</th>
                {view.shouldShowWorkspace && (
                  <th scope="col" className="w-50">
                    {t('auditLogs.columns.workspace')}
                  </th>
                )}
                <th scope="col" className="w-62.5">
                  {t('auditLogs.columns.created')}
                </th>
              </tr>
            </thead>
            <tbody>
              {events.map((event) => (
                <tr key={event.id}>
                  {view.shouldShowAuthor && (
                    <td>
                      <AuditEventAuthorComponent author={event.author} />
                    </td>
                  )}
                  <td>
                    <span className="text-xs break-words text-content">{event.message}</span>
                  </td>
                  {view.shouldShowWorkspace && (
                    <td>
                      <span className="text-xs break-words text-content">
                        {getAuditEventWorkspaceLabel(event) ?? '-'}
                      </span>
                    </td>
                  )}
                  <td>
                    <span className="text-xs text-content">{formatTimestamp(event.createdAt)}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {isLoadingMore && (
        <div className="flex justify-center gap-2 py-4 text-sm text-muted">
          <Loader role="status" />
          {t('auditLogs.loadingMore')}
        </div>
      )}
      {!hasMore && (
        <div className="py-4 text-center text-sm text-muted">{t('auditLogs.allLoaded')}</div>
      )}
    </>
  );
}
