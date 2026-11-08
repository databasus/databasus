// @vitest-environment happy-dom
import { act, screen, waitFor } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

let useVersionCheck: typeof import('@/app/model/useVersionCheck').useVersionCheck;
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
  ({ useVersionCheck } = await import('@/app/model/useVersionCheck'));
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
  it('keeps the next scheduled tick when a version response finishes between ticks', async () => {
    const { systemApi } = await import('@/entities/system');
    const versionRead = vi.spyOn(systemApi, 'getVersion');
    server.use(mockApi.get('/system/version', 200, { version: '1.0.0' }));
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });

    renderScreen(<VersionLifecycle />);
    await act(async () => {
      await versionRead.mock.results[0].value;
    });

    const slowRead = mockApi.pending('get', '/system/version');
    server.use(slowRead.handler);

    await advanceFakeClock(300_000);

    expect(versionRead).toHaveBeenCalledTimes(2);

    await advanceFakeClock(100_000);
    await releaseAndSettle(versionRead, () =>
      slowRead.respond(HttpResponse.json({ version: '1.0.0' })),
    );

    server.use(mockApi.get('/system/version', 200, { version: '1.0.0' }));

    await advanceFakeClock(200_000);

    expect(versionRead).toHaveBeenCalledTimes(3);
  });

  it.each(['read', 'write'] as const)(
    'keeps the interface usable without reloading when cooldown storage fails during %s',
    async (operation) => {
      const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
      const version = mockApi.pending('get', '/system/version');
      server.use(version.handler);
      const onContinue = vi.fn();

      const view = renderScreen(
        <>
          <VersionLifecycle />
          <button onClick={onContinue}>Continue working</button>
        </>,
      );
      await waitFor(() => expect(server.getRequests()).toHaveLength(1));

      const getItem = localStorage.getItem.bind(localStorage);
      const setItem = localStorage.setItem.bind(localStorage);
      const cooldownStorage =
        operation === 'read'
          ? vi.spyOn(localStorage, 'getItem').mockImplementation((key) => {
              if (key === 'lastVersionReload')
                throw new DOMException('Storage is unavailable', 'SecurityError');

              return getItem(key);
            })
          : vi.spyOn(localStorage, 'setItem').mockImplementation((key, value) => {
              if (key === 'lastVersionReload')
                throw new DOMException('Storage is full', 'QuotaExceededError');

              setItem(key, value);
            });

      try {
        await finishApiResponse(server, '/api/v1/system/version', () =>
          version.respond(HttpResponse.json({ version: '2.0.0' })),
        );

        await view.user.click(screen.getByRole('button', { name: 'Continue working' }));

        expect(onContinue).toHaveBeenCalledOnce();
        expect(reload).not.toHaveBeenCalled();
      } finally {
        cooldownStorage.mockRestore();
      }

      server.use(mockApi.get('/system/version', 200, { version: '2.0.0' }));

      await advanceCheck(300_000);

      await waitFor(() => expect(reload).toHaveBeenCalledOnce());
    },
  );

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
    ({ useVersionCheck } = await import('@/app/model/useVersionCheck'));

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

    await finishCancelledApiResponse(server, '/api/v1/system/version', () => version.release());

    expect(reload).not.toHaveBeenCalled();
  });

  it('restarts successfully after setup cleanup and setup with a failed first check', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    server.use(mockApi.network('get', '/system/version'));

    const view = renderScreen(<VersionLifecycle />, { isStrictMode: true });
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    server.use(mockApi.get('/system/version', 200, { version: '2.0.0' }));
    await advanceCheck(300_000);
    await waitFor(() => expect(reload).toHaveBeenCalledOnce());

    view.unmount();
    expect(globalThis.clearInterval).toHaveBeenCalledWith(
      vi.mocked(globalThis.setInterval).mock.results.slice(-1)[0]?.value,
    );
  });
});
