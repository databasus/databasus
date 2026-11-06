import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { WAL_STREAM_CHART_COLORS } from '@/pages/databases/lib/wal-stream-monitoring/chartColors';
import {
  interpolateMonitoringTexts,
  interpolateNumber,
} from '@/pages/databases/lib/wal-stream-monitoring/easedTexts';
import { formatBytesPerSecond } from '@/pages/databases/lib/wal-stream-monitoring/monitoringTextValues';
import type { MonitoringFigures } from '@/pages/databases/model/wal-stream-monitoring/MonitoringFigures';
import type { MonitoringText } from '@/pages/databases/model/wal-stream-monitoring/MonitoringText';
import { MonitoringTextComponent } from '@/pages/databases/ui/wal-stream-monitoring/MonitoringTextComponent';
import { useEasedTransition } from '@/pages/databases/ui/wal-stream-monitoring/useEasedTransition';
import { type TranslationKey, useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';
import { useTheme } from '@/shared/theme';

interface Props {
  figures: MonitoringFigures;
  keepingUpSentences: MonitoringText[];
  isReducedMotion: boolean;
}

interface FigureProps {
  labelKey: TranslationKey;
  // The color of the chart line this figure belongs to.
  lineColor?: string;
  // Rates are averages over a window while amounts are current, which the card has to say, or a
  // rate next to an empty queue reads as a contradiction.
  isAverage?: boolean;
  children: ReactNode;
}

const FigureComponent = ({ labelKey, lineColor, isAverage, children }: FigureProps) => {
  const { t } = useTranslation();

  return (
    <div className="rounded-lg border border-border px-4 py-3">
      <div className="flex items-center gap-2 text-sm text-muted">
        {lineColor && (
          <span
            aria-hidden
            className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
            style={{ backgroundColor: lineColor }}
          />
        )}
        {t(labelKey)}
        {isAverage && (
          <span className="text-xs text-muted">
            {t('backups.physical.monitoring.figures.averageHint')}
          </span>
        )}
      </div>
      <div className="mt-1 text-xl font-semibold tabular-nums">{children}</div>
    </div>
  );
};

export const MonitoringFiguresComponent = ({
  figures,
  keepingUpSentences,
  isReducedMotion,
}: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];

  const easedWaitingBytes = useEasedTransition(
    figures.waitingToBeSavedBytes ?? 0,
    interpolateNumber,
    isReducedMotion,
  );
  const easedWriteRate = useEasedTransition(
    figures.writeBytesPerSecond ?? 0,
    interpolateNumber,
    isReducedMotion,
  );
  const easedSaveRate = useEasedTransition(
    figures.saveBytesPerSecond ?? 0,
    interpolateNumber,
    isReducedMotion,
  );
  const easedSentences = useEasedTransition(
    keepingUpSentences,
    interpolateMonitoringTexts,
    isReducedMotion,
  );

  const notMeasured = t('backups.physical.monitoring.figures.notMeasured');

  return (
    <div className="mt-4">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <FigureComponent labelKey="backups.physical.monitoring.figures.lastSaved">
          {figures.lastSavedAt
            ? formatRelativeTime(figures.lastSavedAt)
            : t('backups.physical.monitoring.path.nothingSavedYet')}
        </FigureComponent>

        <FigureComponent labelKey="backups.physical.monitoring.figures.waitingToBeSaved">
          {figures.waitingToBeSavedBytes === null
            ? notMeasured
            : formatBytes(easedWaitingBytes, formatNumber)}
        </FigureComponent>

        <FigureComponent
          labelKey="backups.physical.monitoring.charts.databaseWrites"
          lineColor={chartColors.sourceWrites}
          isAverage
        >
          {figures.writeBytesPerSecond === null
            ? notMeasured
            : formatBytesPerSecond(easedWriteRate, t, formatNumber)}
        </FigureComponent>

        <FigureComponent
          labelKey="backups.physical.monitoring.charts.savedToStorage"
          lineColor={chartColors.archive}
          isAverage
        >
          {figures.saveBytesPerSecond === null
            ? notMeasured
            : formatBytesPerSecond(easedSaveRate, t, formatNumber)}
        </FigureComponent>
      </div>

      {easedSentences.length > 0 && (
        <div className="mt-2 text-sm text-muted">
          {easedSentences.map((sentence) => (
            <span key={sentence.key} className="mr-1">
              <MonitoringTextComponent text={sentence} />
            </span>
          ))}
        </div>
      )}
    </div>
  );
};
