// @vitest-environment happy-dom
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { renderApp } from '@/app/ui/renderApp';
import { databaseApi } from '@/entities/databases';
import { createDefaultDatabaseSettings, DatabaseType } from '@/entities/databases';
import { notifierApi } from '@/entities/notifiers';
import { storageApi } from '@/entities/storages';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, getSessionSnapshot, invalidateSession } from '@/shared/auth';
import { fr } from '@/shared/i18n/locales/fr';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';

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
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'dev' }),
    ),
  );
};

describe('Application session isolation', () => {
  it('stops old polling and preserves preferences and reload cooldown when the session changes', async () => {
    seedAccount();
    let view: ReturnType<typeof render> | undefined;

    try {
      vi.stubEnv('VITE_APP_VERSION', '1.0.0');
      vi.resetModules();
      const [{ default: App }, session] = await Promise.all([
        import('@/app/ui/App'),
        import('@/shared/auth'),
      ]);
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
      const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
      const initialVersion = mockApi.pending('get', '/system/version');
      server.use(initialVersion.handler);
      localStorage.setItem('databasus-locale', 'fr');
      localStorage.setItem('databasus-theme', 'dark');
      localStorage.setItem('lastVersionReload', String(Date.now()));
      const getDatabaseReads = () =>
        server
          .getRequests()
          .filter((request) => new URL(request.url).pathname === '/api/v1/databases');

      view = render(<App />);
      await waitForCondition(() =>
        expect(screen.getByRole('button', { name: fr.databases.list.addDatabase })).toBeTruthy(),
      );
      await finishApiResponse(server, '/api/v1/system/version', () =>
        initialVersion.respond(HttpResponse.json({ version: '2.0.0' })),
      );

      expect(getDatabaseReads()).toHaveLength(1);
      expect(reload).not.toHaveBeenCalled();

      const obsoleteRefresh = mockApi.pending('get', '/databases');
      server.use(obsoleteRefresh.handler);

      await advanceFakeClock(30_000);
      await waitForCondition(() => expect(getDatabaseReads()).toHaveLength(2));

      const pendingRefresh = getDatabaseReads()[1];
      expect(pendingRefresh.headers.get('Authorization')).toBe('Bearer origin-session');
      expect(pendingRefresh.signal.aborted).toBe(false);

      const replacementAccount = {
        ...account,
        id: 'replacement-owner',
        email: 'replacement@example.com',
      };
      const replacementVersion = mockApi.pending('get', '/system/version');
      const reloadCooldown = String(Date.now());
      server.use(
        mockApi.get('/users/{userId}', 200, replacementAccount),
        mockApi.get('/workspaces', 200, { items: [] }),
        mockApi.get('/databases', 200, { items: [] }),
        replacementVersion.handler,
      );
      localStorage.setItem('lastVersionReload', reloadCooldown);

      act(() =>
        session.activateSession({ token: 'replacement-session', userId: replacementAccount.id }),
      );
      await waitForCondition(() =>
        expect(screen.getByRole('link', { name: fr.app.navigation.profile })).toBeTruthy(),
      );
      fireEvent.click(screen.getByRole('link', { name: fr.app.navigation.profile }));
      await waitForCondition(() =>
        expect(screen.getByDisplayValue(replacementAccount.email)).toBeTruthy(),
      );
      await finishApiResponse(server, '/api/v1/system/version', () =>
        replacementVersion.respond(HttpResponse.json({ version: '2.0.0' })),
      );

      expect(pendingRefresh.signal.aborted).toBe(true);
      expect(reload).not.toHaveBeenCalled();

      await finishCancelledApiResponse(server, '/api/v1/databases', () =>
        obsoleteRefresh.respond(HttpResponse.json({ items: [lists[0].item] })),
      );
      await advanceFakeClock(60_001);

      expect(getDatabaseReads()).toHaveLength(2);
      expect(session.getSessionSnapshot().token).toBe('replacement-session');
      expect(screen.getByDisplayValue(replacementAccount.email)).toBeTruthy();
      expect(screen.queryByText(lists[0].item.name)).toBeNull();
      expect(localStorage.getItem(lists[0].selectionKey)).toBeNull();
      expect(localStorage.getItem('databasus-locale')).toBe('fr');
      expect(document.documentElement.lang).toBe('fr');
      expect(localStorage.getItem('databasus-theme')).toBe('dark');
      expect(document.documentElement.classList.contains('dark')).toBe(true);
      expect(localStorage.getItem('lastVersionReload')).toBe(reloadCooldown);
      expect(reload).not.toHaveBeenCalled();
    } finally {
      view?.unmount();
      vi.unstubAllEnvs();
    }
  });

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
      fireEvent.click(screen.getByRole('link', { name: 'Notifiers' }));
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
      fireEvent.click(await screen.findByRole('link', { name: 'Profile' }));
      await screen.findByDisplayValue('replacement@example.com');

      await (operation === 'load' ? finishCancelledApiResponse : finishApiResponse)(
        server,
        requestPath,
        () => {
          obsoleteOperation.respond(
            operation === 'load'
              ? HttpResponse.json(buildProblem('unauthenticated', 401), { status: 401 })
              : HttpResponse.json({ isDelivered: false, error: 'Obsolete diagnostic' }),
          );
        },
      );

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
      const listTab = await screen.findByRole('link', { name: tab });
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
        fireEvent.click(await screen.findByRole('link', { name: 'Profile' }));
        await screen.findByDisplayValue(replacementAccount.email);
      }

      await finishCancelledApiResponse(server, `/api/v1${path}`, () => {
        obsoleteList.respond(
          HttpResponse.json(
            status === 200 ? { items: [item] } : buildProblem('unauthenticated', status),
            { status },
          ),
        );
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
      const listRead =
        path === '/databases'
          ? vi.spyOn(databaseApi, 'listAllDatabases')
          : path === '/storages'
            ? vi.spyOn(storageApi, 'listAllStorages')
            : vi.spyOn(notifierApi, 'listAllNotifiers');
      renderApp({ isStrictMode: true });
      if (path !== '/databases') {
        await screen.findByRole('button', { name: 'Add database' });
        fireEvent.click(screen.getByRole('link', { name: tab }));
      }
      await screen.findByRole('button', { name: action });
      expect(listRead.mock.calls.length).toBeGreaterThanOrEqual(2);
      expect(listRead.mock.calls[0][1]?.signal?.aborted).toBe(true);

      expect(alert).not.toHaveBeenCalled();
      expect(screen.getByRole('button', { name: action })).toBeTruthy();
      expect(getSessionSnapshot().token).toBe('origin-session');
    },
  );
});

