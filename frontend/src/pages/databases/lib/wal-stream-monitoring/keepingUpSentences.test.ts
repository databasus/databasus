import { describe, expect, it, vi } from 'vitest';

import { type WalStreamKeepingUp } from '../../model/wal-stream-monitoring/WalStreamKeepingUp';
import { WalStreamKeepingUpState } from '../../model/wal-stream-monitoring/WalStreamKeepingUpState';
import { getKeepingUpSentences } from './keepingUpSentences';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const MB = 1000 * 1000;
const GB = 1000 * MB;

const buildKeepingUp = (change: Partial<WalStreamKeepingUp>): WalStreamKeepingUp => ({
  state: WalStreamKeepingUpState.KEEPING_UP,
  sourceWriteBytesPerSecond: 10 * MB,
  archiveBytesPerSecond: 10 * MB,
  unarchivedWalBytes: 16 * MB,
  unarchivedWalChangeBytesPerSecond: 0,
  slotLagBytes: MB,
  slotLagChangeBytesPerSecond: 0,
  slotRebuildLagBytes: 256 * 1024 * 1024,
  estimatedCatchUpSeconds: null,
  estimatedSlotRebuildSeconds: null,
  estimatedSlotLimitSeconds: null,
  ...change,
});

describe('getKeepingUpSentences', () => {
  it('explains falling behind with the difference, the backlog and its growth an hour', () => {
    const sentences = getKeepingUpSentences(
      buildKeepingUp({
        state: WalStreamKeepingUpState.FALLING_BEHIND,
        sourceWriteBytesPerSecond: 21 * MB,
        archiveBytesPerSecond: 12 * MB,
        unarchivedWalBytes: 3.4 * GB,
        unarchivedWalChangeBytesPerSecond: 9 * MB,
      }),
    );

    expect(sentences.map((sentence) => sentence.key)).toEqual([
      'backups.physical.monitoring.keepingUp.fallingBehind',
      'backups.physical.monitoring.keepingUp.fallingBehindBacklog',
    ]);

    expect(sentences[0].params?.differenceRate).toEqual({
      kind: 'bytesPerSecond',
      bytesPerSecond: 9 * MB,
    });
    expect(sentences[1].params?.unsavedAmount).toEqual({ kind: 'bytes', bytes: 3.4 * GB });

    const growthPerHour = sentences[1].params?.growthPerHour;
    expect(growthPerHour?.kind).toBe('bytes');
    expect(growthPerHour?.kind === 'bytes' && growthPerHour.bytes / GB).toBeCloseTo(32.4, 1);
  });

  it('adds what happens next only when it is under a day away', () => {
    const nearSentences = getKeepingUpSentences(
      buildKeepingUp({ estimatedSlotRebuildSeconds: 40 * 60 }),
    );
    const farSentences = getKeepingUpSentences(
      buildKeepingUp({ estimatedSlotRebuildSeconds: 3 * 24 * 3600 }),
    );

    expect(nearSentences.map((sentence) => sentence.key)).toContain(
      'backups.physical.monitoring.keepingUp.rebuildConsequence',
    );
    expect(farSentences.map((sentence) => sentence.key)).not.toContain(
      'backups.physical.monitoring.keepingUp.rebuildConsequence',
    );
  });

  it('warns about the allowance running out when that is under a day away', () => {
    const sentences = getKeepingUpSentences(buildKeepingUp({ estimatedSlotLimitSeconds: 50 * 60 }));

    expect(sentences.map((sentence) => sentence.key)).toContain(
      'backups.physical.monitoring.keepingUp.limitConsequence',
    );
  });

  it.each([
    [WalStreamKeepingUpState.KEEPING_UP, 'backups.physical.monitoring.keepingUp.keepingUp'],
    [WalStreamKeepingUpState.IDLE, 'backups.physical.monitoring.keepingUp.idle'],
  ])('describes %s in one sentence', (state, expectedKey) => {
    expect(
      getKeepingUpSentences(buildKeepingUp({ state })).map((sentence) => sentence.key),
    ).toEqual([expectedKey]);
  });

  it('adds no sentence while the state is unknown', () => {
    expect(
      getKeepingUpSentences(buildKeepingUp({ state: WalStreamKeepingUpState.UNKNOWN })),
    ).toEqual([]);
  });

  it('says how much backlog is left and how long it takes while catching up', () => {
    const [sentence] = getKeepingUpSentences(
      buildKeepingUp({
        state: WalStreamKeepingUpState.CATCHING_UP,
        unarchivedWalBytes: 3 * GB,
        estimatedCatchUpSeconds: 50 * 60,
      }),
    );

    expect(sentence.key).toBe('backups.physical.monitoring.keepingUp.catchingUp');
    expect(sentence.params?.unsavedAmount).toEqual({ kind: 'bytes', bytes: 3 * GB });
  });
});
