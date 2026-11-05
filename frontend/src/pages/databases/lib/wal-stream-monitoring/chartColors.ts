import type { ResolvedTheme } from '../../../../shared/theme';

interface WalStreamChartColors {
  sourceWrites: string;
  archive: string;
  notSavedYet: string;
  notReceivedYet: string;
  buffer: string;
  pauseLimit: string;
  restartMarker: string;
  pauseMarker: string;
}

// The lines also differ in dash pattern, so they stay apart in grayscale and for color-blind
// readers. The dark variants are lighter, so they keep their contrast on a dark background.
export const WAL_STREAM_CHART_COLORS: Record<ResolvedTheme, WalStreamChartColors> = {
  light: {
    sourceWrites: '#1677ff',
    archive: '#fa8c16',
    notSavedYet: '#722ed1',
    notReceivedYet: '#13c2c2',
    buffer: '#722ed1',
    pauseLimit: '#8c8c8c',
    restartMarker: '#f5222d',
    pauseMarker: '#8c8c8c',
  },
  dark: {
    sourceWrites: '#4096ff',
    archive: '#ffa940',
    notSavedYet: '#9254de',
    notReceivedYet: '#36cfc9',
    buffer: '#9254de',
    pauseLimit: '#8c8c8c',
    restartMarker: '#ff4d4f',
    pauseMarker: '#8c8c8c',
  },
};
