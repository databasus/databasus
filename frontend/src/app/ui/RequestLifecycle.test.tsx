// @vitest-environment happy-dom
import { focusManager, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { useEffect } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { RequestConfiguration } from '@/app/ui/RequestConfiguration';
import { type Storage } from '@/entities/storages';
import { StoragesComponent } from '@/pages/storages';
import { activateSession, invalidateSession } from '@/shared/auth';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const workspace = {
  id: 'workspace',
  name: 'Workspace',
  createdAt: '2026-01-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_OWNER' as const,
};
const storage: Storage = {
  id: 'archive',
  workspaceId: workspace.id,
  name: 'Archive',
  settings: { type: 'LOCAL' },
};
const StorageScreen = () => <StoragesComponent workspace={workspace} isCanManageStorages />;
const mockDetails = () =>
  server.use(
    mockApi.get('/storages/{storageId}', 200, storage),
    mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 0 }),
  );
const getCollectionReads = () =>
  server.getRequests().filter((request) => new URL(request.url).pathname === '/api/v1/storages');
const processEvent = async (target: Document | Window, name: string) => {
  await act(async () => {
    fireEvent(target, new Event(name));
  });
};

const ClientObserver = ({ onClient }: { onClient: (client: QueryClient) => void }) => {
  const client = useQueryClient();
  useEffect(() => {
    onClient(client);
  }, [client, onClient]);

  return null;
};

