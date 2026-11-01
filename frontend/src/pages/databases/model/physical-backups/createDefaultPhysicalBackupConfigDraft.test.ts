import { describe, expect, it } from 'vitest';

import { createDefaultPhysicalBackupConfigDraft } from './createDefaultPhysicalBackupConfigDraft';
import { isPhysicalBackupConfigComplete } from './isPhysicalBackupConfigComplete';

describe('createDefaultPhysicalBackupConfigDraft', () => {
  it.each([
    {
      backupType: 'FULL',
      policy: 'FULL_BACKUPS',
      hasIncrementalSchedule: false,
      walLagThresholdBytes: 0,
    },
    {
      backupType: 'FULL_INCREMENTAL',
      policy: 'CHAINS_AND_FULL_BACKUPS',
      hasIncrementalSchedule: true,
      walLagThresholdBytes: 0,
    },
    {
      backupType: 'FULL_INCREMENTAL_WAL_STREAM',
      policy: 'CHAINS_AND_FULL_BACKUPS',
      hasIncrementalSchedule: true,
      walLagThresholdBytes: 256 * 1024 * 1024,
    },
  ] as const)(
    'fits the rules of $backupType',
    ({ backupType, policy, hasIncrementalSchedule, walLagThresholdBytes }) => {
      const draft = createDefaultPhysicalBackupConfigDraft(backupType);

      expect(draft.settings.retention.policy).toBe(policy);
      expect(draft.settings.incrementalBackupSchedule !== null).toBe(hasIncrementalSchedule);
      expect(draft.settings.walLagThresholdBytes).toBe(walLagThresholdBytes);
      expect(isPhysicalBackupConfigComplete(draft, backupType)).toBe(true);
    },
  );
});
