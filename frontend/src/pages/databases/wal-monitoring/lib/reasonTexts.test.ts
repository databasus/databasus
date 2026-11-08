import { describe, expect, it, vi } from 'vitest';

import {
  getNoticeTexts,
  getReasonTexts,
  getTipTexts,
  sortReasonsBrokenFirst,
} from '@/pages/databases/wal-monitoring/lib/reasonTexts';
import { type WalStreamReason } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import { WalStreamNoticeCode } from '@/pages/databases/wal-monitoring/model/WalStreamNoticeCode';
import { WalStreamReasonCode } from '@/pages/databases/wal-monitoring/model/WalStreamReasonCode';
import { WalStreamReasonLevel } from '@/pages/databases/wal-monitoring/model/WalStreamReasonLevel';
import { DICTIONARIES } from '@/shared/i18n';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const now = new Date('2026-10-18T12:00:00Z');

type Dictionary = { [key: string]: string | Dictionary };

const getEnglishText = (key: string): string | undefined => {
  let node: string | Dictionary | undefined = DICTIONARIES.en as unknown as Dictionary;

  for (const part of key.split('.')) {
    node = typeof node === 'object' ? node[part] : undefined;
  }

  return typeof node === 'string' ? node : undefined;
};

const flattenEnglish = (dictionary: Dictionary, prefix: string): [string, string][] =>
  Object.entries(dictionary).flatMap(([key, value]) =>
    typeof value === 'string'
      ? [[`${prefix}.${key}`, value] as [string, string]]
      : flattenEnglish(value, `${prefix}.${key}`),
  );

const buildReason = (
  code: WalStreamReasonCode,
  level: WalStreamReasonLevel = WalStreamReasonLevel.BROKEN,
): WalStreamReason => ({
  code,
  level,
  since: '2026-10-18T11:50:00Z',
  valueBytes: 300 * 1024 * 1024,
  thresholdBytes: 512 * 1024 * 1024,
  count: 3,
  estimatedSeconds: 40 * 60,
});

describe('getReasonTexts', () => {
  it.each(Object.values(WalStreamReasonCode))(
    'gives %s a title, what happened and what to do',
    (code) => {
      for (const isFullBackupPending of [false, true]) {
        const texts = getReasonTexts(buildReason(code), isFullBackupPending, now);

        expect(getEnglishText(texts.titleKey)).toBeDefined();
        expect(getEnglishText(texts.whatHappened.key)).toBeDefined();
        expect(getEnglishText(texts.whatToDo.key)).toBeDefined();
      }
    },
  );

  it.each(Object.values(WalStreamNoticeCode))(
    'gives notice %s a title, what happened and what to do',
    (code) => {
      const texts = getNoticeTexts({ code, level: WalStreamReasonLevel.INFO });

      expect(getEnglishText(texts.titleKey)).toBeDefined();
      expect(getEnglishText(texts.whatHappened.key)).toBeDefined();
      expect(getEnglishText(texts.whatToDo.key)).toBeDefined();
    },
  );

  it.each([WalStreamReasonCode.WAL_CHAIN_BROKEN, WalStreamReasonCode.SLOT_MISSING])(
    'asks for a full backup on %s only while none is requested or running',
    (code) => {
      const withoutPendingFull = getReasonTexts(buildReason(code), false, now).whatToDo.key;
      const withPendingFull = getReasonTexts(buildReason(code), true, now).whatToDo.key;

      expect(getEnglishText(withoutPendingFull)).toContain('Start a full backup');
      expect(getEnglishText(withPendingFull)).toContain('on its way');
      expect(getEnglishText(withPendingFull)).not.toContain('Start a full backup');
    },
  );

  it('does not claim that Databasus repairs a broken chain by itself', () => {
    const whatToDo = getReasonTexts(
      buildReason(WalStreamReasonCode.WAL_CHAIN_BROKEN),
      false,
      now,
    ).whatToDo;

    expect(getEnglishText(whatToDo.key)).not.toMatch(/by itself/);
  });

  it('passes the age, the amounts and the count of a reason to its sentence', () => {
    const queueTexts = getReasonTexts(
      buildReason(WalStreamReasonCode.LOCAL_QUEUE_FILLING, WalStreamReasonLevel.NEEDS_ATTENTION),
      false,
      now,
    );

    expect(queueTexts.whatHappened.params?.amount).toEqual({
      kind: 'bytes',
      bytes: 300 * 1024 * 1024,
    });
    expect(queueTexts.whatHappened.params?.limit).toEqual({
      kind: 'bytes',
      bytes: 512 * 1024 * 1024,
    });

    const stoppedTexts = getReasonTexts(
      {
        code: WalStreamReasonCode.STREAMING_STOPPED,
        level: WalStreamReasonLevel.BROKEN,
        since: '2026-10-18T11:50:00Z',
      },
      false,
      now,
    );

    expect(stoppedTexts.whatHappened.params?.duration).toEqual({
      kind: 'text',
      text: { key: 'backups.physical.monitoring.durations.minutes', params: { minutes: 10 } },
    });
  });
});

describe('sortReasonsBrokenFirst', () => {
  it('lists the missing slot before the filling queue', () => {
    const sortedReasons = sortReasonsBrokenFirst([
      {
        code: WalStreamReasonCode.LOCAL_QUEUE_FILLING,
        level: WalStreamReasonLevel.NEEDS_ATTENTION,
      },
      { code: WalStreamReasonCode.SLOT_MISSING, level: WalStreamReasonLevel.BROKEN },
    ]);

    expect(sortedReasons.map((reason) => reason.code)).toEqual([
      WalStreamReasonCode.SLOT_MISSING,
      WalStreamReasonCode.LOCAL_QUEUE_FILLING,
    ]);
  });
});

describe('getTipTexts', () => {
  it('turns notices into tips, leaving the missing first full backup to the restore sentence', () => {
    const tips = getTipTexts([
      { code: WalStreamNoticeCode.NO_SLOT_WAL_LIMIT, level: WalStreamReasonLevel.INFO },
      { code: WalStreamNoticeCode.NO_FULL_BACKUP_YET, level: WalStreamReasonLevel.INFO },
    ]);

    expect(tips.map((tip) => tip.titleKey)).toEqual([
      'backups.physical.monitoring.notices.noSlotWalLimit.title',
    ]);
  });
});

describe('the English monitoring copy', () => {
  // Technical names belong only where the reader asks for them: the collapsed details, the
  // explanations on hover, and a "what to do" sentence that names the setting to change.
  const isTechnicalPlace = (key: string, text: string) =>
    key.includes('.details.') ||
    key.includes('.tooltips.') ||
    (key.endsWith('.whatToDo') && text.includes('<code>'));

  it('avoids WAL, slot, LSN and lag in every default label and sentence', () => {
    const monitoringCopy = flattenEnglish(
      (DICTIONARIES.en as unknown as { backups: { physical: { monitoring: Dictionary } } }).backups
        .physical.monitoring,
      'backups.physical.monitoring',
    );

    const technicalTexts = monitoringCopy
      .filter(([key, text]) => !isTechnicalPlace(key, text))
      .filter(([, text]) => /\b(WAL|slot|LSN|lag)\b/i.test(text));

    expect(technicalTexts).toEqual([]);
  });
});
