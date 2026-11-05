import { formatDuration } from '../../../../shared/lib';

export const formatOptionalDuration = (durationMs: number | undefined): string =>
  durationMs === undefined ? '-' : formatDuration(durationMs);
