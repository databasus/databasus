import type { ReactNode } from 'react';

import {
  type BackupConfig,
  type LogicalBackupConfigInput,
  toLogicalBackupConfigInput,
} from '../../../../entities/backup-configs';
import type { Database } from '../../../../entities/databases';
import { isLogicalBackupConfigChanged } from '../../model/isLogicalBackupConfigChanged';
import { isLogicalBackupConfigComplete } from '../../model/isLogicalBackupConfigComplete';
import { EditBackupConfigSection } from './EditBackupConfigSection';
import { LogicalBackupConfigFields } from './LogicalBackupConfigFields';

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
