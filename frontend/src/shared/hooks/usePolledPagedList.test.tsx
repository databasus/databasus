// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { PolledPagedListScreen } from '@/shared/hooks/pagedListScreen.fixture';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();

const setPageVisibility = (visibility: DocumentVisibilityState) => {
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue(visibility);
  fireEvent(document, new Event('visibilitychange'));
};

const workspace = (name: string) => ({
  id: name,
  name,
  currentUserRole: 'WORKSPACE_OWNER' as const,
  createdAt: '2026-10-01T00:00:00Z',
});

const renderReadyList = async (props: ComponentProps<typeof PolledPagedListScreen> = {}) => {
  server.use(mockApi.get('/workspaces', 200, { items: [] }));
  vi.useFakeTimers();
  const pollingStartedAt = Date.now();

  const view = renderScreen(<PolledPagedListScreen {...props} />);
  await waitForCondition(() => expect(screen.getByText('Ready:Idle')).toBeTruthy());

  return { ...view, pollingStartedAt };
};

const finishResponse = async (release: () => void) => {
  const responseProcessed = new Promise<void>((resolve) => {
    const finish = () => {
      server.events.removeListener('response:mocked', finish);
      resolve();
    };
    server.events.on('response:mocked', finish);
  });

  await act(async () => {
    release();
    await responseProcessed;
  });
};

