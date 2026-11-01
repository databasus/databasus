import { vi } from 'vitest';

// A notice closes itself by a timer that outlives its screen unless the notice clears it on
// unmount, and a timer firing after the test environment is gone fails the whole run.
export const watchNoticeTimer = (durationMs: number) => {
  const startedTimers = vi.spyOn(globalThis, 'setTimeout');
  const clearedTimers = vi.spyOn(globalThis, 'clearTimeout');

  const getNoticeTimers = () =>
    startedTimers.mock.calls.flatMap(([, delay], callIndex) =>
      delay === durationMs ? [startedTimers.mock.results[callIndex].value] : [],
    );

  return {
    getStartedCount: () => getNoticeTimers().length,
    getLeftRunningCount: () =>
      getNoticeTimers().filter(
        (timer) => !clearedTimers.mock.calls.some(([clearedTimer]) => clearedTimer === timer),
      ).length,
  };
};
