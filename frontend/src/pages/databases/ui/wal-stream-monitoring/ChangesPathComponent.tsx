import {
  ArrowDownOutlined,
  ArrowRightOutlined,
  CloseCircleFilled,
  CloudOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  ExclamationCircleFilled,
  InfoCircleOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons';
import { Tooltip, theme } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { type TranslationKey, useLocale } from '../../../../shared/i18n';
import { formatBytes } from '../../../../shared/lib';
import { useTheme } from '../../../../shared/theme';
import { WAL_STREAM_CHART_COLORS } from '../../lib/wal-stream-monitoring/chartColors';
import { formatBytesPerSecond } from '../../lib/wal-stream-monitoring/monitoringTextValues';
import { getToneColors } from '../../lib/wal-stream-monitoring/toneColors';
import { MonitoringTone } from '../../model/wal-stream-monitoring/MonitoringTone';
import { PathStage } from '../../model/wal-stream-monitoring/PathStage';
import { WalStreamReasonLevel } from '../../model/wal-stream-monitoring/WalStreamReasonLevel';
import { WAL_STREAM_REASON_LEVEL_LABEL_KEYS } from '../../model/wal-stream-monitoring/WalStreamReasonLevelLabelKeys';
import { WalStreamSourceState } from '../../model/wal-stream-monitoring/WalStreamSourceState';
import { type WalStreamStatistics } from '../../model/wal-stream-monitoring/WalStreamStatistics';
import { PathAmountComponent } from './PathAmountComponent';

interface Props {
  statistics: WalStreamStatistics;
  stageTones: Record<PathStage, MonitoringTone>;
  isReducedMotion: boolean;
}

interface StageHeaderProps {
  icon: ReactNode;
  titleKey: TranslationKey;
  tone: MonitoringTone;
  // The line to the next stage starts right after the title, so the strip reads as one pipe.
  hasOutgoingLine?: boolean;
}

interface StageDetailsProps {
  labelKey: TranslationKey;
  tooltipKey: TranslationKey;
  children: ReactNode;
}

interface NoteProps {
  tooltipKey?: TranslationKey;
  children: ReactNode;
}

interface LinkHeaderProps {
  tooltipKey: TranslationKey;
  bytesPerSecond: number | null;
}

interface LinkLabelProps {
  labelKey: TranslationKey;
  lineColor: string;
}

// A stage without a problem stays neutral: a healthy color on every stage would read as a selection
// or an alarm. A problem adds its icon and its word, so color is never the only signal.
const PROBLEM_TONE_LEVELS: Partial<Record<MonitoringTone, WalStreamReasonLevel>> = {
  [MonitoringTone.WARNING]: WalStreamReasonLevel.NEEDS_ATTENTION,
  [MonitoringTone.ERROR]: WalStreamReasonLevel.BROKEN,
};

// On a wide section every stage and link spans the strip's two rows through a subgrid, so the
// icons and the lines share the first row and the details share the second; on a narrow one they
// stack in reading order.
// eslint-disable-next-line i18next/no-literal-string -- Tailwind classes, not copy
const STRIP_CELL_CLASS = '@2xl:row-span-2 @2xl:grid @2xl:grid-rows-subgrid';

const StageHeaderComponent = ({ icon, titleKey, tone, hasOutgoingLine }: StageHeaderProps) => {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const problemLevel = PROBLEM_TONE_LEVELS[tone];
  const toneColors = getToneColors(token, tone);

  return (
    <div className="flex items-center gap-x-2 @2xl:h-9">
      <span
        aria-hidden
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border text-base"
        style={
          problemLevel
            ? {
                borderColor: toneColors.border,
                backgroundColor: toneColors.background,
                color: toneColors.icon,
              }
            : { borderColor: token.colorBorderSecondary, color: token.colorTextSecondary }
        }
      >
        {icon}
      </span>

      <span className="font-semibold">{t(titleKey)}</span>

      {problemLevel && (
        <span
          className="flex items-center gap-1 text-xs font-medium"
          style={{ color: toneColors.icon }}
        >
          {problemLevel === WalStreamReasonLevel.BROKEN ? (
            <CloseCircleFilled aria-hidden />
          ) : (
            <ExclamationCircleFilled aria-hidden />
          )}
          {t(WAL_STREAM_REASON_LEVEL_LABEL_KEYS[problemLevel])}
        </span>
      )}

      {hasOutgoingLine && (
        <span className="ml-1 hidden h-px min-w-3 flex-1 bg-border @2xl:block" aria-hidden />
      )}
    </div>
  );
};

const InfoTooltipComponent = ({ tooltipKey }: { tooltipKey: TranslationKey }) => {
  const { t } = useTranslation();

  return (
    <Tooltip title={t(tooltipKey)}>
      <InfoCircleOutlined
        tabIndex={0}
        aria-label={t(tooltipKey)}
        className="cursor-help text-muted"
      />
    </Tooltip>
  );
};

const StageDetailsComponent = ({ labelKey, tooltipKey, children }: StageDetailsProps) => {
  const { t } = useTranslation();

  return (
    <div className="mt-2 space-y-1.5 pr-3 pl-11">
      {/* Inline, so the info icon follows the last word when the label wraps. */}
      <div className="text-xs text-muted">
        {t(labelKey)} <InfoTooltipComponent tooltipKey={tooltipKey} />
      </div>
      {children}
    </div>
  );
};

const NoteComponent = ({ tooltipKey, children }: NoteProps) => (
  <div className="flex items-start gap-1.5 text-xs text-muted">
    <span>{children}</span>
    {tooltipKey && <InfoTooltipComponent tooltipKey={tooltipKey} />}
  </div>
);

// The rate sits on the line between two stages, like the throughput of a pipe.
const LinkHeaderComponent = ({ tooltipKey, bytesPerSecond }: LinkHeaderProps) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  return (
    <div className="flex items-center justify-center pt-2 text-muted @2xl:h-9 @2xl:pt-0 @2xl:pr-2">
      {/* The container breakpoint classes sit on wrappers: AntD's own icon display rule would
      override them on the icon element and show both arrows. */}
      <span className="hidden h-px flex-1 bg-border @2xl:block" aria-hidden />

      <Tooltip title={t(tooltipKey)}>
        <span
          tabIndex={0}
          className="mx-1 cursor-help rounded-full border border-border px-2 py-0.5 text-sm font-semibold whitespace-nowrap text-content tabular-nums"
        >
          {bytesPerSecond === null
            ? t('backups.physical.monitoring.path.unknownAmount')
            : formatBytesPerSecond(bytesPerSecond, t, formatNumber)}
        </span>
      </Tooltip>

      <span className="hidden h-px flex-1 bg-border @2xl:block" aria-hidden />
      <span className="hidden @2xl:block" aria-hidden>
        <ArrowRightOutlined />
      </span>
    </div>
  );
};

