// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { PagedListScreen } from '@/shared/hooks/pagedListScreen.fixture';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const workspace = (name: string) => ({
  id: name,
  name,
  currentUserRole: 'WORKSPACE_OWNER' as const,
  createdAt: '2026-10-01T00:00:00Z',
});

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

describe('usePagedList', () => {
  it('blocks scrolling with an old cursor after reset and accepts the new first page', async () => {
    server.use(mockApi.get('/workspaces', 200, { items: [workspace('Old')], nextCursor: 'old' }));
    const view = renderScreen(<PagedListScreen />);
    await screen.findByText('Old');

    const replacement = mockApi.delayed({
      method: 'get',
      path: '/workspaces',
      status: 200,
      body: {
        items: [workspace('New')],
        nextCursor: 'new',
      },
    });
    server.use(replacement.handler);

    view.rerender(<PagedListScreen resetKey="second" />);
    await waitFor(() => expect(server.getRequests()).toHaveLength(2));
    fireEvent.scroll(screen.getByTestId('scroll-list'));
    expect(server.getRequests()).toHaveLength(2);

    await finishResponse(replacement.release);
    expect(await screen.findByText('New')).toBeTruthy();
    expect(screen.queryByText('Old')).toBeNull();

    server.use(mockApi.get('/workspaces', 200, { items: [workspace('Last')] }));
    fireEvent.scroll(screen.getByTestId('scroll-list'));
    await screen.findByText('Last');
    expect(new URL(server.getRequests()[2].url).searchParams.get('after')).toBe('new');
  });

  it.each(['success', 'error'] as const)(
    'ignores a pending %s after unmount and removes the scroll listener',
    async (outcome) => {
      const response =
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

      server.use(response.handler);
      const onError = vi.fn();
      const onPublish = vi.fn();
      const view = renderScreen(<PagedListScreen onError={onError} onPublish={onPublish} />);
      await waitFor(() => expect(server.getRequests()).toHaveLength(1));
      const scrollContainer = screen.getByTestId('scroll-list');
      const removeListener = vi.spyOn(scrollContainer, 'removeEventListener');

      view.unmount();
      const publications = onPublish.mock.calls.length;
      await finishResponse(response.release);

      expect(onError).not.toHaveBeenCalled();
      expect(onPublish).toHaveBeenCalledTimes(publications);
      expect(removeListener).toHaveBeenCalledWith('scroll', expect.any(Function));
      fireEvent.scroll(scrollContainer);
      expect(server.getRequests()).toHaveLength(1);
    },
  );

  it('keeps callbacks stable while using current options', async () => {
    server.use(mockApi.get('/workspaces', 200, { items: [workspace('First')] }));
    const onPublish = vi.fn();
    const view = renderScreen(<PagedListScreen onPublish={onPublish} />);
    await screen.findByText('First');
    const callbacks = onPublish.mock.calls[0][0];

    const onError = vi.fn();
    view.rerender(<PagedListScreen onPublish={onPublish} onError={onError} />);
    server.use(mockApi.get('/workspaces', 500, buildProblem('internal', 500)));
    fireEvent.click(screen.getByRole('button', { name: 'Reload' }));
    await waitFor(() => expect(onError).toHaveBeenCalledOnce());

    const current = onPublish.mock.lastCall![0];
    expect(current.reload).toBe(callbacks.reload);
    expect(current.refreshFirstPage).toBe(callbacks.refreshFirstPage);
    expect(current.replaceOrPrependItem).toBe(callbacks.replaceOrPrependItem);
  });

  it('loads an active list after StrictMode replay and ignores the first setup response', async () => {
    const abandoned = mockApi.pending('get', '/workspaces');
    let requestCount = 0;
    server.use(
      http.get('*/api/v1/workspaces', () => {
        requestCount++;
        return requestCount === 1
          ? abandoned.response
          : HttpResponse.json({ items: [workspace('Active')] });
      }),
    );

    const onError = vi.fn();
    renderScreen(<PagedListScreen onError={onError} />, { isStrictMode: true });

    expect(await screen.findByText('Active')).toBeTruthy();
    expect(screen.getByText('Ready:Idle')).toBeTruthy();
    expect(requestCount).toBe(2);

    await finishResponse(() =>
      abandoned.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
    );

    expect(onError).not.toHaveBeenCalled();
    expect(screen.getByText('Active')).toBeTruthy();
  });
});
