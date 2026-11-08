import { Button } from '@mantine/core';
import { CircleAlert, CircleCheck, CircleHelp, CirclePause, CircleX, Clock3 } from 'lucide-react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import {
  getMeasurementAge,
  getSecondsBetween,
} from '@/pages/databases/wal-monitoring/lib/roundedDuration';
import { getToneColors } from '@/pages/databases/wal-monitoring/lib/toneColors';
import { UNCONFIRMED_STATE_AFTER_MS } from '@/pages/databases/wal-monitoring/lib/verdictBanner';
import { FullBackupAction } from '@/pages/databases/wal-monitoring/model/FullBackupAction';
import type { ReasonTexts } from '@/pages/databases/wal-monitoring/model/ReasonTexts';
import {
  type VerdictBanner,
  VerdictIcon,
} from '@/pages/databases/wal-monitoring/model/VerdictBanner';
import { WalStreamReasonLevel } from '@/pages/databases/wal-monitoring/model/WalStreamReasonLevel';
import { MonitoringTextComponent } from '@/pages/databases/wal-monitoring/ui/MonitoringTextComponent';
import { translateLocalizedText } from '@/shared/i18n';
import { THEME_COLOR_VARIABLES } from '@/shared/theme';
interface Props {
  banner: VerdictBanner;
  fullBackupAction: FullBackupAction;
  sampledAt: string | null;
  now: Date;
  isReducedMotion: boolean;
  onStartFullBackup: () => void;
  onOpenConfig: () => void;
}

const VERDICT_ICONS: Record<VerdictIcon, ReactNode> = {
  [VerdictIcon.CHECK]: <CircleCheck size="1em" className="inline-block shrink-0 align-middle" />,
  [VerdictIcon.EXCLAMATION]: (
    <CircleAlert size="1em" className="inline-block shrink-0 align-middle" />
  ),
  [VerdictIcon.CLOSE]: <CircleX size="1em" className="inline-block shrink-0 align-middle" />,
  [VerdictIcon.CLOCK]: <Clock3 size="1em" className="inline-block shrink-0 align-middle" />,
  [VerdictIcon.PAUSE]: <CirclePause size="1em" className="inline-block shrink-0 align-middle" />,
  [VerdictIcon.QUESTION]: <CircleHelp size="1em" className="inline-block shrink-0 align-middle" />,
};

const OtherReasonComponent = ({ reason }: { reason: ReasonTexts }) => {
  const { t } = useTranslation();
  const isBroken = reason.level === WalStreamReasonLevel.BROKEN;

  return (
    <li className="flex items-start gap-2 text-sm">
      <span
        className="mt-0.5 leading-none"
        style={{
          color: isBroken
            ? THEME_COLOR_VARIABLES['error-fill']
            : THEME_COLOR_VARIABLES['warning-fill'],
        }}
        aria-hidden
      >
        {isBroken ? (
          <CircleX size="1em" className="inline-block shrink-0 align-middle" />
        ) : (
          <CircleAlert size="1em" className="inline-block shrink-0 align-middle" />
        )}
      </span>

      <div>
        <div className="font-semibold">{t(reason.titleKey)}</div>
        <div>
          <span className="mr-1">
            <MonitoringTextComponent text={reason.whatHappened} />
          </span>
          <span className="text-muted">
            <MonitoringTextComponent text={reason.whatToDo} />
          </span>
        </div>
      </div>
    </li>
  );
};

export const VerdictBannerComponent = ({
  banner,
  fullBackupAction,
  sampledAt,
  now,
  isReducedMotion,
  onStartFullBackup,
  onOpenConfig,
}: Props) => {
  const { t } = useTranslation();
  const toneColors = getToneColors(banner.tone);
  const measuredSeconds = sampledAt === null ? null : getSecondsBetween(sampledAt, now);
  const isLive = measuredSeconds !== null && measuredSeconds * 1000 < UNCONFIRMED_STATE_AFTER_MS;
  const isFullBackupActionShown = fullBackupAction !== FullBackupAction.HIDDEN;

  return (
    <div
      className="w-full rounded-lg border px-4 py-3"
      style={{
        backgroundColor: toneColors.background,
        borderColor: toneColors.border,
        transition: isReducedMotion ? undefined : 'background-color 300ms, border-color 300ms',
      }}
    >
      <div className="flex min-w-0 items-start gap-2.5">
        <div className="pt-0.5 text-xl leading-none" style={{ color: toneColors.icon }} aria-hidden>
          {VERDICT_ICONS[banner.icon]}
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5">
            <div aria-live="polite" className="text-base font-semibold">
              {t(banner.headlineKey)}
            </div>

            <div className="flex items-center gap-1.5 text-xs text-muted">
              <span
                aria-hidden
                className={`inline-block h-2 w-2 rounded-full ${isLive && !isReducedMotion ? 'animate-pulse' : ''}`}
                style={{
                  backgroundColor: isLive
                    ? THEME_COLOR_VARIABLES['success-fill']
                    : THEME_COLOR_VARIABLES['disabled'],
                }}
              />

              {measuredSeconds === null
                ? t('backups.physical.monitoring.banner.notMeasuredYet')
                : t('backups.physical.monitoring.banner.measuredAgo', {
                    duration: translateLocalizedText(getMeasurementAge(measuredSeconds), t),
                  })}
            </div>
          </div>

          <div className="max-w-3xl text-sm">
            {banner.description && (
              <div className="mt-1">
                <MonitoringTextComponent text={banner.description} onOpenConfig={onOpenConfig} />
              </div>
            )}

            {banner.headlineReason && (
              <div className="mt-1">
                <div>
                  <MonitoringTextComponent text={banner.headlineReason.whatHappened} />
                </div>

                {/* The button below is the action, so its sentence would only repeat it. */}
                {!isFullBackupActionShown && (
                  <div className="text-muted">
                    <MonitoringTextComponent text={banner.headlineReason.whatToDo} />
                  </div>
                )}
              </div>
            )}

            <div className="mt-0.5">
              <MonitoringTextComponent text={banner.restoreSentence} />
            </div>

            {isFullBackupActionShown && (
              <Button
                className="mt-3"
                variant="filled"
                loading={fullBackupAction === FullBackupAction.REQUESTING}
                onClick={onStartFullBackup}
                aria-busy={fullBackupAction === FullBackupAction.REQUESTING}
              >
                {t('backups.physical.monitoring.banner.startFullBackup')}
              </Button>
            )}

            {banner.otherReasons.length > 0 && (
              <ul
                className="mt-3 space-y-2 border-t pt-3"
                style={{ borderColor: toneColors.border }}
              >
                {banner.otherReasons.map((reason) => (
                  <OtherReasonComponent key={reason.titleKey} reason={reason} />
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
