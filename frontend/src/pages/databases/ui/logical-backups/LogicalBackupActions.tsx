import {
  CloseCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  SafetyOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { DatabaseType } from '@/entities/databases';
import type { LogicalBackup } from '@/pages/databases/model/LogicalBackup';
import { LogicalBackupStatus } from '@/pages/databases/model/LogicalBackupStatus';
import { RowActionButton } from '@/pages/databases/ui/RowActionButton';
import type { TranslationKey } from '@/shared/i18n';

// Physical databases have their own backups list; the generic text keeps the map exhaustive.
const DOWNLOAD_TOOLTIP_KEYS: Record<DatabaseType, TranslationKey> = {
  [DatabaseType.POSTGRES_LOGICAL]: 'backups.logical.list.actions.download.postgresql',
  [DatabaseType.POSTGRES_PHYSICAL]: 'backups.logical.list.actions.download.other',
  [DatabaseType.MYSQL]: 'backups.logical.list.actions.download.mysql',
  [DatabaseType.MARIADB]: 'backups.logical.list.actions.download.mariadb',
  [DatabaseType.MONGODB]: 'backups.logical.list.actions.download.mongodb',
};

export interface LogicalBackupRowActions {
  databaseType: DatabaseType;
  isCanManageDBs: boolean;
  cancellingBackupId?: string;
  deletingBackupId?: string;
  downloadingBackupId?: string;
  verifyingBackupId?: string;
  onCancel: (backupId: string) => void;
  onDelete: (backupId: string) => void;
  onVerify: (backupId: string) => void;
  onRestore: (backup: LogicalBackup) => void;
  onDownload: (backupId: string) => void;
}

interface Props {
  backup: LogicalBackup;
  actions: LogicalBackupRowActions;
}

export const LogicalBackupActions = ({ backup, actions }: Props) => {
  const { t } = useTranslation();

  const { cancellingBackupId, deletingBackupId, downloadingBackupId, verifyingBackupId } = actions;

  if (backup.status === LogicalBackupStatus.IN_PROGRESS && actions.isCanManageDBs) {
    return (
      <div className="flex gap-2">
        <RowActionButton
          title={t('backups.list.actions.cancelBackup')}
          icon={<CloseCircleOutlined />}
          className="!text-danger"
          isPending={cancellingBackupId === backup.id}
          isDisabled={Boolean(cancellingBackupId)}
          onClick={() => actions.onCancel(backup.id)}
        />
      </div>
    );
  }

  if (backup.status !== LogicalBackupStatus.COMPLETED) return <div className="flex gap-2" />;

  if (deletingBackupId === backup.id) {
    return (
      <div className="flex gap-2">
        <RowActionButton
          title={t('backups.list.actions.deleteBackup')}
          icon={<DeleteOutlined />}
          className="!text-danger"
          isPending
          onClick={() => {}}
        />
      </div>
    );
  }

  return (
    <div className="flex gap-2">
      {actions.isCanManageDBs && (
        <RowActionButton
          title={t('backups.list.actions.deleteBackup')}
          icon={<DeleteOutlined />}
          className="!text-danger"
          isDisabled={Boolean(deletingBackupId)}
          onClick={() => actions.onDelete(backup.id)}
        />
      )}

      {actions.isCanManageDBs && actions.databaseType === DatabaseType.POSTGRES_LOGICAL && (
        <RowActionButton
          title={t('backups.logical.list.actions.verifyRestore')}
          icon={<SafetyOutlined />}
          className="!text-accent"
          isPending={verifyingBackupId === backup.id}
          isDisabled={Boolean(verifyingBackupId)}
          onClick={() => actions.onVerify(backup.id)}
        />
      )}

      <RowActionButton
        title={t('backups.logical.list.actions.restore')}
        icon={<CloudUploadOutlined />}
        className="!text-accent"
        onClick={() => actions.onRestore(backup)}
      />

      <RowActionButton
        title={t(DOWNLOAD_TOOLTIP_KEYS[actions.databaseType])}
        icon={<DownloadOutlined />}
        className="!text-accent"
        isPending={downloadingBackupId === backup.id}
        isDisabled={Boolean(downloadingBackupId)}
        onClick={() => actions.onDownload(backup.id)}
      />
    </div>
  );
};
