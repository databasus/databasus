import { CircleCheck } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { interpolateNumber } from '@/pages/databases/wal-monitoring/lib/easedTexts';
import { getAmountProgress } from '@/pages/databases/wal-monitoring/lib/pathStages';
import { useEasedTransition } from '@/pages/databases/wal-monitoring/ui/useEasedTransition';
import { useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';
import { THEME_COLOR_VARIABLES } from '@/shared/theme';
interface Props {
  valueBytes: number | null;
  limitBytes?: number;
  isReducedMotion: boolean;
}

const easeAmount = (fromBytes: number, toBytes: number, progress: number) =>
  interpolateNumber(fromBytes, toBytes, progress);

// An empty queue says "nothing waiting" instead of a bold 0 B over an empty meter, which read as
// broken. A filled queue shows the amount and, with a limit, "X of Y" under a thin meter.
export const PathAmountComponent = ({ valueBytes, limitBytes, isReducedMotion }: Props) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const easedBytes = useEasedTransition(valueBytes ?? 0, easeAmount, isReducedMotion);

  if (valueBytes === null) {
    return (
      <div className="text-sm text-muted">
        {t('backups.physical.monitoring.path.unknownAmount')}
      </div>
    );
  }

  if (valueBytes === 0) {
    return (
      <div className="flex items-center gap-1.5 text-sm text-muted">
        <CircleCheck
          aria-hidden
          style={{ color: THEME_COLOR_VARIABLES['success-fill'] }}
          size={14}
        />
        {t('backups.physical.monitoring.path.nothingWaiting')}
      </div>
    );
  }

  const progress = limitBytes === undefined ? null : getAmountProgress(valueBytes, limitBytes);

  return (
    <div>
      <div className="text-base font-semibold tabular-nums">
        {formatBytes(easedBytes, formatNumber)}
      </div>

      {progress && limitBytes !== undefined && (
        <div className="mt-1 max-w-48">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-hover">
            <div
              className="h-full rounded-full"
              style={{
                width: `${progress.percent}%`,
                backgroundColor: progress.isOverLimit
                  ? THEME_COLOR_VARIABLES['error-fill']
                  : THEME_COLOR_VARIABLES['primary'],
                transition: isReducedMotion ? undefined : 'width 300ms',
              }}
            />
          </div>

          <div className="mt-0.5 text-xs text-muted">
            {progress.isOverLimit ? (
              <span className="font-semibold text-danger">
                {t('backups.physical.monitoring.path.overLimit')}
              </span>
            ) : (
              t('backups.physical.monitoring.path.ofLimit', {
                amount: formatBytes(valueBytes, formatNumber),
                limit: formatBytes(limitBytes, formatNumber),
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
};
