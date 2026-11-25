import { CircleX, CloudUpload, Download, ShieldCheck, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { DatabaseType } from '@/entities/databases';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import { LogicalBackupStatus } from '@/pages/databases/logical-backups/model/LogicalBackupStatus';
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

  if (backup.status === LogicalBackupStatus.IN_PROGRESS) {
    if (!actions.isCanManageDBs) return <div className="flex flex-wrap gap-2" />;

    return (
      <div className="flex flex-wrap gap-2">
        <RowActionButton
          title={t('backups.list.actions.cancelBackup')}
          icon={<CircleX size={18} />}
          className="!text-danger"
          isPending={cancellingBackupId === backup.id}
          isDisabled={Boolean(cancellingBackupId)}
          onClick={() => actions.onCancel(backup.id)}
        />
      </div>
    );
  }

  const isDeletingThisBackup = deletingBackupId === backup.id;

  const deleteButton = actions.isCanManageDBs && (
    <RowActionButton
      title={t('backups.list.actions.deleteBackup')}
      icon={<Trash2 size={18} />}
      className="!text-danger"
      isPending={isDeletingThisBackup}
      isDisabled={Boolean(deletingBackupId) && !isDeletingThisBackup}
      onClick={() => actions.onDelete(backup.id)}
    />
  );

  if (backup.status !== LogicalBackupStatus.COMPLETED || isDeletingThisBackup) {
    return <div className="flex flex-wrap gap-2">{deleteButton}</div>;
  }

  return (
    <div className="flex flex-wrap gap-2">
      {deleteButton}

      {actions.isCanManageDBs && actions.databaseType === DatabaseType.POSTGRES_LOGICAL && (
        <RowActionButton
          title={t('backups.logical.list.actions.verifyRestore')}
          icon={<ShieldCheck size={18} />}
          className="!text-accent"
          isPending={verifyingBackupId === backup.id}
          isDisabled={Boolean(verifyingBackupId)}
          onClick={() => actions.onVerify(backup.id)}
        />
      )}

      <RowActionButton
        title={t('backups.logical.list.actions.restore')}
        icon={<CloudUpload size={18} />}
        className="!text-accent"
        onClick={() => actions.onRestore(backup)}
      />

      <RowActionButton
        title={t(DOWNLOAD_TOOLTIP_KEYS[actions.databaseType])}
        icon={<Download size={18} />}
        className="!text-accent"
        isPending={downloadingBackupId === backup.id}
        isDisabled={Boolean(downloadingBackupId)}
        onClick={() => actions.onDownload(backup.id)}
      />
    </div>
  );
};
