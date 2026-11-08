import { Alert, Skeleton } from '@mantine/core';
import { TriangleAlert } from 'lucide-react';
import {
  memo,
  type ReactNode,
  useCallback,
  useEffect,
  useEffectEvent,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';

import type { Database } from '@/entities/databases';
import { physicalBackupsApi } from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { physicalWalStreamStatisticsApi } from '@/pages/databases/wal-monitoring/api/physicalWalStreamStatisticsApi';
import { getFullBackupAction } from '@/pages/databases/wal-monitoring/lib/fullBackupAction';
import { FullBackupRequestController } from '@/pages/databases/wal-monitoring/lib/fullBackupRequestController';
import { getKeepingUpSentences } from '@/pages/databases/wal-monitoring/lib/keepingUpSentences';
import { applyLiveValues } from '@/pages/databases/wal-monitoring/lib/liveOverlay';
import { getMonitoringFigures } from '@/pages/databases/wal-monitoring/lib/monitoringFigures';
import { getStageTones } from '@/pages/databases/wal-monitoring/lib/pathStages';
import { getTipTexts } from '@/pages/databases/wal-monitoring/lib/reasonTexts';
import { getVerdictBanner } from '@/pages/databases/wal-monitoring/lib/verdictBanner';
import { WalStreamLiveController } from '@/pages/databases/wal-monitoring/lib/walStreamLiveController';
import {
  type PageVisibility,
  type RefreshTimer,
  WalStreamRefreshController,
} from '@/pages/databases/wal-monitoring/lib/walStreamRefreshController';
import { type WalStreamLive } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import type { WalStreamRefreshState } from '@/pages/databases/wal-monitoring/model/WalStreamRefreshState';
import { WalStreamStatisticsPeriod } from '@/pages/databases/wal-monitoring/model/WalStreamStatisticsPeriod';
import { MonitoringFiguresComponent } from '@/pages/databases/wal-monitoring/ui/MonitoringFiguresComponent';
import { RestoreTimelineComponent } from '@/pages/databases/wal-monitoring/ui/RestoreTimelineComponent';
import { TechnicalDetailsComponent } from '@/pages/databases/wal-monitoring/ui/TechnicalDetailsComponent';
import { useCurrentTime } from '@/pages/databases/wal-monitoring/ui/useCurrentTime';
import { usePrefersReducedMotion } from '@/pages/databases/wal-monitoring/ui/usePrefersReducedMotion';
import { VerdictBannerComponent } from '@/pages/databases/wal-monitoring/ui/VerdictBannerComponent';
import { WalStreamKeepingUpChartComponent } from '@/pages/databases/wal-monitoring/ui/WalStreamChartsComponent';
import { useSessionMutation } from '@/shared/api';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  onOpenConfig: () => void;
}

// eslint-disable-next-line i18next/no-literal-string -- DOM event name
const VISIBILITY_CHANGE_EVENT = 'visibilitychange';

const browserPageVisibility: PageVisibility = {
  isVisible: () => document.visibilityState !== 'hidden',
  subscribe: (onVisibilityChange) => {
    document.addEventListener(VISIBILITY_CHANGE_EVENT, onVisibilityChange);

    return () => document.removeEventListener(VISIBILITY_CHANGE_EVENT, onVisibilityChange);
  },
};

const browserRefreshTimer: RefreshTimer = {
  setInterval: (callback, intervalMs) => window.setInterval(callback, intervalMs),
  clearInterval: (handle) => window.clearInterval(handle as number),
};

// Longer than any chart's entry animation, so the animation of a new period finishes before
// later refreshes stop animating.
const CHART_ENTRY_ANIMATION_MS = 1500;

// The charts re-render only when a refresh brings a new series, not on every tick of the clock.
const MemoizedWalStreamKeepingUpChartComponent = memo(WalStreamKeepingUpChartComponent);

