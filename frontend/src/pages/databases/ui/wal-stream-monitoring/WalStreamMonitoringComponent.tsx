import { Alert, App, Skeleton } from 'antd';
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
import { physicalBackupsApi } from '@/pages/databases/api/physicalBackupsApi';
import { physicalWalStreamStatisticsApi } from '@/pages/databases/api/physicalWalStreamStatisticsApi';
import { getFullBackupAction } from '@/pages/databases/lib/wal-stream-monitoring/fullBackupAction';
import { FullBackupRequestController } from '@/pages/databases/lib/wal-stream-monitoring/fullBackupRequestController';
import { getKeepingUpSentences } from '@/pages/databases/lib/wal-stream-monitoring/keepingUpSentences';
import { applyLiveValues } from '@/pages/databases/lib/wal-stream-monitoring/liveOverlay';
import { getMonitoringFigures } from '@/pages/databases/lib/wal-stream-monitoring/monitoringFigures';
import { getStageTones } from '@/pages/databases/lib/wal-stream-monitoring/pathStages';
import { getTipTexts } from '@/pages/databases/lib/wal-stream-monitoring/reasonTexts';
import { getVerdictBanner } from '@/pages/databases/lib/wal-stream-monitoring/verdictBanner';
import { WalStreamLiveController } from '@/pages/databases/lib/wal-stream-monitoring/walStreamLiveController';
import {
  type PageVisibility,
  type RefreshTimer,
  WalStreamRefreshController,
} from '@/pages/databases/lib/wal-stream-monitoring/walStreamRefreshController';
import { type WalStreamLive } from '@/pages/databases/model/wal-stream-monitoring/WalStreamLive';
import type { WalStreamRefreshState } from '@/pages/databases/model/wal-stream-monitoring/WalStreamRefreshState';
import { WalStreamStatisticsPeriod } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriod';
import { MonitoringFiguresComponent } from '@/pages/databases/ui/wal-stream-monitoring/MonitoringFiguresComponent';
import { RestoreTimelineComponent } from '@/pages/databases/ui/wal-stream-monitoring/RestoreTimelineComponent';
import { TechnicalDetailsComponent } from '@/pages/databases/ui/wal-stream-monitoring/TechnicalDetailsComponent';
import { useCurrentTime } from '@/pages/databases/ui/wal-stream-monitoring/useCurrentTime';
import { usePrefersReducedMotion } from '@/pages/databases/ui/wal-stream-monitoring/usePrefersReducedMotion';
import { VerdictBannerComponent } from '@/pages/databases/ui/wal-stream-monitoring/VerdictBannerComponent';
import { WalStreamKeepingUpChartComponent } from '@/pages/databases/ui/wal-stream-monitoring/WalStreamChartsComponent';
import { translateApiError } from '@/shared/i18n';
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
      <Skeleton active title paragraph={{ rows: 1 }} />
    </div>
    <div className="h-36 rounded-lg border border-border p-4">
      <Skeleton active paragraph={{ rows: 3 }} />
    </div>
    <div className="h-[760px] rounded-lg border border-border p-4">
      <Skeleton active paragraph={{ rows: 12 }} />
    </div>
  </div>
);

export const WalStreamMonitoringComponent = ({ database, isCanManageDBs, onOpenConfig }: Props) => {
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();

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

  const reportFullBackupRequested = useEffectEvent(() => {
    message.success(i18n.t('backups.physical.monitoring.banner.fullBackupRequested'));
  });
  const reportFullBackupError = useEffectEvent((error: unknown) => {
    message.error(translateApiError(error, i18n.t));
  });

  useEffect(() => {
    const requestController = new FullBackupRequestController(
      {
        requestFullBackup: () => physicalBackupsApi.triggerPhysicalBackup(database.id, 'FULL'),
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
          type="warning"
          showIcon
          message={t('backups.physical.monitoring.banner.refreshFailed')}
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
