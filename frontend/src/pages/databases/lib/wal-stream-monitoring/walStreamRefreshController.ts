import type { WalStreamRefreshState } from '../../model/wal-stream-monitoring/WalStreamRefreshState';
import type { WalStreamStatistics } from '../../model/wal-stream-monitoring/WalStreamStatistics';
import type { WalStreamStatisticsPeriod } from '../../model/wal-stream-monitoring/WalStreamStatisticsPeriod';

// Samples arrive every 30 seconds, so refreshing faster shows nothing new and refreshing slower
// lets the screen fall behind the verdict.
export const WAL_STREAM_REFRESH_INTERVAL_MS = 30_000;

export interface RefreshTimer {
  setInterval: (callback: () => void, intervalMs: number) => unknown;
  clearInterval: (handle: unknown) => void;
}

export interface PageVisibility {
  isVisible: () => boolean;

  // Returns the unsubscribe function.
  subscribe: (onVisibilityChange: () => void) => () => void;
}

interface WalStreamRefreshDependencies {
  fetchStatistics: (period: WalStreamStatisticsPeriod) => Promise<WalStreamStatistics>;
  timer: RefreshTimer;
  pageVisibility: PageVisibility;
  getNow: () => Date;
}

// Refreshes on start, every 30 seconds in every period, at once on a period change and at once
// when the browser tab becomes visible again, and never while it is hidden. A response that
// arrives after a newer request started belongs to an older period or moment and is dropped. The
// class holds no React state, so the schedule is testable with fake timers.
export class WalStreamRefreshController {
  private state: WalStreamRefreshState = {
    statistics: null,
    lastSuccessfulRefreshAt: null,
    firstLoadError: null,
    isLastRefreshFailed: false,
  };

  private requestGeneration = 0;
  private intervalHandle: unknown = null;
  private unsubscribeFromVisibility: (() => void) | null = null;

  constructor(
    private readonly dependencies: WalStreamRefreshDependencies,
    private period: WalStreamStatisticsPeriod,
    private readonly onStateChange: (state: WalStreamRefreshState) => void,
  ) {}

  start() {
    this.unsubscribeFromVisibility = this.dependencies.pageVisibility.subscribe(() => {
      if (this.dependencies.pageVisibility.isVisible()) {
        this.refreshNowAndRestartInterval();
      }
    });

    this.refreshNowAndRestartInterval();
  }

  stop() {
    this.requestGeneration += 1;
    this.clearRefreshInterval();
    this.unsubscribeFromVisibility?.();
    this.unsubscribeFromVisibility = null;
  }

  setPeriod(period: WalStreamStatisticsPeriod) {
    this.period = period;
    this.refreshNowAndRestartInterval();
  }

  retry() {
    this.refreshNowAndRestartInterval();
  }

  getState(): WalStreamRefreshState {
    return this.state;
  }

  private refreshNowAndRestartInterval() {
    this.clearRefreshInterval();

    this.intervalHandle = this.dependencies.timer.setInterval(() => {
      if (this.dependencies.pageVisibility.isVisible()) {
        void this.refresh();
      }
    }, WAL_STREAM_REFRESH_INTERVAL_MS);

    void this.refresh();
  }

  private clearRefreshInterval() {
    if (this.intervalHandle !== null) {
      this.dependencies.timer.clearInterval(this.intervalHandle);
      this.intervalHandle = null;
    }
  }

  private async refresh() {
    this.requestGeneration += 1;
    const generation = this.requestGeneration;

    try {
      const statistics = await this.dependencies.fetchStatistics(this.period);
      if (generation !== this.requestGeneration) return;

      this.updateState({
        statistics,
        lastSuccessfulRefreshAt: this.dependencies.getNow(),
        firstLoadError: null,
        isLastRefreshFailed: false,
      });
    } catch (e) {
      if (generation !== this.requestGeneration) return;

      this.updateState({
        firstLoadError: this.state.statistics === null ? e : null,
        isLastRefreshFailed: true,
      });
    }
  }

  private updateState(change: Partial<WalStreamRefreshState>) {
    this.state = { ...this.state, ...change };
    this.onStateChange(this.state);
  }
}
