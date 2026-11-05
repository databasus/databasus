import type { ReactNode } from 'react';

import {
  type BackupConfig,
  toPhysicalBackupConfigInput,
} from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
} from '../../../../entities/databases';
import { isPhysicalBackupConfigChanged } from '../../model/physical-backups/isPhysicalBackupConfigChanged';
import { isPhysicalBackupConfigComplete } from '../../model/physical-backups/isPhysicalBackupConfigComplete';
import type { PhysicalBackupConfigDraft } from '../../model/physical-backups/PhysicalBackupConfigDraft';
import { toPhysicalBackupConfigDraft } from '../../model/physical-backups/toPhysicalBackupConfigDraft';
import { toPhysicalBackupConfigInputFromDraft } from '../../model/physical-backups/toPhysicalBackupConfigInputFromDraft';
import { EditBackupConfigSection } from './EditBackupConfigSection';
import { PhysicalBackupConfigFields } from './PhysicalBackupConfigFields';

interface Props {
  database: Database;
  onCancel: () => void;
  onSaved: (config: BackupConfig) => void;
  storageCleanupNotice?: ReactNode;
}

export const EditPhysicalBackupConfigSection = ({
  database,
  onCancel,
  onSaved,
  storageCleanupNotice,
}: Props) => {
  const backupType =
    database.settings.type === DatabaseType.POSTGRES_PHYSICAL
      ? database.settings.backupType
      : PhysicalDatabaseBackupType.FULL;

  return (
    <EditBackupConfigSection<PhysicalBackupConfigDraft>
      database={database}
      isPhysical
      toDraft={(config) => {
        const input = toPhysicalBackupConfigInput(config);

        return input && toPhysicalBackupConfigDraft(input);
      }}
      toInput={toPhysicalBackupConfigInputFromDraft}
      isChanged={(comparison) => isPhysicalBackupConfigChanged({ ...comparison, backupType })}
      isComplete={(draft) => isPhysicalBackupConfigComplete(draft, backupType)}
      renderFields={(draft, onChange) => (
        <PhysicalBackupConfigFields
          draft={draft}
          onChange={onChange}
          workspaceId={database.workspaceId}
          backupType={backupType}
          canClearStorage
          storageCleanupNotice={storageCleanupNotice}
        />
      )}
      onCancel={onCancel}
      onSaved={onSaved}
    />
  );
};
