import type {
  MonitoringText,
  MonitoringTextParam,
} from '@/pages/databases/wal-monitoring/model/MonitoringText';

const easeOutCubic = (progress: number): number =>
  1 - (1 - Math.min(1, Math.max(0, progress))) ** 3;

export const interpolateNumber = (fromValue: number, toValue: number, progress: number): number =>
  fromValue + (toValue - fromValue) * easeOutCubic(progress);

// Moves the amounts of the same sentence towards their new values, so a refresh reads as a change
// rather than a jump. A sentence whose key changed is replaced at once: its numbers mean
// something else now.
export const interpolateMonitoringTexts = (
  fromTexts: MonitoringText[],
  toTexts: MonitoringText[],
  progress: number,
): MonitoringText[] =>
  toTexts.map((toText, textIndex) => {
    const fromText = fromTexts[textIndex];

    if (progress >= 1 || fromText === undefined || fromText.key !== toText.key || !toText.params) {
      return toText;
    }

    const params: Record<string, MonitoringTextParam> = {};

    for (const [name, toParam] of Object.entries(toText.params)) {
      params[name] = interpolateParam(fromText.params?.[name], toParam, progress);
    }

    return { key: toText.key, params };
  });

const interpolateParam = (
  fromParam: MonitoringTextParam | undefined,
  toParam: MonitoringTextParam,
  progress: number,
): MonitoringTextParam => {
  if (fromParam?.kind === 'bytes' && toParam.kind === 'bytes') {
    return { kind: 'bytes', bytes: interpolateNumber(fromParam.bytes, toParam.bytes, progress) };
  }

  if (fromParam?.kind === 'bytesPerSecond' && toParam.kind === 'bytesPerSecond') {
    return {
      kind: 'bytesPerSecond',
      bytesPerSecond: interpolateNumber(fromParam.bytesPerSecond, toParam.bytesPerSecond, progress),
    };
  }

  return toParam;
};
