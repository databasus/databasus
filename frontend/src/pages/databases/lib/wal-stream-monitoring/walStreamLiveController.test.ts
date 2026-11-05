import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { WalStreamLive } from '../../model/wal-stream-monitoring/WalStreamLive';
import type { WalStreamLiveResponse } from '../../model/wal-stream-monitoring/WalStreamLiveResponse';
import { FakePageVisibility } from './testPageVisibility';
import { WAL_STREAM_LIVE_INTERVAL_MS, WalStreamLiveController } from './walStreamLiveController';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const buildLive = (localQueueBytes: number): WalStreamLive =>
  ({ measuredAt: '2026-10-18T12:00:00Z', localQueueBytes }) as WalStreamLive;

const runningResponse = (localQueueBytes: number): WalStreamLiveResponse => ({
  isStreamerRunning: true,
  current: buildLive(localQueueBytes),
});

const createHarness = (fetchLive: () => Promise<WalStreamLiveResponse>) => {
  const pageVisibility = new FakePageVisibility();
  const onLiveChange = vi.fn();
  const fetchLiveMock = vi.fn(fetchLive);

  const controller = new WalStreamLiveController(
    {
      fetchLive: fetchLiveMock,
      timer: {
        setInterval: (callback, intervalMs) => setInterval(callback, intervalMs),
        clearInterval: (handle) => clearInterval(handle as ReturnType<typeof setInterval>),
      },
      pageVisibility,
    },
    onLiveChange,
  );

  return { controller, fetchLive: fetchLiveMock, onLiveChange, pageVisibility };
};

describe('WalStreamLiveController', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('reads at once and then every second', async () => {
    let localQueueBytes = 0;
    const harness = createHarness(async () => runningResponse(localQueueBytes++));

    harness.controller.start();
    await vi.advanceTimersByTimeAsync(0);
    await vi.advanceTimersByTimeAsync(3 * WAL_STREAM_LIVE_INTERVAL_MS);

    expect(harness.fetchLive).toHaveBeenCalledTimes(4);
    expect(harness.onLiveChange).toHaveBeenLastCalledWith(buildLive(3));

    harness.controller.stop();
  });

  it('reports no values while no streamer runs', async () => {
    const harness = createHarness(async () => ({ isStreamerRunning: false, current: null }));

    harness.controller.start();
    await vi.advanceTimersByTimeAsync(0);

    expect(harness.onLiveChange).toHaveBeenCalledWith(null);

    harness.controller.stop();
  });

  it('does not read while the browser tab is hidden, and reads at once when it shows again', async () => {
    const harness = createHarness(async () => runningResponse(1));

    harness.controller.start();
    await vi.advanceTimersByTimeAsync(0);
    harness.pageVisibility.setVisible(false);
    await vi.advanceTimersByTimeAsync(10 * WAL_STREAM_LIVE_INTERVAL_MS);

    expect(harness.fetchLive).toHaveBeenCalledTimes(1);

    harness.pageVisibility.setVisible(true);
    await vi.advanceTimersByTimeAsync(0);

    expect(harness.fetchLive).toHaveBeenCalledTimes(2);

    harness.controller.stop();
  });

  it('keeps the last values when a reading fails', async () => {
    let isFailing = false;
    const harness = createHarness(async () => {
      if (isFailing) {
        throw new Error('network down');
      }

      return runningResponse(5);
    });

    harness.controller.start();
    await vi.advanceTimersByTimeAsync(0);
    isFailing = true;
    await vi.advanceTimersByTimeAsync(2 * WAL_STREAM_LIVE_INTERVAL_MS);

    expect(harness.onLiveChange).toHaveBeenCalledTimes(1);
    expect(harness.onLiveChange).toHaveBeenLastCalledWith(buildLive(5));

    harness.controller.stop();
  });

  it('skips a tick while the previous reading is still on its way', async () => {
    let resolveSlowReading: (response: WalStreamLiveResponse) => void = () => {};
    const harness = createHarness(
      () =>
        new Promise((resolve) => {
          resolveSlowReading = resolve;
        }),
    );

    harness.controller.start();
    await vi.advanceTimersByTimeAsync(3 * WAL_STREAM_LIVE_INTERVAL_MS);

    expect(harness.fetchLive).toHaveBeenCalledTimes(1);

    resolveSlowReading(runningResponse(1));
    await vi.advanceTimersByTimeAsync(WAL_STREAM_LIVE_INTERVAL_MS);

    expect(harness.fetchLive).toHaveBeenCalledTimes(2);

    harness.controller.stop();
  });

  it('stops reading and ignores a reading that lands after stop', async () => {
    let resolveReading: (response: WalStreamLiveResponse) => void = () => {};
    const harness = createHarness(
      () =>
        new Promise((resolve) => {
          resolveReading = resolve;
        }),
    );

    harness.controller.start();
    harness.controller.stop();
    resolveReading(runningResponse(9));
    await vi.advanceTimersByTimeAsync(5 * WAL_STREAM_LIVE_INTERVAL_MS);

    expect(harness.fetchLive).toHaveBeenCalledTimes(1);
    expect(harness.onLiveChange).not.toHaveBeenCalled();
  });
});