const MonitoringContainerComponent = ({ children }: { children: ReactNode }) => (
  // The width stops where a line of text and a chart still read at a glance; past it the tab only
  // grows empty space.
  <div className="w-full max-w-6xl rounded-md bg-surface p-3 shadow md:p-5">{children}</div>
);

const MonitoringSkeletonComponent = () => (
  <div className="space-y-4">
    <div className="h-24 rounded-lg border border-border p-4">
      <div aria-busy="true">
        {Array.from({ length: 1 }, (_, index) => (
          <Skeleton key={index} height={16} className="mb-4" />
        ))}
      </div>
    </div>
    <div className="h-36 rounded-lg border border-border p-4">
      <div aria-busy="true">
        {Array.from({ length: 3 }, (_, index) => (
          <Skeleton key={index} height={16} className="mb-4" />
        ))}
      </div>
    </div>
    <div className="h-[760px] rounded-lg border border-border p-4">
      <div aria-busy="true">
        {Array.from({ length: 12 }, (_, index) => (
          <Skeleton key={index} height={16} className="mb-4" />
        ))}
      </div>
    </div>
  </div>
);

export const WalStreamMonitoringComponent = ({ database, isCanManageDBs, onOpenConfig }: Props) => {
  const triggerPhysicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof physicalBackupsApi.triggerPhysicalBackup>) =>
      physicalBackupsApi.triggerPhysicalBackup(...parameters),
  });

  const { t, i18n } = useTranslation();
  const notify = useFeedback();

  const [period, setPeriod] = useState<WalStreamStatisticsPeriod>(WalStreamStatisticsPeriod.HOUR);
  const [refreshState, setRefreshState] = useState<WalStreamRefreshState | null>(null);
  const [lastAnimatedPeriod, setLastAnimatedPeriod] = useState<WalStreamStatisticsPeriod | null>(
    null,
  );
  const [isRequestingFullBackup, setIsRequestingFullBackup] = useState(false);
  const [live, setLive] = useState<WalStreamLive | null>(null);

  const controllerRef = useRef<WalStreamRefreshController | null>(null);
  const fullBackupRequestControllerRef = useRef<FullBackupRequestController | null>(null);

  const retryFirstLoad = () => {
    controllerRef.current?.retry();
  };

  const startFullBackup = () => {
    void fullBackupRequestControllerRef.current?.requestFullBackup();
  };

  const now = useCurrentTime();
  const isReducedMotion = usePrefersReducedMotion();

  const changePeriod = useCallback((newPeriod: WalStreamStatisticsPeriod) => {
    setPeriod(newPeriod);
    controllerRef.current?.setPeriod(newPeriod);
  }, []);

  const getPeriod = useEffectEvent(() => period);

  useEffect(() => {
    const controller = new WalStreamRefreshController(
      {
        fetchStatistics: (requestedPeriod) =>
          physicalWalStreamStatisticsApi.getWalStreamStatistics(database.id, requestedPeriod),
        timer: browserRefreshTimer,
        pageVisibility: browserPageVisibility,
        getNow: () => new Date(),
      },
      getPeriod(),
      setRefreshState,
    );

    controllerRef.current = controller;
    controller.start();

    return () => controller.stop();
  }, [database.id]);

  useEffect(() => {
    setLive(null);

    const liveController = new WalStreamLiveController(
      {
        fetchLive: () => physicalWalStreamStatisticsApi.getWalStreamLive(database.id),
        timer: browserRefreshTimer,
        pageVisibility: browserPageVisibility,
      },
      setLive,
    );

    liveController.start();

    return () => liveController.stop();
  }, [database.id]);

  const requestFullBackup = useEffectEvent(() =>
    triggerPhysicalBackupMutation.mutateAsync([database.id, 'FULL']),
  );

  const reportFullBackupRequested = useEffectEvent(() => {
    notify({
      kind: 'success',
      title: i18n.t('backups.physical.monitoring.banner.fullBackupRequested'),
    });
  });
  const reportFullBackupError = useEffectEvent((error: unknown) => {
    notify({ kind: 'error', title: translateApiError(error, i18n.t) });
  });

  useEffect(() => {
    const requestController = new FullBackupRequestController(
      {
        requestFullBackup: () => requestFullBackup(),
        refreshNow: () => controllerRef.current?.retry(),
        reportRequested: () => reportFullBackupRequested(),
        reportError: (e) => reportFullBackupError(e),
      },
      setIsRequestingFullBackup,
    );

    setIsRequestingFullBackup(false);
    fullBackupRequestControllerRef.current = requestController;

    return () => requestController.stopReporting();
  }, [database.id]);

  useEffect(() => {
    fullBackupRequestControllerRef.current?.handleRefreshLanded();
  }, [refreshState]);

  const shownSeriesPeriod = refreshState?.statistics?.series.period ?? null;

  useEffect(() => {
    if (shownSeriesPeriod === null || shownSeriesPeriod === lastAnimatedPeriod) {
      return;
    }

    const timeout = setTimeout(
      () => setLastAnimatedPeriod(shownSeriesPeriod),
      CHART_ENTRY_ANIMATION_MS,
    );

    return () => clearTimeout(timeout);
  }, [shownSeriesPeriod, lastAnimatedPeriod]);

  const statistics = refreshState?.statistics
    ? applyLiveValues(refreshState.statistics, live)
    : null;

  if (!statistics || !refreshState?.lastSuccessfulRefreshAt) {
    return (
      <MonitoringContainerComponent>
        {refreshState?.firstLoadError ? (
          <LoadFailure error={refreshState.firstLoadError} onRetry={retryFirstLoad} />
        ) : (
          <MonitoringSkeletonComponent />
        )}
      </MonitoringContainerComponent>
    );
  }

  const banner = getVerdictBanner({
    statistics,
    lastSuccessfulRefreshAt: refreshState.lastSuccessfulRefreshAt,
    now,
  });
  const figures = getMonitoringFigures(statistics);
  const isChartAnimationActive =
    !isReducedMotion && statistics.series.period !== lastAnimatedPeriod;

  return (
    <MonitoringContainerComponent>
      <VerdictBannerComponent
        banner={banner}
        fullBackupAction={getFullBackupAction({
          level: statistics.health.level,
          isStateConfirmed: banner.isStateConfirmed,
          reasons: statistics.health.reasons,
          latestRestorableAt: statistics.latestRestorableAt,
          isFullBackupPending: statistics.health.isFullBackupPending,
          isCanManageDBs,
          isRequestingFullBackup,
        })}
        sampledAt={statistics.current?.sampledAt ?? null}
        now={now}
        isReducedMotion={isReducedMotion}
        onStartFullBackup={startFullBackup}
        onOpenConfig={onOpenConfig}
      />

      {refreshState.isLastRefreshFailed && (
        <Alert
          className="mt-3"
          color="warning"
          icon={
            <TriangleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={t('backups.physical.monitoring.banner.refreshFailed')}
        />
      )}

      {figures && (
        <MonitoringFiguresComponent
          figures={figures}
          keepingUpSentences={getKeepingUpSentences(statistics.keepingUp)}
          isReducedMotion={isReducedMotion}
        />
      )}

      <RestoreTimelineComponent ranges={statistics.restorableRanges} now={now} />

      <MemoizedWalStreamKeepingUpChartComponent
        series={statistics.series}
        period={period}
        onPeriodChange={changePeriod}
        isAnimationActive={isChartAnimationActive}
      />

      <TechnicalDetailsComponent
        statistics={statistics}
        stageTones={getStageTones(statistics.health.level, statistics.health.reasons)}
        tips={getTipTexts(statistics.health.notices)}
        isReducedMotion={isReducedMotion}
        isChartAnimationActive={isChartAnimationActive}
      />
    </MonitoringContainerComponent>
  );
};
