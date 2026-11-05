import { Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { type TranslationKey, useLocale } from '@/shared/i18n';
import { getUserTimeFormat, getUserShortTimeFormat } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';

import { type AuditEvent } from '../model/AuditEvent';
import { getAuditEventWorkspaceLabel } from '../model/getAuditEventWorkspaceLabel';
import { AuditEventAuthorComponent } from './AuditEventAuthorComponent';

interface Props {
  events: AuditEvent[];
  failure?: { error: unknown; onRetry: () => void };
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
  const columns: ColumnsType<AuditEvent> = [
    ...(view.shouldShowAuthor
      ? [
          {
            title: t('auditLogs.columns.user'),
            key: 'author',
            width: 300,
            render: (_: unknown, event: AuditEvent) => (
              <AuditEventAuthorComponent author={event.author} />
            ),
          },
        ]
      : []),
    {
      title: t('auditLogs.columns.message'),
      key: 'message',
      render: (_: unknown, event: AuditEvent) => (
        <span className="text-xs break-words text-content">{event.message}</span>
      ),
    },
    ...(view.shouldShowWorkspace
      ? [
          {
            title: t('auditLogs.columns.workspace'),
            key: 'workspace',
            width: 200,
            render: (_: unknown, event: AuditEvent) => (
              <span className="text-xs break-words text-accent">
                {getAuditEventWorkspaceLabel(event) ?? '-'}
              </span>
            ),
          },
        ]
      : []),
    {
      title: t('auditLogs.columns.created'),
      key: 'createdAt',
      width: 250,
      render: (_: unknown, event: AuditEvent) => (
        <span className="text-xs text-content">{formatTimestamp(event.createdAt)}</span>
      ),
    },
  ];

  if (isLoading)
    return (
      <div className="flex h-64 items-center justify-center">
        <Spin size="large" />
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
              <div className="text-xs text-accent">{getAuditEventWorkspaceLabel(event) ?? '-'}</div>
            )}
          </article>
        ))
      ) : (
        <div className="overflow-x-auto">
          <Table
            columns={columns}
            dataSource={events}
            pagination={false}
            rowKey="id"
            size="small"
          />
        </div>
      )}
      {isLoadingMore && (
        <div className="flex justify-center gap-2 py-4 text-sm text-muted">
          <Spin />
          {t('auditLogs.loadingMore')}
        </div>
      )}
      {!hasMore && (
        <div className="py-4 text-center text-sm text-muted">{t('auditLogs.allLoaded')}</div>
      )}
    </>
  );
}
