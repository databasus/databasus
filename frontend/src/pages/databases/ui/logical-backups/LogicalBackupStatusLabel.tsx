import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  LockOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import { BackupEncryption } from '../../../../entities/backup-configs';
import type { LogicalBackup } from '../../model/LogicalBackup';
import { LogicalBackupStatus } from '../../model/LogicalBackupStatus';
import { LOGICAL_BACKUP_STATUS_LABEL_KEYS } from '../../model/LogicalBackupStatusLabelKeys';
import { LogicalBackupVerificationStatus } from '../../model/LogicalBackupVerificationStatus';
import { LOGICAL_BACKUP_VERIFICATION_STATUS_LABEL_KEYS } from '../../model/LogicalBackupVerificationStatusLabelKeys';
import { VERIFICATION_STATUS_BADGE_STYLES } from '../../model/verificationStatusBadgeStyles';

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
        <Tooltip title={t('backups.list.clickToSeeErrorDetails')}>
          <Button
            type="link"
            size="small"
            danger
            className="!h-auto !px-0 underline"
            icon={<ExclamationCircleOutlined />}
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
          <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
          {backup.encryption === BackupEncryption.ENCRYPTED && (
            <Tooltip title={t('backups.logical.list.encrypted')}>
              <LockOutlined className="ml-1" style={{ fontSize: 14 }} />
            </Tooltip>
          )}
        </div>
      );
    }

    if (status === LogicalBackupStatus.IN_PROGRESS) {
      return (
        <div className="flex items-center font-bold text-accent">
          <SyncOutlined spin />
          <span className="ml-2">{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</span>
        </div>
      );
    }

    if (status === LogicalBackupStatus.CANCELED) {
      return (
        <div className="flex items-center text-muted">
          <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
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
