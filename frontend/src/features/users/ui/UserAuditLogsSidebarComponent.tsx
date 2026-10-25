import { LoadingOutlined } from '@ant-design/icons';
import { App, Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { auditLogApi } from '../../../entity/audit-logs/api/auditLogApi';
import type { AuditLog } from '../../../entity/audit-logs/model/AuditLog';
import type { GetAuditLogsRequest } from '../../../entity/audit-logs/model/GetAuditLogsRequest';
import type { User } from '../../../entity/users';
import { translateApiError, useLocale } from '../../../shared/i18n';
import { getUserTimeFormat } from '../../../shared/time';

interface Props {
  user: User;
}

export function UserAuditLogsSidebarComponent({ user }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const { message } = App.useApp();
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [total, setTotal] = useState(0);

  const pageSize = 50;

  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  useEffect(() => {
    loadAuditLogs(true);
  }, [user.id]);

  const handleScroll = useCallback(() => {
    if (!scrollContainerRef.current || isLoadingMore || !hasMore || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
    const threshold = 100;

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadAuditLogs(false);
    }
  }, [isLoadingMore, hasMore]);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
      return () => scrollContainer.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

  const loadAuditLogs = async (isInitialLoad = false) => {
    if (!isInitialLoad && loadingRef.current) {
      return;
    }

    loadingRef.current = true;

    if (isInitialLoad) {
      setIsLoading(true);
      setAuditLogs([]);
    } else {
      setIsLoadingMore(true);
    }

    try {
      const offset = isInitialLoad ? 0 : auditLogs.length;
      const request: GetAuditLogsRequest = {
        limit: pageSize,
        offset: offset,
      };

      const response = await auditLogApi.getUserAuditLogs(user.id, request);

      if (isInitialLoad) {
        setAuditLogs(response.auditLogs);
      } else {
        setAuditLogs((prev) => {
          const existingIds = new Set(prev.map((log) => log.id));
          const newLogs = response.auditLogs.filter((log) => !existingIds.has(log.id));
          return [...prev, ...newLogs];
        });
      }

      setTotal(response.total);
      setHasMore(response.auditLogs.length === pageSize);
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  const columns: ColumnsType<AuditLog> = [
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
      dataIndex: 'workspaceName',
      key: 'workspaceName',
      width: 200,
      render: (workspaceId: string | undefined) => (
        <span
          className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
            workspaceId
              ? 'bg-blue-100 text-blue-800'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-white'
          }`}
        >
          {workspaceId || '-'}
        </span>
      ),
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
        <div className="mb-4 flex items-center justify-between">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            {isLoading ? (
              <Spin indicator={<LoadingOutlined spin />} />
            ) : (
              t('auditLogs.loadedOfTotal', { loaded: auditLogs.length, total })
            )}
          </div>
        </div>

        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          </div>
        ) : (
          <>
            <Table
              columns={columns}
              dataSource={auditLogs}
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

            {!hasMore && auditLogs.length > 0 && (
              <div className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                {t('auditLogs.allLoaded', { count: total })}
              </div>
            )}

            {!isLoading && auditLogs.length === 0 && (
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