describe('usePolledPagedList', () => {
  it('stops polling while the page is hidden', async () => {
    await renderReadyList();
    setPageVisibility('hidden');

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3_000);
    });

    expect(server.getRequests()).toHaveLength(1);
    expect(screen.getByText('Ready:Idle')).toBeTruthy();
  });

  it('refreshes at once when the page becomes visible', async () => {
    const { pollingStartedAt } = await renderReadyList();
    const nextTickAt = pollingStartedAt + 1_000;
    let refreshRequestedAt = 0;
    server.events.on('request:start', () => {
      refreshRequestedAt = Date.now();
    });
    setPageVisibility('hidden');
    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });

    setPageVisibility('visible');
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));

    expect(refreshRequestedAt).toBeGreaterThan(0);
    expect(refreshRequestedAt).toBeLessThan(nextTickAt);

    await advanceFakeClock(1_000);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(3));
  });

  it('loads the initial page when mounted hidden and waits for it before refreshing', async () => {
    setPageVisibility('hidden');
    const initialPage = mockApi.delayed({
      method: 'get',
      path: '/workspaces',
      status: 200,
      body: { items: [workspace('Initial')] },
    });
    server.use(initialPage.handler);
    vi.useFakeTimers();

    renderScreen(<PolledPagedListScreen />);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(1));

    await advanceFakeClock(3_000);
    setPageVisibility('visible');
    await advanceFakeClock(1_000);

    expect(server.getRequests()).toHaveLength(1);

    setPageVisibility('hidden');
    await act(async () => initialPage.release());
    await waitForCondition(() => expect(screen.getByText('Initial')).toBeTruthy());
    await advanceFakeClock(2_000);

    expect(server.getRequests()).toHaveLength(1);

    server.use(mockApi.get('/workspaces', 200, { items: [workspace('Refreshed')] }));
    setPageVisibility('visible');
    await waitForCondition(() => expect(screen.getByText('Refreshed')).toBeTruthy());
  });

  it('requires an explicit retry when the first page fails before a visibility change', async () => {
    server.use(mockApi.get('/workspaces', 500, buildProblem('internal', 500)));
    const onError = vi.fn();
    vi.useFakeTimers();

    renderScreen(<PolledPagedListScreen onError={onError} />);
    await waitForCondition(() => expect(onError).toHaveBeenCalledOnce());

    setPageVisibility('hidden');
    setPageVisibility('visible');
    await advanceFakeClock(3_000);

    expect(server.getRequests()).toHaveLength(1);

    server.use(mockApi.get('/workspaces', 200, { items: [workspace('Recovered')] }));
    fireEvent.click(screen.getByRole('button', { name: 'Reload' }));
    await waitForCondition(() => expect(screen.getByText('Recovered')).toBeTruthy());

    expect(onError).toHaveBeenCalledOnce();
  });

  it.each(['Refresh', 'Reload'])(
    'accepts an explicit %s while the page is hidden',
    async (action) => {
      await renderReadyList();
      setPageVisibility('hidden');
      server.use(mockApi.get('/workspaces', 200, { items: [workspace('Updated')] }));

      fireEvent.click(screen.getByRole('button', { name: action }));
      await waitForCondition(() => expect(screen.getByText('Updated')).toBeTruthy());
      await advanceFakeClock(3_000);

      expect(server.getRequests()).toHaveLength(2);
    },
  );

  it('keeps a pending refresh when the page is hidden and shown again', async () => {
    await renderReadyList();
    const pendingRefresh = mockApi.delayed({
      method: 'get',
      path: '/workspaces',
      status: 200,
      body: { items: [workspace('Refreshed')] },
    });
    server.use(pendingRefresh.handler);
    await advanceFakeClock(1_000);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));

    setPageVisibility('hidden');
    await advanceFakeClock(3_000);
    setPageVisibility('visible');
    await advanceFakeClock(1_000);

    expect(server.getRequests()).toHaveLength(2);

    await act(async () => pendingRefresh.release());
    await waitForCondition(() => expect(screen.getByText('Refreshed')).toBeTruthy());
    server.use(mockApi.get('/workspaces', 200, { items: [workspace('Next')] }));
    await advanceFakeClock(1_000);
    await waitForCondition(() => expect(screen.getByText('Next')).toBeTruthy());

    expect(server.getRequests()).toHaveLength(3);
  });

  it.each(['success', 'error'] as const)(
    'ignores a pending refresh %s after reset and uses the latest options on return',
    async (outcome) => {
      const onError = vi.fn();
      const view = await renderReadyList({ pageSize: 10, onError });
      const staleRefresh =
        outcome === 'success'
          ? mockApi.delayed({
              method: 'get',
              path: '/workspaces',
              status: 200,
              body: { items: [workspace('Stale')] },
            })
          : mockApi.delayed({
              method: 'get',
              path: '/workspaces',
              status: 500,
              body: buildProblem('internal', 500),
            });
      server.use(staleRefresh.handler);
      await advanceFakeClock(1_000);
      await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));
      setPageVisibility('hidden');

      server.use(mockApi.get('/workspaces', 200, { items: [workspace('Current')] }));
      view.rerender(<PolledPagedListScreen resetKey="second" pageSize={20} onError={onError} />);
      await waitForCondition(() => expect(screen.getByText('Current')).toBeTruthy());
      await finishResponse(staleRefresh.release);

      expect(screen.queryByText('Stale')).toBeNull();
      expect(screen.getByText('Current')).toBeTruthy();
      expect(onError).not.toHaveBeenCalled();

      view.rerender(<PolledPagedListScreen resetKey="second" pageSize={30} onError={onError} />);
      server.use(mockApi.get('/workspaces', 200, { items: [workspace('Latest')] }));
      setPageVisibility('visible');
      await waitForCondition(() => expect(screen.getByText('Latest')).toBeTruthy());

      expect(server.getRequests()).toHaveLength(4);
      expect(new URL(server.getRequests()[3].url).searchParams.get('limit')).toBe('30');
    },
  );

  it('replaces its timer and visibility listener when the polling interval changes', async () => {
    const addListener = vi.spyOn(document, 'addEventListener');
    const removeListener = vi.spyOn(document, 'removeEventListener');
    const view = await renderReadyList();
    const initialVisibilityListener = addListener.mock.calls.find(
      ([event]) => event === 'visibilitychange',
    )![1];
    addListener.mockClear();

    view.rerender(<PolledPagedListScreen refreshIntervalMs={5_000} />);

    expect(removeListener).toHaveBeenCalledWith('visibilitychange', initialVisibilityListener);
    expect(addListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));

    const updatedVisibilityListener = addListener.mock.calls.find(
      ([event]) => event === 'visibilitychange',
    )![1];

    expect(vi.getTimerCount()).toBe(1);

    await advanceFakeClock(1_000);

    expect(server.getRequests()).toHaveLength(1);

    setPageVisibility('hidden');
    setPageVisibility('visible');
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));
    await advanceFakeClock(5_000);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(3));

    view.unmount();

    expect(removeListener).toHaveBeenCalledWith('visibilitychange', updatedVisibilityListener);
    expect(vi.getTimerCount()).toBe(0);
  });

  it.each(['success', 'error'] as const)(
    'sends no requests or errors after unmount with a pending refresh %s',
    async (outcome) => {
      const onError = vi.fn();
      const view = await renderReadyList({ onError });
      const pendingRefresh =
        outcome === 'success'
          ? mockApi.delayed({
              method: 'get',
              path: '/workspaces',
              status: 200,
              body: { items: [workspace('Late')] },
            })
          : mockApi.delayed({
              method: 'get',
              path: '/workspaces',
              status: 500,
              body: buildProblem('internal', 500),
            });
      server.use(pendingRefresh.handler);
      await advanceFakeClock(1_000);
      await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));
      const removeListener = vi.spyOn(document, 'removeEventListener');

      view.unmount();
      await finishResponse(pendingRefresh.release);
      setPageVisibility('hidden');
      setPageVisibility('visible');
      await advanceFakeClock(3_000);

      expect(onError).not.toHaveBeenCalled();
      expect(removeListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
      expect(vi.getTimerCount()).toBe(0);
      expect(server.getRequests()).toHaveLength(2);
    },
  );

  it('keeps one polling lifecycle when Strict Mode replays setup', async () => {
    const abandonedPage = mockApi.pending('get', '/workspaces');
    let initialRequestCount = 0;
    server.use(
      http.get('*/api/v1/workspaces', () => {
        initialRequestCount += 1;
        return initialRequestCount === 1
          ? abandonedPage.response
          : HttpResponse.json({ items: [workspace('Active')] });
      }),
    );
    const onError = vi.fn();
    const addListener = vi.spyOn(document, 'addEventListener');
    const removeListener = vi.spyOn(document, 'removeEventListener');
    vi.useFakeTimers();

    const view = renderScreen(<PolledPagedListScreen onError={onError} />, { isStrictMode: true });
    await waitForCondition(() => expect(screen.getByText('Active')).toBeTruthy());

    expect(initialRequestCount).toBe(2);
    expect(vi.getTimerCount()).toBe(1);

    const visibilityListeners = addListener.mock.calls.filter(
      ([event]) => event === 'visibilitychange',
    );

    expect(visibilityListeners).toHaveLength(2);
    expect(removeListener).toHaveBeenCalledWith('visibilitychange', visibilityListeners[0][1]);

    await finishResponse(() =>
      abandonedPage.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
    );

    expect(onError).not.toHaveBeenCalled();

    setPageVisibility('hidden');
    await advanceFakeClock(2_000);

    expect(initialRequestCount).toBe(2);

    setPageVisibility('visible');
    await waitForCondition(() => expect(initialRequestCount).toBe(3));
    await advanceFakeClock(1_000);
    await waitForCondition(() => expect(initialRequestCount).toBe(4));

    view.unmount();

    expect(removeListener).toHaveBeenCalledWith('visibilitychange', visibilityListeners[1][1]);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('waits for the initial page before polling and stops its timer on unmount', async () => {
    const firstPage = mockApi.delayed({
      method: 'get',
      path: '/workspaces',
      status: 200,
      body: { items: [], nextCursor: 'next' },
    });
    server.use(firstPage.handler);
    vi.useFakeTimers();

    const controlledView = renderScreen(<PolledPagedListScreen />);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(1));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3_000);
    });

    expect(server.getRequests()).toHaveLength(1);

    await act(async () => {
      firstPage.release();
    });
    await waitForCondition(() => expect(screen.getByText('Ready:Idle')).toBeTruthy());

    server.use(mockApi.get('/workspaces', 200, { items: [] }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));

    const removeListener = vi.spyOn(screen.getByTestId('scroll-list'), 'removeEventListener');
    controlledView.unmount();
    expect(removeListener).toHaveBeenCalledWith('scroll', expect.any(Function));
    expect(vi.getTimerCount()).toBe(0);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5_000);
    });

    expect(server.getRequests()).toHaveLength(2);

    fireEvent.scroll(document.body);
  });
});
