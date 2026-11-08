import { Button, Tooltip } from '@mantine/core';
import { CircleAlert, CircleCheck, CircleX, Lock, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { BackupEncryption } from '@/entities/backup-configs';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import {
  LogicalBackupStatus,
  LOGICAL_BACKUP_STATUS_LABEL_KEYS,
} from '@/pages/databases/logical-backups/model/LogicalBackupStatus';
import {
  LogicalBackupVerificationStatus,
  LOGICAL_BACKUP_VERIFICATION_STATUS_LABEL_KEYS,
} from '@/pages/databases/logical-backups/model/LogicalBackupVerificationStatus';
import { VERIFICATION_STATUS_BADGE_STYLES } from '@/pages/databases/logical-backups/model/verificationStatusBadgeStyles';

interface Props {
  backup: LogicalBackup;
  onShowError: (backup: LogicalBackup) => void;
}

export const LogicalBackupStatusLabel = ({ backup, onShowError }: Props) => {
  const { t } = useTranslation();

  const renderStatus = () => {
    const { status } = backup;

    if (status === LogicalBackupStatus.FAILED) {
      return (
        <Tooltip label={t('backups.list.clickToSeeErrorDetails')}>
          <Button
            variant="transparent"
            size="xs"
            color="danger"
            className="!h-auto !px-0 underline"
            leftSection={<CircleAlert size="1em" className="inline-block shrink-0 align-middle" />}
            onClick={() => onShowError(backup)}
          >
            {t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}
          </Button>
        </Tooltip>
      );
    }

    if (status === LogicalBackupStatus.COMPLETED) {
      return (
        <div className="flex items-center text-success">
          <CircleCheck
            className="mr-2 inline-block shrink-0 align-middle"
            style={{ fontSize: 16 }}
            size="1em"
          />
          <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
          {backup.encryption === BackupEncryption.ENCRYPTED && (
            <Tooltip label={t('backups.logical.list.encrypted')}>
              <Lock
                className="ml-1 inline-block shrink-0 align-middle"
                style={{ fontSize: 14 }}
                size="1em"
              />
            </Tooltip>
          )}
        </div>
      );
    }

    if (status === LogicalBackupStatus.IN_PROGRESS) {
      return (
        <div className="flex items-center font-bold text-accent">
          <RefreshCw className="inline-block shrink-0 animate-spin align-middle" size="1em" />
          <span className="ml-2">{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</span>
        </div>
      );
    }

    if (status === LogicalBackupStatus.CANCELED) {
      return (
        <div className="flex items-center text-muted">
          <CircleX
            className="mr-2 inline-block shrink-0 align-middle"
            style={{ fontSize: 16 }}
            size="1em"
          />
          <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
        </div>
      );
    }

    return <span className="font-bold">{status}</span>;
  };

  const badgeStyle = VERIFICATION_STATUS_BADGE_STYLES[backup.verificationStatus];
  const verificationLabelKey =
    LOGICAL_BACKUP_VERIFICATION_STATUS_LABEL_KEYS[backup.verificationStatus];
  const isVerificationShown =
    backup.verificationStatus !== LogicalBackupVerificationStatus.NOT_VERIFIED &&
    badgeStyle &&
    verificationLabelKey;

  if (!isVerificationShown) return renderStatus();

  return (
    <div className="flex items-center">
      {renderStatus()}
      <span
        className={`ml-3 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${badgeStyle.pillClasses}`}
      >
        <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
        {t(verificationLabelKey)}
      </span>
    </div>
  );
};
