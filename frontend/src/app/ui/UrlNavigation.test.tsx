// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { renderApp } from '@/app/ui/renderApp';
import { type BackupConfig, backupConfigApi } from '@/entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
  databaseApi,
} from '@/entities/databases';
import { userApi } from '@/entities/users';
import { type Workspace, workspaceApi } from '@/entities/workspaces';
import { activateSession, getSessionSnapshot, invalidateSession } from '@/shared/auth';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';

const server = setUpMockApi();
const workspace: Workspace = {
  id: 'team-id',
  name: 'Team',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T00:00:00Z',
};
const database: Database = {
  id: 'shop-id',
  name: 'Shop',
  workspaceId: workspace.id,
  notifierIds: [],
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
};
const databasePath = `/workspaces/${workspace.id}/databases/${database.id}`;
const initialAddress = window.location.href;

afterEach(() => window.history.replaceState(null, '', initialAddress));

const prepareNavigation = (databases: Database[] = [database]) => {
  activateSession({ token: 'navigation-session', userId: 'owner-id' });
  server.use(
    mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
    mockApi.get('/users/{userId}', 200, {
      id: 'owner-id',
      name: 'Owner',
      email: 'owner@example.com',
      role: 'MEMBER',
      status: 'ACTIVE',
      createdAt: workspace.createdAt,
    }),
    mockApi.get('/instance-settings', 200, {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isTwoFactorAuthRequired: false,
      isEmailConfigured: false,
    }),
    mockApi.get('/workspaces', 200, { items: [workspace] }),
    mockApi.get('/system/disk-usages', 200, {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    }),
    http.get('*/api/v1/databases', ({ request }) =>
      HttpResponse.json({
        items: databases.filter(
          (item) => item.workspaceId === new URL(request.url).searchParams.get('workspaceId'),
        ),
      }),
    ),
    http.get('*/api/v1/databases/:databaseId', ({ params }) =>
      HttpResponse.json(databases.find((item) => item.id === params.databaseId)),
    ),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
    mockApi.get('/notifiers', 200, { items: [] }),
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    mockApi.pending('get', '/healthcheck-attempts').handler,
    mockApi.pending('get', '/logical-backups').handler,
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'dev' }),
    ),
  );
};

