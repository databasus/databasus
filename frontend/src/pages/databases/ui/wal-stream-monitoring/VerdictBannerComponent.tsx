import {
  CheckCircleFilled,
  ClockCircleOutlined,
  CloseCircleFilled,
  ExclamationCircleFilled,
  PauseCircleFilled,
  QuestionCircleFilled,
} from '@ant-design/icons';
import { Button, theme } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { translateLocalizedText } from '../../../../shared/i18n';
import {
  getMeasurementAge,
  getSecondsBetween,
} from '../../lib/wal-stream-monitoring/roundedDuration';
import { getToneColors } from '../../lib/wal-stream-monitoring/toneColors';
import { UNCONFIRMED_STATE_AFTER_MS } from '../../lib/wal-stream-monitoring/verdictBanner';
import { FullBackupAction } from '../../model/wal-stream-monitoring/FullBackupAction';
import type { ReasonTexts } from '../../model/wal-stream-monitoring/ReasonTexts';
import type { VerdictBanner } from '../../model/wal-stream-monitoring/VerdictBanner';
import { VerdictIcon } from '../../model/wal-stream-monitoring/VerdictIcon';
import { WalStreamReasonLevel } from '../../model/wal-stream-monitoring/WalStreamReasonLevel';
import { MonitoringTextComponent } from './MonitoringTextComponent';

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
  [VerdictIcon.CHECK]: <CheckCircleFilled />,
  [VerdictIcon.EXCLAMATION]: <ExclamationCircleFilled />,
  [VerdictIcon.CLOSE]: <CloseCircleFilled />,
  [VerdictIcon.CLOCK]: <ClockCircleOutlined />,
  [VerdictIcon.PAUSE]: <PauseCircleFilled />,
  [VerdictIcon.QUESTION]: <QuestionCircleFilled />,
};

const OtherReasonComponent = ({ reason }: { reason: ReasonTexts }) => {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const isBroken = reason.level === WalStreamReasonLevel.BROKEN;

  return (
    <li className="flex items-start gap-2 text-sm">
      <span
        className="mt-0.5 leading-none"
        style={{ color: isBroken ? token.colorError : token.colorWarning }}
        aria-hidden
      >
        {isBroken ? <CloseCircleFilled /> : <ExclamationCircleFilled />}
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
  const { token } = theme.useToken();

  const toneColors = getToneColors(token, banner.tone);

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
                style={{ backgroundColor: isLive ? token.colorSuccess : token.colorTextQuaternary }}
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
                type="primary"
                loading={fullBackupAction === FullBackupAction.REQUESTING}
                onClick={onStartFullBackup}
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
