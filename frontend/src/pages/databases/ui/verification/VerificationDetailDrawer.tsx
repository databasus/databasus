import { Alert, Drawer, Skeleton, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useEffectEvent, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type LocaleContextValue, useLocale } from '../../../../shared/i18n';
import { RequestRevision } from '../../../../shared/lib/RequestRevision';
import { getUserTimeFormat } from '../../../../shared/time';
import { LoadFailure } from '../../../../shared/ui';
import { verificationRunsApi } from '../../api/verificationRunsApi';
import { type VerificationDetails } from '../../model/verification/VerificationDetails';
import { VerificationStatus } from '../../model/verification/VerificationStatus';
import { type VerificationTableStat } from '../../model/verification/VerificationTableStat';
import { formatOptionalDuration } from './formatOptionalDuration';
import { VerificationStatusLabel } from './VerificationStatusLabel';
import { VerificationTriggerTag } from './VerificationTriggerTag';

interface Props {
  verificationId: string;
  isOpen: boolean;
  onClose: () => void;
  onClosed: () => void;
}

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

const renderTimestamp = (
  iso: string,
  formatRelativeTime: LocaleContextValue['formatRelativeTime'],
) => (
  <span className="flex flex-col items-end">
    <span>{dayjs.utc(iso).local().format(getUserTimeFormat().format)}</span>
    <span className="text-xs text-muted">({formatRelativeTime(dayjs.utc(iso))})</span>
  </span>
);

const renderInfoRow = (label: string, value: React.ReactNode) => (
  <div className="flex items-baseline justify-between border-b border-border py-1 last:border-b-0">
    <span className="text-xs text-muted">{label}</span>
    <div className="text-sm text-content">{value}</div>
  </div>
);

const renderSection = (title: string, children: React.ReactNode) => (
  <div className="mb-4">
    <div className="mb-1 text-xs font-semibold tracking-wide text-muted uppercase">{title}</div>
    {children}
  </div>
);

export const VerificationDetailDrawer = ({ verificationId, isOpen, onClose, onClosed }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const [verification, setVerification] = useState<VerificationDetails | undefined>();
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<unknown>();
  const [loadRevision] = useState(() => new RequestRevision());

  const loadVerification = async () => {
    const revision = loadRevision.Invalidate();

    setIsLoading(true);
    setLoadError(undefined);

    try {
      const loadedVerification = await verificationRunsApi.getById(verificationId);
      if (loadRevision.IsCurrent(revision)) setVerification(loadedVerification);
    } catch (error) {
      if (loadRevision.IsCurrent(revision)) setLoadError(error);
    }

    if (loadRevision.IsCurrent(revision)) setIsLoading(false);
  };

  const startVerificationLoad = useEffectEvent(() => {
    void loadVerification();
  });

  useEffect(() => {
    startVerificationLoad();

    return () => {
      loadRevision.Invalidate();
    };
  }, [verificationId, loadRevision]);

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
      open={isOpen}
      afterOpenChange={(isOpen) => {
        if (!isOpen) onClosed();
      }}
      maskClosable={true}
    >
      {isLoading ? (
        <Skeleton active paragraph={{ rows: 10 }} />
      ) : loadError !== undefined || !verification ? (
        <LoadFailure error={loadError} onRetry={() => void loadVerification()} />
      ) : (
        <div>
          {verification.failMessage &&
            (verification.status === VerificationStatus.CANCELED ? (
              <div className="mb-4 rounded-md border border-border bg-raised p-3 text-sm">
                <div className="mb-1 font-semibold">
                  {t('verification.runs.details.cancellationReason')}
                </div>
                <div className="break-words whitespace-pre-wrap">{verification.failMessage}</div>
              </div>
            ) : (
              <Alert
                className="mb-4"
                type="error"
                message={t('verification.runs.details.failure')}
                description={
                  <div className="break-words whitespace-pre-wrap">{verification.failMessage}</div>
                }
              />
            ))}

          {renderSection(
            t('verification.runs.details.sections.status'),
            <>
              {renderInfoRow(
                t('common.fields.status'),
                <VerificationStatusLabel status={verification.status} />,
              )}
              {renderInfoRow(
                t('verification.runs.fields.trigger'),
                <VerificationTriggerTag trigger={verification.trigger} />,
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
                formatOptionalDuration(verification.restoreDurationMs),
              )}
              {renderInfoRow(
                t('verification.runs.details.fields.verifyDuration'),
                formatOptionalDuration(verification.verifyDurationMs),
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

          <h3 className="mt-2 mb-2 text-base font-semibold">
            {t('verification.runs.details.tableStats.title')}
          </h3>
          {sortedStats.length === 0 ? (
            <div className="text-sm text-muted">
              {t('verification.runs.details.tableStats.empty')}
            </div>
          ) : (
            <Table
              bordered
              size="small"
              columns={tableStatColumns}
              dataSource={sortedStats}
              rowKey={(stat) => `${stat.schemaName}.${stat.name}`}
              pagination={false}
            />
          )}
        </div>
      )}
    </Drawer>
  );
};
