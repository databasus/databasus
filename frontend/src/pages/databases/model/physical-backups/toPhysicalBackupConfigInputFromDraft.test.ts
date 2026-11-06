import { describe, expect, it } from 'vitest';

import { buildPhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/buildPhysicalBackupConfigDraft.fixture';
import { toPhysicalBackupConfigInputFromDraft } from '@/pages/databases/model/physical-backups/toPhysicalBackupConfigInputFromDraft';

const ENTERED_FULL_BACKUPS = {
  count: 5,
  gfsHours: 1,
  gfsDays: 2,
  gfsWeeks: 3,
  gfsMonths: 4,
  gfsYears: 6,
};

describe('toPhysicalBackupConfigInputFromDraft', () => {
  it('sends no full-backup retention for CHAINS', () => {
    const draft = buildPhysicalBackupConfigDraft({
      retention: { policy: 'CHAINS', chainsCount: 3 },
      fullBackups: ENTERED_FULL_BACKUPS,
    });

    expect(toPhysicalBackupConfigInputFromDraft(draft).settings.retention).toEqual({
      policy: 'CHAINS',
      chainsCount: 3,
      fullBackups: null,
    });
  });

  it('sends a chain count of zero for FULL_BACKUPS', () => {
    const draft = buildPhysicalBackupConfigDraft({
      retention: { policy: 'FULL_BACKUPS', chainsCount: 3 },
    });

    expect(toPhysicalBackupConfigInputFromDraft(draft).settings.retention.chainsCount).toBe(0);
  });

  it('sends the count and zero GFS periods for LAST_N', () => {
    const draft = buildPhysicalBackupConfigDraft({
      retention: { policy: 'CHAINS_AND_FULL_BACKUPS', chainsCount: 3 },
      fullBackups: { ...ENTERED_FULL_BACKUPS, policy: 'LAST_N' },
    });

    expect(toPhysicalBackupConfigInputFromDraft(draft).settings.retention).toEqual({
      policy: 'CHAINS_AND_FULL_BACKUPS',
      chainsCount: 3,
      fullBackups: {
        policy: 'LAST_N',
        count: 5,
        gfsHours: 0,
        gfsDays: 0,
        gfsWeeks: 0,
        gfsMonths: 0,
        gfsYears: 0,
      },
    });
  });

  it('sends the GFS periods and a count of zero for GFS', () => {
    const draft = buildPhysicalBackupConfigDraft({
      retention: { policy: 'FULL_BACKUPS' },
      fullBackups: { ...ENTERED_FULL_BACKUPS, policy: 'GFS' },
    });

    expect(toPhysicalBackupConfigInputFromDraft(draft).settings.retention.fullBackups).toEqual({
      ...ENTERED_FULL_BACKUPS,
      policy: 'GFS',
      count: 0,
    });
  });

  it('keeps walLagThresholdBytes and the other settings unchanged', () => {
    const draft = buildPhysicalBackupConfigDraft({
      config: { storageId: 'storage-id', encryption: 'ENCRYPTED' },
      settings: { walLagThresholdBytes: 4096, notificationTriggers: ['WAL_GAP'] },
      backupType: 'FULL_INCREMENTAL_WAL_STREAM',
    });

    const input = toPhysicalBackupConfigInputFromDraft(draft);

    expect(input.storageId).toBe('storage-id');
    expect(input.encryption).toBe('ENCRYPTED');
    expect(input.settings.walLagThresholdBytes).toBe(4096);
    expect(input.settings.notificationTriggers).toEqual(['WAL_GAP']);
    expect(input.settings.incrementalBackupSchedule).toEqual(
      draft.settings.incrementalBackupSchedule,
    );
  });
});