const LinkLabelComponent = ({ labelKey, lineColor }: LinkLabelProps) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center gap-1 pb-2 @2xl:pb-0">
      <span className="flex items-center gap-1.5 text-xs text-muted">
        <span
          aria-hidden
          className="inline-block h-2 w-2 shrink-0 rounded-full"
          style={{ backgroundColor: lineColor }}
        />
        {t(labelKey)}
      </span>
      <span className="block text-muted @2xl:hidden" aria-hidden>
        <ArrowDownOutlined />
      </span>
    </div>
  );
};

export const ChangesPathComponent = ({ statistics, stageTones, isReducedMotion }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];
  const { current, keepingUp } = statistics;
  const slotRebuildLagBytes = keepingUp.slotRebuildLagBytes;
  const slotLagBytes = current?.slotLagBytes ?? null;
  const localQueueBytes = current?.localQueueBytes ?? null;

  return (
    <div className="rounded-lg border border-border p-4">
      <div className="@container">
        <ol className="grid grid-cols-1 @2xl:grid-cols-[minmax(0,1fr)_minmax(104px,140px)_minmax(0,1fr)_minmax(104px,140px)_minmax(0,1fr)] @2xl:grid-rows-[auto_auto]">
          <li className={STRIP_CELL_CLASS}>
            <StageHeaderComponent
              icon={<DatabaseOutlined />}
              titleKey="backups.physical.monitoring.path.database"
              hasOutgoingLine
              tone={stageTones[PathStage.SOURCE]}
            />
            <StageDetailsComponent
              labelKey="backups.physical.monitoring.path.notReceivedYet"
              tooltipKey="backups.physical.monitoring.tooltips.notReceivedYet"
            >
              <PathAmountComponent
                valueBytes={slotLagBytes}
                limitBytes={slotRebuildLagBytes > 0 ? slotRebuildLagBytes : undefined}
                isReducedMotion={isReducedMotion}
              />

              {/* A limit matters once something waits; until then it stays in the tooltip. */}
              {slotRebuildLagBytes > 0 && slotLagBytes !== null && slotLagBytes > 0 && (
                <NoteComponent tooltipKey="backups.physical.monitoring.tooltips.rebuildLimit">
                  {t('backups.physical.monitoring.path.rebuildLimit', {
                    amount: formatBytes(slotRebuildLagBytes, formatNumber),
                  })}
                </NoteComponent>
              )}

              {current?.sourceState === WalStreamSourceState.REACHABLE &&
                current.slotSafeWalSizeBytes != null && (
                  <NoteComponent tooltipKey="backups.physical.monitoring.tooltips.allowanceLeft">
                    {t('backups.physical.monitoring.path.allowanceLeft', {
                      amount: formatBytes(current.slotSafeWalSizeBytes, formatNumber),
                    })}
                  </NoteComponent>
                )}
            </StageDetailsComponent>
          </li>

          <li className={STRIP_CELL_CLASS}>
            <LinkHeaderComponent
              tooltipKey="backups.physical.monitoring.tooltips.writeRate"
              bytesPerSecond={keepingUp.sourceWriteBytesPerSecond}
            />
            <LinkLabelComponent
              labelKey="backups.physical.monitoring.charts.databaseWrites"
              lineColor={chartColors.sourceWrites}
            />
          </li>

          <li className={STRIP_CELL_CLASS}>
            <StageHeaderComponent
              icon={<CloudServerOutlined />}
              titleKey="backups.physical.monitoring.path.databasus"
              hasOutgoingLine
              tone={stageTones[PathStage.DATABASUS]}
            />
            <StageDetailsComponent
              labelKey="backups.physical.monitoring.path.waitingInBuffer"
              tooltipKey="backups.physical.monitoring.tooltips.waitingInBuffer"
            >
              <PathAmountComponent
                valueBytes={localQueueBytes}
                limitBytes={current?.localQueuePauseBytes}
                isReducedMotion={isReducedMotion}
              />

              {current && localQueueBytes !== null && localQueueBytes > 0 && (
                <NoteComponent tooltipKey="backups.physical.monitoring.tooltips.pauseLimit">
                  {t('backups.physical.monitoring.path.pauseLimit', {
                    amount: formatBytes(current.localQueuePauseBytes, formatNumber),
                  })}
                </NoteComponent>
              )}

              {current && !current.isReceiverRunning && (
                <div className="flex items-center gap-1.5 text-xs font-semibold text-warning">
                  <PauseCircleOutlined aria-hidden />
                  {t('backups.physical.monitoring.path.receivingPaused')}
                </div>
              )}
            </StageDetailsComponent>
          </li>

          <li className={STRIP_CELL_CLASS}>
            <LinkHeaderComponent
              tooltipKey="backups.physical.monitoring.tooltips.saveRate"
              bytesPerSecond={keepingUp.archiveBytesPerSecond}
            />
            <LinkLabelComponent
              labelKey="backups.physical.monitoring.charts.savedToStorage"
              lineColor={chartColors.archive}
            />
          </li>

          <li className={STRIP_CELL_CLASS}>
            <StageHeaderComponent
              icon={<CloudOutlined />}
              titleKey="backups.physical.monitoring.path.storage"
              tone={stageTones[PathStage.STORAGE]}
            />
            <StageDetailsComponent
              labelKey="backups.physical.monitoring.figures.lastSaved"
              tooltipKey="backups.physical.monitoring.tooltips.lastSaved"
            >
              <div className="text-base font-semibold">
                {current?.lastArchivedWalAt
                  ? formatRelativeTime(current.lastArchivedWalAt)
                  : t('backups.physical.monitoring.path.nothingSavedYet')}
              </div>

              {current?.unarchivedWalBytes != null && (
                <NoteComponent tooltipKey="backups.physical.monitoring.tooltips.notSavedYet">
                  {t('backups.physical.monitoring.path.notSavedYetAmount', {
                    amount: formatBytes(current.unarchivedWalBytes, formatNumber),
                  })}
                </NoteComponent>
              )}
            </StageDetailsComponent>
          </li>
        </ol>
      </div>

      <p className="mt-4 mb-0 text-xs text-muted">
        {t('backups.physical.monitoring.path.ratesCaption')}
      </p>
    </div>
  );
};
