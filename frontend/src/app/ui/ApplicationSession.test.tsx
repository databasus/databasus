// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { createDefaultDatabaseSettings, DatabaseType } from '@/entities/databases';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, getSessionSnapshot, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';

import { renderApp } from './renderApp';

const server = setUpMockApi();
const account = {
  id: 'owner-id',
  name: 'Owner',
  email: 'owner@example.com',
  role: 'ADMIN' as const,
  status: 'ACTIVE' as const,
  createdAt: '2026-10-01T00:00:00Z',
};
const workspace = {
  id: 'workspace-id',
  name: 'Workspace',
  currentUserRole: 'WORKSPACE_OWNER' as const,
  createdAt: '2026-10-01T00:00:00Z',
};
const lists = [
  {
    path: '/databases',
    tab: 'Databases',
    action: 'Add database',
    selectionKey: 'selectedDatabaseId_workspace-id',
    item: {
      id: 'obsolete-database',
      name: 'Obsolete database',
      workspaceId: workspace.id,
      settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
      notifierIds: [],
      healthStatus: 'AVAILABLE',
    },
  },
  {
    path: '/storages',
    tab: 'Storages',
    action: 'Add storage',
    selectionKey: 'selectedStorageId_workspace-id',
    item: {
      id: 'obsolete-storage',
      name: 'Obsolete storage',
      workspaceId: workspace.id,
      settings: { type: 'LOCAL' },
    },
  },
  {
    path: '/notifiers',
    tab: 'Notifiers',
    action: 'Add notifier',
    selectionKey: 'selectedNotifierId_workspace-id',
    item: {
      id: 'obsolete-notifier',
      name: 'Obsolete notifier',
      workspaceId: workspace.id,
      lastSendError: null,
      settings: {
        type: 'EMAIL',
        targetEmail: 'ops@example.com',
        smtpHost: 'smtp.example.com',
        smtpPort: 587,
        security: 'STARTTLS',
        isInsecureSkipVerify: false,
      },
    },
  },
] as const;
const seedAccount = () => {
  activateSession({ token: 'origin-session', userId: account.id });
  server.use(
    mockApi.get('/users/{userId}', 200, account),
    mockApi.get('/workspaces', 200, { items: [workspace] }),
    mockApi.get('/instance-settings', 200, {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isTwoFactorAuthRequired: false,
      isEmailConfigured: true,
    }),
    mockApi.get('/system/disk-usages', 200, {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    }),
    mockApi.get('/system/version', 200, { version: 'dev' }),
    mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
    mockApi.get('/databases', 200, { items: [] }),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
    mockApi.get('/notifiers', 200, { items: [] }),
    mockApi.get('/storage-switches', 200, { items: [] }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'dev' }),
    ),
  );
};

