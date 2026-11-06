import { describe, expect, it } from 'vitest';

import type { PhysicalBackupConfigInput } from '@/entities/backup-configs';
import { buildPhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/buildPhysicalBackupConfigDraft.fixture';
import { DEFAULT_FULL_BACKUPS_RETENTION } from '@/pages/databases/model/physical-backups/defaultFullBackupsRetention';
import { toPhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/toPhysicalBackupConfigDraft';

describe('toPhysicalBackupConfigDraft', () => {
  it('fills the full-backup retention with the defaults when the server returned none', () => {
    const storedDraft = buildPhysicalBackupConfigDraft({ retention: { policy: 'CHAINS' } });
    const input: PhysicalBackupConfigInput = {
      ...storedDraft,
      settings: {
        ...storedDraft.settings,
        retention: { policy: 'CHAINS', chainsCount: 3, fullBackups: null },
      },
    };

    const draft = toPhysicalBackupConfigDraft(input);

    expect(draft.settings.retention).toEqual({
      policy: 'CHAINS',
      chainsCount: 3,
      fullBackups: DEFAULT_FULL_BACKUPS_RETENTION,
    });
  });

  it('keeps a stored full-backup retention and the settings the form does not edit', () => {
    const storedFullBackups = { ...DEFAULT_FULL_BACKUPS_RETENTION, gfsYears: 9 };
    const input = buildPhysicalBackupConfigDraft({
      settings: { walLagThresholdBytes: 1024 },
      fullBackups: storedFullBackups,
    });

    const draft = toPhysicalBackupConfigDraft(input);

    expect(draft.settings.retention.fullBackups).toEqual(storedFullBackups);
    expect(draft.settings.walLagThresholdBytes).toBe(1024);
  });
});