describe('Request lifecycle', () => {
  it.each(['focus', 'reconnect'])(
    'starts no ordinary read after %s on a screen without a refresh rule',
    async (event) => {
      server.use(mockApi.get('/storages', 200, { items: [] }));

      renderScreen(<StorageScreen />);
      await screen.findByRole('button', { name: 'Add storage' });

      expect(getCollectionReads()).toHaveLength(1);

      if (event === 'focus') {
        const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
        const focusChanges = vi.fn();
        const unsubscribe = focusManager.subscribe(focusChanges);

        try {
          await processEvent(window, 'visibilitychange');

          expect(focusChanges).toHaveBeenLastCalledWith(false);

          visibility.mockReturnValue('visible');
          await processEvent(window, 'visibilitychange');

          expect(focusChanges).toHaveBeenLastCalledWith(true);
        } finally {
          const isCurrentlyFocused = focusManager.isFocused();
          visibility.mockReturnValue('visible');
          if (!isCurrentlyFocused) await processEvent(window, 'visibilitychange');

          unsubscribe();
        }
      } else {
        await processEvent(window, 'offline');
        await processEvent(window, 'online');
      }

      expect(getCollectionReads()).toHaveLength(1);
      expect(screen.getByRole('button', { name: 'Add storage' })).toBeTruthy();
    },
  );

  it('waits for fresh data after immediate remount before garbage collection', async () => {
    mockDetails();
    server.use(mockApi.get('/storages', 200, { items: [storage] }));

    const view = renderScreen(<StorageScreen key="first-visit" />);
    await screen.findByRole('button', { name: storage.name });

    const currentRead = mockApi.pending('get', '/storages');
    server.use(currentRead.handler);

    view.rerender(<StorageScreen key="next-visit" />);
    await waitFor(() => expect(getCollectionReads()).toHaveLength(2));

    expect(screen.queryByRole('button', { name: storage.name })).toBeNull();
    expect(screen.getByRole('status')).toBeTruthy();

    await finishApiResponse(server, '/api/v1/storages', () =>
      currentRead.respond(
        HttpResponse.json({ items: [{ ...storage, name: 'Changed on server' }] }),
      ),
    );

    expect(screen.getByRole('button', { name: 'Changed on server' })).toBeTruthy();
  });

  it('waits for a fresh sibling read and keeps it alive when another consumer closes', async () => {
    mockDetails();
    server.use(mockApi.get('/storages', 200, { items: [storage] }));

    const view = renderScreen(
      <section aria-label="First screen">
        <StorageScreen />
      </section>,
    );
    await screen.findByRole('button', { name: storage.name });

    const currentRead = mockApi.pending('get', '/storages');
    server.use(currentRead.handler);

    view.rerender(
      <>
        <section aria-label="First screen">
          <StorageScreen />
        </section>
        <section aria-label="Second screen">
          <StorageScreen />
        </section>
      </>,
    );
    await waitFor(() => expect(getCollectionReads()).toHaveLength(2));

    expect(
      within(screen.getByRole('region', { name: 'First screen' })).getByRole('button', {
        name: storage.name,
      }),
    ).toBeTruthy();
    expect(
      within(screen.getByRole('region', { name: 'Second screen' })).queryByRole('button', {
        name: storage.name,
      }),
    ).toBeNull();
    expect(
      within(screen.getByRole('region', { name: 'Second screen' })).getByRole('status'),
    ).toBeTruthy();

    view.rerender(
      <section aria-label="First screen">
        <StorageScreen />
      </section>,
    );

    expect(getCollectionReads()[1].signal.aborted).toBe(false);

    await finishApiResponse(server, '/api/v1/storages', () =>
      currentRead.respond(
        HttpResponse.json({ items: [{ ...storage, name: 'Fresh shared archive' }] }),
      ),
    );

    expect(screen.getByRole('button', { name: 'Fresh shared archive' })).toBeTruthy();
  });

  it('shows no earlier sibling data when the new visit read fails', async () => {
    mockDetails();
    server.use(mockApi.get('/storages', 200, { items: [storage] }));

    const view = renderScreen(
      <section aria-label="First screen">
        <StorageScreen />
      </section>,
    );
    await screen.findByRole('button', { name: storage.name });

    const freshRead = mockApi.pending('get', '/storages');
    server.use(freshRead.handler);

    view.rerender(
      <>
        <section aria-label="First screen">
          <StorageScreen />
        </section>
        <section aria-label="Second screen">
          <StorageScreen />
        </section>
      </>,
    );
    await waitFor(() => expect(getCollectionReads()).toHaveLength(2));

    await finishApiResponse(server, '/api/v1/storages', () =>
      freshRead.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
    );

    const firstScreen = within(screen.getByRole('region', { name: 'First screen' }));
    const secondScreen = within(screen.getByRole('region', { name: 'Second screen' }));
    expect(firstScreen.getByRole('button', { name: storage.name })).toBeTruthy();
    expect(secondScreen.queryByRole('button', { name: storage.name })).toBeNull();
    expect(secondScreen.getByRole('button', { name: 'Retry' })).toBeTruthy();

    server.use(
      mockApi.get('/storages', 200, { items: [{ ...storage, name: 'Fresh after retry' }] }),
    );

    fireEvent.click(secondScreen.getByRole('button', { name: 'Retry' }));
    await secondScreen.findByRole('button', { name: 'Fresh after retry' });

    expect(firstScreen.getByRole('button', { name: 'Fresh after retry' })).toBeTruthy();
  });

  it('starts no next cursor page after the final consumer cancels the read', async () => {
    const firstPage = mockApi.pending('get', '/storages');
    server.use(firstPage.handler);

    const view = renderScreen(<StorageScreen />);
    await waitFor(() => expect(getCollectionReads()).toHaveLength(1));

    view.unmount();
    await finishCancelledApiResponse(server, '/api/v1/storages', () =>
      firstPage.respond(HttpResponse.json({ items: [], nextCursor: 'next-page' })),
    );

    expect(getCollectionReads()).toHaveLength(1);
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('publishes no partial collection and starts no third page after cancellation on page two', async () => {
    const secondPage = mockApi.pending('get', '/storages');
    server.use(
      http.get('*/api/v1/storages', ({ request }) =>
        new URL(request.url).searchParams.has('after')
          ? secondPage.response
          : HttpResponse.json({ items: [storage], nextCursor: 'second-page' }),
      ),
    );

    const view = renderScreen(<StorageScreen />);
    await waitFor(() => expect(getCollectionReads()).toHaveLength(2));

    expect(new URL(getCollectionReads()[1].url).searchParams.get('after')).toBe('second-page');
    expect(screen.queryByRole('button', { name: storage.name })).toBeNull();
    expect(screen.getByRole('status')).toBeTruthy();

    view.unmount();

    expect(getCollectionReads()[1].signal.aborted).toBe(true);

    await finishCancelledApiResponse(server, '/api/v1/storages', () =>
      secondPage.respond(
        HttpResponse.json({
          items: [{ ...storage, id: 'second-storage' }],
          nextCursor: 'third-page',
        }),
      ),
    );

    expect(getCollectionReads()).toHaveLength(2);
    expect(screen.queryByRole('button', { name: storage.name })).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('shows fresh data after a mutation invalidates the list', async () => {
    mockDetails();
    let hasRenamed = false;
    server.use(
      http.get('*/api/v1/storages', () =>
        HttpResponse.json({
          items: hasRenamed
            ? [
                { ...storage, name: 'Renamed archive' },
                { ...storage, id: 'server-created', name: 'Added on server' },
              ]
            : [storage],
        }),
      ),
      http.put('*/api/v1/storages/:storageId', () => {
        hasRenamed = true;
        return HttpResponse.json({ ...storage, name: 'Renamed archive' });
      }),
    );

    const view = renderScreen(<StorageScreen />);
    await screen.findByRole('button', { name: 'Archive' });

    await view.user.click((await screen.findAllByRole('button', { name: 'Edit' }))[0]);
    await view.user.clear(screen.getByPlaceholderText('Enter name...'));
    await view.user.type(screen.getByPlaceholderText('Enter name...'), 'Renamed archive');
    await view.user.click(screen.getByRole('button', { name: 'Save' }));

    await screen.findByRole('button', { name: 'Added on server' });

    expect(screen.getByRole('button', { name: 'Renamed archive' })).toBeTruthy();
    expect(getCollectionReads()).toHaveLength(2);
  });

  it('attempts an offline action once and replays nothing after reconnect', async () => {
    mockDetails();
    server.use(
      mockApi.get('/storages', 200, { items: [storage] }),
      mockApi.post('/storages/{storageId}/test', 500, buildProblem('internal', 500)),
    );

    const view = renderScreen(<StorageScreen />);
    await screen.findByRole('button', { name: 'Test connection' });

    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false);
    await processEvent(window, 'offline');

    await view.user.click(screen.getByRole('button', { name: 'Test connection' }));
    await screen.findByRole('alert');

    const getTestReads = () => server.getRequests().filter((request) => request.method === 'POST');
    expect(getTestReads()).toHaveLength(1);

    await processEvent(window, 'online');

    expect(getTestReads()).toHaveLength(1);
  });

  it.each([500, 403])(
    'keeps a save acknowledged when its refresh fails with HTTP %s',
    async (status) => {
      mockDetails();
      const onUnavailable = vi.fn();
      let hasRenamed = false;
      server.use(
        http.get('*/api/v1/storages', () =>
          hasRenamed
            ? HttpResponse.json(
                buildProblem(status === 403 ? 'permission_denied' : 'internal', status),
                { status },
              )
            : HttpResponse.json({ items: [storage] }),
        ),
        http.put('*/api/v1/storages/:storageId', () => {
          hasRenamed = true;
          return HttpResponse.json({ ...storage, name: 'Saved archive' });
        }),
      );

      const view = renderScreen(
        <StoragesComponent
          workspace={workspace}
          isCanManageStorages
          onUnavailable={onUnavailable}
        />,
      );

      await view.user.click((await screen.findAllByRole('button', { name: 'Edit' }))[0]);
      await view.user.clear(screen.getByPlaceholderText('Enter name...'));
      await view.user.type(screen.getByPlaceholderText('Enter name...'), 'Saved archive');
      await view.user.click(screen.getByRole('button', { name: 'Save' }));
      await screen.findByRole('button', { name: 'Retry' });

      expect(server.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(1);
      expect(screen.queryByPlaceholderText('Enter name...')).toBeNull();

      if (status === 403) {
        expect(onUnavailable).toHaveBeenCalledWith(workspace.id);
        expect(screen.queryByRole('button', { name: 'Saved archive' })).toBeNull();
        expect(screen.queryByRole('button', { name: 'Add storage' })).toBeNull();
      } else {
        expect(onUnavailable).not.toHaveBeenCalled();
        expect(screen.getByRole('button', { name: 'Saved archive' })).toBeTruthy();
      }
    },
  );

  it('attempts an ordinary read when the browser reports offline', async () => {
    server.use(mockApi.get('/storages', 200, { items: [] }));
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false);
    await processEvent(window, 'offline');

    renderScreen(<StorageScreen />);
    await screen.findByRole('button', { name: 'Add storage' });

    expect(getCollectionReads()).toHaveLength(1);

    await processEvent(window, 'online');

    expect(getCollectionReads()).toHaveLength(1);
  });

  it('sends no action with replacement credentials when the session changes before dispatch', async () => {
    activateSession({ token: 'first-session', userId: 'owner' });
    mockDetails();
    server.use(mockApi.get('/storages', 200, { items: [storage] }));

    renderScreen(<StorageScreen />);
    const test = await screen.findByRole('button', { name: 'Test connection' });

    await act(async () => {
      fireEvent.click(test);
      activateSession({ token: 'replacement-session', userId: 'another-owner' });
    });

    expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(0);
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('clears cached data when the session ends', async () => {
    activateSession({ token: 'first-session', userId: 'owner' });
    mockDetails();
    const action = mockApi.pending('post', '/storages/{storageId}/test');
    server.use(mockApi.get('/storages', 200, { items: [storage] }), action.handler);
    let client: QueryClient | undefined;
    const captureClient = (value: QueryClient) => {
      client = value;
    };

    const view = renderScreen(
      <RequestConfiguration>
        <ClientObserver onClient={captureClient} />
        <StorageScreen />
      </RequestConfiguration>,
    );
    await view.user.click(await screen.findByRole('button', { name: 'Test connection' }));
    await waitFor(() =>
      expect(server.getRequests().some((request) => request.method === 'POST')).toBe(true),
    );

    expect(
      client
        ?.getQueryCache()
        .getAll()
        .some((query) => query.state.data !== undefined),
    ).toBe(true);
    expect(client?.getMutationCache().getAll().length).toBeGreaterThan(0);

    act(() => {
      invalidateSession();
    });

    expect(
      client
        ?.getQueryCache()
        .getAll()
        .every((query) => query.state.data === undefined),
    ).toBe(true);
    expect(client?.getMutationCache().getAll()).toHaveLength(0);
    expect(screen.queryByRole('button', { name: storage.name })).toBeNull();

    await finishApiResponse(server, `/api/v1/storages/${storage.id}/test`, () =>
      action.respond(HttpResponse.json({ isConnected: true })),
    );

    expect(screen.queryByText('Connection test successful!')).toBeNull();
  });
});
