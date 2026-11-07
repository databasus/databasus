// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { UsersPage } from '@/pages/users/ui/UsersPage';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const buildUser = (email: string): ApiComponents['schemas']['User'] => ({
  id: `${email}-id`,
  email,
  name: email,
  role: 'MEMBER',
  status: 'ACTIVE',
  createdAt: '2026-10-01T10:00:00Z',
});

describe('UsersPage', () => {
  it.each([
    { query: '', locale: 'en' },
    { query: 'member', locale: 'fr' },
  ] as const)(
    'reloads the same query without an empty-success state when the $locale list request for "$query" fails',
    async ({ query, locale }) => {
      const dictionary = DICTIONARIES[locale];
      mockServer.use(
        query
          ? mockApi.get('/users', 200, { items: [] })
          : mockApi.get('/users', 500, buildProblem('internal', 500)),
      );

      const { user } = renderScreen(<UsersPage />, { locale });
      const search = screen.getByRole('textbox', { name: dictionary.users.list.searchPlaceholder });
      if (query) {
        await screen.findByText(dictionary.users.list.empty);
        mockServer.use(mockApi.get('/users', 500, buildProblem('internal', 500)));
        await user.type(search, query);
      }

      await screen.findByText(dictionary.errors.internal);
      const retry = await screen.findByRole('button', { name: dictionary.common.actions.retry });
      expect(screen.queryByText(dictionary.users.list.empty)).toBeNull();
      expect(screen.queryByText(dictionary.users.list.loaded.replace('{{count}}', '0'))).toBeNull();

      const failedRequestCount = mockServer.getRequests().length;
      const failedRequest = new URL(mockServer.getRequests()[failedRequestCount - 1].url);
      const retriedPage = mockApi.delayed({
        method: 'get',
        path: '/users',
        status: 200,
        body: { items: [buildUser('member@example.com')] },
      });
      mockServer.use(retriedPage.handler);

      await user.click(retry);
      await waitFor(() => expect(mockServer.getRequests()).toHaveLength(failedRequestCount + 1));

      expect(new URL(mockServer.getRequests()[failedRequestCount].url).search).toBe(
        failedRequest.search,
      );
      expect(failedRequest.searchParams.get('query')).toBe(query || null);
      expect(failedRequest.searchParams.get('limit')).toBe('20');
      expect(failedRequest.searchParams.has('after')).toBe(false);
      expect((search as HTMLInputElement).value).toBe(query);
      expect(screen.queryByText(dictionary.users.list.empty)).toBeNull();

      retriedPage.release();
      await screen.findByText(/member@example.com/);
      expect(screen.queryByText(dictionary.errors.internal)).toBeNull();
      expect(screen.queryByRole('button', { name: dictionary.common.actions.retry })).toBeNull();
    },
  );

  it('removes the retry and sends one request while the reload is pending', async () => {
    mockServer.use(mockApi.get('/users', 500, buildProblem('internal', 500)));
    const { user } = renderScreen(<UsersPage />);
    const retry = await screen.findByRole('button', { name: 'Retry' });
    const failedRequestCount = mockServer.getRequests().length;
    const retriedPage = mockApi.delayed({
      method: 'get',
      path: '/users',
      status: 200,
      body: { items: [buildUser('member@example.com')] },
    });
    mockServer.use(retriedPage.handler);

    await user.click(retry);
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(failedRequestCount + 1));
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull());

    retriedPage.release();
    await screen.findByText(/member@example.com/);

    expect(mockServer.getRequests()).toHaveLength(failedRequestCount + 1);
  });

  it('loads the next page with the cursor of the previous one when the list is scrolled down', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/users`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        const userPage: ApiComponents['schemas']['UserPage'] =
          cursor === null
            ? { items: [buildUser('first@example.com')], nextCursor: 'first-page-cursor' }
            : { items: [buildUser('second@example.com')] };

        return HttpResponse.json(userPage);
      }),
    );
    const { container } = renderScreen(<UsersPage />);
    await screen.findByText(/first@example\.com/);

    fireEvent.scroll(container.querySelector('.overflow-y-auto') as Element);

    expect(await screen.findByText(/second@example\.com/)).toBeTruthy();
    await waitFor(() => expect(requestedCursors).toEqual([null, 'first-page-cursor']));
  });
});

const processedResponse = () =>
  new Promise<void>((resolve) => {
    const onResponse = () => {
      mockServer.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    mockServer.events.on('response:mocked', onResponse);
  });

describe('UsersPage asynchronous ownership', () => {
  it.each([200, 500] as const)(
    'keeps the current rows after a superseded search returns HTTP %s',
    async (status) => {
      const earlier = mockApi.pending('get', '/users');
      mockServer.use(earlier.handler);

      const { container, user } = renderScreen(<UsersPage />);
      await waitFor(() => expect(mockServer.getRequests()).toHaveLength(1));

      mockServer.use(
        mockApi.get('/users', 200, {
          items: [buildUser('current@example.com')],
          nextCursor: 'current-cursor',
        }),
      );
      await user.type(
        screen.getByRole('textbox', { name: 'Search by email or name...' }),
        'current',
      );
      await screen.findByText(/current@example.com/);

      const response = processedResponse();
      await act(async () => {
        earlier.respond(
          HttpResponse.json(
            status === 200
              ? { items: [buildUser('obsolete@example.com')], nextCursor: 'obsolete-cursor' }
              : buildProblem('internal', 500),
            { status },
          ),
        );
        await response;
      });

      expect(screen.queryByText(/obsolete@example.com/)).toBeNull();
      expect(screen.queryByText(DICTIONARIES.en.errors.internal)).toBeNull();

      fireEvent.scroll(container.querySelector('.overflow-y-auto')!);
      await waitFor(() => expect(mockServer.getRequests()).toHaveLength(3));
      const pageUrl = new URL(mockServer.getRequests()[2].url);
      expect(pageUrl.searchParams.get('query')).toBe('current');
      expect(pageUrl.searchParams.get('after')).toBe('current-cursor');
      expect(pageUrl.searchParams.get('limit')).toBe('20');
    },
  );

  it('discards old paging and deduplicates rows when a new search starts during scrolling', async () => {
    const firstUser = buildUser('first@example.com');
    mockServer.use(mockApi.get('/users', 200, { items: [firstUser], nextCursor: 'old-cursor' }));
    const { container, user } = renderScreen(<UsersPage />);
    await screen.findByText(/first@example.com/);

    const oldPage = mockApi.delayed({
      method: 'get',
      path: '/users',
      status: 200,
      body: { items: [buildUser('obsolete@example.com')] },
    });
    mockServer.use(oldPage.handler);
    const scroller = container.querySelector('.overflow-y-auto')!;
    fireEvent.scroll(scroller);
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(2));

    const newSearch = mockApi.pending('get', '/users');
    mockServer.use(newSearch.handler);
    await user.type(screen.getByRole('textbox', { name: 'Search by email or name...' }), 'new');
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(3));
    fireEvent.scroll(scroller);
    expect(mockServer.getRequests()).toHaveLength(3);

    const response = processedResponse();
    await act(async () => {
      oldPage.release();
      await response;
    });
    await act(async () => {
      newSearch.respond(
        HttpResponse.json({ items: [buildUser('new@example.com'), buildUser('new@example.com')] }),
      );
    });

    await screen.findByText(/new@example.com/);
    expect(screen.queryByText(/obsolete@example.com/)).toBeNull();
    expect(screen.getAllByRole('switch')).toHaveLength(1);
  });

  it('retains acknowledged status when an older list page finishes after a mutation', async () => {
    const account = buildUser('member@example.com');
    mockServer.use(mockApi.get('/users', 200, { items: [account], nextCursor: 'next' }));
    const { container, user } = renderScreen(<UsersPage />);
    const toggle = await screen.findByRole('switch');

    const page = mockApi.delayed({
      method: 'get',
      path: '/users',
      status: 200,
      body: { items: [account, buildUser('second@example.com')] },
    });
    const mutation = mockApi.delayed({
      method: 'post',
      path: '/users/{userId}/deactivate',
      status: 200,
      body: { ...account, status: 'INACTIVE' },
    });
    mockServer.use(page.handler, mutation.handler);

    fireEvent.scroll(container.querySelector('.overflow-y-auto')!);
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(2));
    await user.click(toggle);
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(3));
    await user.click(toggle);
    expect(mockServer.getRequests().filter((request) => request.method === 'POST')).toHaveLength(1);
    expect((toggle as HTMLInputElement).checked).toBe(true);

    mutation.release();
    await screen.findByText('User deactivated successfully');
    expect((toggle as HTMLInputElement).checked).toBe(false);

    page.release();
    await screen.findByText(/second@example.com/);
    expect(
      (screen.getByRole('switch', { name: 'Is active? (member@example.com)' }) as HTMLInputElement)
        .checked,
    ).toBe(false);
    expect(screen.getAllByRole('switch')).toHaveLength(2);
  });

  it.each(['status', 'role'] as const)(
    'preserves displayed values and permits retry when the %s action is refused',
    async (action) => {
      const account = buildUser('member@example.com');
      mockServer.use(
        mockApi.get('/users', 200, { items: [account] }),
        mockApi.post('/users/{userId}/deactivate', 403, buildProblem('permission_denied', 403)),
        mockApi.put('/users/{userId}/role', 403, buildProblem('permission_denied', 403)),
      );
      const { user } = renderScreen(<UsersPage />);
      await screen.findByText(/member@example.com/);

      if (action === 'status') await user.click(screen.getByRole('switch'));
      else {
        await user.click(screen.getByRole('combobox'));
        await user.click(await screen.findByText('Admin'));
      }
      await screen.findByText('You do not have permission to do this.');

      expect((screen.getByRole('switch') as HTMLInputElement).checked).toBe(true);
      expect(screen.getByRole('combobox').hasAttribute('disabled')).toBe(false);
      expect(screen.getByRole('combobox', { name: /^System role/ }).textContent).toBe('Member');
      expect(screen.getByRole('switch').hasAttribute('disabled')).toBe(false);
    },
  );

  it('publishes no mutation feedback after the page is unmounted with a pending action', async () => {
    const account = buildUser('member@example.com');
    const mutation = mockApi.delayed({
      method: 'post',
      path: '/users/{userId}/deactivate',
      status: 200,
      body: { ...account, status: 'INACTIVE' },
    });
    mockServer.use(mockApi.get('/users', 200, { items: [account] }), mutation.handler);

    const view = renderScreen(<UsersPage />);
    await view.user.click(await screen.findByRole('switch'));
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(2));

    view.rerender(<div>Users closed</div>);
    const response = processedResponse();
    await act(async () => {
      mutation.release();
      await response;
    });

    expect(screen.queryByText('User deactivated successfully')).toBeNull();
    expect(screen.getByText('Users closed')).toBeTruthy();
  });
});

it('acknowledges role and activation from returned users when each action succeeds', async () => {
  const account = { ...buildUser('inactive@example.com'), status: 'INACTIVE' as const };
  mockServer.use(
    mockApi.get('/users', 200, { items: [account] }),
    mockApi.post('/users/{userId}/activate', 200, { ...account, status: 'ACTIVE' }),
    mockApi.put('/users/{userId}/role', 200, { ...account, status: 'ACTIVE', role: 'ADMIN' }),
  );

  const { user } = renderScreen(<UsersPage />);
  await user.click(await screen.findByRole('switch'));
  await screen.findByText('User activated successfully');
  expect((screen.getByRole('switch') as HTMLInputElement).checked).toBe(true);

  await user.click(screen.getByRole('combobox'));
  await user.click(await screen.findByText('Admin'));
  await screen.findByText('User role changed successfully');

  expect(screen.getByRole('combobox', { name: /^System role/ }).textContent).toBe('Admin');
  expect(
    await mockServer
      .getRequests()
      .find((request) => request.method === 'PUT')!
      .json(),
  ).toEqual({ role: 'ADMIN' });
});

it.each([
  { action: 'status', isRefused: false },
  { action: 'role', isRefused: false },
  { action: 'status', isRefused: true },
  { action: 'role', isRefused: true },
] as const)(
  'ignores the obsolete $action completion when search returns to the same text with refusal=$isRefused',
  async ({ action, isRefused }) => {
    const account = buildUser('member@example.com');
    let firstQueryCount = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/users`, ({ request }) => {
        const query = new URL(request.url).searchParams.get('query');
        const name =
          query === 'a'
            ? ++firstQueryCount === 1
              ? 'First A'
              : 'Current A'
            : query === 'b'
              ? 'Current B'
              : 'Initial';
        return HttpResponse.json({ items: [{ ...account, name }] });
      }),
    );
    const mutation =
      action === 'status'
        ? mockApi.pending('post', '/users/{userId}/deactivate')
        : mockApi.pending('put', '/users/{userId}/role');
    mockServer.use(mutation.handler);

    const { user } = renderScreen(<UsersPage />);
    await screen.findByText(/Initial/);
    const search = screen.getByRole('textbox', { name: 'Search by email or name...' });
    fireEvent.change(search, { target: { value: 'a' } });
    await screen.findByText(/First A/);

    if (action === 'status') await user.click(screen.getByRole('switch'));
    else {
      await user.click(screen.getByRole('combobox'));
      await user.click(await screen.findByText('Admin'));
    }
    await waitFor(() =>
      expect(mockServer.getRequests().some((request) => request.method !== 'GET')).toBe(true),
    );

    fireEvent.change(search, { target: { value: 'b' } });
    await screen.findByText(/Current B/);
    fireEvent.change(search, { target: { value: 'a' } });
    await screen.findByText(/Current A/);

    const processed = processedResponse();
    await act(async () => {
      mutation.respond(
        HttpResponse.json(
          isRefused
            ? buildProblem('permission_denied', 403)
            : {
                ...account,
                name: 'Obsolete A',
                status: action === 'status' ? 'INACTIVE' : 'ACTIVE',
                role: action === 'role' ? 'ADMIN' : 'MEMBER',
              },
          { status: isRefused ? 403 : 200 },
        ),
      );
      await processed;
    });

    await waitFor(() => expect(screen.getByRole('switch').hasAttribute('disabled')).toBe(false));
    expect(screen.getByText(/Current A/)).toBeTruthy();
    expect(screen.queryByText(/Obsolete A/)).toBeNull();
    expect((screen.getByRole('switch') as HTMLInputElement).checked).toBe(true);
    expect(screen.getByRole('combobox', { name: /^System role/ }).textContent).toBe('Member');
    expect(screen.queryByText(DICTIONARIES.en.errors.permissionDenied)).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('status')).toBeNull();
  },
);
