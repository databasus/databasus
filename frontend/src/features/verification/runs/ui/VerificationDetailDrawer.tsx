import { Button, Drawer, Skeleton, Table, Tag, type TagProps } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { TFunction } from 'i18next';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type VerificationDetails,
  type VerificationTableStat,
  VERIFICATION_STATUS_LABEL_KEYS,
  VERIFICATION_TRIGGER_LABEL_KEYS,
  VerificationStatus,
  VerificationTrigger,
  verificationRunsApi,
} from '../../../../entity/verification/runs';
import { type LocaleContextValue, translateApiError, useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';

interface Props {
  verificationId: string;
  onClose: () => void;
}

const formatDurationMs = (durationMs?: number) => {
  if (durationMs === undefined || durationMs === null) {
    return '-';
  }

  const totalSeconds = Math.floor(durationMs / 1000);

  if (totalSeconds < 60) {
    // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
    return `${totalSeconds}s`;
  }

  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
  return `${minutes}m ${seconds}s`;
};

const formatSizeBytes = (
  sizeBytes: number | undefined,
  formatNumber: LocaleContextValue['formatNumber'],
) => {
  if (sizeBytes === undefined || sizeBytes === null || sizeBytes <= 0) {
    return '-';
  }

  const sizeMb = sizeBytes / (1024 * 1024);

  if (sizeMb >= 1024) {
    return `${formatNumber(Number((sizeMb / 1024).toFixed(2)))} GB`;
  }

  return `${formatNumber(Number(sizeMb.toFixed(2)))} MB`;
};

const VERIFICATION_STATUS_TAG_COLORS: Record<VerificationStatus, TagProps['color']> = {
  [VerificationStatus.COMPLETED]: 'green',
  [VerificationStatus.FAILED]: 'red',
  [VerificationStatus.RUNNING]: 'blue',
  [VerificationStatus.PENDING]: 'default',
  [VerificationStatus.CANCELED]: 'default',
};

const renderStatusTag = (status: VerificationStatus, t: TFunction) => (
  <Tag color={VERIFICATION_STATUS_TAG_COLORS[status]}>
    {t(VERIFICATION_STATUS_LABEL_KEYS[status])}
  </Tag>
);

const renderTriggerTag = (trigger: VerificationTrigger, t: TFunction) => (
  <Tag color={trigger === VerificationTrigger.MANUAL ? 'blue' : 'purple'}>
    {t(VERIFICATION_TRIGGER_LABEL_KEYS[trigger])}
  </Tag>
);

const renderTimestamp = (
  iso: string,
  formatRelativeTime: LocaleContextValue['formatRelativeTime'],
) => (
  <span className="flex flex-col items-end">
    <span>{dayjs.utc(iso).local().format(getUserTimeFormat().format)}</span>
    <span className="text-xs text-gray-500 dark:text-gray-400">
      ({formatRelativeTime(dayjs.utc(iso))})
    </span>
  </span>
);

const renderInfoRow = (label: string, value: React.ReactNode) => (
  <div className="flex items-baseline justify-between border-b border-gray-100 py-1 last:border-b-0 dark:border-gray-700">
    <span className="text-xs text-gray-500 dark:text-gray-400">{label}</span>
    <span className="text-sm text-gray-800 dark:text-gray-200">{value}</span>
  </div>
);

const renderSection = (title: string, children: React.ReactNode) => (
  <div className="mb-4">
    <div className="mb-1 text-xs font-semibold tracking-wide text-gray-400 uppercase dark:text-gray-500">
      {title}
    </div>
    {children}
  </div>
);

export const VerificationDetailDrawer = ({ verificationId, onClose }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const [verification, setVerification] = useState<VerificationDetails | undefined>();
  const [isLoading, setIsLoading] = useState(true);
  const [hasLoadError, setHasLoadError] = useState(false);

  const loadVerification = () => {
    setIsLoading(true);
    setHasLoadError(false);

    verificationRunsApi
      .getById(verificationId)
      .then(setVerification)
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
        setHasLoadError(true);
      })
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadVerification();
  }, [verificationId]);

  const sortedStats = useMemo(
    () => [...(verification?.tableStats ?? [])].sort((a, b) => b.rowCount - a.rowCount),
    [verification],
  );

  const tableStatColumns: ColumnsType<VerificationTableStat> = [
    {
      title: t('verification.runs.details.tableStats.schema'),
      dataIndex: 'schemaName',
      key: 'schemaName',
      width: 140,
      render: (schemaName: string) => <span className="font-mono text-xs">{schemaName}</span>,
    },
    {
      title: t('verification.runs.details.tableStats.table'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <span className="font-mono text-xs">{name}</span>,
    },
    {
      title: t('verification.runs.details.tableStats.rows'),
      dataIndex: 'rowCount',
      key: 'rowCount',
      width: 120,
      align: 'right',
      render: (rowCount: number) => <span className="tabular-nums">{formatNumber(rowCount)}</span>,
      sorter: (a, b) => a.rowCount - b.rowCount,
    },
  ];

  return (
    <Drawer
      title={t('verification.runs.details.title')}
      placement="right"
      width={520}
      onClose={onClose}
      open={true}
      maskClosable={true}
    >
      {isLoading ? (
        <Skeleton active paragraph={{ rows: 10 }} />
      ) : hasLoadError || !verification ? (
        <div className="flex flex-col items-center gap-3 py-12 text-center">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            {t('verification.runs.details.loadFailed')}
          </div>
          <Button onClick={loadVerification}>{t('common.actions.retry')}</Button>
        </div>
      ) : (
        <div>
          {verification.failMessage &&
            (verification.status === VerificationStatus.CANCELED ? (
              <div className="mb-4 rounded-md border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
                <div className="mb-1 font-semibold">
                  {t('verification.runs.details.cancellationReason')}
                </div>
                <div className="break-words whitespace-pre-wrap">{verification.failMessage}</div>
              </div>
            ) : (
              <div className="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
                <div className="mb-1 font-semibold">{t('verification.runs.details.failure')}</div>
                <div className="break-words whitespace-pre-wrap">{verification.failMessage}</div>
              </div>
            ))}

          {renderSection(
            t('verification.runs.details.sections.status'),
            <>
              {renderInfoRow(t('common.fields.status'), renderStatusTag(verification.status, t))}
              {renderInfoRow(
                t('verification.runs.fields.trigger'),
                renderTriggerTag(verification.trigger, t),
              )}
              {renderInfoRow(
                t('verification.runs.details.fields.attempt'),
                verification.attemptCount,
              )}
            </>,
          )}

          {renderSection(
            t('verification.runs.details.sections.timeline'),
            <>
              {renderInfoRow(
                t('verification.runs.fields.createdAt'),
                renderTimestamp(verification.createdAt, formatRelativeTime),
              )}
              {verification.startedAt &&
                renderInfoRow(
                  t('verification.runs.details.fields.startedAt'),
                  renderTimestamp(verification.startedAt, formatRelativeTime),
                )}
              {verification.finishedAt &&
                renderInfoRow(
                  t('verification.runs.details.fields.finishedAt'),
                  renderTimestamp(verification.finishedAt, formatRelativeTime),
                )}
            </>,
          )}

          {renderSection(
            t('verification.runs.details.sections.results'),
            <>
              {renderInfoRow(
                t('verification.runs.details.fields.restoreDuration'),
                formatDurationMs(verification.restoreDurationMs),
              )}
              {renderInfoRow(
                t('verification.runs.details.fields.verifyDuration'),
                formatDurationMs(verification.verifyDurationMs),
              )}
              {renderInfoRow(
                t('verification.runs.details.fields.restoredDbSize'),
                formatSizeBytes(verification.restoredDatabaseSizeBytes, formatNumber),
              )}
              {renderInfoRow(
                t('verification.runs.details.fields.schemas'),
                verification.schemaCount ?? '-',
              )}
              {renderInfoRow(
                t('verification.runs.details.fields.tables'),
                verification.tableCount ?? '-',
              )}
              {verification.pgRestoreExitCode !== undefined &&
                verification.pgRestoreExitCode !== null &&
                renderInfoRow(
                  t('verification.runs.details.fields.pgRestoreExitCode'),
                  verification.pgRestoreExitCode,
                )}
            </>,
          )}

          <h3 className="mt-2 mb-2 text-base font-semibold dark:text-white">
            {t('verification.runs.details.tableStats.title')}
          </h3>
          {sortedStats.length === 0 ? (
            <div className="text-sm text-gray-500 dark:text-gray-400">
              {t('verification.runs.details.tableStats.empty')}
            </div>
          ) : (
            <Table
              bordered
              size="small"
              columns={tableStatColumns}
              dataSource={sortedStats}
              rowKey="id"
              pagination={false}
            />
          )}
        </div>
      )}
    </Drawer>
  );
};
