import { PhysicalDatabaseBackupType } from '../../../../entities/databases';
import { createDefaultPhysicalBackupConfigDraft } from './createDefaultPhysicalBackupConfigDraft';
import type { FullBackupsRetention } from './FullBackupsRetention';
import type { PhysicalBackupConfigDraft } from './PhysicalBackupConfigDraft';

type PhysicalDraftSettings = PhysicalBackupConfigDraft['settings'];

interface DraftChange {
  config?: Partial<Omit<PhysicalBackupConfigDraft, 'settings'>>;
  settings?: Partial<Omit<PhysicalDraftSettings, 'retention'>>;
  retention?: Partial<Omit<PhysicalDraftSettings['retention'], 'fullBackups'>>;
  fullBackups?: Partial<FullBackupsRetention>;
  backupType?: PhysicalDatabaseBackupType;
}

export const buildPhysicalBackupConfigDraft = ({
  config,
  settings,
  retention,
  fullBackups,
  backupType = PhysicalDatabaseBackupType.FULL_INCREMENTAL,
}: DraftChange = {}): PhysicalBackupConfigDraft => {
  const defaults = createDefaultPhysicalBackupConfigDraft(backupType);

  return {
    ...defaults,
    ...config,
    settings: {
      ...defaults.settings,
      ...settings,
      retention: {
        ...defaults.settings.retention,
        ...retention,
        fullBackups: { ...defaults.settings.retention.fullBackups, ...fullBackups },
      },
    },
  };
};
