import { Alert, Drawer, Skeleton } from '@mantine/core';
import dayjs from 'dayjs';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { verificationRunsApi } from '@/pages/databases/verification/api/verificationRunsApi';
import { VerificationStatus } from '@/pages/databases/verification/model/VerificationStatus';
import { formatOptionalDuration } from '@/pages/databases/verification/ui/formatOptionalDuration';
import { VerificationFailureDetails } from '@/pages/databases/verification/ui/VerificationFailureDetails';
import { VerificationStatusLabel } from '@/pages/databases/verification/ui/VerificationStatusLabel';
import { VerificationTriggerTag } from '@/pages/databases/verification/ui/VerificationTriggerTag';
import { useSessionQuery } from '@/shared/api';
import { type LocaleContextValue, useLocale } from '@/shared/i18n';
import { getUserTimeFormat } from '@/shared/time';
import { closeOverlayOnEscape, OverlayFocusReturn, LoadFailure } from '@/shared/ui';
import tableStyles from '@/shared/ui/nativeTable.module.css';

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
  <div>
    <div className="mb-1 text-xs font-semibold tracking-wide text-muted uppercase">{title}</div>
    {children}
  </div>
);

export const VerificationDetailDrawer = ({ verificationId, isOpen, onClose, onClosed }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const query = useSessionQuery({
    queryKey: ['verification-details', verificationId],
    queryFn: (signal) => verificationRunsApi.getById(verificationId, { signal }),
  });
  const verification = query.data;
  const isLoading = query.isPending;
  const loadError = query.error ?? undefined;
  const loadVerification = () => query.refetch();

  const sortedStats = useMemo(
    () => [...(verification?.tableStats ?? [])].sort((a, b) => b.rowCount - a.rowCount),
    [verification],
  );

  const [sortDirection, setSortDirection] = useState<'ascend' | 'descend' | null>(null);
  const SortIcon =
    sortDirection === 'ascend' ? ArrowUp : sortDirection === 'descend' ? ArrowDown : ArrowUpDown;
  const visibleStats = sortDirection
    ? [...sortedStats].sort(
        (first, second) =>
          (first.rowCount - second.rowCount) * (sortDirection === 'ascend' ? 1 : -1),
      )
    : sortedStats;

  return (
    <Drawer
      closeOnEscape={false}
      onKeyDown={(event) => closeOverlayOnEscape(event, { isOpen: isOpen, onClose: onClose })}
      title={t('verification.runs.details.title')}
      position="right"
      size={520}
      onClose={onClose}
      opened={isOpen}
      onExitTransitionEnd={onClosed}
      closeOnClickOutside={true}
      closeButtonProps={{ 'aria-label': t('common.actions.close') }}
    >
      {isOpen && <OverlayFocusReturn />}
      {isLoading ? (
        <div aria-busy="true">
          {Array.from({ length: 10 }, (_, index) => (
            <Skeleton key={index} height={16} className="mb-4" />
          ))}
        </div>
      ) : loadError !== undefined || !verification ? (
        <LoadFailure error={loadError} onRetry={() => void loadVerification()} />
      ) : (
        <div className="flex flex-col gap-4">
          {verification.failMessage &&
            (verification.status === VerificationStatus.CANCELED ? (
              <div className="rounded-md border border-border bg-raised px-3 py-2 text-sm">
                <div className="mb-1 font-semibold">
                  {t('verification.runs.details.cancellationReason')}
                </div>
                <div className="break-words whitespace-pre-wrap">{verification.failMessage}</div>
              </div>
            ) : (
              <Alert
                color="danger"
                title={
                  <div className="space-y-1">
                    <div className="font-semibold">{t('verification.runs.details.failure')}</div>
                    <div className="break-words whitespace-pre-wrap">
                      {verification.failMessage}
                    </div>
                  </div>
                }
              />
            ))}

          {isOpen &&
            verification.status === VerificationStatus.FAILED &&
            verification.failureDiagnostic && (
              <VerificationFailureDetails
                key={verificationId}
                diagnostic={verification.failureDiagnostic}
              />
            )}

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

          <div className="space-y-2">
            <h3 className="text-base font-semibold">
              {t('verification.runs.details.tableStats.title')}
            </h3>
            {sortedStats.length === 0 ? (
              <div className="text-sm text-muted">
                {t('verification.runs.details.tableStats.empty')}
              </div>
            ) : (
              <div
                className="relative overflow-x-auto"
                role="region"
                aria-label={t('verification.runs.details.tableStats.title')}
                tabIndex={0}
              >
                <table className={`${tableStyles.table} ${tableStyles.bordered}`}>
                  <thead>
                    <tr>
                      <th scope="col">{t('verification.runs.details.tableStats.schema')}</th>
                      <th scope="col">{t('verification.runs.details.tableStats.table')}</th>
                      <th
                        scope="col"
                        style={{ textAlign: 'right' }}
                        aria-sort={
                          sortDirection === 'ascend'
                            ? 'ascending'
                            : sortDirection === 'descend'
                              ? 'descending'
                              : 'none'
                        }
                      >
                        <button
                          type="button"
                          className="inline-flex max-w-full items-center gap-1 text-start"
                          onClick={() =>
                            setSortDirection((direction) =>
                              direction === 'ascend'
                                ? 'descend'
                                : direction === 'descend'
                                  ? null
                                  : 'ascend',
                            )
                          }
                        >
                          {t('verification.runs.details.tableStats.rows')}
                          <SortIcon
                            aria-hidden
                            size={12}
                            className={
                              sortDirection ? 'shrink-0 text-accent' : 'shrink-0 text-muted'
                            }
                          />
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibleStats.map((stat) => (
                      <tr key={((stat) => `${stat.schemaName}.${stat.name}`)(stat)}>
                        <td>
                          <span className="font-mono text-xs">{stat.schemaName}</span>
                        </td>
                        <td>
                          <span className="font-mono text-xs">{stat.name}</span>
                        </td>
                        <td style={{ textAlign: 'right' }}>
                          <span className="tabular-nums">{formatNumber(stat.rowCount)}</span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
    </Drawer>
  );
};
