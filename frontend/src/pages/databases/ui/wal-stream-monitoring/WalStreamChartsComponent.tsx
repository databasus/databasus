import { Segmented, theme } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceArea,
  ReferenceLine,
  ResponsiveContainer,
  Scatter,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { WAL_STREAM_CHART_COLORS } from '@/pages/databases/lib/wal-stream-monitoring/chartColors';
import {
  getBinaryAxisTicks,
  getChartTimeDomain,
  getDataSinceMs,
  getMissingDataRanges,
  getRowsMaximum,
  hasEnoughChartPoints,
  toChartRows,
} from '@/pages/databases/lib/wal-stream-monitoring/chartRows';
import { formatBytesPerSecond } from '@/pages/databases/lib/wal-stream-monitoring/monitoringTextValues';
import type { WalStreamChartRow } from '@/pages/databases/model/wal-stream-monitoring/WalStreamChartRow';
import { type WalStreamSeries } from '@/pages/databases/model/wal-stream-monitoring/WalStreamSeries';
import { WalStreamStatisticsPeriod } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriod';
import { WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriodLabelKeys';
import { type TranslationKey, useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';
import { useTheme } from '@/shared/theme';
import { getUserClockFormat } from '@/shared/time';

interface KeepingUpChartProps {
  series: WalStreamSeries;
  period: WalStreamStatisticsPeriod;
  onPeriodChange: (period: WalStreamStatisticsPeriod) => void;
  isAnimationActive: boolean;
}

interface TechnicalChartsProps {
  series: WalStreamSeries;
  walSegmentSizeBytes: number;
  isAnimationActive: boolean;
}

enum ChartLineStyle {
  SOLID = 'SOLID',
  DASHED = 'DASHED',
  // A wide translucent band under a thin line, so two equal series stay visible as a line inside
  // a band instead of one line hiding the other.
  BAND = 'BAND',
}

interface ChartLine {
  dataKey: keyof WalStreamChartRow;
  labelKey: TranslationKey;
  color: string;
  style: ChartLineStyle;
}

interface ChartMarker {
  labelKey: TranslationKey;
  color: string;
  isDiamond: boolean;
}

interface ChartFrameProps {
  titleKey: TranslationKey;
  subtitleKey?: TranslationKey;
  headerExtra?: ReactNode;
  heightClassName: string;
  rows: WalStreamChartRow[];
  series: WalStreamSeries;
  lines: ChartLine[];
  isRate: boolean;
  minimumTop: number;
  isAnimationActive: boolean;
  pauseLimitBytes?: number;
}

// The tab has a fixed maximum width, so the plots keep fixed heights that leave the rest of the tab
// on screen.
// eslint-disable-next-line i18next/no-literal-string -- Tailwind class, not copy
const MAIN_CHART_HEIGHT_CLASS = 'h-[260px]';
// eslint-disable-next-line i18next/no-literal-string -- Tailwind class, not copy
const TECHNICAL_CHART_HEIGHT_CLASS = 'h-[200px]';

const LINE_STROKE_WIDTH = 2;
const BAND_STROKE_WIDTH = 6;
const BAND_STROKE_OPACITY = 0.35;

const MAX_MISSING_DATA_CAPTIONS = 3;

// A rate below this is "almost nothing" for a database, so the rate axis never zooms in further and
// a near-idle database does not fill the chart with noise.
const RATE_AXIS_MINIMUM_TOP = 64 * 1024;

// eslint-disable-next-line i18next/no-literal-string -- recharts sync group id
const CHARTS_SYNC_ID = 'walStreamMonitoring';

const PERIODS = [
  WalStreamStatisticsPeriod.HOUR,
  WalStreamStatisticsPeriod.DAY,
  WalStreamStatisticsPeriod.WEEK,
];

const getTimeTickFormat = (period: WalStreamStatisticsPeriod): string => {
  const clockFormat = getUserClockFormat().format;

  // eslint-disable-next-line i18next/no-literal-string -- dayjs format token, not copy
  return period === WalStreamStatisticsPeriod.WEEK ? `ddd ${clockFormat}` : clockFormat;
};

const LegendLineSampleComponent = ({ line }: { line: ChartLine }) => (
  <svg width="24" height="10" aria-hidden className="shrink-0">
    <line
      x1="0"
      y1="5"
      x2="24"
      y2="5"
      stroke={line.color}
      strokeWidth={line.style === ChartLineStyle.BAND ? BAND_STROKE_WIDTH : LINE_STROKE_WIDTH}
      strokeOpacity={line.style === ChartLineStyle.BAND ? BAND_STROKE_OPACITY : 1}
      strokeDasharray={line.style === ChartLineStyle.DASHED ? '6 4' : undefined}
    />
  </svg>
);

const ChartLegendComponent = ({
  lines,
  markers,
}: {
  lines: ChartLine[];
  markers: ChartMarker[];
}) => {
  const { t } = useTranslation();

  return (
    <ul className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
      {lines.map((line) => (
        <li key={line.dataKey} className="flex items-center gap-1.5">
          <LegendLineSampleComponent line={line} />
          {t(line.labelKey)}
        </li>
      ))}

      {markers.map((marker) => (
        <li key={marker.labelKey} className="flex items-center gap-1.5">
          <span
            aria-hidden
            className={`inline-block h-2 w-2 shrink-0 ${marker.isDiamond ? 'rotate-45' : 'rounded-full'}`}
            style={{ backgroundColor: marker.color }}
          />
          {t(marker.labelKey)}
        </li>
      ))}
    </ul>
  );
};

const ChartFrameComponent = ({
  titleKey,
  subtitleKey,
  headerExtra,
  heightClassName,
  rows,
  series,
  lines,
  isRate,
  minimumTop,
  isAnimationActive,
  pauseLimitBytes,
}: ChartFrameProps) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const { token } = theme.useToken();
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];
  const timeTickFormat = getTimeTickFormat(series.period);
  const amountTicks = getBinaryAxisTicks(
    getRowsMaximum(
      rows,
      lines.map((line) => line.dataKey),
    ),
    minimumTop,
  );
  const hasMarkers = pauseLimitBytes !== undefined;
  const markers: ChartMarker[] = hasMarkers
    ? [
        {
          labelKey: 'backups.physical.monitoring.charts.restarts',
          color: chartColors.restartMarker,
          isDiamond: false,
        },
        {
          labelKey: 'backups.physical.monitoring.charts.pauses',
          color: chartColors.pauseMarker,
          isDiamond: true,
        },
      ]
    : [];

  const formatAmount = (value: number) =>
    isRate ? formatBytesPerSecond(value, t, formatNumber) : formatBytes(value, formatNumber);

  return (
    <div>
      <div className="flex flex-wrap items-start justify-between gap-x-6 gap-y-2">
        <div className="min-w-0 space-y-1">
          <div className="font-semibold">{t(titleKey)}</div>
          {subtitleKey && <div className="text-xs text-muted">{t(subtitleKey)}</div>}
          <ChartLegendComponent lines={lines} markers={markers} />
        </div>

        {headerExtra}
      </div>

      <div className={`mt-2 w-full ${heightClassName}`}>
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart
            data={rows}
            syncId={CHARTS_SYNC_ID}
            margin={{ top: 8, right: 16, left: 8, bottom: 0 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke={token.colorBorderSecondary} />
            <XAxis
              dataKey="timestampMs"
              type="number"
              scale="time"
              domain={getChartTimeDomain(series)}
              allowDataOverflow
              tickFormatter={(timestampMs: number) => dayjs(timestampMs).format(timeTickFormat)}
              tick={{ fontSize: 11, fill: token.colorTextSecondary }}
              stroke={token.colorBorder}
            />
            <YAxis
              width={80}
              domain={[0, amountTicks[amountTicks.length - 1]]}
              ticks={amountTicks}
              tickFormatter={formatAmount}
              tick={{ fontSize: 11, fill: token.colorTextSecondary }}
              stroke={token.colorBorder}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: token.colorBgElevated,
                borderColor: token.colorBorder,
                color: token.colorText,
              }}
              labelFormatter={(timestampMs) => dayjs(Number(timestampMs)).format(timeTickFormat)}
              formatter={(value) => (typeof value === 'number' ? formatAmount(value) : value)}
            />

            {getMissingDataRanges(series).map((range) => (
              <ReferenceArea
                key={range.fromMs}
                x1={range.fromMs}
                x2={range.toMs}
                fill={token.colorFillSecondary}
                fillOpacity={1}
                ifOverflow="hidden"
              />
            ))}

            {lines.map((line) => (
              <Line
                key={line.dataKey}
                type="monotone"
                dataKey={line.dataKey}
                name={t(line.labelKey)}
                stroke={line.color}
                strokeWidth={
                  line.style === ChartLineStyle.BAND ? BAND_STROKE_WIDTH : LINE_STROKE_WIDTH
                }
                strokeOpacity={line.style === ChartLineStyle.BAND ? BAND_STROKE_OPACITY : 1}
                strokeDasharray={line.style === ChartLineStyle.DASHED ? '6 4' : undefined}
                dot={false}
                connectNulls={false}
                isAnimationActive={isAnimationActive}
              />
            ))}

            {hasMarkers && (
              <ReferenceLine
                y={pauseLimitBytes}
                stroke={chartColors.pauseLimit}
                strokeDasharray="4 4"
                label={{
                  value: t('backups.physical.monitoring.charts.pauseLimit'),
                  position: 'insideTopRight',
                  fill: token.colorTextSecondary,
                  fontSize: 11,
                }}
              />
            )}

            {hasMarkers && (
              <Scatter
                dataKey="restartMarkerBytes"
                name={t('backups.physical.monitoring.charts.restarts')}
                fill={chartColors.restartMarker}
                isAnimationActive={isAnimationActive}
              />
            )}

            {hasMarkers && (
              <Scatter
                dataKey="pauseMarkerBytes"
                name={t('backups.physical.monitoring.charts.pauses')}
                fill={chartColors.pauseMarker}
                shape="diamond"
                isAnimationActive={isAnimationActive}
              />
            )}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

// A chart drawn from a handful of points says nothing, so until enough data exists the reader gets
// an explanation instead.
export const ChartAvailabilityComponent = ({
  series,
  children,
}: {
  series: WalStreamSeries;
  children: ReactNode;
}) => {
  const { t } = useTranslation();

  if (!hasEnoughChartPoints(series)) {
    return (
      <div className="mt-4 rounded-lg border border-dashed border-border p-6 text-center text-muted">
        {t('backups.physical.monitoring.charts.notEnoughData')}
      </div>
    );
  }

  const dataSinceMs = getDataSinceMs(series);
  const missingDataRanges = getMissingDataRanges(series).slice(0, MAX_MISSING_DATA_CAPTIONS);
  const timeFormat = getTimeTickFormat(series.period);

  return (
    <div className="mt-4 space-y-6">
      {children}

      {(dataSinceMs !== null || missingDataRanges.length > 0) && (
        <div className="space-y-0.5 text-xs text-muted">
          {dataSinceMs !== null && (
            <div>
              {t('backups.physical.monitoring.charts.dataSince', {
                time: dayjs(dataSinceMs).format(timeFormat),
              })}
            </div>
          )}

          {missingDataRanges.map((range) => (
            <div key={range.fromMs}>
              {t('backups.physical.monitoring.charts.noMeasurements', {
                from: dayjs(range.fromMs).format(timeFormat),
                to: dayjs(range.toMs).format(timeFormat),
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export const WalStreamKeepingUpChartComponent = ({
  series,
  period,
  onPeriodChange,
  isAnimationActive,
}: KeepingUpChartProps) => {
  const { t } = useTranslation();
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];

  const periodSwitch = (
    <Segmented
      aria-label={t('healthcheck.attempts.period')}
      value={period}
      onChange={(value) => onPeriodChange(value as WalStreamStatisticsPeriod)}
      options={PERIODS.map((periodOption) => ({
        value: periodOption,
        label: t(WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS[periodOption]),
      }))}
    />
  );

  if (!hasEnoughChartPoints(series)) {
    return (
      <div className="mt-6">
        <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
          <div className="font-semibold">
            {t('backups.physical.monitoring.charts.keepingUpTitle')}
          </div>
          {periodSwitch}
        </div>

        <ChartAvailabilityComponent series={series}>{null}</ChartAvailabilityComponent>
      </div>
    );
  }

  return (
    <div className="mt-6">
      <ChartAvailabilityComponent series={series}>
        <ChartFrameComponent
          titleKey="backups.physical.monitoring.charts.keepingUpTitle"
          headerExtra={periodSwitch}
          heightClassName={MAIN_CHART_HEIGHT_CLASS}
          rows={toChartRows(series.points)}
          series={series}
          isRate
          minimumTop={RATE_AXIS_MINIMUM_TOP}
          isAnimationActive={isAnimationActive}
          lines={[
            {
              dataKey: 'archiveBytesPerSecond',
              labelKey: 'backups.physical.monitoring.charts.savedToStorage',
              color: chartColors.archive,
              style: ChartLineStyle.BAND,
            },
            {
              dataKey: 'sourceWriteBytesPerSecond',
              labelKey: 'backups.physical.monitoring.charts.databaseWrites',
              color: chartColors.sourceWrites,
              style: ChartLineStyle.SOLID,
            },
          ]}
        />
      </ChartAvailabilityComponent>
    </div>
  );
};

export const WalStreamTechnicalChartsComponent = ({
  series,
  walSegmentSizeBytes,
  isAnimationActive,
}: TechnicalChartsProps) => {
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];
  const rows = toChartRows(series.points);
  const latestPauseLimitBytes =
    [...series.points].reverse().find((point) => point.localQueuePauseBytes !== null)
      ?.localQueuePauseBytes ?? 0;

  return (
    <ChartAvailabilityComponent series={series}>
      <ChartFrameComponent
        titleKey="backups.physical.monitoring.charts.unsavedTitle"
        subtitleKey="backups.physical.monitoring.charts.unsavedSubtitle"
        heightClassName={TECHNICAL_CHART_HEIGHT_CLASS}
        rows={rows}
        series={series}
        isRate={false}
        minimumTop={walSegmentSizeBytes}
        isAnimationActive={isAnimationActive}
        lines={[
          {
            dataKey: 'maxUnarchivedWalBytes',
            labelKey: 'backups.physical.monitoring.charts.notSavedYet',
            color: chartColors.notSavedYet,
            style: ChartLineStyle.SOLID,
          },
          {
            dataKey: 'maxSlotLagBytes',
            labelKey: 'backups.physical.monitoring.charts.notReceivedYet',
            color: chartColors.notReceivedYet,
            style: ChartLineStyle.DASHED,
          },
        ]}
      />

      <ChartFrameComponent
        titleKey="backups.physical.monitoring.charts.bufferTitle"
        heightClassName={TECHNICAL_CHART_HEIGHT_CLASS}
        rows={rows}
        series={series}
        isRate={false}
        minimumTop={Math.max(walSegmentSizeBytes, latestPauseLimitBytes)}
        isAnimationActive={isAnimationActive}
        pauseLimitBytes={latestPauseLimitBytes}
        lines={[
          {
            dataKey: 'maxLocalQueueBytes',
            labelKey: 'backups.physical.monitoring.charts.waitingInBuffer',
            color: chartColors.buffer,
            style: ChartLineStyle.SOLID,
          },
        ]}
      />
    </ChartAvailabilityComponent>
  );
};
