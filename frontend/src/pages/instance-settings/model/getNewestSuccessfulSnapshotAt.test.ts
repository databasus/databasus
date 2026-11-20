import { describe, expect, it } from 'vitest';

import type { InstanceBackupDestinationHealth } from '@/entities/instance-backups';
import { getNewestSuccessfulSnapshotAt } from '@/pages/instance-settings/model/getNewestSuccessfulSnapshotAt';

const buildDestination = (
  lastSuccessfulSnapshotAt: string | null,
): InstanceBackupDestinationHealth => ({
  storageId: crypto.randomUUID(),
  storageName: 'Offsite S3',
  workspaceId: crypto.randomUUID(),
  workspaceName: 'Production',
  health: lastSuccessfulSnapshotAt ? 'HEALTHY' : 'INITIALIZING',
  lastSuccessfulSnapshotAt,
});

describe('getNewestSuccessfulSnapshotAt', () => {
  it('returns the newest snapshot across storages', () => {
    const destinations = [
      buildDestination('2026-11-14T10:00:00Z'),
      buildDestination('2026-11-15T03:00:00Z'),
      buildDestination(null),
    ];

    expect(getNewestSuccessfulSnapshotAt(destinations)).toBe('2026-11-15T03:00:00Z');
  });

  it('returns nothing when no storage holds a successful copy', () => {
    expect(getNewestSuccessfulSnapshotAt([buildDestination(null)])).toBeUndefined();
    expect(getNewestSuccessfulSnapshotAt([])).toBeUndefined();
  });
});