const observeListResponse = (path: string) =>
  new Promise<void>((resolve) => {
    const onResponse = ({ request }: { request: Request }) => {
      if (new URL(request.url).pathname !== `/api/v1${path}`) return;
      server.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    server.events.on('response:mocked', onResponse);
  });

describe('Application session isolation', () => {
  it.each(['load', 'test'] as const)(
    'suppresses obsolete notifier details feedback after account replacement during $0',
    async (operation) => {
      seedAccount();
      const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
      const notifier = lists[2].item;
      const obsoleteOperation =
        operation === 'load'
          ? mockApi.pending('get', '/notifiers/{notifierId}')
          : mockApi.pending('post', '/notifiers/{notifierId}/test');
      server.use(
        mockApi.get('/notifiers', 200, { items: [notifier] }),
        mockApi.get('/notifiers/{notifierId}', 200, notifier),
      );
      server.use(obsoleteOperation.handler);

      renderApp();
      await screen.findByRole('button', { name: 'Add database' });
      fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));
      if (operation === 'test')
        fireEvent.click(await screen.findByRole('button', { name: 'Send test notification' }));
      const requestPath = `/api/v1/notifiers/${notifier.id}${operation === 'test' ? '/test' : ''}`;
      await waitFor(() =>
        expect(
          server.getRequests().some((request) => new URL(request.url).pathname === requestPath),
        ).toBe(true),
      );

      act(() => invalidateSession());
      await screen.findByRole('button', { name: 'Sign in' });

      server.use(
        mockApi.get('/users/{userId}', 200, {
          ...account,
          id: 'replacement-owner',
          email: 'replacement@example.com',
        }),
        mockApi.get('/workspaces', 200, { items: [] }),
      );
      act(() => activateSession({ token: 'replacement-session', userId: 'replacement-owner' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Profile' }));
      await screen.findByDisplayValue('replacement@example.com');

      const processed = observeListResponse(requestPath.replace('/api/v1', ''));
      await act(async () => {
        obsoleteOperation.respond(
          operation === 'load'
            ? HttpResponse.json(buildProblem('unauthenticated', 401), { status: 401 })
            : HttpResponse.json({ isDelivered: false, error: 'Obsolete diagnostic' }),
        );
        await processed;
      });

      expect(alert).not.toHaveBeenCalled();
      expect(getSessionSnapshot().token).toBe('replacement-session');
      expect(screen.getByDisplayValue('replacement@example.com')).toBeTruthy();
    },
  );

  it.each(
    lists.flatMap((list) =>
      [200, 401].flatMap((status) =>
        [false, true].map((hasReplacementSession) => ({ ...list, status, hasReplacementSession })),
      ),
    ),
  )(
    'suppresses the old $tab HTTP $status completion after logout with replacement session=$hasReplacementSession',
    async ({ path, tab, selectionKey, item, status, hasReplacementSession }) => {
      seedAccount();
      const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
      const obsoleteList = mockApi.pending('get', path);
      if (path === '/databases') server.use(obsoleteList.handler);

      renderApp();
      const listTab = await screen.findByRole('button', { name: tab });
      if (path !== '/databases') {
        await screen.findByRole('button', { name: 'Add database' });
        server.use(obsoleteList.handler);
        fireEvent.click(listTab);
      }

      await waitFor(() =>
        expect(
          server
            .getRequests()
            .some((request) => new URL(request.url).pathname === `/api/v1${path}`),
        ).toBe(true),
      );

      act(() => invalidateSession());
      await screen.findByRole('button', { name: 'Sign in' });

      const replacementAccount = {
        ...account,
        id: 'replacement-owner',
        email: 'replacement@example.com',
        name: 'Replacement owner',
      };

      server.use(
        mockApi.get(path, 200, { items: [] }),
        mockApi.get('/users/{userId}', 200, replacementAccount),
        mockApi.get('/workspaces', 200, { items: [] }),
      );
      if (hasReplacementSession) {
        act(() => activateSession({ token: 'replacement-session', userId: replacementAccount.id }));
        fireEvent.click(await screen.findByRole('button', { name: 'Profile' }));
        await screen.findByDisplayValue(replacementAccount.email);
      }

      const processed = observeListResponse(path);
      await act(async () => {
        obsoleteList.respond(
          HttpResponse.json(
            status === 200 ? { items: [item] } : buildProblem('unauthenticated', status),
            { status },
          ),
        );
        await processed;
      });

      expect(alert).not.toHaveBeenCalled();
      expect(getSessionSnapshot().token).toBe(
        hasReplacementSession ? 'replacement-session' : undefined,
      );
      if (hasReplacementSession)
        expect(screen.getByDisplayValue(replacementAccount.email)).toBeTruthy();
      else expect(screen.getByRole('button', { name: 'Sign in' })).toBeTruthy();
      expect(localStorage.getItem(selectionKey)).toBeNull();
    },
  );

  it.each(lists)(
    'retains the active $tab load without old feedback after root StrictMode repeats setup',
    async ({ path, tab, action }) => {
      seedAccount();
      const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
      const obsoleteSetup = mockApi.pending('get', path);
      let listReadCount = 0;
      const listHandler = http.get(`*${API_BASE_PATH}${path}`, () =>
        ++listReadCount === 1 ? obsoleteSetup.response : HttpResponse.json({ items: [] }),
      );
      if (path === '/databases') server.use(listHandler);

      renderApp({ isStrictMode: true });
      if (path !== '/databases') {
        await screen.findByRole('button', { name: 'Add database' });
        server.use(listHandler);
        fireEvent.click(screen.getByRole('button', { name: tab }));
      }
      await screen.findByRole('button', { name: action });
      expect(listReadCount).toBe(2);

      const processed = observeListResponse(path);
      await act(async () => {
        obsoleteSetup.respond(
          HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
        );
        await processed;
      });

      expect(alert).not.toHaveBeenCalled();
      expect(screen.getByRole('button', { name: action })).toBeTruthy();
      expect(getSessionSnapshot().token).toBe('origin-session');
    },
  );
});
