import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import dayjs from 'dayjs';
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
            <Tooltip title={t('restores.card.errorDetailsTooltip')}>
              <Button
                type="link"
                size="small"
                danger
                className="!h-auto !px-0 underline"
                icon={<ExclamationCircleOutlined />}
                onClick={onShowError}
              >
                {t(RESTORE_STATUS_LABEL_KEYS[restore.status])}
              </Button>
            </Tooltip>
          )}

          {restore.status === RestoreStatus.COMPLETED && (
            <div className="flex items-center text-success">
              <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />

              <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
            </div>
          )}

          {restore.status === RestoreStatus.CANCELED && (
            <div className="flex items-center text-muted">
              <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />

              <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
            </div>
          )}

          {restore.status === RestoreStatus.IN_PROGRESS && (
            <div className="flex items-center font-bold text-accent">
              <SyncOutlined spin />
              <span className="ml-2">{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</span>
            </div>
          )}
        </div>

        {restore.status === RestoreStatus.IN_PROGRESS && (
          <div className="ml-2">
            <Tooltip title={t('restores.card.cancelTooltip')}>
              <Button
                type="text"
                size="small"
                className="!text-danger"
                aria-label={t('restores.card.cancelTooltip')}
                icon={<CloseCircleOutlined style={{ fontSize: 16 }} />}
                loading={isCancelling}
                disabled={isCancelDisabled}
                onClick={onCancel}
              />
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
