import { ActionIcon, Button, Tooltip } from '@mantine/core';
import dayjs from 'dayjs';
import { CircleAlert, CircleCheck, CircleX, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { RESTORE_STATUS_LABEL_KEYS, type Restore, RestoreStatus } from '@/entities/restores';
import { useLocale } from '@/shared/i18n';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  restore: Restore;
  backupDurationMs: number;
  isCancelling: boolean;
  isCancelDisabled: boolean;
  onCancel: () => void;
  onShowError: () => void;
}

const EXPECTED_RESTORE_TO_BACKUP_DURATION_RATIO = 5;

export const RestoreCard = ({
  restore,
  backupDurationMs,
  isCancelling,
  isCancelDisabled,
  onCancel,
  onShowError,
}: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const restoreDurationMs =
    restore.status === RestoreStatus.IN_PROGRESS
      ? // eslint-disable-next-line react/purity -- the elapsed time updates with each poll, and a slightly old value does no harm
        Date.now() - new Date(restore.createdAt).getTime()
      : restore.durationMs;

  const minutes = Math.floor(restoreDurationMs / 60000);
  const seconds = Math.floor((restoreDurationMs % 60000) / 1000);
  const milliseconds = restoreDurationMs % 1000;
  // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
  const duration = `${minutes}m ${seconds}s ${milliseconds}ms`;

  const expectedRestoreDurationMs = backupDurationMs * EXPECTED_RESTORE_TO_BACKUP_DURATION_RATIO;
  // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
  const expectedRestoreDuration = `${Math.floor(expectedRestoreDurationMs / 60000)}m ${Math.floor((expectedRestoreDurationMs % 60000) / 1000)}s`;

  return (
    <div className="mb-1 rounded border border-border p-3 text-sm">
      <div className="mb-1 flex items-center justify-between">
        <div className="flex flex-1">
          <div className="w-[75px] min-w-[75px] pr-2">{t('common.fields.status')}</div>

          {restore.status === RestoreStatus.FAILED && (
            <Tooltip label={t('restores.card.errorDetailsTooltip')}>
              <Button
                variant="transparent"
                size="xs"
                color="danger"
                className="!h-auto !px-0 underline"
                leftSection={
                  <CircleAlert size="1em" className="inline-block shrink-0 align-middle" />
                }
                onClick={onShowError}
              >
                {t(RESTORE_STATUS_LABEL_KEYS[restore.status])}
              </Button>
            </Tooltip>
          )}

          {restore.status === RestoreStatus.COMPLETED && (
            <div className="flex items-center text-success">
              <CircleCheck
                className="mr-2 inline-block shrink-0 align-middle"
                style={{ fontSize: 16 }}
                size="1em"
              />

              <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
            </div>
          )}

          {restore.status === RestoreStatus.CANCELED && (
            <div className="flex items-center text-muted">
              <CircleX
                className="mr-2 inline-block shrink-0 align-middle"
                style={{ fontSize: 16 }}
                size="1em"
              />

              <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
            </div>
          )}

          {restore.status === RestoreStatus.IN_PROGRESS && (
            <div className="flex items-center font-bold text-accent">
              <RefreshCw className="inline-block shrink-0 animate-spin align-middle" size="1em" />
              <span className="ml-2">{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</span>
            </div>
          )}
        </div>

        {restore.status === RestoreStatus.IN_PROGRESS && (
          <div className="ml-2">
            <Tooltip label={t('restores.card.cancelTooltip')}>
              <ActionIcon
                variant="subtle"
                size="xs"
                className="!text-danger"
                aria-label={t('restores.card.cancelTooltip')}
                loading={isCancelling}
                disabled={isCancelDisabled}
                onClick={onCancel}
                aria-busy={isCancelling}
              >
                <CircleX
                  style={{ fontSize: 16 }}
                  size="1em"
                  className="inline-block shrink-0 align-middle"
                />
              </ActionIcon>
            </Tooltip>
          </div>
        )}
      </div>

      <div className="mb-1 flex">
        <div className="w-[75px] min-w-[75px] pr-2">{t('restores.card.startedAt')}</div>
        <div>
          {dayjs.utc(restore.createdAt).local().format(getUserTimeFormat().format)} (
          {formatRelativeTime(dayjs.utc(restore.createdAt).local())})
        </div>
      </div>

      {restore.status === RestoreStatus.IN_PROGRESS && (
        <div className="flex">
          <div className="w-[75px] min-w-[75px] pr-2">{t('restores.card.duration')}</div>
          <div>
            <div>{duration}</div>
            <div className="mt-2 text-xs text-muted">
              {t('restores.card.expectedDurationHint')}
              <br />
              <br />
              {t('restores.card.expectedDurationLimit', { duration: expectedRestoreDuration })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
