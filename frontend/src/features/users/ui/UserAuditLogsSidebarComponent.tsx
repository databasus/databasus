import { LoadingOutlined } from '@ant-design/icons';
import { Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useRef } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type AuditEvent,
  getAuditEventWorkspaceLabel,
  useAuditEventPages,
} from '../../../entities/audit-logs';
import type { User } from '../../../entities/users';
import { useLocale } from '../../../shared/i18n';
import { getUserTimeFormat } from '../../../shared/time';

interface Props {
  user: User;
}

export function UserAuditLogsSidebarComponent({ user }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const {
    items: auditEvents,
    isLoading,
    isLoadingMore,
    hasMore,
  } = useAuditEventPages({ authorUserId: user.id, pageSize: 50, scrollContainerRef });

  const columns: ColumnsType<AuditEvent> = [
    {
      title: t('auditLogs.columns.message'),
      dataIndex: 'message',
      key: 'message',
      width: 350,
      render: (message: string) => (
        <span className="text-xs text-gray-900 dark:text-white">{message}</span>
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
            className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
              workspaceLabel
                ? 'bg-blue-100 text-blue-800'
                : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-white'
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
      width: 200,
      render: (createdAt: string) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserTimeFormat();
        return (
          <span className="text-xs text-gray-700 dark:text-white">
            {`${date.format(timeFormat.format)} (${formatRelativeTime(date)})`}
          </span>
        );
      },
    },
  ];

  return (
    <div className="h-full">
      <div ref={scrollContainerRef} className="h-full overflow-y-auto">
        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          </div>
        ) : (
          <>
            <Table
              columns={columns}
              dataSource={auditEvents}
              pagination={false}
              rowKey="id"
              size="small"
              className="mb-4"
            />

            {isLoadingMore && (
              <div className="flex justify-center py-4">
                <Spin indicator={<LoadingOutlined spin />} />
                <span className="ml-2 text-sm text-gray-500 dark:text-gray-400">
                  {t('auditLogs.loadingMore')}
                </span>
              </div>
            )}

            {!hasMore && auditEvents.length > 0 && (
              <div className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                {t('auditLogs.allLoaded')}
              </div>
            )}

            {auditEvents.length === 0 && (
              <div className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                {t('users.auditLogs.empty')}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
