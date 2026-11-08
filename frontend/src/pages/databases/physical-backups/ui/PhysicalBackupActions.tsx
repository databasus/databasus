import { CircleX, CloudUpload, Trash2 } from 'lucide-react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import { PhysicalBackupStatus } from '@/pages/databases/physical-backups/model/PhysicalBackupStatus';
import { PhysicalBackupType } from '@/pages/databases/physical-backups/model/PhysicalBackupType';
import { RowActionButton } from '@/pages/databases/ui/RowActionButton';

export interface PhysicalBackupRowActions {
  isCanManageDBs: boolean;
  cancellingBackupId?: string;
  deletingBackupId?: string;
  onCancel: (backupId: string) => void;
  onDelete: (backup: PhysicalBackupListItem) => void;
  onRestore: (backup: PhysicalBackupListItem) => void;
}

interface Props {
  backup: PhysicalBackupListItem;
  actions: PhysicalBackupRowActions;
}

export const PhysicalBackupActions = ({ backup, actions }: Props): JSX.Element => {
  const { t } = useTranslation();

  const { isCanManageDBs, cancellingBackupId, deletingBackupId } = actions;

  // WAL ranges have no standalone backup identity: they are not restored or deleted on their own,
  // point-in-time restore covers them.
  if (backup.type === PhysicalBackupType.WAL) return <div className="flex gap-2" />;

  return (
    <div className="flex gap-2">
      {backup.status === PhysicalBackupStatus.IN_PROGRESS && isCanManageDBs && (
        <RowActionButton
          title={t('backups.list.actions.cancelBackup')}
          icon={<CircleX size={18} />}
          className="!text-danger"
          isPending={cancellingBackupId === backup.id}
          isDisabled={Boolean(cancellingBackupId)}
          onClick={() => actions.onCancel(backup.id)}
        />
      )}

      {backup.status === PhysicalBackupStatus.COMPLETED && (
        <>
          <RowActionButton
            title={t('backups.physical.list.restoreFromThisBackup')}
            icon={<CloudUpload size={18} />}
            className="!text-accent"
            onClick={() => actions.onRestore(backup)}
          />

          {isCanManageDBs && (
            <RowActionButton
              title={t('backups.list.actions.deleteBackup')}
              icon={<Trash2 size={18} />}
              className="!text-danger"
              isPending={deletingBackupId === backup.id}
              isDisabled={Boolean(deletingBackupId)}
              onClick={() => actions.onDelete(backup)}
            />
          )}
        </>
      )}
    </div>
  );
};
