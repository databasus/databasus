import { LoadingOutlined } from '@ant-design/icons';
import { Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type AuditEvent,
  AuditEventAuthorComponent,
  getAuditEventWorkspaceLabel,
  useAuditEventPages,
} from '../../../entity/audit-logs';
import { useIsMobile } from '../../../shared/hooks';
import { useLocale } from '../../../shared/i18n';
import { getUserTimeFormat } from '../../../shared/time';

interface Props {
  scrollContainerRef: RefObject<HTMLDivElement | null>;
}

export function AuditLogsComponent({ scrollContainerRef }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const isMobile = useIsMobile();
  const {
    items: auditEvents,
    isLoading,
    isLoadingMore,
    hasMore,
  } = useAuditEventPages({ pageSize: 50, scrollContainerRef });

  const columns: ColumnsType<AuditEvent> = [
    {
      title: t('auditLogs.columns.user'),
      key: 'user',
      width: 300,
      render: (_, event: AuditEvent) => <AuditEventAuthorComponent author={event.author} />,
    },
    {
      title: t('auditLogs.columns.message'),
      dataIndex: 'message',
      key: 'message',
      render: (message: string) => (
        <span className="text-xs text-gray-900 dark:text-gray-100">{message}</span>
      ),
    },
    {
      title: t('auditLogs.columns.workspace'),
      key: 'workspace',
      width: 200,
      render: (_, event: AuditEvent) => {
        const workspaceLabel = getAuditEventWorkspaceLabel(event);

        return (
          <span
            className={`inline-block rounded-full px-1.5 py-0.5 text-xs font-medium ${
              workspaceLabel
                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
            }`}
          >
            {workspaceLabel ?? '-'}
          </span>
        );
      },
    },
    {
      title: t('auditLogs.columns.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 250,
      render: (createdAt: string) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserTimeFormat();
        return (
          <span className="text-xs text-gray-700 dark:text-gray-300">
            {`${date.format(timeFormat.format)} (${formatRelativeTime(date)})`}
          </span>
        );
      },
    },
  ];

  const renderAuditEventCard = (event: AuditEvent) => {
    const date = dayjs(event.createdAt);
    const timeFormat = getUserTimeFormat();
    const workspaceLabel = getAuditEventWorkspaceLabel(event);

    return (
      <div
        key={event.id}
        className="mb-3 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800"
      >
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <AuditEventAuthorComponent author={event.author} />
          </div>
          <div className="text-right text-xs text-gray-500 dark:text-gray-400">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-gray-400 dark:text-gray-500">{formatRelativeTime(date)}</div>
          </div>
        </div>
        <div className="mt-2 text-sm text-gray-900 dark:text-gray-100">{event.message}</div>
        {workspaceLabel && (
          <div className="mt-2">
            <span className="inline-block rounded-full bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
              {workspaceLabel}
            </span>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="max-w-[1200px]">
      <h2 className="mb-4 text-xl font-bold dark:text-white">{t('auditLogs.title')}</h2>

      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : auditEvents.length === 0 ? (
        <div className="flex h-32 items-center justify-center text-gray-500 dark:text-gray-400">
          {t('auditLogs.empty')}
        </div>
      ) : (
        <>
          {isMobile ? (
            <div>{auditEvents.map(renderAuditEventCard)}</div>
          ) : (
            <Table
              columns={columns}
              dataSource={auditEvents}
              pagination={false}
              rowKey="id"
              size="small"
              className="mb-4"
            />
          )}

          {isLoadingMore && (
            <div className="flex justify-center py-4">
              <Spin indicator={<LoadingOutlined spin />} />
              <span className="ml-2 text-sm text-gray-500 dark:text-gray-400">
                {t('auditLogs.loadingMore')}
              </span>
            </div>
          )}

          {!hasMore && (
            <div className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
              {t('auditLogs.allLoaded')}
            </div>
          )}
        </>
      )}
    </div>
  );
}
