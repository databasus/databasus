import { describe, expect, it } from 'vitest';

import type { MonitoringText } from '../../model/wal-stream-monitoring/MonitoringText';
import { interpolateMonitoringTexts } from './easedTexts';

const writeSentence = (bytesPerSecond: number): MonitoringText => ({
  key: 'backups.physical.monitoring.keepingUp.keepingUp',
  params: {
    writeRate: { kind: 'bytesPerSecond', bytesPerSecond },
    archiveRate: { kind: 'bytesPerSecond', bytesPerSecond },
  },
});

describe('interpolateMonitoringTexts', () => {
  it('moves the amounts of the same sentence between their old and new values', () => {
    const [halfway] = interpolateMonitoringTexts([writeSentence(10)], [writeSentence(20)], 0.5);
    const halfwayRate = halfway.params?.writeRate;

    expect(halfwayRate?.kind === 'bytesPerSecond' && halfwayRate.bytesPerSecond).toBeGreaterThan(
      10,
    );
    expect(halfwayRate?.kind === 'bytesPerSecond' && halfwayRate.bytesPerSecond).toBeLessThan(20);
  });

  it('ends on the new values', () => {
    expect(interpolateMonitoringTexts([writeSentence(10)], [writeSentence(20)], 1)).toEqual([
      writeSentence(20),
    ]);
  });

  it('replaces a sentence whose meaning changed at once', () => {
    const idleSentence: MonitoringText = { key: 'backups.physical.monitoring.keepingUp.idle' };

    expect(interpolateMonitoringTexts([writeSentence(10)], [idleSentence], 0.5)).toEqual([
      idleSentence,
    ]);
  });
});
