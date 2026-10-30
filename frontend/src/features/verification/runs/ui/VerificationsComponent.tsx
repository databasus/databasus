import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  EyeOutlined,
  PauseCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { Spin, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { TFunction } from 'i18next';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { Database } from '../../../../entities/databases';
import { verificationAgentApi } from '../../../../entities/verification/agents';
import {
  type Verification,
  VERIFICATION_STATUS_LABEL_KEYS,
  VERIFICATION_TRIGGER_LABEL_KEYS,
  VerificationStatus,
  VerificationTrigger,
  verificationRunsApi,
} from '../../../../entities/verification/runs';
import { usePolledPagedList } from '../../../../shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';
import { VerificationDetailDrawer } from './VerificationDetailDrawer';

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const AGENT_AVAILABILITY_REFRESH_MS = 5_000;
const VERIFICATIONS_REFRESH_MS = 1_000;
const VERIFICATIONS_PAGE_SIZE = 50;

const formatDurationMs = (durationMs?: number) => {
  if (durationMs === undefined || durationMs === null) {
    return '-';
  }

  const totalSeconds = Math.floor(durationMs / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
    return `${hours}h ${minutes}m ${seconds}s`;
  }

  if (minutes > 0) {
    // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
    return `${minutes}m ${seconds}s`;
  }

  // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
  return `${seconds}s`;
};

const renderStatus = (status: VerificationStatus, t: TFunction) => {
  const label = t(VERIFICATION_STATUS_LABEL_KEYS[status]);

  if (status === VerificationStatus.COMPLETED) {
    return (
      <div className="flex items-center text-green-600">
        <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>{label}</span>
      </div>
    );
  }

  if (status === VerificationStatus.FAILED) {
    return (
      <div className="flex items-center text-red-600">
        <ExclamationCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>{label}</span>
      </div>
    );
  }

  if (status === VerificationStatus.RUNNING) {
    return (
      <div className="flex items-center font-bold text-blue-600">
        <SyncOutlined spin />
        <span className="ml-2">{label}</span>
      </div>
    );
  }

  if (status === VerificationStatus.PENDING) {
    return (
      <div className="flex items-center text-gray-600">
        <ClockCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>{label}</span>
      </div>
    );
  }

  if (status === VerificationStatus.CANCELED) {
    return (
      <div className="flex items-center text-gray-500">
        <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>{label}</span>
      </div>
    );
  }

  return (
    <div className="flex items-center">
      <PauseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
      <span>{label}</span>
    </div>
  );
};

const renderTrigger = (trigger: VerificationTrigger, t: TFunction) => (
  <Tag color={trigger === VerificationTrigger.MANUAL ? 'blue' : 'purple'}>
    {t(VERIFICATION_TRIGGER_LABEL_KEYS[trigger])}
  </Tag>
);

export const VerificationsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
}: Props) => {
  const { t } = useTranslation();
  const { locale, formatRelativeTime } = useLocale();
  const [selectedVerificationId, setSelectedVerificationId] = useState<string | undefined>();
  const [hasAgents, setHasAgents] = useState<boolean | undefined>();
  const [cancellingVerificationId, setCancellingVerificationId] = useState<string | undefined>();

  const fetchVerificationsPage = async (after: string | undefined) => {
    const response = await verificationRunsApi.listByDatabase(
      database.id,
      VERIFICATIONS_PAGE_SIZE,
      after,
    );
    return { items: response.items, nextCursor: response.nextCursor };
  };

  const cancelVerification = async (id: string) => {
    setCancellingVerificationId(id);

    try {
      await verificationRunsApi.cancel(id);
      await refreshVerifications();
    } catch (e) {
      alert(translateApiError(e, t));
    } finally {
      setCancellingVerificationId(undefined);
    }
  };

  const renderActions = (record: Verification) => {
    if (
      record.status === VerificationStatus.PENDING ||
      record.status === VerificationStatus.RUNNING
    ) {
      if (!isCanManageDBs) {
        return <span className="text-xs text-gray-400">-</span>;
      }

      const isCancelling = cancellingVerificationId === record.id;

      return (
        <div className="flex gap-2 text-lg">
          {isCancelling ? (
            <SyncOutlined spin />
          ) : (
            <Tooltip title={t('verification.runs.actions.cancel')}>
              <CloseCircleOutlined
                className="cursor-pointer"
                onClick={() => {
                  if (cancellingVerificationId) return;
                  cancelVerification(record.id);
                }}
                style={{ color: '#ff0000', opacity: cancellingVerificationId ? 0.2 : 1 }}
              />
            </Tooltip>
          )}
        </div>
      );
    }

    if (
      record.status === VerificationStatus.COMPLETED ||
      record.status === VerificationStatus.FAILED ||
      record.status === VerificationStatus.CANCELED
    ) {
      return (
        <div className="flex gap-2 text-lg">
          <Tooltip title={t('verification.runs.actions.viewDetails')}>
            <EyeOutlined
              className="cursor-pointer"
              onClick={() => setSelectedVerificationId(record.id)}
              style={{ color: '#155dfc' }}
            />
          </Tooltip>
        </div>
      );
    }

    return <span className="text-xs text-gray-400">-</span>;
  };

  const {
    items: verifications,
    isLoading,
    isLoadingMore,
    hasMore,
    refreshFirstPage: refreshVerifications,
  } = usePolledPagedList({
    fetchPage: fetchVerificationsPage,
    getItemKey: (verification: Verification) => verification.id,
    refreshIntervalMs: VERIFICATIONS_REFRESH_MS,
    scrollContainerRef,
    resetKey: database.id,
    onError: (e) => alert(translateApiError(e, t)),
  });

  useEffect(() => {
    let isCancelled = false;

    const loadAvailability = () => {
      verificationAgentApi
        .getAgentCount()
        .then((availability) => {
          if (isCancelled) return;
          setHasAgents(availability.agentCount > 0);
        })
        .catch(() => {
          if (isCancelled) return;
          setHasAgents(undefined);
        });
    };

    loadAvailability();
    const intervalId = setInterval(loadAvailability, AGENT_AVAILABILITY_REFRESH_MS);

    return () => {
      isCancelled = true;
      clearInterval(intervalId);
    };
  }, []);

  const columns: ColumnsType<Verification> = [
    {
      title: t('verification.runs.fields.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 220,
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-gray-500 dark:text-gray-400">
            ({formatRelativeTime(dayjs.utc(createdAt))})
          </span>
        </div>
      ),
    },
    {
      title: t('common.fields.status'),
      dataIndex: 'status',
      key: 'status',
      width: 160,
      render: (status: VerificationStatus) => renderStatus(status, t),
    },
    {
      title: t('verification.runs.fields.trigger'),
      dataIndex: 'trigger',
      key: 'trigger',
      width: 110,
      render: (trigger: VerificationTrigger) => renderTrigger(trigger, t),
    },
    {
      title: (
        <div className="flex items-center">
          {t('verification.runs.fields.duration')}
          <Tooltip className="ml-1" title={t('verification.runs.durationTooltip')}>
            <ExclamationCircleOutlined />
          </Tooltip>
        </div>
      ),
      key: 'duration',
      width: 130,
      render: (_, record: Verification) => {
        const restoreMs = record.restoreDurationMs ?? 0;
        const verifyMs = record.verifyDurationMs ?? 0;
        const total = restoreMs + verifyMs;

        if (total === 0) {
          return '-';
        }

        return (
          <div className="text-sm">
            <div>{formatDurationMs(total)}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {t('verification.runs.restoreDuration', { duration: formatDurationMs(restoreMs) })}
            </div>
          </div>
        );
      },
    },
    {
      title: t('verification.runs.fields.actions'),
      key: 'actions',
      width: 120,
      render: (_, record: Verification) => renderActions(record),
    },
  ];

  return (
    <div
      className={`w-full bg-white p-3 shadow md:p-5 dark:bg-gray-800 ${isDirectlyUnderTab ? 'rounded-tr-md rounded-br-md rounded-bl-md' : 'rounded-md'}`}
    >
      <h2 className="text-lg font-bold md:text-xl dark:text-white">
        {t('verification.runs.title')}
      </h2>

      <p className="mt-2 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
        {t('verification.runs.description')}
      </p>

      {hasAgents === false && (
        <div className="mt-3 flex max-w-2xl max-w-[400px] items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
          <ExclamationCircleOutlined className="shrink-0" />
          <span>
            <Trans
              i18nKey="verification.runs.noAgentsWarning"
              components={{
                docsLink: (
                  <a
                    href={getWebsitePageUrl('restoreVerification', locale)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:no-underline"
                  />
                ),
              }}
            />
          </span>
        </div>
      )}

      <div className="mt-5 w-full md:max-w-[900px]">
        {/* Desktop table */}
        <div className="hidden md:block">
          <Table
            bordered
            columns={columns}
            dataSource={verifications}
            rowKey="id"
            loading={isLoading}
            size="small"
            pagination={false}
          />
          {!isLoading && verifications.length === 0 && (
            <div className="mt-3 text-center text-sm text-gray-500 dark:text-gray-400">
              {t('verification.runs.empty')}
            </div>
          )}
          {isLoadingMore && (
            <div className="mt-2 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && verifications.length > 0 && (
            <div className="mt-2 text-center text-gray-500 dark:text-gray-400">
              {t('verification.runs.allLoaded', { total: verifications.length })}
            </div>
          )}
        </div>

        {/* Mobile card list */}
        <div className="md:hidden">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <Spin />
            </div>
          ) : verifications.length === 0 ? (
            <div className="py-8 text-center text-gray-500 dark:text-gray-400">
              {t('verification.runs.empty')}
            </div>
          ) : (
            verifications.map((verification) => (
              <div
                key={verification.id}
                className="mb-2 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      {t('verification.runs.fields.createdAt')}
                    </div>
                    <div className="text-sm font-medium">
                      {dayjs.utc(verification.createdAt).local().format(getUserTimeFormat().format)}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      ({formatRelativeTime(dayjs.utc(verification.createdAt))})
                    </div>
                  </div>
                  <div>{renderStatus(verification.status, t)}</div>
                </div>

                <div className="mt-3 flex items-center gap-2">
                  {renderTrigger(verification.trigger, t)}
                  {verification.attemptCount > 1 && (
                    <span className="text-xs text-gray-500">
                      {t('verification.runs.attempt', { attempt: verification.attemptCount })}
                    </span>
                  )}
                </div>

                {verification.failMessage && (
                  <div
                    className={`mt-2 text-xs ${
                      verification.status === VerificationStatus.CANCELED
                        ? 'text-gray-500'
                        : 'text-red-600'
                    }`}
                  >
                    {verification.failMessage}
                  </div>
                )}

                <div className="mt-3 flex items-center justify-end border-t border-gray-200 pt-3 dark:border-gray-700">
                  {renderActions(verification)}
                </div>
              </div>
            ))
          )}

          {isLoadingMore && (
            <div className="mt-3 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && verifications.length > 0 && (
            <div className="mt-3 text-center text-sm text-gray-500 dark:text-gray-400">
              {t('verification.runs.allLoaded', { total: verifications.length })}
            </div>
          )}
        </div>
      </div>

      {selectedVerificationId && (
        <VerificationDetailDrawer
          verificationId={selectedVerificationId}
          onClose={() => setSelectedVerificationId(undefined)}
        />
      )}
    </div>
  );
};
