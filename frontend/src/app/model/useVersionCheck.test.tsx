// @vitest-environment happy-dom
import { act, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

let useVersionCheck: typeof import('./useVersionCheck').useVersionCheck;
const server = setUpMockApi();
let clockTime = 0;
const advanceCheck = async (elapsed: number) => {
  clockTime += elapsed;
  const intervals = vi
    .mocked(globalThis.setInterval)
    .mock.calls.filter((call) => call[1] === 300_000);
  const tick = intervals[intervals.length - 1]?.[0];
  if (typeof tick !== 'function') throw new Error('Version polling interval was not installed');
  await act(async () => {
    tick();
  });
};
function VersionLifecycle() {
  useVersionCheck();
  return null;
}
beforeEach(async () => {
  vi.stubEnv('VITE_APP_VERSION', '1.0.0');
  vi.resetModules();
  ({ useVersionCheck } = await import('./useVersionCheck'));
  clockTime = new Date('2026-10-31T00:00:00Z').getTime();
  vi.spyOn(Date, 'now').mockImplementation(() => clockTime);
  vi.spyOn(globalThis, 'setInterval');
  vi.spyOn(globalThis, 'clearInterval');
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
});

describe('useVersionCheck', () => {
  it.each(['1.0.0', ''])('keeps the interface when the returned version is %s', async (version) => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    server.use(mockApi.get('/system/version', 200, { version }));

    renderScreen(<VersionLifecycle />);
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));
    await advanceCheck(300_000);
    await waitFor(() => expect(server.getRequests()).toHaveLength(2));

    expect(reload).not.toHaveBeenCalled();
  });

  it('starts no requests or intervals when the build is a development version', async () => {
    vi.stubEnv('VITE_APP_VERSION', 'dev');
    vi.resetModules();
    ({ useVersionCheck } = await import('./useVersionCheck'));

    renderScreen(<VersionLifecycle />);
    expect(server.getRequests()).toHaveLength(0);
    expect(
      vi.mocked(globalThis.setInterval).mock.calls.filter((call) => call[1] === 300_000),
    ).toHaveLength(0);
  });

  it('honors the persisted reload cooldown when a different version is returned repeatedly', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    server.use(mockApi.get('/system/version', 200, { version: '2.0.0' }));

    const firstScreen = renderScreen(<VersionLifecycle />);
    await waitFor(() => expect(reload).toHaveBeenCalledOnce());
    firstScreen.unmount();

    const nextVersion = mockApi.delayed({
      method: 'get',
      path: '/system/version',
      status: 200,
      body: { version: '2.0.0' },
    });
    server.use(nextVersion.handler);
    renderScreen(<VersionLifecycle />);
    await waitFor(() => expect(server.getRequests()).toHaveLength(2));

    const processed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );
    await act(async () => {
      nextVersion.release();
      await processed;
    });

    expect(reload).toHaveBeenCalledOnce();
    expect(localStorage.getItem('lastVersionReload')).toBe(String(clockTime));
    server.use(mockApi.get('/system/version', 200, { version: '2.0.0' }));
    await advanceCheck(300_000);
    await waitFor(() => expect(reload).toHaveBeenCalledTimes(2));
  });

  it('starts no overlapping poll and suppresses a mismatch after cleanup with a pending response', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const version = mockApi.delayed({
      method: 'get',
      path: '/system/version',
      status: 200,
      body: { version: '2.0.0' },
    });
    server.use(version.handler);

    const view = renderScreen(<VersionLifecycle />);
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));
    await advanceCheck(900_000);
    expect(server.getRequests()).toHaveLength(1);
    view.unmount();

    expect(globalThis.clearInterval).toHaveBeenCalledWith(
      vi.mocked(globalThis.setInterval).mock.results.slice(-1)[0]?.value,
    );

    const responded = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );

    await act(async () => {
      version.release();
      await responded;
    });

    expect(reload).not.toHaveBeenCalled();
  });

  it('restarts successfully after setup cleanup and setup with a failed first check', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    server.use(mockApi.network('get', '/system/version'));

    const view = renderScreen(<VersionLifecycle />, { isStrictMode: true });
    await waitFor(() => expect(server.getRequests()).toHaveLength(2));

    server.use(mockApi.get('/system/version', 200, { version: '2.0.0' }));
    await advanceCheck(300_000);
    await waitFor(() => expect(reload).toHaveBeenCalledOnce());

    view.unmount();
    expect(globalThis.clearInterval).toHaveBeenCalledWith(
      vi.mocked(globalThis.setInterval).mock.results.slice(-1)[0]?.value,
    );
  });
});