const readLogicalBackupFilters = () =>
  server
    .getRequests()
    .map((request) => new URL(request.url))
    .filter(({ pathname }) => pathname === '/api/v1/logical-backups')
    .map(({ searchParams }) => searchParams.get('createdBefore'));

describe('Application logout', () => {
  it('clears the account state and keeps the preferences when Logout is clicked', async () => {
    seedAccount();
    localStorage.setItem('selectedDatabaseId_workspace-id', 'previous-database');
    localStorage.setItem('selectedStorageId_workspace-id', 'previous-storage');
    localStorage.setItem('selectedNotifierId_workspace-id', 'previous-notifier');
    localStorage.setItem('logicalBackupListFilter', JSON.stringify({ statuses: ['FAILED'] }));
    localStorage.setItem('physicalBackupsFilters', JSON.stringify({ types: ['FULL'] }));
    localStorage.setItem('databasus-locale', 'en');
    localStorage.setItem('databasus-theme', 'dark');
    localStorage.setItem('lastVersionReload', '123');
    sessionStorage.setItem('databasus_oauth_sign_in_attempt', 'previous-attempt');

    renderApp();
    fireEvent.click(await screen.findByRole('link', { name: 'Profile' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Logout' }));
    await screen.findByRole('button', { name: 'Sign in' });

    expect(window.location.pathname).toBe('/');
    expect(getSessionSnapshot()).toMatchObject({ token: undefined, userId: undefined });
    for (const clearedKey of [
      'databasus_user_token',
      'databasus_user_id',
      'selected_workspace_id',
      'selectedDatabaseId_workspace-id',
      'selectedStorageId_workspace-id',
      'selectedNotifierId_workspace-id',
      'logicalBackupListFilter',
      'physicalBackupsFilters',
    ])
      expect(localStorage.getItem(clearedKey), clearedKey).toBeNull();
    expect(sessionStorage.getItem('databasus_oauth_sign_in_attempt')).toBeNull();
    expect(localStorage.getItem('databasus-locale')).toBe('en');
    expect(localStorage.getItem('databasus-theme')).toBe('dark');
    expect(localStorage.getItem('lastVersionReload')).toBe('123');
  });

  it('waits for fresh startup data after the same account signs in again', async () => {
    seedAccount();
    renderApp();
    fireEvent.click(await screen.findByRole('link', { name: 'Profile' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Logout' }));
    await screen.findByRole('button', { name: 'Sign in' });

    const freshAccount = mockApi.delayed({
      method: 'get',
      path: '/users/{userId}',
      status: 200,
      body: { ...account, name: 'Updated owner' },
    });
    server.use(freshAccount.handler);
    const previousReads = server
      .getRequests()
      .filter((request) => new URL(request.url).pathname.endsWith(`/users/${account.id}`)).length;

    act(() => activateSession({ token: 'second-session', userId: account.id }));
    await waitFor(() =>
      expect(
        server
          .getRequests()
          .filter((request) => new URL(request.url).pathname.endsWith(`/users/${account.id}`)),
      ).toHaveLength(previousReads + 1),
    );

    expect(screen.queryByRole('link', { name: 'Profile' })).toBeNull();

    await act(async () => freshAccount.release());
    await screen.findByRole('link', { name: 'Profile' });

    server.use(mockApi.get('/users/{userId}', 200, { ...account, name: 'Updated owner' }));

    fireEvent.click(await screen.findByRole('link', { name: 'Profile' }));
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: 'Name' })).toHaveProperty(
        'value',
        'Updated owner',
      ),
    );
  });

  it.each([false, true])(
    'opens the next account on the first database with no filter in a shared workspace after explicit logout=%s',
    async (shouldLogOut) => {
      const buildSharedDatabase = (id: string, name: string) => ({
        id,
        name,
        workspaceId: workspace.id,
        settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
        notifierIds: [],
        healthStatus: 'AVAILABLE' as const,
      });
      const firstDatabase = buildSharedDatabase('first-database', 'First database');
      const secondDatabase = buildSharedDatabase('second-database', 'Second database');
      const previousFilterDate = '2026-01-01T00:00:00.000Z';

      seedAccount();
      server.use(
        mockApi.get('/databases', 200, { items: [firstDatabase, secondDatabase] }),
        http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
          HttpResponse.json(
            params.databaseId === firstDatabase.id ? firstDatabase : secondDatabase,
          ),
        ),
        mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: [] }),
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
          HttpResponse.json(buildProblem('not_found', 404), { status: 404 }),
        ),
        http.get(`*${API_BASE_PATH}/healthcheck-configs/:databaseId`, () =>
          HttpResponse.json(buildProblem('not_found', 404), { status: 404 }),
        ),
      );
      localStorage.setItem(
        'logicalBackupListFilter',
        JSON.stringify({ createdBefore: previousFilterDate }),
      );

      renderApp();
      fireEvent.click(await screen.findByText('Second database'));
      await waitFor(() =>
        expect(localStorage.getItem('selectedDatabaseId_workspace-id')).toBe(secondDatabase.id),
      );
      await waitFor(() => expect(readLogicalBackupFilters()).toContain(previousFilterDate));

      if (shouldLogOut) {
        fireEvent.click(screen.getByRole('link', { name: 'Profile' }));
        fireEvent.click(await screen.findByRole('button', { name: 'Logout' }));
        await screen.findByRole('button', { name: 'Sign in' });
      }

      server.use(
        mockApi.get('/users/{userId}', 200, {
          ...account,
          id: 'replacement-owner',
          email: 'replacement@example.com',
        }),
      );
      act(() => activateSession({ token: 'replacement-session', userId: 'replacement-owner' }));

      await waitFor(() =>
        expect(localStorage.getItem('selectedDatabaseId_workspace-id')).toBe(firstDatabase.id),
      );
      await waitFor(() => expect(readLogicalBackupFilters().pop()).toBeNull());
      expect(window.location.pathname).toBe(
        `/workspaces/${workspace.id}/databases/${firstDatabase.id}/backups`,
      );
    },
  );
});
