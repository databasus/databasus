import type { ReactNode } from 'react';

import {
  type BackupConfig,
  type LogicalBackupConfigInput,
  toLogicalBackupConfigInput,
} from '@/entities/backup-configs';
import type { Database } from '@/entities/databases';
import { isLogicalBackupConfigChanged } from '@/pages/databases/model/isLogicalBackupConfigChanged';
import { isLogicalBackupConfigComplete } from '@/pages/databases/model/isLogicalBackupConfigComplete';
import { EditBackupConfigSection } from '@/pages/databases/ui/backup-config/EditBackupConfigSection';
import { LogicalBackupConfigFields } from '@/pages/databases/ui/backup-config/LogicalBackupConfigFields';

interface Props {
  database: Database;
  onCancel: () => void;
  onSaved: (config: BackupConfig) => void;
  storageCleanupNotice?: ReactNode;
}

export const EditLogicalBackupConfigSection = ({
  database,
  onCancel,
  onSaved,
  storageCleanupNotice,
}: Props) => (
  <EditBackupConfigSection<LogicalBackupConfigInput>
    database={database}
    isPhysical={false}
    toDraft={toLogicalBackupConfigInput}
    toInput={(draft) => draft}
    isChanged={isLogicalBackupConfigChanged}
    isComplete={isLogicalBackupConfigComplete}
    renderFields={(draft, onChange) => (
      <LogicalBackupConfigFields
        config={draft}
        onChange={onChange}
        workspaceId={database.workspaceId}
        canClearStorage
        storageCleanupNotice={storageCleanupNotice}
      />
    )}
    onCancel={onCancel}
    onSaved={onSaved}
  />
);
