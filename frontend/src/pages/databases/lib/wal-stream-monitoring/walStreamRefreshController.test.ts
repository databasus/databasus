import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { FakePageVisibility } from '@/pages/databases/lib/wal-stream-monitoring/testPageVisibility';
import {
  WAL_STREAM_REFRESH_INTERVAL_MS,
  WalStreamRefreshController,
} from '@/pages/databases/lib/wal-stream-monitoring/walStreamRefreshController';
import type { WalStreamRefreshState } from '@/pages/databases/model/wal-stream-monitoring/WalStreamRefreshState';
import { type WalStreamStatistics } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatistics';
import { WalStreamStatisticsPeriod } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatisticsPeriod';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const buildStatistics = (marker: string): WalStreamStatistics =>
  ({ series: { from: marker } }) as unknown as WalStreamStatistics;

interface ControllerHarness {
  controller: WalStreamRefreshController;
  fetchStatistics: ReturnType<typeof vi.fn>;
  pageVisibility: FakePageVisibility;
  getLatestState: () => WalStreamRefreshState | undefined;
}

const createHarness = (): ControllerHarness => {
  const fetchStatistics = vi.fn(async (period: WalStreamStatisticsPeriod) =>
    buildStatistics(period),
  );
  const pageVisibility = new FakePageVisibility();
  let latestState: WalStreamRefreshState | undefined;

  const controller = new WalStreamRefreshController(
    {
      fetchStatistics,
      timer: {
        setInterval: (callback, intervalMs) => setInterval(callback, intervalMs),
        clearInterval: (handle) => clearInterval(handle as ReturnType<typeof setInterval>),
      },
      pageVisibility,
      getNow: () => new Date(),
    },
    WalStreamStatisticsPeriod.DAY,
    (state) => {
      latestState = state;
    },
  );

  return { controller, fetchStatistics, pageVisibility, getLatestState: () => latestState };
};

describe('WalStreamRefreshController', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('refreshes on open and every 30 seconds in every period', async () => {
    const { controller, fetchStatistics } = createHarness();

    controller.start();
    expect(fetchStatistics).toHaveBeenCalledTimes(1);
    expect(fetchStatistics).toHaveBeenLastCalledWith(WalStreamStatisticsPeriod.DAY);

    await vi.advanceTimersByTimeAsync(3 * WAL_STREAM_REFRESH_INTERVAL_MS);
    expect(fetchStatistics).toHaveBeenCalledTimes(4);

    controller.stop();
  });

  it('does not refresh while the browser tab is hidden and refreshes at once when it is shown', async () => {
    const { controller, fetchStatistics, pageVisibility } = createHarness();

    controller.start();
    pageVisibility.setVisible(false);
    fetchStatistics.mockClear();

    await vi.advanceTimersByTimeAsync(10 * 60 * 1000);
    expect(fetchStatistics).not.toHaveBeenCalled();

    pageVisibility.setVisible(true);
    expect(fetchStatistics).toHaveBeenCalledTimes(1);

    controller.stop();
  });

  it('refreshes at once with the new period when the period changes', async () => {
    const { controller, fetchStatistics } = createHarness();

    controller.start();
    await vi.advanceTimersByTimeAsync(10_000);
    fetchStatistics.mockClear();

    controller.setPeriod(WalStreamStatisticsPeriod.WEEK);

    expect(fetchStatistics).toHaveBeenCalledTimes(1);
    expect(fetchStatistics).toHaveBeenLastCalledWith(WalStreamStatisticsPeriod.WEEK);

    controller.stop();
  });

  it('keeps the previous response after a failed refresh and still runs the next one', async () => {
    const { controller, fetchStatistics, getLatestState } = createHarness();

    controller.start();
    await vi.advanceTimersByTimeAsync(0);

    const firstStatistics = getLatestState()?.statistics;
    expect(firstStatistics).not.toBeNull();

    fetchStatistics.mockRejectedValueOnce(new Error('network down'));
    await vi.advanceTimersByTimeAsync(WAL_STREAM_REFRESH_INTERVAL_MS);

    expect(getLatestState()?.statistics).toBe(firstStatistics);
    expect(getLatestState()?.isLastRefreshFailed).toBe(true);
    expect(getLatestState()?.firstLoadError).toBeNull();

    await vi.advanceTimersByTimeAsync(WAL_STREAM_REFRESH_INTERVAL_MS);

    expect(fetchStatistics).toHaveBeenCalledTimes(3);
    expect(getLatestState()?.isLastRefreshFailed).toBe(false);

    controller.stop();
  });

  it('reports a failed first load as an error with nothing to show', async () => {
    const { controller, fetchStatistics, getLatestState } = createHarness();
    const loadError = new Error('forbidden');

    fetchStatistics.mockRejectedValueOnce(loadError);
    controller.start();
    await vi.advanceTimersByTimeAsync(0);

    expect(getLatestState()?.statistics).toBeNull();
    expect(getLatestState()?.firstLoadError).toBe(loadError);

    controller.retry();
    await vi.advanceTimersByTimeAsync(0);

    expect(getLatestState()?.statistics).not.toBeNull();
    expect(getLatestState()?.firstLoadError).toBeNull();

    controller.stop();
  });

  it('drops a response to the previous period that arrives after the period changed', async () => {
    const { controller, fetchStatistics, getLatestState } = createHarness();
    let resolveDayResponse: (statistics: WalStreamStatistics) => void = () => {};

    fetchStatistics.mockImplementationOnce(
      () => new Promise<WalStreamStatistics>((resolve) => (resolveDayResponse = resolve)),
    );

    controller.start();
    controller.setPeriod(WalStreamStatisticsPeriod.HOUR);
    await vi.advanceTimersByTimeAsync(0);

    resolveDayResponse(buildStatistics(WalStreamStatisticsPeriod.DAY));
    await vi.advanceTimersByTimeAsync(0);

    expect(getLatestState()?.statistics).toEqual(buildStatistics(WalStreamStatisticsPeriod.HOUR));

    controller.stop();
  });

  it('stops refreshing once stopped', async () => {
    const { controller, fetchStatistics } = createHarness();

    controller.start();
    controller.stop();
    fetchStatistics.mockClear();

    await vi.advanceTimersByTimeAsync(5 * WAL_STREAM_REFRESH_INTERVAL_MS);

    expect(fetchStatistics).not.toHaveBeenCalled();
  });
});
