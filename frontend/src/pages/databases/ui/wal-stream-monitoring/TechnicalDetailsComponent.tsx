import { useTranslation } from 'react-i18next';

import type { MonitoringTone } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTone';
import type { PathStage } from '@/pages/databases/model/wal-stream-monitoring/PathStage';
import type { ReasonTexts } from '@/pages/databases/model/wal-stream-monitoring/ReasonTexts';
import type { WalStreamStatistics } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatistics';
import { ChangesPathComponent } from '@/pages/databases/ui/wal-stream-monitoring/ChangesPathComponent';
import { MonitoringTipsComponent } from '@/pages/databases/ui/wal-stream-monitoring/MonitoringTipsComponent';
import { WalStreamTechnicalChartsComponent } from '@/pages/databases/ui/wal-stream-monitoring/WalStreamChartsComponent';
import { WalStreamRawValuesComponent } from '@/pages/databases/ui/wal-stream-monitoring/WalStreamRawValuesComponent';

interface Props {
  statistics: WalStreamStatistics;
  stageTones: Record<PathStage, MonitoringTone>;
  tips: ReasonTexts[];
  isReducedMotion: boolean;
  isChartAnimationActive: boolean;
}

// Without a sample the segment size is unknown; 16 MiB is PostgreSQL's default.
const DEFAULT_WAL_SEGMENT_SIZE_BYTES = 16 * 1024 * 1024;

// The figures stay in view under the verdict instead of behind a click: a reader checking why the
// verdict says what it does should not have to look for them.
export const TechnicalDetailsComponent = ({
  statistics,
  stageTones,
  tips,
  isReducedMotion,
  isChartAnimationActive,
}: Props) => {
  const { t } = useTranslation();

  return (
    <section className="mt-8 space-y-6 border-t border-border pt-6">
      <h3 className="text-base font-semibold">
        {t('backups.physical.monitoring.technicalDetails.title')}
      </h3>

      <ChangesPathComponent
        statistics={statistics}
        stageTones={stageTones}
        isReducedMotion={isReducedMotion}
      />

      <MonitoringTipsComponent tips={tips} />

      <WalStreamTechnicalChartsComponent
        series={statistics.series}
        walSegmentSizeBytes={
          statistics.current?.walSegmentSizeBytes ?? DEFAULT_WAL_SEGMENT_SIZE_BYTES
        }
        isAnimationActive={isChartAnimationActive}
      />

      <WalStreamRawValuesComponent statistics={statistics} />
    </section>
  );
};
