import { Tooltip, theme } from 'antd';
import { useTranslation } from 'react-i18next';

import { formatMoment } from '@/pages/databases/lib/wal-stream-monitoring/monitoringTextValues';
import {
  getLostStretchTexts,
  getMomentParam,
  getRestoreTimeline,
  getSegmentRangeText,
} from '@/pages/databases/lib/wal-stream-monitoring/restoreTimeline';
import { RestoreTimelineSegmentKind } from '@/pages/databases/model/wal-stream-monitoring/RestoreTimelineSegmentKind';
import type { WalStreamTimeRange } from '@/pages/databases/model/wal-stream-monitoring/WalStreamTimeRange';
import { MonitoringTextComponent } from '@/pages/databases/ui/wal-stream-monitoring/MonitoringTextComponent';
import type { TranslationKey } from '@/shared/i18n';

interface Props {
  ranges: WalStreamTimeRange[];
  now: Date;
}

const SEGMENT_LABEL_KEYS: Record<RestoreTimelineSegmentKind, TranslationKey> = {
  [RestoreTimelineSegmentKind.RESTORABLE]: 'backups.physical.monitoring.restoreTimeline.restorable',
  [RestoreTimelineSegmentKind.LOST]: 'backups.physical.monitoring.restoreTimeline.lost',
  [RestoreTimelineSegmentKind.NOT_SAVED_YET]:
    'backups.physical.monitoring.restoreTimeline.notSavedYet',
};

const SEGMENT_KINDS = [
  RestoreTimelineSegmentKind.RESTORABLE,
  RestoreTimelineSegmentKind.LOST,
  RestoreTimelineSegmentKind.NOT_SAVED_YET,
];

const useSegmentBackgrounds = (): Record<RestoreTimelineSegmentKind, string> => {
  const { token } = theme.useToken();

  return {
    [RestoreTimelineSegmentKind.RESTORABLE]: token.colorSuccess,
    // Hatched as well as muted red, so the lost stretch stands apart without color and an old gap
    // does not shout next to a green banner.
    // eslint-disable-next-line i18next/no-literal-string -- CSS value, not copy
    [RestoreTimelineSegmentKind.LOST]: `repeating-linear-gradient(135deg, ${token.colorErrorBorder} 0 3px, ${token.colorErrorBg} 3px 6px)`,
    [RestoreTimelineSegmentKind.NOT_SAVED_YET]: token.colorTextQuaternary,
  };
};

export const RestoreTimelineComponent = ({ ranges, now }: Props) => {
  const { t } = useTranslation();
  const segmentBackgrounds = useSegmentBackgrounds();

  const segments = getRestoreTimeline(ranges, now);

  if (segments.length === 0) {
    return null;
  }

  const shownKinds = SEGMENT_KINDS.filter((kind) =>
    segments.some((segment) => segment.kind === kind),
  );
  const lostStretchTexts = getLostStretchTexts(segments, now);

  return (
    <div className="mt-6">
      <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-1">
        <div className="font-semibold">
          {t('backups.physical.monitoring.restoreTimeline.title')}
        </div>

        <ul className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
          {shownKinds.map((kind) => (
            <li key={kind} className="flex items-center gap-1.5">
              <span
                aria-hidden
                className="inline-block h-2.5 w-4 shrink-0 rounded-sm"
                style={{ background: segmentBackgrounds[kind] }}
              />
              {t(SEGMENT_LABEL_KEYS[kind])}
            </li>
          ))}
        </ul>
      </div>

      <div className="relative mt-2 h-3 w-full overflow-hidden rounded-full bg-hover">
        {segments.map((segment) => (
          <Tooltip
            key={`${segment.kind}-${segment.fromMs}`}
            title={
              <div>
                <div className="font-semibold">{t(SEGMENT_LABEL_KEYS[segment.kind])}</div>
                <MonitoringTextComponent text={getSegmentRangeText(segment, now)} />
              </div>
            }
          >
            <div
              className="absolute top-0 h-full"
              style={{
                left: `${segment.startPercent}%`,
                width: `${segment.widthPercent}%`,
                background: segmentBackgrounds[segment.kind],
              }}
            />
          </Tooltip>
        ))}
      </div>

      <div className="mt-1 flex justify-between text-xs text-muted">
        <span>{formatMoment(getMomentParam(segments[0].fromMs, now))}</span>
        <span>{formatMoment(getMomentParam(segments[segments.length - 1].toMs, now))}</span>
      </div>

      {lostStretchTexts.length > 0 && (
        <ul className="mt-2 space-y-1 text-sm">
          {lostStretchTexts.map((text, index) => (
            <li key={index}>
              <MonitoringTextComponent text={text} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
