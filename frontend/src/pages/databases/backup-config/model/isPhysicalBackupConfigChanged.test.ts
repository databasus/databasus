import { describe, expect, it } from 'vitest';

import { buildPhysicalBackupConfigDraft } from '@/pages/databases/backup-config/model/buildPhysicalBackupConfigDraft.fixture';
import { isPhysicalBackupConfigChanged } from '@/pages/databases/backup-config/model/isPhysicalBackupConfigChanged';

const backupType = 'FULL_INCREMENTAL';

describe('isPhysicalBackupConfigChanged', () => {
  it('finds no change between equal drafts', () => {
    const acknowledged = buildPhysicalBackupConfigDraft();

    expect(
      isPhysicalBackupConfigChanged({
        acknowledged,
        shown: buildPhysicalBackupConfigDraft(),
        backupType,
      }),
    ).toBe(false);
  });

  it.each([
    { name: 'storage', shown: { config: { storageId: 'storage-id' } } },
    { name: 'encryption', shown: { config: { encryption: 'ENCRYPTED' as const } } },
    { name: 'backups switch', shown: { config: { isBackupsEnabled: false } } },
    { name: 'retention policy', shown: { retention: { policy: 'CHAINS' as const } } },
    { name: 'chain count', shown: { retention: { chainsCount: 8 } } },
    { name: 'GFS period', shown: { fullBackups: { gfsYears: 9 } } },
    { name: 'full-backup policy', shown: { fullBackups: { policy: 'LAST_N' as const } } },
    {
      name: 'full schedule',
      shown: { settings: { fullBackupSchedule: { type: 'HOURLY' as const } } },
    },
    {
      name: 'incremental schedule',
      shown: { settings: { incrementalBackupSchedule: { type: 'HOURLY' as const } } },
    },
    {
      name: 'empty incremental schedule',
      shown: { settings: { incrementalBackupSchedule: null } },
    },
    { name: 'trigger', shown: { settings: { notificationTriggers: [] } } },
  ])('counts a change of the $name', ({ shown }) => {
    expect(
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft(),
        shown: buildPhysicalBackupConfigDraft(shown),
        backupType,
      }),
    ).toBe(true);
  });

  it('ignores the full-backup retention when the policy keeps only chains', () => {
    const retention = { policy: 'CHAINS' as const };

    expect(
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({ retention }),
        shown: buildPhysicalBackupConfigDraft({ retention, fullBackups: { gfsYears: 9 } }),
        backupType,
      }),
    ).toBe(false);
  });

  it('ignores the chain count when the policy keeps only full backups', () => {
    expect(
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({
          retention: { policy: 'FULL_BACKUPS', chainsCount: 0 },
          backupType: 'FULL',
        }),
        shown: buildPhysicalBackupConfigDraft({
          retention: { policy: 'FULL_BACKUPS', chainsCount: 7 },
          backupType: 'FULL',
        }),
        backupType: 'FULL',
      }),
    ).toBe(false);
  });

  it('ignores the count with GFS and the GFS periods with LAST_N', () => {
    const compare = (policy: 'GFS' | 'LAST_N', change: object) =>
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({ fullBackups: { policy } }),
        shown: buildPhysicalBackupConfigDraft({ fullBackups: { policy, ...change } }),
        backupType,
      });

    expect(compare('GFS', { count: 4 })).toBe(false);
    expect(compare('LAST_N', { gfsYears: 9 })).toBe(false);
    expect(compare('LAST_N', { count: 4 })).toBe(true);
  });

  it('ignores schedules and triggers with backups off', () => {
    const config = { isBackupsEnabled: false };

    expect(
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({ config }),
        shown: buildPhysicalBackupConfigDraft({
          config,
          settings: { fullBackupSchedule: { type: 'HOURLY' }, notificationTriggers: [] },
        }),
        backupType,
      }),
    ).toBe(false);
  });

  it('ignores the triggers and the incremental schedule a full-only backup type does not show', () => {
    expect(
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({
          settings: { notificationTriggers: ['WAL_GAP', 'CHAIN_BROKEN'] },
          backupType: 'FULL',
        }),
        shown: buildPhysicalBackupConfigDraft({
          settings: { notificationTriggers: [], incrementalBackupSchedule: { type: 'HOURLY' } },
          backupType: 'FULL',
        }),
        backupType: 'FULL',
      }),
    ).toBe(false);
  });

  it('counts the WAL gap trigger only for a WAL streaming backup type', () => {
    const compare = (shownBackupType: 'FULL_INCREMENTAL' | 'FULL_INCREMENTAL_WAL_STREAM') =>
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({
          settings: { notificationTriggers: [] },
        }),
        shown: buildPhysicalBackupConfigDraft({
          settings: { notificationTriggers: ['WAL_GAP'] },
        }),
        backupType: shownBackupType,
      });

    expect(compare('FULL_INCREMENTAL')).toBe(false);
    expect(compare('FULL_INCREMENTAL_WAL_STREAM')).toBe(true);
  });

  it('ignores the order of the triggers', () => {
    expect(
      isPhysicalBackupConfigChanged({
        acknowledged: buildPhysicalBackupConfigDraft({
          settings: { notificationTriggers: ['BACKUP_FAILED', 'BACKUP_SUCCESS'] },
        }),
        shown: buildPhysicalBackupConfigDraft({
          settings: { notificationTriggers: ['BACKUP_SUCCESS', 'BACKUP_FAILED'] },
        }),
        backupType,
      }),
    ).toBe(false);
  });
});
