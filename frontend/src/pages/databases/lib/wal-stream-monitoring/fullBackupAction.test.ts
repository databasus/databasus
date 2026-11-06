import { describe, expect, it, vi } from 'vitest';

import { getFullBackupAction } from '@/pages/databases/lib/wal-stream-monitoring/fullBackupAction';
import { FullBackupAction } from '@/pages/databases/model/wal-stream-monitoring/FullBackupAction';
import { WalStreamHealthLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamHealthLevel';
import { WalStreamReasonCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonCode';
import { WalStreamReasonLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonLevel';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const brokenChainInput = {
  level: WalStreamHealthLevel.BROKEN,
  isStateConfirmed: true,
  reasons: [{ code: WalStreamReasonCode.WAL_CHAIN_BROKEN, level: WalStreamReasonLevel.BROKEN }],
  latestRestorableAt: '2026-10-18T07:50:00Z',
  isFullBackupPending: false,
  isCanManageDBs: true,
  isRequestingFullBackup: false,
};

describe('getFullBackupAction', () => {
  it('offers a full backup for a broken chain', () => {
    expect(getFullBackupAction(brokenChainInput)).toBe(FullBackupAction.AVAILABLE);
  });

  it('offers a full backup for a missing slot', () => {
    const action = getFullBackupAction({
      ...brokenChainInput,
      reasons: [{ code: WalStreamReasonCode.SLOT_MISSING, level: WalStreamReasonLevel.BROKEN }],
    });

    expect(action).toBe(FullBackupAction.AVAILABLE);
  });

  it('offers a full backup to a healthy database without a completed full backup', () => {
    const action = getFullBackupAction({
      ...brokenChainInput,
      level: WalStreamHealthLevel.HEALTHY,
      reasons: [],
      latestRestorableAt: null,
    });

    expect(action).toBe(FullBackupAction.AVAILABLE);
  });

  it('hides the button from a user who cannot manage the database', () => {
    expect(getFullBackupAction({ ...brokenChainInput, isCanManageDBs: false })).toBe(
      FullBackupAction.HIDDEN,
    );
  });

  it('hides the button when no reason calls for a full backup', () => {
    const action = getFullBackupAction({
      ...brokenChainInput,
      level: WalStreamHealthLevel.NEEDS_ATTENTION,
      reasons: [
        {
          code: WalStreamReasonCode.LOCAL_QUEUE_FILLING,
          level: WalStreamReasonLevel.NEEDS_ATTENTION,
        },
      ],
    });

    expect(action).toBe(FullBackupAction.HIDDEN);
  });

  it.each([WalStreamHealthLevel.NOT_STREAMING, WalStreamHealthLevel.COLLECTING_DATA])(
    'hides the button for %s even without a completed full backup',
    (level) => {
      const action = getFullBackupAction({
        ...brokenChainInput,
        level,
        reasons: [],
        latestRestorableAt: null,
      });

      expect(action).toBe(FullBackupAction.HIDDEN);
    },
  );

  it('hides the button while the current state cannot be confirmed', () => {
    expect(getFullBackupAction({ ...brokenChainInput, isStateConfirmed: false })).toBe(
      FullBackupAction.HIDDEN,
    );
  });

  it('hides the button while a full backup is requested or running', () => {
    expect(getFullBackupAction({ ...brokenChainInput, isFullBackupPending: true })).toBe(
      FullBackupAction.HIDDEN,
    );
  });

  it('shows the request as running while it is sent', () => {
    expect(getFullBackupAction({ ...brokenChainInput, isRequestingFullBackup: true })).toBe(
      FullBackupAction.REQUESTING,
    );
  });

  it('offers the button again once a failed request settles', () => {
    const whileRequesting = getFullBackupAction({
      ...brokenChainInput,
      isRequestingFullBackup: true,
    });
    const afterFailure = getFullBackupAction({
      ...brokenChainInput,
      isRequestingFullBackup: false,
    });

    expect(whileRequesting).toBe(FullBackupAction.REQUESTING);
    expect(afterFailure).toBe(FullBackupAction.AVAILABLE);
  });
});
