// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { storageUsageApi } from '@/pages/storages/api/storageUsageApi';
import { StorageBackupUsage } from '@/pages/storages/ui/StorageBackupUsage';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { advanceClockUntil } from '@/shared/lib/test/advanceClockUntil';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const renderUsage = (isStrictMode = false) =>
  renderScreen(<StorageBackupUsage storageId="archive-a" workspaceId="workspace-a" />, {
    isStrictMode,
  });

const LocaleControl = () => {
  const { setLocale } = useLocale();

  return <button onClick={() => setLocale('ru')}>Switch language</button>;
};

const observeUsageSubscriptions = () => {
  const addListener = vi.spyOn(document, 'addEventListener');
  const removeListener = vi.spyOn(document, 'removeEventListener');

  return {
    removeListener,
    getSubscriptions: () =>
      addListener.mock.calls
        .filter(([event]) => event === 'visibilitychange')
        .map(([, listener]) => ({ listener })),
  };
};

describe('StorageBackupUsage', () => {
  it('keeps the next scheduled tick when a polling response finishes between ticks', async () => {
    const usageRead = vi.spyOn(storageUsageApi, 'getUsage');
    server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 104 }));
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });

    renderUsage();
    await screen.findByText('Total usage: 104 MB');

    const slowRead = mockApi.pending('get', '/storages/{storageId}/backup-usage');
    server.use(slowRead.handler);

    await advanceFakeClock(1000);

    expect(usageRead).toHaveBeenCalledTimes(2);

    await advanceFakeClock(500);
    await releaseAndSettle(usageRead, () =>
      slowRead.respond(HttpResponse.json({ totalUsageMb: 200 })),
    );

    server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 200 }));

    await advanceFakeClock(500);

    expect(usageRead).toHaveBeenCalledTimes(3);
  });

  it('updates totals every second without reloading when catalog contents change', async () => {
    let totalUsageMb = 104.4;
    server.use(
      http.get('*/api/v1/storages/:storageId/backup-usage', () =>
        HttpResponse.json({ totalUsageMb }),
      ),
    );
    vi.useFakeTimers();
    renderUsage();
    await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());

    totalUsageMb = 10752;
    await advanceClockUntil(() => expect(screen.getByText('Total usage: 10.5 GB')).toBeTruthy());

    totalUsageMb = 0;
    await advanceClockUntil(() => expect(screen.getByText('Total usage: 0 MB')).toBeTruthy());
  });

  it.each(
    (['initial', 'background'] as const).flatMap((phase) =>
      ([403, 500] as const).map((status) => ({ phase, status })),
    ),
  )(
    'waits for explicit retry after a failed $phase read with HTTP $status',
    async ({ phase, status }) => {
      let hasFailed = phase === 'initial';
      let totalUsageMb = 104;
      let requestCount = 0;
      server.use(
        http.get('*/api/v1/storages/:storageId/backup-usage', () => {
          requestCount++;

          return hasFailed
            ? HttpResponse.json(
                buildProblem(status === 403 ? 'permission_denied' : 'internal', status),
                { status },
              )
            : HttpResponse.json({ totalUsageMb });
        }),
      );
      vi.useFakeTimers();
      renderUsage();

      if (phase === 'background') {
        await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());

        hasFailed = true;
        await advanceFakeClock(1000);
      }

      await waitForCondition(() =>
        expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
      );
      expect(screen.queryByText('Total usage: 0 MB')).toBeNull();

      if (phase === 'background' && status === 500) {
        expect(screen.getByText('Total usage: 104 MB')).toBeTruthy();
      } else {
        expect(screen.queryByText('Total usage: 104 MB')).toBeNull();
      }

      const requestsBeforeVisibility = requestCount;

      const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
      fireEvent(document, new Event('visibilitychange'));
      await advanceFakeClock(3000);
      expect(requestCount).toBe(requestsBeforeVisibility);

      visibility.mockReturnValue('visible');
      fireEvent(document, new Event('visibilitychange'));
      await advanceFakeClock(3000);
      expect(requestCount).toBe(requestsBeforeVisibility);

      hasFailed = false;
      totalUsageMb = 200;
      fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
      await waitForCondition(() => expect(screen.getByText('Total usage: 200 MB')).toBeTruthy());
      await advanceClockUntil(() =>
        expect(requestCount).toBeGreaterThan(requestsBeforeVisibility + 1),
      );
    },
  );

  it.each(
    ([403, 404] as const).flatMap((status) =>
      (['HTTP 500', 'network error'] as const).map((retryFailure) => ({ status, retryFailure })),
    ),
  )(
    'keeps totals hidden when retry fails with $retryFailure after an HTTP $status refusal',
    async ({ status, retryFailure }) => {
      server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 104 }));
      const usageRead = vi.spyOn(storageUsageApi, 'getUsage');
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });

      renderUsage();
      await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());

      server.use(
        mockApi.get(
          '/storages/{storageId}/backup-usage',
          status,
          buildProblem(status === 403 ? 'permission_denied' : 'not_found', status),
        ),
      );

      await advanceFakeClock(1000);
      await waitForCondition(() =>
        expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
      );

      expect(screen.queryByText('Total usage: 104 MB')).toBeNull();

      const retryRead = mockApi.pending('get', '/storages/{storageId}/backup-usage');
      server.use(retryRead.handler);

      fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
      await waitForCondition(() => expect(usageRead).toHaveBeenCalledTimes(3));

      expect(screen.queryByText('Total usage: 104 MB')).toBeNull();

      await releaseAndSettle(usageRead, () =>
        retryRead.respond(
          retryFailure === 'network error'
            ? HttpResponse.error()
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
        ),
      );

      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy();
      expect(screen.queryByText('Total usage: 104 MB')).toBeNull();

      await advanceFakeClock(3000);

      expect(usageRead).toHaveBeenCalledTimes(3);
      expect(screen.queryByText('Total usage: 104 MB')).toBeNull();

      server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 104 }));

      fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
      await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());
      await advanceClockUntil(() => expect(usageRead.mock.calls.length).toBeGreaterThan(4));
    },
  );

  it('keeps the last total with feedback after a transient refresh failure', async () => {
    let hasFailed = false;
    server.use(
      http.get('*/api/v1/storages/:storageId/backup-usage', () =>
        hasFailed
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json({ totalUsageMb: 104 }),
      ),
    );
    vi.useFakeTimers();
    renderUsage();
    await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());

    hasFailed = true;
    await advanceClockUntil(() =>
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
    );
    expect(screen.getByText('Total usage: 104 MB')).toBeTruthy();
    expect(screen.getByText('Could not refresh backup usage.')).toBeTruthy();
  });

  it('resumes usage reads when a hidden page becomes visible', async () => {
    let totalUsageMb = 104;
    let requestCount = 0;
    server.use(
      http.get('*/api/v1/storages/:storageId/backup-usage', () => {
        requestCount++;
        return HttpResponse.json({ totalUsageMb });
      }),
    );
    vi.useFakeTimers();
    renderUsage();
    await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());

    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    totalUsageMb = 200;

    await advanceFakeClock(3000);
    expect(requestCount).toBe(1);

    visibility.mockReturnValue('visible');
    fireEvent(document, new Event('visibilitychange'));

    await waitForCondition(() => expect(screen.getByText('Total usage: 200 MB')).toBeTruthy());
    expect(requestCount).toBe(2);
  });

  it('keeps one request in flight when visibility events overlap polling ticks', async () => {
    const pendingUsage = mockApi.pending('get', '/storages/{storageId}/backup-usage');
    server.use(pendingUsage.handler);
    const usageRead = vi.spyOn(storageUsageApi, 'getUsage');
    vi.useFakeTimers();
    renderUsage();
    await waitForCondition(() => expect(usageRead).toHaveBeenCalledTimes(1));

    fireEvent(document, new Event('visibilitychange'));
    await advanceFakeClock(3000);
    expect(usageRead).toHaveBeenCalledTimes(1);

    await releaseAndSettle(usageRead, () =>
      pendingUsage.respond(HttpResponse.json({ totalUsageMb: 104 })),
    );
    expect(screen.getByText('Total usage: 104 MB')).toBeTruthy();
  });

  it.each(
    ['storage', 'workspace', 'session'].flatMap((context) =>
      ['success', 'error'].map((outcome) => ({ context, outcome })),
    ),
  )('ignores an earlier $outcome when the $context changes', async ({ context, outcome }) => {
    const pendingUsage = mockApi.pending('get', '/storages/{storageId}/backup-usage');
    server.use(pendingUsage.handler);
    const usageRead = vi.spyOn(storageUsageApi, 'getUsage');
    const view = renderUsage();
    await waitFor(() => expect(usageRead).toHaveBeenCalledTimes(1));

    server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 200 }));

    if (context === 'session') {
      act(() => activateSession({ token: 'another-session', userId: 'another-user' }));
    } else {
      view.rerender(
        <StorageBackupUsage
          storageId={context === 'storage' ? 'archive-b' : 'archive-a'}
          workspaceId={context === 'workspace' ? 'workspace-b' : 'workspace-a'}
        />,
      );
    }

    await screen.findByText('Total usage: 200 MB');

    await releaseAndSettle(
      usageRead,
      () =>
        pendingUsage.respond(
          outcome === 'success'
            ? HttpResponse.json({ totalUsageMb: 999 })
            : HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
        ),
      0,
    );

    expect(screen.getByText('Total usage: 200 MB')).toBeTruthy();
    expect(screen.queryByText('Total usage: 999 MB')).toBeNull();
  });

  it('stops reads and feedback after unmount with an outstanding request', async () => {
    const pendingUsage = mockApi.pending('get', '/storages/{storageId}/backup-usage');
    server.use(pendingUsage.handler);
    const usageRead = vi.spyOn(storageUsageApi, 'getUsage');
    vi.useFakeTimers();
    const subscriptions = observeUsageSubscriptions();
    const view = renderUsage();
    await waitForCondition(() => expect(usageRead).toHaveBeenCalledTimes(1));

    const [subscription] = subscriptions.getSubscriptions();

    view.unmount();
    expect(subscriptions.removeListener).toHaveBeenCalledWith(
      'visibilitychange',
      subscription.listener,
    );

    await releaseAndSettle(usageRead, () =>
      pendingUsage.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
    );
    fireEvent(document, new Event('visibilitychange'));
    await advanceFakeClock(3000);

    expect(usageRead).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('loads normally after setup-cleanup-setup', async () => {
    server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 104 }));
    vi.useFakeTimers();
    const subscriptions = observeUsageSubscriptions();
    const usageRead = vi.spyOn(storageUsageApi, 'getUsage');
    const view = renderUsage(true);
    await waitForCondition(() => expect(screen.getByText('Total usage: 104 MB')).toBeTruthy());

    const registered = subscriptions.getSubscriptions();
    expect(registered).toHaveLength(2);

    const [earlier, current] = registered;

    expect(subscriptions.removeListener).toHaveBeenCalledWith('visibilitychange', earlier.listener);
    expect(subscriptions.removeListener).not.toHaveBeenCalledWith(
      'visibilitychange',
      current.listener,
    );

    view.unmount();

    expect(subscriptions.removeListener).toHaveBeenCalledWith('visibilitychange', current.listener);

    const readsBeforeClock = usageRead.mock.calls.length;

    await advanceFakeClock(3000);

    expect(usageRead).toHaveBeenCalledTimes(readsBeforeClock);
  });

  it('changes labels and compact decimal separators when the interface language changes', async () => {
    server.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 10752 }));
    const view = renderScreen(
      <>
        <LocaleControl />
        <StorageBackupUsage storageId="archive-a" workspaceId="workspace-a" />
      </>,
    );
    await screen.findByText('Total usage: 10.5 GB');

    await view.user.click(screen.getByRole('button', { name: 'Switch language' }));

    expect(
      await screen.findByText(
        DICTIONARIES.ru.backups.list.totalUsage.replace('{{size}}', '10,5 GB'),
      ),
    ).toBeTruthy();
  });
});
