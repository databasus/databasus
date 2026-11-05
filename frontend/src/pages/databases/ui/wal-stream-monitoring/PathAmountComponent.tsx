import { CheckCircleOutlined } from '@ant-design/icons';
import { theme } from 'antd';
import { useTranslation } from 'react-i18next';

import { useLocale } from '../../../../shared/i18n';
import { formatBytes } from '../../../../shared/lib';
import { interpolateNumber } from '../../lib/wal-stream-monitoring/easedTexts';
import { getAmountProgress } from '../../lib/wal-stream-monitoring/pathStages';
import { useEasedTransition } from './useEasedTransition';

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
  const { token } = theme.useToken();

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
        <CheckCircleOutlined aria-hidden style={{ color: token.colorSuccess }} />
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
                backgroundColor: progress.isOverLimit ? token.colorError : token.colorPrimary,
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