describe('Application URL navigation', () => {
  it.each([
    { isEmptyWorkspace: true, shouldReturnToSource: false },
    { isEmptyWorkspace: false, shouldReturnToSource: false },
    { isEmptyWorkspace: false, shouldReturnToSource: true },
  ])(
    'selects a wizard-created database only on accepted closure with empty workspace=$isEmptyWorkspace and returning to source=$shouldReturnToSource',
    async ({ isEmptyWorkspace, shouldReturnToSource }) => {
      const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
      const createdDatabase = { ...database, id: 'created-id', name: 'Created' };
      const databases = isEmptyWorkspace ? [] : [database, otherDatabase];
      prepareNavigation(databases);
      const archiveStorage = {
        id: 'archive-id',
        workspaceId: workspace.id,
        name: 'Archive',
        settings: { type: 'LOCAL' as const },
      };
      const savedConfig: BackupConfig = {
        databaseId: createdDatabase.id,
        isBackupsEnabled: true,
        storageId: archiveStorage.id,
        encryption: 'NONE',
        cleanup: { pendingFileCount: 0, failedFileCount: 0 },
        settings: {
          type: 'LOGICAL',
          schedule: { type: 'DAILY', timeOfDay: '04:00' },
          retention: {
            policy: 'COUNT',
            period: 'MONTH',
            count: 10,
            gfsHours: 0,
            gfsDays: 0,
            gfsWeeks: 0,
            gfsMonths: 0,
            gfsYears: 0,
          },
          isRetryIfFailed: false,
          maxFailedTriesCount: 1,
          notificationTriggers: [],
        },
      };
      const configuration = mockApi.pending('put', '/backup-configs/{databaseId}');
      const configurationSaves = vi.spyOn(backupConfigApi, 'updateBackupConfig');
      const collectionReads = vi.spyOn(databaseApi, 'listAllDatabases');
      server.use(
        mockApi.get('/storages', 200, { items: [archiveStorage] }),
        mockApi.get('/storages/{storageId}', 200, archiveStorage),
        mockApi.post('/database-tests', 200, { isConnected: true }),
        mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
        http.post('*/api/v1/databases', () => {
          databases.push(createdDatabase);
          return HttpResponse.json(createdDatabase, { status: 201 });
        }),
        configuration.handler,
        mockApi.post('/logical-backups', 202, {
          id: 'backup-id',
          databaseId: createdDatabase.id,
          storageId: archiveStorage.id,
          status: 'IN_PROGRESS',
          sizeMb: 0,
          rawDatabaseSizeMb: 0,
          durationMs: 0,
          encryption: 'NONE',
          verificationStatus: 'NOT_VERIFIED',
          createdAt: workspace.createdAt,
        }),
        mockApi.get('/logical-backups', 200, { items: [], totalUsageMb: 0 }),
        mockApi.get('/backup-configs/{databaseId}', 200, savedConfig),
      );
      const listPath = `/workspaces/${workspace.id}/databases`;
      window.history.replaceState(
        null,
        '',
        isEmptyWorkspace ? listPath : `${listPath}/${otherDatabase.id}/backups`,
      );
      if (!isEmptyWorkspace)
        window.history.pushState({ key: 'source-database' }, '', `${databasePath}/backups`);
      const application = renderApp();
      try {
        fireEvent.click(await screen.findByRole('button', { name: 'Add database' }));
        fireEvent.change(await screen.findByPlaceholderText('My favourite DB'), {
          target: { value: createdDatabase.name },
        });
        fireEvent.click(screen.getByText('MySQL'));
        fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
        fireEvent.change(await screen.findByPlaceholderText(/host$/), {
          target: { value: 'db.example.com' },
        });
        fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'backup' } });
        fireEvent.change(screen.getByPlaceholderText(/password$/), { target: { value: 'secret' } });
        fireEvent.change(screen.getByPlaceholderText(/database name$/), {
          target: { value: 'shop' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
        fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
        const storageLabel = await screen.findByText('Storage');
        fireEvent.click(storageLabel.parentElement!.querySelector('[role="combobox"]')!);
        fireEvent.click(await screen.findByRole('option', { name: archiveStorage.name }));
        fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
        fireEvent.click(await screen.findByRole('button', { name: 'Complete' }));
        await waitFor(() => expect(configurationSaves).toHaveBeenCalledTimes(1));

        if (!isEmptyWorkspace) await act(async () => window.history.back());
        if (shouldReturnToSource) {
          await waitFor(() =>
            expect(window.location.pathname).toBe(`${listPath}/${otherDatabase.id}/backups`),
          );
          await act(async () => window.history.forward());
        }
        const retainedPath = isEmptyWorkspace
          ? listPath
          : shouldReturnToSource
            ? `${databasePath}/backups`
            : `${listPath}/${otherDatabase.id}/backups`;
        await waitFor(() => expect(window.location.pathname).toBe(retainedPath));
        const previousReadCount = collectionReads.mock.calls.length;
        fireEvent(document, new Event('visibilitychange'));
        await waitFor(() =>
          expect(collectionReads.mock.calls.length).toBeGreaterThan(previousReadCount),
        );
        await screen.findByRole('link', { name: createdDatabase.name, hidden: true });
        expect(window.location.pathname).toBe(retainedPath);
        expect(
          server
            .getRequests()
            .some((request) => new URL(request.url).pathname === '/api/v1/databases/created-id'),
        ).toBe(false);

        const pushHistory = vi.spyOn(window.history, 'pushState');
        const replaceHistory = vi.spyOn(window.history, 'replaceState');
        await releaseAndSettle(configurationSaves, () =>
          configuration.respond(HttpResponse.json(savedConfig)),
        );
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
        const destination = isEmptyWorkspace
          ? `${listPath}/${createdDatabase.id}/backups`
          : retainedPath;
        await waitFor(() => expect(window.location.pathname).toBe(destination));
        expect(
          screen.getByRole('link', { name: createdDatabase.name }).getAttribute('aria-current'),
        ).toBe(String(isEmptyWorkspace));
        expect(pushHistory).toHaveBeenCalledTimes(isEmptyWorkspace ? 1 : 0);
        expect(replaceHistory).not.toHaveBeenCalled();
        if (isEmptyWorkspace)
          expect(await within(screen.getByRole('table')).findByText('No backups yet')).toBeTruthy();
      } finally {
        application.unmount();
        configuration.cancel();
      }
    },
  );

  it('shows reconciliation recovery before a pending sibling read settles', async () => {
    const replacementDatabase = { ...database, id: 'replacement-id', name: 'Replacement' };
    prepareNavigation([database, replacementDatabase]);
    const pendingConfigs = createPendingApiResponse();
    const currentCollection = createPendingApiResponse();
    const databaseReads = vi.spyOn(databaseApi, 'listAllDatabases');
    let collectionReads = 0;
    let configReads = 0;
    server.use(
      http.get('*/api/v1/databases', () =>
        ++collectionReads === 1
          ? HttpResponse.json({ items: [database, replacementDatabase] })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
      ),
      http.get('*/api/v1/backup-configs', () =>
        ++configReads === 2 ? pendingConfigs.response : HttpResponse.json({ items: [] }),
      ),
      mockApi.get('/databases/{databaseId}', 404, buildProblem('not_found', 404)),
    );
    window.history.replaceState(null, '', `${databasePath}/config`);
    const application = renderApp();
    try {
      await waitFor(() => expect(configReads).toBe(2));
      const retry = await screen.findByRole('button', { name: 'Retry' });
      expect(window.location.pathname).toBe(`${databasePath}/config`);
      expect(screen.queryByRole('tab')).toBeNull();

      server.use(
        http.get('*/api/v1/databases', () => currentCollection.response),
        mockApi.get('/databases/{databaseId}', 200, replacementDatabase),
      );

      fireEvent.click(retry);
      await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(3));
      expect(window.location.pathname).toBe(`${databasePath}/config`);

      await releaseAndSettle(
        databaseReads,
        () => currentCollection.respond(HttpResponse.json({ items: [replacementDatabase] })),
        2,
      );
      await screen.findByRole('tab', { name: 'Config' });
      expect(window.location.pathname).toBe(
        `/workspaces/${workspace.id}/databases/${replacementDatabase.id}/config`,
      );
    } finally {
      application.unmount();
      pendingConfigs.cancel();
      currentCollection.cancel();
    }
  });

  it('replaces a tabless database address with the addressed database backups', async () => {
    const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
    prepareNavigation([otherDatabase, database]);
    localStorage.setItem(`selectedDatabaseId_${workspace.id}`, otherDatabase.id);
    window.history.replaceState(null, '', databasePath);
    const pushHistory = vi.spyOn(window.history, 'pushState');
    const replaceHistory = vi.spyOn(window.history, 'replaceState');
    renderApp();

    expect(
      (await screen.findByRole('tab', { name: 'Backups' })).getAttribute('aria-selected'),
    ).toBe('true');
    expect(window.location.pathname).toBe(`${databasePath}/backups`);
    expect(screen.getByRole('link', { name: database.name }).getAttribute('aria-current')).toBe(
      'true',
    );
    expect(pushHistory).not.toHaveBeenCalled();
    expect(replaceHistory).toHaveBeenCalledWith(expect.anything(), '', `${databasePath}/backups`);
  });

  it('preserves the addressed database and tab when the viewport crosses the mobile boundary', async () => {
    const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
    prepareNavigation([otherDatabase, database]);
    localStorage.setItem(`selectedDatabaseId_${workspace.id}`, otherDatabase.id);
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    await screen.findByRole('tab', { name: 'Config' });

    for (const width of [320, 1280]) {
      vi.stubGlobal('innerWidth', width);
      fireEvent(window, new Event('resize'));
      await waitFor(() =>
        expect(screen.queryByRole('button', { name: '← Back to databases' }) !== null).toBe(
          width === 320,
        ),
      );
      expect(screen.getByRole('tab', { name: 'Config' }).getAttribute('aria-selected')).toBe(
        'true',
      );
      expect(window.location.pathname).toBe(`${databasePath}/config`);
      expect(
        server
          .getRequests()
          .some(
            (request) => new URL(request.url).pathname === `/api/v1/databases/${otherDatabase.id}`,
          ),
      ).toBe(false);
    }
  });

  it('opens a mobile database link over another remembered database and returns to its list', async () => {
    vi.stubGlobal('innerWidth', 320);
    const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
    prepareNavigation([otherDatabase, database]);
    localStorage.setItem(`selectedDatabaseId_${workspace.id}`, otherDatabase.id);
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();

    expect((await screen.findByRole('tab', { name: 'Config' })).getAttribute('aria-selected')).toBe(
      'true',
    );
    expect(await screen.findByRole('button', { name: 'Copy' })).toBeTruthy();
    expect(window.location.pathname).toBe(`${databasePath}/config`);
    expect(
      server
        .getRequests()
        .some(
          (request) => new URL(request.url).pathname === `/api/v1/databases/${otherDatabase.id}`,
        ),
    ).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: '← Back to databases' }));
    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${workspace.id}/databases`),
    );
    expect(await screen.findByRole('link', { name: database.name })).toBeTruthy();
    expect(screen.getByRole('link', { name: otherDatabase.name })).toBeTruthy();
    expect(screen.queryByRole('tab')).toBeNull();
  });

  it.each([200, 403, 404])(
    'preserves the new database tab after an obsolete detail response with HTTP %i',
    async (status) => {
      const logicalDatabase = {
        ...database,
        settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
      };
      prepareNavigation([logicalDatabase]);
      const detailReads = vi.spyOn(databaseApi, 'getDatabase');
      const oldDetail = createPendingApiResponse();
      let hasHeldDetail = false;
      server.use(
        http.get('*/api/v1/databases/:databaseId', () => {
          if (!hasHeldDetail) {
            hasHeldDetail = true;
            return oldDetail.response;
          }
          return HttpResponse.json(logicalDatabase);
        }),
        mockApi.get('/verifications', 200, { items: [] }),
        mockApi.get('/verification-agent-counts', 200, { agentCount: 0 }),
      );
      window.history.replaceState(null, '', `${databasePath}/verifications`);
      window.history.pushState({ key: 'config-entry' }, '', `${databasePath}/config`);
      const application = renderApp();

      try {
        await waitFor(() => expect(detailReads).toHaveBeenCalledTimes(1));
        await act(async () => window.history.back());
        await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/verifications`));
        await releaseAndSettle(detailReads, () =>
          oldDetail.respond(
            status === 200
              ? HttpResponse.json(database)
              : HttpResponse.json(buildProblem('not_found', status), { status }),
          ),
        );

        expect(window.location.pathname).toBe(`${databasePath}/verifications`);
        expect(
          (await screen.findByRole('tab', { name: 'Verifications' })).getAttribute('aria-selected'),
        ).toBe('true');
        expect(detailReads).toHaveBeenCalledTimes(2);
        expect(
          server
            .getRequests()
            .filter((request) => new URL(request.url).pathname === '/api/v1/databases'),
        ).toHaveLength(1);
      } finally {
        application.unmount();
        oldDetail.cancel();
      }
    },
  );

  it.each(['network', 'server'])(
    'retains the refused workspace address through pending reconciliation and a %s failure until retry succeeds',
    async (failure) => {
      const otherWorkspace = { ...workspace, id: 'other-team', name: 'Other team' };
      prepareNavigation();
      server.use(mockApi.get('/workspaces', 200, { items: [workspace, otherWorkspace] }));
      window.history.replaceState(null, '', `${databasePath}/config`);
      const application = renderApp();
      await screen.findByRole('tab', { name: 'Config' });
      const workspaceReads = vi.spyOn(workspaceApi, 'listAllWorkspaces');
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(
        reconciliation.handler,
        mockApi.get('/databases', 403, buildProblem('permission_denied', 403)),
      );

      try {
        fireEvent(document, new Event('visibilitychange'));
        await waitFor(() => expect(workspaceReads).toHaveBeenCalledTimes(1));
        expect(screen.queryByRole('tab', { name: 'Config' })).toBeNull();
        expect(screen.queryByRole('link', { name: database.name })).toBeNull();
        expect(window.location.pathname).toBe(`${databasePath}/config`);

        await releaseAndSettle(workspaceReads, () =>
          reconciliation.respond(
            failure === 'network'
              ? HttpResponse.error()
              : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
          ),
        );

        const retry = await screen.findByRole('button', { name: 'Retry' });
        expect(window.location.pathname).toBe(`${databasePath}/config`);
        expect(screen.queryByRole('tab', { name: 'Config' })).toBeNull();
        server.use(
          mockApi.get('/workspaces', 200, { items: [otherWorkspace] }),
          mockApi.get('/databases', 200, { items: [] }),
        );
        fireEvent.click(retry);

        await screen.findByRole('button', { name: 'Add database' });
        expect(window.location.pathname).toBe(`/workspaces/${otherWorkspace.id}/databases`);
        expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      } finally {
        application.unmount();
        reconciliation.cancel();
      }
    },
  );

  it('reconciles a late transfer immediately after browser history changes the database tab', async () => {
    const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
    const targetWorkspace = { ...workspace, id: 'target-team', name: 'Target team' };
    const targetStorage = {
      id: 'target-storage',
      workspaceId: targetWorkspace.id,
      name: 'Target archive',
      settings: { type: 'LOCAL' },
    };
    const databases = [database, otherDatabase];
    prepareNavigation(databases);
    const transfer = mockApi.pending('post', '/databases/{databaseId}/transfer');
    const transferRequests = vi.spyOn(backupConfigApi, 'transferDatabase');
    server.use(
      transfer.handler,
      mockApi.get('/workspaces', 200, { items: [workspace, targetWorkspace] }),
      http.get('*/api/v1/storages', ({ request }) =>
        HttpResponse.json({
          items:
            new URL(request.url).searchParams.get('workspaceId') === targetWorkspace.id
              ? [targetStorage]
              : [],
        }),
      ),
    );
    window.history.replaceState(null, '', `${databasePath}/backups`);
    const application = renderApp();
    fireEvent.click(await screen.findByRole('tab', { name: 'Config' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Transfer' }));
    const dialog = within(await screen.findByRole('dialog'));
    fireEvent.click(dialog.getByRole('combobox'));
    fireEvent.click(await screen.findByRole('option', { name: targetWorkspace.name }));
    await waitFor(() => expect(dialog.getAllByRole('combobox')).toHaveLength(2));
    fireEvent.click(dialog.getAllByRole('combobox')[1]);
    fireEvent.click(await screen.findByRole('option', { name: targetStorage.name }));
    fireEvent.click(dialog.getByRole('button', { name: 'Transfer' }));

    try {
      await waitFor(() => expect(transferRequests).toHaveBeenCalledTimes(1));
      await act(async () => window.history.back());
      await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/backups`));
      expect(screen.queryByRole('dialog')).toBeNull();
      const pushHistory = vi.spyOn(window.history, 'pushState');
      const replaceHistory = vi.spyOn(window.history, 'replaceState');
      databases.splice(0, 1);

      await releaseAndSettle(transferRequests, () =>
        transfer.respond(new HttpResponse(null, { status: 204 })),
      );

      expect(window.location.pathname).toBe(
        `/workspaces/${workspace.id}/databases/${otherDatabase.id}/backups`,
      );
      expect(screen.queryByRole('link', { name: database.name })).toBeNull();
      expect(
        screen.getByRole('link', { name: otherDatabase.name }).getAttribute('aria-current'),
      ).toBe('true');
      expect(pushHistory).not.toHaveBeenCalled();
      expect(replaceHistory).toHaveBeenCalledWith(
        expect.anything(),
        '',
        `/workspaces/${workspace.id}/databases/${otherDatabase.id}/backups`,
      );
      expect(screen.queryByText('Database transferred successfully!')).toBeNull();
    } finally {
      application.unmount();
      transfer.cancel();
    }
  });

  it.each([200, 403, 404])(
    'rejects an obsolete collection HTTP %i after selecting another database',
    async (status) => {
      const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
      prepareNavigation([database, otherDatabase]);
      window.history.replaceState(null, '', `${databasePath}/config`);
      const application = renderApp();
      await screen.findByRole('tab', { name: 'Config' });
      const collectionReads = vi.spyOn(databaseApi, 'listAllDatabases');
      const pendingCollection = createPendingApiResponse();
      let hasHeldRead = false;
      server.use(
        http.get('*/api/v1/databases', () => {
          if (!hasHeldRead) {
            hasHeldRead = true;
            return pendingCollection.response;
          }
          return HttpResponse.json({ items: [database, otherDatabase] });
        }),
      );

      try {
        fireEvent(document, new Event('visibilitychange'));
        await waitFor(() => expect(collectionReads).toHaveBeenCalledTimes(1));
        fireEvent.click(screen.getByRole('link', { name: otherDatabase.name }));
        const otherPath = `/workspaces/${workspace.id}/databases/${otherDatabase.id}/config`;
        await waitFor(() => expect(window.location.pathname).toBe(otherPath));

        await releaseAndSettle(
          collectionReads,
          () =>
            pendingCollection.respond(
              status === 200
                ? HttpResponse.json({ items: [database] })
                : HttpResponse.json(
                    buildProblem(status === 403 ? 'permission_denied' : 'not_found', status),
                    { status },
                  ),
            ),
          0,
        );

        expect(window.location.pathname).toBe(otherPath);
        expect(
          screen.getByRole('link', { name: otherDatabase.name }).getAttribute('aria-current'),
        ).toBe('true');
        expect(collectionReads).toHaveBeenCalledTimes(2);
        expect(
          server
            .getRequests()
            .filter((request) => new URL(request.url).pathname === '/api/v1/workspaces'),
        ).toHaveLength(1);
      } finally {
        application.unmount();
        pendingCollection.cancel();
      }
    },
  );

  it.each([403, 404])(
    'exhausts rejected workspaces when the accessible list stays stale after HTTP %i',
    async (status) => {
      prepareNavigation();
      const accessibleWorkspaces = vi.spyOn(workspaceApi, 'listAllWorkspaces');
      const reconciliation = createPendingApiResponse();
      const stalledCollection = createPendingApiResponse();
      let workspaceReads = 0;
      let collectionReads = 0;
      server.use(
        http.get('*/api/v1/workspaces', () =>
          ++workspaceReads === 1
            ? HttpResponse.json({ items: [workspace] })
            : reconciliation.response,
        ),
        http.get('*/api/v1/databases', () => {
          collectionReads += 1;
          return collectionReads === 1
            ? HttpResponse.json(buildProblem('not_found', status), { status })
            : stalledCollection.response;
        }),
      );
      window.history.replaceState(null, '', `${databasePath}/config`);
      const application = renderApp();
      try {
        await waitFor(() => expect(accessibleWorkspaces).toHaveBeenCalledTimes(2));
        await releaseAndSettle(
          accessibleWorkspaces,
          () => reconciliation.respond(HttpResponse.json({ items: [workspace] })),
          1,
        );

        await screen.findByRole('link', { name: 'Profile' });
        expect(window.location.pathname).toBe('/');
        expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
        expect(collectionReads).toBe(1);
      } finally {
        application.unmount();
        reconciliation.cancel();
        stalledCollection.cancel();
      }
    },
  );

  it('uses the latest selected database as the default within the mounted workspace', async () => {
    const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
    prepareNavigation([database, otherDatabase]);
    localStorage.setItem(`selectedDatabaseId_${workspace.id}`, database.id);
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    await screen.findByRole('tab', { name: 'Config' });
    fireEvent.click(screen.getByRole('link', { name: otherDatabase.name }));
    await waitFor(() => expect(window.location.pathname).toContain(otherDatabase.id));
    await act(async () => {
      window.history.pushState(
        { key: 'unspecified-database' },
        '',
        `/workspaces/${workspace.id}/databases`,
      );
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    await waitFor(() =>
      expect(window.location.pathname).toBe(
        `/workspaces/${workspace.id}/databases/${otherDatabase.id}/backups`,
      ),
    );
    expect(
      screen.getByRole('link', { name: otherDatabase.name }).getAttribute('aria-current'),
    ).toBe('true');
  });

  it.each(['password', 'verification'] as const)(
    'abandons pending %s sign-in when browser history opens another deep link',
    async (step) => {
      prepareNavigation();
      invalidateSession();
      const password = mockApi.pending('post', '/sessions');
      const verification = mockApi.pending('post', '/pending-sign-ins/{pendingSignInId}/verify');
      server.use(
        step === 'password'
          ? password.handler
          : mockApi.post('/sessions', 200, {
              outcome: 'CODE_REQUIRED',
              pendingSignInId: 'pending-id',
              email: 'owner@example.com',
            }),
        verification.handler,
      );
      const request = step === 'password' ? password : verification;
      const signIn = vi.spyOn(userApi, step === 'password' ? 'signIn' : 'verifySignInCode');
      window.history.replaceState(null, '', '/profile');
      window.history.pushState({ key: 'password-link' }, '', `${databasePath}/config`);
      const application = renderApp();
      fireEvent.change(await screen.findByPlaceholderText('your@email.com'), {
        target: { value: 'owner@example.com' },
      });
      fireEvent.change(screen.getByPlaceholderText('********'), {
        target: { value: 'password-value' },
      });
      fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
      if (step === 'verification') {
        fireEvent.change(await screen.findByPlaceholderText('123456'), {
          target: { value: '123456' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
      }
      try {
        await waitFor(() => expect(signIn).toHaveBeenCalledTimes(1));
        await act(async () => window.history.back());
        await waitFor(() => expect(window.location.pathname).toBe('/profile'));

        await releaseAndSettle(signIn, () =>
          request.respond(
            HttpResponse.json({
              ...(step === 'password' ? { outcome: 'COMPLETED' } : {}),
              token: 'obsolete-session',
              userId: 'owner-id',
              email: 'owner@example.com',
            }),
          ),
        );

        expect(getSessionSnapshot().token).toBeUndefined();
        expect(window.location.pathname).toBe('/profile');
        expect(await screen.findByPlaceholderText('your@email.com')).toHaveProperty('value', '');
        expect(screen.queryByPlaceholderText('123456')).toBeNull();
      } finally {
        application.unmount();
        password.cancel();
        verification.cancel();
      }
    },
  );

  it('invalidates an older OAuth destination when a password sign-in takes ownership', async () => {
    prepareNavigation();
    invalidateSession();
    const password = mockApi.pending('post', '/sessions');
    server.use(password.handler);
    window.history.replaceState(null, '', '/profile');
    const application = renderApp();
    await screen.findByPlaceholderText('your@email.com');
    const state = 'e'.repeat(64);
    sessionStorage.setItem(
      'databasus_oauth_sign_in_attempt',
      JSON.stringify({
        state,
        provider: 'github',
        createdAt: Date.now(),
        redirectUri: `${window.location.origin}/auth/callback`,
        returnPath: `${databasePath}/config`,
      }),
    );
    fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
      target: { value: 'owner@example.com' },
    });
    fireEvent.change(screen.getByPlaceholderText('********'), {
      target: { value: 'password-value' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));

    try {
      await waitFor(() =>
        expect(
          server
            .getRequests()
            .some((request) => request.method === 'POST' && request.url.endsWith('/sessions')),
        ).toBe(true),
      );
      expect(sessionStorage.getItem('databasus_oauth_sign_in_attempt')).toBeNull();
      await act(async () => {
        window.history.pushState(
          { key: 'abandoned-callback' },
          '',
          `/auth/callback?state=${state}&code=abandoned-code`,
        );
        window.dispatchEvent(new PopStateEvent('popstate'));
      });

      await screen.findByRole('button', { name: 'Return to sign in' });
      expect(getSessionSnapshot().token).toBeUndefined();
      expect(server.getRequests().some((request) => request.url.endsWith('/github-sessions'))).toBe(
        false,
      );
    } finally {
      application.unmount();
      password.cancel();
    }
  });

  it('restores the previous workspace and Storages address with Back and Forward', async () => {
    const otherWorkspace = { ...workspace, id: 'other-team', name: 'Other team' };
    prepareNavigation();
    server.use(mockApi.get('/workspaces', 200, { items: [workspace, otherWorkspace] }));
    window.history.replaceState(null, '', `/workspaces/${workspace.id}/storages`);
    renderApp();
    await screen.findByRole('button', { name: 'Add storage' });
    fireEvent.click(screen.getByRole('combobox', { name: 'Selected workspace' }));
    fireEvent.click(await screen.findByRole('option', { name: otherWorkspace.name }));
    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${otherWorkspace.id}/storages`),
    );

    await act(async () => window.history.back());

    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${workspace.id}/storages`),
    );
    expect(screen.getByRole('combobox', { name: 'Selected workspace' }).textContent).toContain(
      workspace.name,
    );
    expect(await screen.findByRole('button', { name: 'Add storage' })).toBeTruthy();
    await act(async () => window.history.forward());
    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${otherWorkspace.id}/storages`),
    );
    expect(screen.getByRole('combobox', { name: 'Selected workspace' }).textContent).toContain(
      otherWorkspace.name,
    );
    expect(await screen.findByRole('button', { name: 'Add storage' })).toBeTruthy();
  });

  it('skips a rejected database address with Back and restores its canonical fallback with Forward', async () => {
    prepareNavigation();
    window.history.replaceState(null, '', `/workspaces/${workspace.id}/storages`);
    renderApp();
    await screen.findByRole('button', { name: 'Add storage' });
    await act(async () => {
      window.history.pushState(
        { key: 'missing-database' },
        '',
        `/workspaces/${workspace.id}/databases/missing-id/config`,
      );
      window.dispatchEvent(new PopStateEvent('popstate'));
    });
    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/config`));
    await screen.findByRole('tab', { name: 'Config' });

    await act(async () => window.history.back());

    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${workspace.id}/storages`),
    );
    expect(await screen.findByRole('button', { name: 'Add storage' })).toBeTruthy();
    await act(async () => window.history.forward());
    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/config`));
    expect((await screen.findByRole('tab', { name: 'Config' })).getAttribute('aria-selected')).toBe(
      'true',
    );
  });

  it('clears the old deep link before another sign-in after active-session refusal', async () => {
    prepareNavigation();
    server.use(
      mockApi.get('/databases/{databaseId}', 401, buildProblem('unauthenticated', 401)),
      mockApi.post('/sessions', 200, {
        outcome: 'COMPLETED',
        token: 'replacement-session',
        userId: 'owner-id',
        email: 'owner@example.com',
      }),
    );
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    await screen.findByRole('button', { name: 'Sign in' });
    expect(window.location.pathname).toBe('/');
    server.use(mockApi.get('/databases/{databaseId}', 200, database));

    fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
      target: { value: 'owner@example.com' },
    });
    fireEvent.change(screen.getByPlaceholderText('********'), {
      target: { value: 'password-value' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(
      (await screen.findByRole('tab', { name: 'Backups' })).getAttribute('aria-selected'),
    ).toBe('true');
    expect(window.location.pathname).toBe(`${databasePath}/backups`);
  });

  it('keeps history unchanged when the displayed section, database and tab are selected again', async () => {
    prepareNavigation();
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    await screen.findByRole('tab', { name: 'Config' });
    const historyLength = window.history.length;

    fireEvent.click(screen.getByRole('tab', { name: 'Config' }));
    fireEvent.click(screen.getByRole('link', { name: database.name }));
    fireEvent.click(screen.getByRole('link', { name: 'Databases' }));

    expect(window.location.pathname).toBe(`${databasePath}/config`);
    expect(window.history.length).toBe(historyLength);
    expect(screen.getByRole('tab', { name: 'Config' }).getAttribute('aria-selected')).toBe('true');
  });

  it('retries a previously refused database when a new navigation returns after access recovers', async () => {
    const replacementDatabase = { ...database, id: 'replacement-id', name: 'Replacement' };
    prepareNavigation([database, replacementDatabase]);
    server.use(
      http.get('*/api/v1/databases/:databaseId', ({ params }) =>
        params.databaseId === database.id
          ? HttpResponse.json(buildProblem('not_found', 404), { status: 404 })
          : HttpResponse.json(replacementDatabase),
      ),
    );
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    await waitFor(() => expect(window.location.pathname).toContain(replacementDatabase.id));
    await screen.findByRole('tab', { name: 'Config' });
    server.use(mockApi.get('/databases/{databaseId}', 200, database));

    await act(async () => {
      window.history.pushState({ key: 'recovered-link' }, '', `${databasePath}/config`);
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    expect(await screen.findByRole('link', { name: database.name })).toBeTruthy();
    expect((await screen.findByRole('tab', { name: 'Config' })).getAttribute('aria-selected')).toBe(
      'true',
    );
    expect(window.location.pathname).toBe(`${databasePath}/config`);
    expect(screen.getByRole('link', { name: database.name }).getAttribute('aria-current')).toBe(
      'true',
    );
  });

  it('uses an explicitly addressed workspace and database instead of remembered selections', async () => {
    const otherWorkspace = { ...workspace, id: 'other-team', name: 'Other team' };
    const otherDatabase = { ...database, id: 'other-id', workspaceId: otherWorkspace.id };
    const rememberedDatabase = {
      ...otherDatabase,
      id: 'remembered-id',
      name: 'Remembered database',
    };
    prepareNavigation([database, rememberedDatabase, otherDatabase]);
    server.use(mockApi.get('/workspaces', 200, { items: [workspace, otherWorkspace] }));
    localStorage.setItem('selected_workspace_id', workspace.id);
    localStorage.setItem(`selectedDatabaseId_${otherWorkspace.id}`, rememberedDatabase.id);
    const path = `/workspaces/${otherWorkspace.id}/databases/${otherDatabase.id}/config`;
    window.history.replaceState(null, '', path);

    renderApp();

    expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
    expect(window.location.pathname).toBe(path);
    expect(screen.getByRole('combobox', { name: 'Selected workspace' }).textContent).toContain(
      otherWorkspace.name,
    );
    expect(localStorage.getItem('selected_workspace_id')).toBe(otherWorkspace.id);
  });

  it('keeps the address and offers recovery when availability reconciliation fails', async () => {
    const replacementDatabase = { ...database, id: 'replacement-id', name: 'Replacement' };
    prepareNavigation([database, replacementDatabase]);
    let collectionReads = 0;
    server.use(
      http.get('*/api/v1/databases', () => {
        collectionReads += 1;
        return collectionReads === 1
          ? HttpResponse.json({ items: [database, replacementDatabase] })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
      }),
      mockApi.get('/databases/{databaseId}', 404, buildProblem('not_found', 404)),
    );
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    const retry = await screen.findByRole('button', { name: 'Retry' });
    expect(window.location.pathname).toBe(`${databasePath}/config`);
    expect(screen.queryByRole('tab')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Copy' })).toBeNull();
    server.use(
      mockApi.get('/databases', 200, { items: [replacementDatabase] }),
      mockApi.get('/databases/{databaseId}', 200, replacementDatabase),
    );

    fireEvent.click(retry);

    expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
    expect(window.location.pathname).toBe(
      `/workspaces/${workspace.id}/databases/${replacementDatabase.id}/config`,
    );
  });

  it('hides details transferred to another workspace after collection validation', async () => {
    const replacementDatabase = { ...database, id: 'replacement-id', name: 'Replacement' };
    const foreignDatabase = { ...database, workspaceId: 'other-team', name: 'Private database' };
    prepareNavigation([database, replacementDatabase]);
    server.use(
      http.get('*/api/v1/databases/:databaseId', ({ params }) =>
        HttpResponse.json(
          params.databaseId === database.id ? foreignDatabase : replacementDatabase,
        ),
      ),
    );
    window.history.replaceState(null, '', `${databasePath}/config`);

    renderApp();

    expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
    expect(window.location.pathname).toBe(
      `/workspaces/${workspace.id}/databases/${replacementDatabase.id}/config`,
    );
    expect(screen.queryByText(foreignDatabase.name)).toBeNull();
  });

  it.each(['tab', 'database-return', 'section-return'] as const)(
    'accepts a late copy selection only within the same database lifetime after %s navigation',
    async (navigation) => {
      const otherDatabase = { ...database, id: 'other-id', name: 'Other' };
      const copiedDatabase = { ...database, id: 'copy-id', name: 'Copied Shop' };
      const databases = [database, otherDatabase];
      prepareNavigation(databases);
      const copy = mockApi.pending('post', '/databases/{databaseId}/copy');
      server.use(copy.handler);
      window.history.replaceState(null, '', `${databasePath}/config`);
      const application = renderApp();
      fireEvent.click(await screen.findByRole('button', { name: 'Copy' }));
      fireEvent.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
      );
      await waitFor(() =>
        expect(
          server.getRequests().some((request) => request.url.endsWith(`/${database.id}/copy`)),
        ).toBe(true),
      );

      try {
        if (navigation === 'tab') fireEvent.click(screen.getByRole('tab', { name: 'Backups' }));
        else if (navigation === 'database-return') {
          fireEvent.click(screen.getByRole('link', { name: otherDatabase.name }));
          await waitFor(() => expect(window.location.pathname).toContain(otherDatabase.id));
          fireEvent.click(screen.getByRole('link', { name: database.name }));
          await screen.findByRole('button', { name: 'Copy' });
        } else {
          fireEvent.click(screen.getByRole('link', { name: 'Storages' }));
          await screen.findByRole('button', { name: 'Add storage' });
          fireEvent.click(screen.getByRole('link', { name: 'Databases' }));
          await screen.findByRole('tab', { name: 'Backups' });
        }

        databases.push(copiedDatabase);
        await finishApiResponse(server, `/api/v1/databases/${database.id}/copy`, () =>
          copy.respond(HttpResponse.json(copiedDatabase, { status: 201 })),
        );
        if (navigation === 'section-return') fireEvent(document, new Event('visibilitychange'));

        expect(await screen.findByRole('link', { name: copiedDatabase.name })).toBeTruthy();
        const selectedDatabaseId = navigation === 'tab' ? copiedDatabase.id : database.id;
        const tab = navigation === 'database-return' ? 'config' : 'backups';
        expect(window.location.pathname).toBe(
          `/workspaces/${workspace.id}/databases/${selectedDatabaseId}/${tab}`,
        );
        expect(screen.queryByText('Database copied successfully!')).toBeNull();
      } finally {
        application.unmount();
        copy.cancel();
      }
    },
  );

  it.each(
    [false, true].flatMap((hasSecondFactor) =>
      [false, true].map((hasExtraAddressData) => ({ hasSecondFactor, hasExtraAddressData })),
    ),
  )(
    'keeps a safe signed-out deep link with second factor=$hasSecondFactor and extra address data=$hasExtraAddressData',
    async ({ hasSecondFactor, hasExtraAddressData }) => {
      prepareNavigation();
      invalidateSession();
      server.use(
        mockApi.post(
          '/sessions',
          200,
          hasSecondFactor
            ? {
                outcome: 'CODE_REQUIRED',
                pendingSignInId: 'pending-id',
                email: 'owner@example.com',
              }
            : {
                outcome: 'COMPLETED',
                token: 'signed-in-session',
                userId: 'owner-id',
                email: 'owner@example.com',
              },
        ),
        mockApi.post('/pending-sign-ins/{pendingSignInId}/verify', 200, {
          token: 'signed-in-session',
          userId: 'owner-id',
          email: 'owner@example.com',
        }),
      );
      window.history.replaceState(
        null,
        '',
        `${databasePath}/config${hasExtraAddressData ? '?token=secret#fragment' : ''}`,
      );
      renderApp();
      fireEvent.change(await screen.findByPlaceholderText('your@email.com'), {
        target: { value: 'owner@example.com' },
      });
      fireEvent.change(screen.getByPlaceholderText('********'), {
        target: { value: 'password-value' },
      });
      fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
      if (hasSecondFactor) {
        fireEvent.change(await screen.findByPlaceholderText('123456'), {
          target: { value: '123456' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
      }

      expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
      expect(window.location.pathname).toBe(`${databasePath}/config`);
      expect(window.location.search).toBe('');
      expect(window.location.hash).toBe('');
    },
  );

  it.each(['github', 'google'] as const)(
    'returns from %s sign-in to the database link and skips the callback in browser history',
    async (provider) => {
      prepareNavigation();
      invalidateSession();
      const state = 'b'.repeat(64);
      sessionStorage.setItem(
        'databasus_oauth_sign_in_attempt',
        JSON.stringify({
          state,
          provider,
          createdAt: Date.now(),
          redirectUri: `${window.location.origin}/auth/callback`,
          returnPath: `${databasePath}/config`,
        }),
      );
      server.use(
        mockApi.post(provider === 'github' ? '/github-sessions' : '/google-sessions', 200, {
          token: 'oauth-session',
          userId: 'owner-id',
          email: 'owner@example.com',
        }),
      );
      const previousPath = `/workspaces/${workspace.id}/storages`;
      window.history.replaceState({ idx: 0, key: 'before-oauth' }, '', previousPath);
      window.history.pushState(
        { idx: 1, key: 'oauth-callback' },
        '',
        `/auth/callback?state=${state}&code=private-code`,
      );

      renderApp({ isStrictMode: true });

      expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
      expect(window.location.pathname).toBe(`${databasePath}/config`);
      expect(window.location.search).toBe('');
      expect(sessionStorage.getItem('databasus_oauth_sign_in_attempt')).toBeNull();

      await act(async () => window.history.back());

      await waitFor(() => expect(window.location.pathname).toBe(previousPath));
      expect(await screen.findByRole('button', { name: 'Add storage' })).toBeTruthy();

      await act(async () => window.history.forward());

      expect(
        (await screen.findByRole('tab', { name: 'Config' })).getAttribute('aria-selected'),
      ).toBe('true');
      expect(window.location.pathname).toBe(`${databasePath}/config`);
      expect(
        server
          .getRequests()
          .filter((request) => new URL(request.url).pathname === `/api/v1/${provider}-sessions`),
      ).toHaveLength(1);
    },
  );

  it.each([
    'https://outside.example/profile',
    '//outside.example/profile',
    '/auth/callback',
    '/storages/google-oauth',
    '/profile?token=secret',
    '/%2f%2foutside.example',
    '/workspaces/%252foutside/databases',
  ])('rejects an unsafe OAuth return destination %s', async (returnPath) => {
    prepareNavigation();
    invalidateSession();
    const state = 'c'.repeat(64);
    sessionStorage.setItem(
      'databasus_oauth_sign_in_attempt',
      JSON.stringify({
        state,
        provider: 'github',
        createdAt: Date.now(),
        redirectUri: `${window.location.origin}/auth/callback`,
        returnPath,
      }),
    );
    server.use(
      mockApi.post('/github-sessions', 200, {
        token: 'oauth-session',
        userId: 'owner-id',
        email: 'owner@example.com',
      }),
    );
    window.history.replaceState(null, '', `/auth/callback?state=${state}&code=private-code`);
    renderApp();

    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/backups`));
    expect(window.location.origin).toBe(new URL(initialAddress).origin);
  });

  it('restores the selected application section after a reload', async () => {
    prepareNavigation();
    window.history.replaceState(null, '', `/workspaces/${workspace.id}/storages`);
    const application = renderApp();
    await screen.findByRole('button', { name: 'Add storage' });
    application.unmount();

    renderApp();

    await screen.findByRole('button', { name: 'Add storage' });
    expect(window.location.pathname).toBe(`/workspaces/${workspace.id}/storages`);
    expect(screen.getByRole('link', { name: 'Storages' }).getAttribute('aria-current')).toBe(
      'page',
    );
  });

  it('opens mobile database details directly and returns to the list without earlier history', async () => {
    vi.stubGlobal('innerWidth', 320);
    prepareNavigation();
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    await screen.findByRole('tab', { name: 'Config' });

    fireEvent.click(screen.getByRole('button', { name: '← Back to databases' }));

    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${workspace.id}/databases`),
    );
    expect(await screen.findByRole('link', { name: database.name })).toBeTruthy();
    expect(screen.queryByRole('tab', { name: 'Config' })).toBeNull();
  });

  it.each([
    '/unknown',
    '/users',
    '/databasus-settings',
    '/workspaces/foreign-team/databases/foreign-id/config',
  ])(
    'replaces an unavailable destination with the default accessible database at %s',
    async (path) => {
      prepareNavigation();
      window.history.replaceState(null, '', path);
      renderApp();

      await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/backups`));
      expect(await screen.findByRole('tab', { name: 'Backups' })).toBeTruthy();
      expect(server.getRequests().some((request) => request.url.includes('foreign-team'))).toBe(
        false,
      );
    },
  );

  it('replaces a missing database with the empty list when the workspace has no databases', async () => {
    prepareNavigation([]);
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();

    await waitFor(() =>
      expect(window.location.pathname).toBe(`/workspaces/${workspace.id}/databases`),
    );
    expect(await screen.findByRole('button', { name: 'Add database' })).toBeTruthy();
    expect(screen.queryByRole('tab')).toBeNull();
  });

  it('resolves to the empty workspace experience when no workspace is accessible', async () => {
    prepareNavigation();
    server.use(mockApi.get('/workspaces', 200, { items: [] }));
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();

    await waitFor(() => expect(window.location.pathname).toBe('/'));
    expect(await screen.findByRole('link', { name: 'Profile' })).toBeTruthy();
    expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
  });

  it.each([403, 404])(
    'reconciles confirmed detail unavailability and excludes rejected candidates after HTTP %i',
    async (status) => {
      const replacementDatabase = { ...database, id: 'replacement-id', name: 'Replacement' };
      prepareNavigation([database, replacementDatabase]);
      server.use(
        http.get('*/api/v1/databases/:databaseId', ({ params }) =>
          params.databaseId === database.id
            ? HttpResponse.json(buildProblem('not_found', status), { status })
            : HttpResponse.json(replacementDatabase),
        ),
      );
      window.history.replaceState(null, '', `${databasePath}/config`);
      renderApp();

      await waitFor(() =>
        expect(window.location.pathname).toBe(
          `/workspaces/${workspace.id}/databases/${replacementDatabase.id}/config`,
        ),
      );
      expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
      expect(
        server
          .getRequests()
          .filter(
            (request) => new URL(request.url).pathname === `/api/v1/databases/${database.id}`,
          ),
      ).toHaveLength(1);
    },
  );

  it('keeps the requested address and retries the same database after a temporary failure', async () => {
    prepareNavigation();
    server.use(mockApi.get('/databases/{databaseId}', 500, buildProblem('internal', 500)));
    window.history.replaceState(null, '', `${databasePath}/config`);
    renderApp();
    const retry = await screen.findByRole('button', { name: 'Retry' });
    expect(window.location.pathname).toBe(`${databasePath}/config`);
    server.use(mockApi.get('/databases/{databaseId}', 200, database));

    fireEvent.click(retry);

    expect(await screen.findByRole('tab', { name: 'Config' })).toBeTruthy();
    expect(window.location.pathname).toBe(`${databasePath}/config`);
  });
  it('restores the selected database tab after a reload', async () => {
    prepareNavigation();
    window.history.replaceState(null, '', `${databasePath}/config`);
    const application = renderApp();

    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Config' }).getAttribute('aria-selected')).toBe(
        'true',
      ),
    );
    expect(window.location.pathname).toBe(`${databasePath}/config`);
    application.unmount();

    renderApp();

    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Config' }).getAttribute('aria-selected')).toBe(
        'true',
      ),
    );
    expect(window.location.pathname).toBe(`${databasePath}/config`);
  });

  it('replaces an unavailable database tab with its backups address', async () => {
    prepareNavigation();
    window.history.replaceState(null, '', `${databasePath}/verifications`);

    renderApp();

    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/backups`));
    expect(
      (await screen.findByRole('tab', { name: 'Backups' })).getAttribute('aria-selected'),
    ).toBe('true');
  });

  it('replaces a foreign database link with a database of the addressed workspace', async () => {
    const foreignDatabase = { ...database, id: 'foreign-id', workspaceId: 'foreign-team' };
    prepareNavigation([database, foreignDatabase]);
    window.history.replaceState(
      null,
      '',
      `/workspaces/${workspace.id}/databases/${foreignDatabase.id}/config`,
    );

    renderApp();

    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/config`));
    expect(
      server
        .getRequests()
        .some((request) =>
          new URL(request.url).pathname.endsWith(`/databases/${foreignDatabase.id}`),
        ),
    ).toBe(false);
  });

  it('returns to the previous database tab with the back button', async () => {
    prepareNavigation();
    window.history.replaceState(null, '', `${databasePath}/backups`);
    renderApp();
    fireEvent.click(await screen.findByRole('tab', { name: 'Config' }));
    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/config`));

    await act(async () => window.history.back());

    await waitFor(() => expect(window.location.pathname).toBe(`${databasePath}/backups`));
    expect(
      (await screen.findByRole('tab', { name: 'Backups' })).getAttribute('aria-selected'),
    ).toBe('true');
  });
});
