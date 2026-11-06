// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { renderApp } from '@/app/ui/renderApp';
import { backupConfigApi } from '@/entities/backup-configs';
import {
  createDefaultDatabaseSettings,
  databaseApi,
  DatabaseType,
  type Database,
} from '@/entities/databases';
import { notifierApi } from '@/entities/notifiers';
import { storageApi } from '@/entities/storages';
import {
  workspaceApi,
  workspaceMemberApi,
  type Workspace,
  type WorkspaceMember,
} from '@/entities/workspaces';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';

const server = setUpMockApi();
const firstWorkspace: Workspace = {
  id: 'first-workspace',
  name: 'First team',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T00:00:00Z',
};
const secondWorkspace: Workspace = {
  ...firstWorkspace,
  id: 'second-workspace',
  name: 'Second team',
};
const thirdWorkspace: Workspace = { ...firstWorkspace, id: 'third-workspace', name: 'Third team' };
const firstWorkspaceDatabase: Database = {
  id: 'first-workspace-database',
  name: 'Production database',
  workspaceId: firstWorkspace.id,
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
  healthStatus: 'AVAILABLE',
};
const openApplication = (
  workspaces: Workspace[],
  rememberedWorkspaceId?: string,
  databases: Database[] = [],
) => {
  activateSession({ token: 'workspace-session', userId: 'owner-id' });

  if (rememberedWorkspaceId) localStorage.setItem('selected_workspace_id', rememberedWorkspaceId);

  server.use(
    mockApi.get('/system/disk-usages', 200, {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    }),
    mockApi.get('/users/{userId}', 200, {
      id: 'owner-id',
      name: 'Owner',
      email: 'owner@example.com',
      role: 'ADMIN',
      status: 'ACTIVE',
      createdAt: firstWorkspace.createdAt,
    }),
    mockApi.get('/instance-settings', 200, {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isEmailConfigured: true,
      isTwoFactorAuthRequired: false,
    }),
    mockApi.get('/workspaces', 200, { items: workspaces }),
    http.get(`*${API_BASE_PATH}/workspaces/:workspaceId`, ({ params }) =>
      HttpResponse.json(workspaces.find((workspace) => workspace.id === params.workspaceId)),
    ),
    http.get(`*${API_BASE_PATH}/databases`, ({ request }) =>
      HttpResponse.json({
        items: databases.filter(
          (database) =>
            database.workspaceId === new URL(request.url).searchParams.get('workspaceId'),
        ),
      }),
    ),
    mockApi.get('/databases/{databaseId}', 200, firstWorkspaceDatabase),
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/logical-backups').handler,
    mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    mockApi.get('/storages', 200, { items: [] }),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/workspace-members', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'v0.0.0' }),
    ),
  );

  return renderApp();
};
const getWorkspaceSelection = () =>
  within(
    screen
      .getByRole('combobox', { name: 'Selected workspace' })
      .closest('.ant-select-selector') as HTMLElement,
  );
const selectWorkspace = async (workspace: Workspace) => {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Selected workspace' }));
  const option = await waitFor(() => {
    const option = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
      (option) => option.textContent === workspace.name,
    );

    expect(option).toBeTruthy();

    return option!;
  });

  fireEvent.click(option);
};
const openSettings = async () => {
  fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));
  await screen.findByRole('textbox', { name: 'Workspace name' });
};
const denyFirstWorkspace = () =>
  server.use(
    http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) =>
      new URL(request.url).searchParams.get('workspaceId') === firstWorkspace.id
        ? HttpResponse.json(buildProblem('not_found', 404), { status: 404 })
        : HttpResponse.json({ items: [] }),
    ),
  );
const collectionReadPaths = ['/databases', '/backup-configs', '/storages'] as const;
const holdFirstWorkspaceRead = (path: (typeof collectionReadPaths)[number]) => {
  const pendingRead = createPendingApiResponse();
  server.use(
    http.get(`*${API_BASE_PATH}${path}`, ({ request }) =>
      new URL(request.url).searchParams.get('workspaceId') === firstWorkspace.id
        ? pendingRead.response
        : HttpResponse.json({ items: [] }),
    ),
  );

  return pendingRead;
};
const observeCollectionRead = (path: (typeof collectionReadPaths)[number]) => {
  if (path === '/backup-configs') return vi.spyOn(backupConfigApi, 'listAllWorkspaceBackupConfigs');
  if (path === '/storages') return vi.spyOn(storageApi, 'listAllStorages');
  return vi.spyOn(databaseApi, 'listAllDatabases');
};
const getWorkspaceReadCount = () =>
  server
    .getRequests()
    .filter((request) => new URL(request.url).pathname === `${API_BASE_PATH}/workspaces`).length;
const waitForDatabaseDetails = async () => {
  await screen.findByRole('button', { name: /Production database/ });
  await screen.findByRole('tab', { name: 'Backups' });
};
const expectNoDatabaseContent = () => {
  expect(screen.queryByRole('button', { name: /Production database/ })).toBeNull();
  expect(screen.queryByRole('tab', { name: 'Backups' })).toBeNull();
};
const refuseWorkspaceRead = (status: 403 | 404) =>
  HttpResponse.json(buildProblem(status === 403 ? 'permission_denied' : 'not_found', status), {
    status,
  });

describe('Application database workspace access', () => {
  it.each(
    ['initial', 'background'].flatMap((phase) =>
      collectionReadPaths.flatMap((path) =>
        ([403, 404] as const).flatMap((status) =>
          [false, true].map((hasAnotherWorkspace) => ({
            phase,
            path,
            status,
            hasAnotherWorkspace,
          })),
        ),
      ),
    ),
  )(
    'clears database content when $phase $path answers $status with another workspace=$hasAnotherWorkspace',
    async ({ phase, path, status, hasAnotherWorkspace }) => {
      const collectionReads = observeCollectionRead(path);
      const workspaceReads = vi.spyOn(workspaceApi, 'listAllWorkspaces');
      const application = openApplication(
        hasAnotherWorkspace ? [firstWorkspace, secondWorkspace] : [firstWorkspace],
        undefined,
        [firstWorkspaceDatabase],
      );

      if (phase === 'background') await waitForDatabaseDetails();

      const refusedRead = holdFirstWorkspaceRead(path);
      await screen.findByRole('combobox', { name: 'Selected workspace' });
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(reconciliation.handler);

      try {
        if (phase === 'background') fireEvent(document, new Event('visibilitychange'));

        await waitFor(() =>
          expect(collectionReads).toHaveBeenCalledTimes(phase === 'background' ? 2 : 1),
        );
        await releaseAndSettle(collectionReads, () =>
          refusedRead.respond(refuseWorkspaceRead(status)),
        );

        await waitFor(expectNoDatabaseContent);
        await waitFor(() => expect(workspaceReads).toHaveBeenCalledTimes(2));
        expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();

        if (hasAnotherWorkspace) {
          expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
          expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
        } else {
          expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
        }

        await releaseAndSettle(workspaceReads, () =>
          reconciliation.respond(
            HttpResponse.json({ items: hasAnotherWorkspace ? [secondWorkspace] : [] }),
          ),
        );

        if (hasAnotherWorkspace) {
          await screen.findByRole('button', { name: 'Add database' });
          expect(localStorage.getItem('selected_workspace_id')).toBe(secondWorkspace.id);
        } else {
          expect(
            screen.getAllByRole('button', { name: 'Create workspace' }).length,
          ).toBeGreaterThan(0);
          expect(localStorage.getItem('selected_workspace_id')).toBeNull();
        }

        expectNoDatabaseContent();
      } finally {
        application.unmount();
        refusedRead.cancel();
        reconciliation.cancel();
      }
    },
  );

  it('recovers workspace navigation when reconciliation fails after a database access refusal', async () => {
    openApplication([firstWorkspace, secondWorkspace], undefined, [firstWorkspaceDatabase]);
    await waitForDatabaseDetails();

    const databaseReads = observeCollectionRead('/databases');
    const refusedRead = holdFirstWorkspaceRead('/databases');
    server.use(mockApi.get('/workspaces', 500, buildProblem('internal', 500)));

    fireEvent(document, new Event('visibilitychange'));
    await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(1));
    await releaseAndSettle(databaseReads, () => refusedRead.respond(refuseWorkspaceRead(404)));
    const retry = await screen.findByRole('button', { name: 'Retry' });

    expectNoDatabaseContent();
    expect(
      screen.getByText('An internal server error occurred. Please try again later.'),
    ).toBeTruthy();

    server.use(mockApi.get('/workspaces', 200, { items: [secondWorkspace] }));

    fireEvent.click(retry);
    await screen.findByRole('button', { name: 'Add database' });

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
    expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expectNoDatabaseContent();
  });

  it.each([403, 404] as const)(
    'ignores a late database refusal with HTTP %s after a workspace switch',
    async (status) => {
      openApplication([firstWorkspace, secondWorkspace], undefined, [firstWorkspaceDatabase]);
      await waitForDatabaseDetails();

      const databaseReads = observeCollectionRead('/databases');
      const refusedRead = holdFirstWorkspaceRead('/databases');

      fireEvent(document, new Event('visibilitychange'));
      await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(1));

      await selectWorkspace(secondWorkspace);
      await openSettings();
      fireEvent.change(await screen.findByDisplayValue(secondWorkspace.name), {
        target: { value: 'Current workspace draft' },
      });
      await releaseAndSettle(
        databaseReads,
        () => refusedRead.respond(refuseWorkspaceRead(status)),
        0,
      );

      expect(screen.getByDisplayValue('Current workspace draft')).toBeTruthy();
      expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
      expect(getWorkspaceReadCount()).toBe(1);
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    },
  );

  it.each([403, 404] as const)(
    'ignores a late database refusal with HTTP %s after session replacement',
    async (status) => {
      openApplication([firstWorkspace, secondWorkspace], undefined, [firstWorkspaceDatabase]);
      await waitForDatabaseDetails();

      const databaseReads = observeCollectionRead('/databases');
      const refusedRead = holdFirstWorkspaceRead('/databases');

      fireEvent(document, new Event('visibilitychange'));
      await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(1));

      server.use(mockApi.get('/workspaces', 200, { items: [secondWorkspace] }));

      act(() => activateSession({ token: 'replacement-session', userId: 'owner-id' }));
      await waitFor(() =>
        expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
      );
      const workspaceReadCount = getWorkspaceReadCount();
      await releaseAndSettle(
        databaseReads,
        () => refusedRead.respond(refuseWorkspaceRead(status)),
        0,
      );

      expect(getWorkspaceReadCount()).toBe(workspaceReadCount);
      expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expectNoDatabaseContent();
    },
  );

  it.each([403, 404] as const)(
    'keeps a newer database list when an older refresh answers HTTP %s',
    async (status) => {
      openApplication([firstWorkspace, secondWorkspace], undefined, [firstWorkspaceDatabase]);
      await waitForDatabaseDetails();

      const databaseReads = observeCollectionRead('/databases');
      const refusedRead = holdFirstWorkspaceRead('/databases');

      fireEvent(document, new Event('visibilitychange'));
      await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(1));

      server.use(
        mockApi.get('/databases', 200, {
          items: [{ ...firstWorkspaceDatabase, name: 'Current database' }],
        }),
      );

      fireEvent(document, new Event('visibilitychange'));
      await screen.findByRole('button', { name: /Current database/ });

      await releaseAndSettle(
        databaseReads,
        () => refusedRead.respond(refuseWorkspaceRead(status)),
        0,
      );

      expect(
        screen.getByRole('button', { name: /Current database/ }).getAttribute('aria-pressed'),
      ).toBe('true');
      expect(getWorkspaceSelection().getByText(firstWorkspace.name)).toBeTruthy();
      expect(getWorkspaceReadCount()).toBe(1);
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    },
  );

  it.each(
    ['initial', 'background'].flatMap((phase) =>
      ['network', 'server'].flatMap((temporaryFailure) =>
        ([403, 404] as const).flatMap((status) =>
          (['/backup-configs', '/storages'] as const).map((path) => ({
            phase,
            temporaryFailure,
            status,
            path,
          })),
        ),
      ),
    ),
  )(
    'reconciles access when $phase $path answers $status after a $temporaryFailure failure',
    async ({ phase, temporaryFailure, status, path }) => {
      const application = openApplication([firstWorkspace, secondWorkspace], undefined, [
        firstWorkspaceDatabase,
      ]);

      if (phase === 'background') await waitForDatabaseDetails();

      const databaseReads = observeCollectionRead('/databases');
      const accessReads = observeCollectionRead(path);
      const failedRead = holdFirstWorkspaceRead('/databases');
      const refusedRead = holdFirstWorkspaceRead(path);
      const pendingSibling = holdFirstWorkspaceRead(
        path === '/storages' ? '/backup-configs' : '/storages',
      );
      await screen.findByRole('combobox', { name: 'Selected workspace' });
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(reconciliation.handler);

      try {
        if (phase === 'background') fireEvent(document, new Event('visibilitychange'));

        await waitFor(() => {
          expect(databaseReads).toHaveBeenCalledTimes(1);
          expect(accessReads).toHaveBeenCalledTimes(1);
        });
        await releaseAndSettle(databaseReads, () =>
          failedRead.respond(
            temporaryFailure === 'network'
              ? HttpResponse.error()
              : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
          ),
        );

        if (phase === 'initial') await screen.findByRole('button', { name: 'Retry' });
        else expect(screen.getByRole('button', { name: /Production database/ })).toBeTruthy();

        expect(getWorkspaceReadCount()).toBe(1);

        await releaseAndSettle(accessReads, () => refusedRead.respond(refuseWorkspaceRead(status)));

        await waitFor(expectNoDatabaseContent);
        await waitFor(() => expect(getWorkspaceReadCount()).toBe(2));
        expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
        expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      } finally {
        application.unmount();
        failedRead.cancel();
        refusedRead.cancel();
        pendingSibling.respond(HttpResponse.json({ items: [] }));
        reconciliation.respond(HttpResponse.json({ items: [secondWorkspace] }));
      }
    },
  );

  it('reconciles workspace access once when sibling reads refuse the same refresh', async () => {
    const application = openApplication([firstWorkspace, secondWorkspace], undefined, [
      firstWorkspaceDatabase,
    ]);
    await waitForDatabaseDetails();

    const configReads = observeCollectionRead('/backup-configs');
    const storageReads = observeCollectionRead('/storages');
    const refusedConfigs = holdFirstWorkspaceRead('/backup-configs');
    const refusedStorages = holdFirstWorkspaceRead('/storages');
    const reconciliation = mockApi.pending('get', '/workspaces');
    server.use(reconciliation.handler);

    try {
      fireEvent(document, new Event('visibilitychange'));
      await waitFor(() => {
        expect(configReads).toHaveBeenCalledTimes(1);
        expect(storageReads).toHaveBeenCalledTimes(1);
      });
      await act(async () => {
        refusedConfigs.respond(refuseWorkspaceRead(403));
        refusedStorages.respond(refuseWorkspaceRead(404));
        await Promise.allSettled([
          configReads.mock.results[0].value,
          storageReads.mock.results[0].value,
        ]);
      });

      await waitFor(expectNoDatabaseContent);
      await waitFor(() => expect(getWorkspaceReadCount()).toBe(2));
      expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
    } finally {
      application.unmount();
      refusedConfigs.cancel();
      refusedStorages.cancel();
      reconciliation.cancel();
    }
  });

  it.each(['network', 'server'])(
    'keeps workspace navigation when a database refresh has a $0 failure',
    async (temporaryFailure) => {
      openApplication([firstWorkspace, secondWorkspace], undefined, [firstWorkspaceDatabase]);
      await waitForDatabaseDetails();

      const databaseReads = observeCollectionRead('/databases');
      const failedRead = holdFirstWorkspaceRead('/databases');

      fireEvent(document, new Event('visibilitychange'));
      await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(1));
      await releaseAndSettle(databaseReads, () =>
        failedRead.respond(
          temporaryFailure === 'network'
            ? HttpResponse.error()
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
        ),
      );

      expect(
        screen.getByRole('button', { name: /Production database/ }).getAttribute('aria-pressed'),
      ).toBe('true');
      expect(getWorkspaceSelection().getByText(firstWorkspace.name)).toBeTruthy();
      expect(getWorkspaceReadCount()).toBe(1);
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    },
  );
});

describe('Application workspace navigation', () => {
  it('clears the databases and refreshes workspaces when a refresh answers 403', async () => {
    const application = openApplication([firstWorkspace, secondWorkspace], undefined, [
      firstWorkspaceDatabase,
    ]);
    await screen.findByRole('button', { name: /Production database/ });
    await screen.findByRole('tab', { name: 'Backups' });
    const reconciliation = mockApi.pending('get', '/workspaces');
    server.use(
      http.get(`*${API_BASE_PATH}/databases`, ({ request }) =>
        new URL(request.url).searchParams.get('workspaceId') === firstWorkspace.id
          ? HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 })
          : HttpResponse.json({ items: [] }),
      ),
      reconciliation.handler,
    );

    try {
      fireEvent(document, new Event('visibilitychange'));

      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Production database/ })).toBeNull();
        expect(screen.queryByRole('tab', { name: 'Backups' })).toBeNull();
      });
      expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
      await waitFor(() =>
        expect(
          server
            .getRequests()
            .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')),
        ).toHaveLength(2),
      );
    } finally {
      application.unmount();
      reconciliation.respond(HttpResponse.json({ items: [secondWorkspace] }));
    }
  });

  it.each(
    (['/audit-events', '/workspace-members'] as const).flatMap((path) =>
      [403, 404, 500].map((status) => ({
        path,
        status,
        scenario:
          status === 500
            ? 'retains workspace input when a child read fails'
            : 'reconciles workspace access when a child read refuses access',
      })),
    ),
  )('$scenario for $path with HTTP $status', async ({ path, status }) => {
    openApplication([firstWorkspace, secondWorkspace]);
    const childRead = mockApi.pending('get', path);
    server.use(childRead.handler);
    await openSettings();
    const name = await screen.findByDisplayValue(firstWorkspace.name);

    await waitFor(() =>
      expect(
        server.getRequests().some((request) => new URL(request.url).pathname.endsWith(path)),
      ).toBe(true),
    );

    fireEvent.change(name, { target: { value: 'Obsolete workspace draft' } });
    server.use(
      mockApi.get('/workspaces', 200, { items: [secondWorkspace] }),
      mockApi.get(path, 200, { items: [] }),
    );

    const previousWorkspaceReads = server
      .getRequests()
      .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')).length;

    childRead.respond(
      HttpResponse.json(
        buildProblem(
          status === 404 ? 'not_found' : status === 403 ? 'permission_denied' : 'internal',
          status,
        ),
        { status },
      ),
    );

    if (status === 500) {
      await screen.findByText('An internal server error occurred. Please try again later.');

      expect(screen.getByDisplayValue('Obsolete workspace draft')).toBeTruthy();
      expect(getWorkspaceSelection().getByText(firstWorkspace.name)).toBeTruthy();
      expect(localStorage.getItem('selected_workspace_id')).toBe(firstWorkspace.id);
      expect(
        server
          .getRequests()
          .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')),
      ).toHaveLength(previousWorkspaceReads);

      return;
    }

    await waitFor(() =>
      expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
    );
    expect(await screen.findByDisplayValue(secondWorkspace.name)).toBeTruthy();
    expect(screen.queryByDisplayValue('Obsolete workspace draft')).toBeNull();
    expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
    expect(localStorage.getItem('selected_workspace_id')).toBe(secondWorkspace.id);
    expect(server.getRequests().some((request) => ['PUT', 'DELETE'].includes(request.method))).toBe(
      false,
    );
  });

  it('restores the remembered workspace after application reload', async () => {
    const application = openApplication([firstWorkspace, secondWorkspace], secondWorkspace.id);
    await screen.findByRole('combobox', { name: 'Selected workspace' });

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();

    application.unmount();
    renderApp();
    await screen.findByRole('combobox', { name: 'Selected workspace' });

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
    expect(localStorage.getItem('selected_workspace_id')).toBe(secondWorkspace.id);
  });

  it('updates navigation when a workspace rename is acknowledged', async () => {
    openApplication([firstWorkspace, secondWorkspace]);
    server.use(
      mockApi.put('/workspaces/{workspaceId}', 200, { ...firstWorkspace, name: 'Renamed team' }),
    );

    await openSettings();

    fireEvent.change(screen.getByRole('textbox', { name: 'Workspace name' }), {
      target: { value: 'Renamed team' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));

    await waitFor(() => expect(getWorkspaceSelection().getByText('Renamed team')).toBeTruthy());
  });

  it.each([true, false])(
    'deletes the selected workspace without reloading with another workspace=%s',
    async (hasAnotherWorkspace) => {
      const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
      openApplication(hasAnotherWorkspace ? [firstWorkspace, secondWorkspace] : [firstWorkspace]);
      server.use(mockApi.delete('/workspaces/{workspaceId}', 204, undefined));

      await openSettings();

      fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
      fireEvent.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Delete Workspace' }),
      );

      await screen.findByText('Workspace deleted successfully');
      if (hasAnotherWorkspace) {
        await waitFor(() =>
          expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
        );
        expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
      } else {
        await waitFor(() =>
          expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull(),
        );
        expect(screen.getAllByRole('button', { name: 'Create workspace' }).length).toBeGreaterThan(
          0,
        );
        expect(screen.queryByRole('textbox', { name: 'Workspace name' })).toBeNull();
      }

      expect(reload).not.toHaveBeenCalled();
    },
  );

  it.each([false, true])(
    'retains a newer workspace draft after obsolete access reconciliation with failure=%s',
    async (hasFailed) => {
      openApplication([firstWorkspace, secondWorkspace, thirdWorkspace]);
      await screen.findByRole('combobox', { name: 'Selected workspace' });
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(reconciliation.handler);
      denyFirstWorkspace();

      fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));

      await waitFor(() =>
        expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
      );
      await waitFor(() =>
        expect(
          server
            .getRequests()
            .filter((request) => new URL(request.url).pathname.endsWith('/workspaces')),
        ).toHaveLength(2),
      );

      await selectWorkspace(thirdWorkspace);
      await openSettings();
      const name = await screen.findByDisplayValue(thirdWorkspace.name);

      fireEvent.change(name, { target: { value: 'Current workspace draft' } });

      const processed = new Promise<void>((resolve) => {
        server.events.on('response:mocked', function onResponse({ request }) {
          if (!new URL(request.url).pathname.endsWith('/workspaces')) return;

          server.events.removeListener('response:mocked', onResponse);
          resolve();
        });
      });

      await act(async () => {
        reconciliation.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [secondWorkspace] }),
        );
        await processed;
      });

      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull();
      expect(getWorkspaceSelection().getByText(thirdWorkspace.name)).toBeTruthy();
      expect(screen.getByDisplayValue('Current workspace draft')).toBeTruthy();
      expect(getWorkspaceSelection().queryByText(firstWorkspace.name)).toBeNull();
      expect(localStorage.getItem('selected_workspace_id')).toBe(thirdWorkspace.id);
    },
  );

  it('recovers a failed accessible-workspace refresh without rendering unavailable content', async () => {
    openApplication([firstWorkspace, secondWorkspace]);
    await screen.findByRole('combobox', { name: 'Selected workspace' });
    denyFirstWorkspace();
    server.use(mockApi.get('/workspaces', 500, buildProblem('internal', 500)));

    fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));
    const retry = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Add notifier' })).toBeNull();

    server.use(mockApi.get('/workspaces', 200, { items: [secondWorkspace] }));

    fireEvent.click(retry);
    await screen.findByRole('button', { name: 'Add notifier' });

    expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it.each(
    (['create', 'rename', 'delete'] as const).flatMap((mutation) =>
      [false, true].map((hasFailed) => ({ mutation, hasFailed })),
    ),
  )(
    'retains an acknowledged workspace $mutation after older reconciliation finishes with failure=$hasFailed',
    async ({ mutation, hasFailed }) => {
      openApplication([firstWorkspace, secondWorkspace]);
      await screen.findByRole('combobox', { name: 'Selected workspace' });
      const reconciliation = mockApi.pending('get', '/workspaces');
      server.use(reconciliation.handler);
      denyFirstWorkspace();

      fireEvent.click(screen.getByRole('button', { name: 'Notifiers' }));

      await waitFor(() =>
        expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy(),
      );

      if (mutation === 'create') {
        server.use(mockApi.post('/workspaces', 201, thirdWorkspace));

        fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Selected workspace' }));
        fireEvent.click(await screen.findByRole('button', { name: /Create new workspace/ }));
        const dialog = within(await screen.findByRole('dialog'));

        fireEvent.change(dialog.getByRole('textbox', { name: 'Workspace name' }), {
          target: { value: thirdWorkspace.name },
        });
        fireEvent.click(dialog.getByRole('button', { name: 'Create workspace' }));

        await waitFor(() =>
          expect(getWorkspaceSelection().getByText(thirdWorkspace.name)).toBeTruthy(),
        );
      } else {
        await openSettings();
        if (mutation === 'rename') {
          server.use(
            mockApi.put('/workspaces/{workspaceId}', 200, {
              ...secondWorkspace,
              name: 'Renamed team',
            }),
          );

          fireEvent.change(screen.getByRole('textbox', { name: 'Workspace name' }), {
            target: { value: 'Renamed team' },
          });
          fireEvent.click(screen.getByRole('button', { name: /Save changes/i }));

          await waitFor(() =>
            expect(getWorkspaceSelection().getByText('Renamed team')).toBeTruthy(),
          );
        } else {
          server.use(mockApi.delete('/workspaces/{workspaceId}', 204, undefined));

          fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
          fireEvent.click(
            within(await screen.findByRole('dialog')).getByRole('button', {
              name: 'Delete Workspace',
            }),
          );

          await waitFor(() =>
            expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull(),
          );
        }
      }

      const processed = new Promise<void>((resolve) => {
        server.events.on('response:mocked', function onResponse() {
          server.events.removeListener('response:mocked', onResponse);
          resolve();
        });
      });

      await act(async () => {
        reconciliation.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [secondWorkspace] }),
        );
        await processed;
      });

      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull();

      if (mutation === 'delete') {
        expect(screen.queryByRole('combobox', { name: 'Selected workspace' })).toBeNull();
        expect(localStorage.getItem('selected_workspace_id')).toBeNull();
      } else {
        expect(
          getWorkspaceSelection().getByText(
            mutation === 'create' ? thirdWorkspace.name : 'Renamed team',
          ),
        ).toBeTruthy();
        expect(localStorage.getItem('selected_workspace_id')).toBe(
          mutation === 'create' ? thirdWorkspace.id : secondWorkspace.id,
        );
      }
    },
  );
});

describe('Application workspace switch', () => {
  const buildMember = (email: string): WorkspaceMember => ({
    id: `${email}-member`,
    workspaceId: secondWorkspace.id,
    userId: `${email}-user`,
    email,
    name: email.split('@')[0],
    role: 'WORKSPACE_MEMBER',
    isInvited: false,
    createdAt: firstWorkspace.createdAt,
  });
  const listScreens = [
    {
      tab: 'Databases',
      path: '/databases',
      preferencePrefix: 'selectedDatabaseId',
      searchLabel: 'Search database',
      buildResource: (workspace: Workspace, name: string) => ({
        ...firstWorkspaceDatabase,
        id: `${workspace.id}-${name}`,
        workspaceId: workspace.id,
        name,
      }),
    },
    {
      tab: 'Storages',
      path: '/storages',
      preferencePrefix: 'selectedStorageId',
      buildResource: (workspace: Workspace, name: string) => ({
        id: `${workspace.id}-${name}`,
        workspaceId: workspace.id,
        name,
        settings: { type: 'LOCAL' },
      }),
    },
    {
      tab: 'Notifiers',
      path: '/notifiers',
      preferencePrefix: 'selectedNotifierId',
      searchLabel: 'Search notifier',
      buildResource: (workspace: Workspace, name: string) => ({
        id: `${workspace.id}-${name}`,
        workspaceId: workspace.id,
        name,
        settings: { type: 'DISCORD' },
        lastSendError: null,
      }),
    },
  ];
  const serveResources = (path: string, resources: { id: string; workspaceId: string }[]) =>
    server.use(
      http.get(`*${API_BASE_PATH}${path}`, ({ request }) =>
        HttpResponse.json({
          items: resources.filter(
            (resource) =>
              resource.workspaceId === new URL(request.url).searchParams.get('workspaceId'),
          ),
        }),
      ),
      http.get(`*${API_BASE_PATH}${path}/:resourceId`, ({ params }) =>
        HttpResponse.json(resources.find((resource) => resource.id === params.resourceId)),
      ),
    );
  const findResourceCard = (name: string) =>
    // Only fixed test fixture names reach this selector.
    // nosemgrep: javascript.lang.security.audit.detect-non-literal-regexp.detect-non-literal-regexp
    screen.findByRole('button', { name: new RegExp(`^${name}`) });
  const expectResourceSelected = (name: string) =>
    waitFor(async () =>
      expect((await findResourceCard(name)).getAttribute('aria-pressed')).toBe('true'),
    );
  const openWorkspaceScreen = async (tab: string) => {
    await screen.findByRole('combobox', { name: 'Selected workspace' });

    if (tab !== 'Databases') fireEvent.click(await screen.findByRole('button', { name: tab }));
  };

  const obsoleteName = 'Obsolete answer';
  const heldReads = [
    ...listScreens.map((listScreen) => ({
      tab: listScreen.tab,
      path: listScreen.path,
      listScreen,
      observeRead: () =>
        listScreen.tab === 'Databases'
          ? vi.spyOn(databaseApi, 'listAllDatabases')
          : listScreen.tab === 'Storages'
            ? vi.spyOn(storageApi, 'listAllStorages')
            : vi.spyOn(notifierApi, 'listAllNotifiers'),
      obsoleteBody: {
        items: [listScreen.buildResource(firstWorkspace, obsoleteName)],
      } as object,
    })),
    {
      tab: 'Settings',
      path: '/workspaces/:workspaceId',
      listScreen: undefined,
      observeRead: () => vi.spyOn(workspaceApi, 'getWorkspace'),
      obsoleteBody: { ...firstWorkspace, name: obsoleteName } as object,
    },
    {
      tab: 'Settings',
      path: '/workspace-members',
      listScreen: undefined,
      observeRead: () => vi.spyOn(workspaceMemberApi, 'listAllMembers'),
      obsoleteBody: { items: [buildMember(obsoleteName)] } as object,
    },
  ];

  it.each(
    heldReads.flatMap((heldRead) => [200, 404, 500].map((status) => ({ ...heldRead, status }))),
  )(
    'shows no data of the previous workspace after switching when $path on $tab answers HTTP $status late',
    async ({ tab, path, listScreen, observeRead, obsoleteBody, status }) => {
      const currentMember = buildMember('current@example.com');
      const currentResources = ['Current one', 'Current two'].map((name) =>
        listScreen?.buildResource(secondWorkspace, name),
      );

      openApplication([firstWorkspace, secondWorkspace]);
      server.use(
        http.get(`*${API_BASE_PATH}/workspace-members`, ({ request }) =>
          HttpResponse.json({
            items:
              new URL(request.url).searchParams.get('workspaceId') === secondWorkspace.id
                ? [currentMember]
                : [],
          }),
        ),
      );

      if (listScreen)
        serveResources(listScreen.path, currentResources as { id: string; workspaceId: string }[]);

      const reads = observeRead();
      const oldRead = createPendingApiResponse();
      server.use(
        http.get(`*${API_BASE_PATH}${path}`, ({ request, params }) => {
          const readWorkspaceId =
            params.workspaceId ?? new URL(request.url).searchParams.get('workspaceId');

          return readWorkspaceId === firstWorkspace.id ? oldRead.response : undefined;
        }),
      );

      await openWorkspaceScreen(tab);
      await waitFor(() =>
        expect(reads.mock.calls.some(([id]) => id === firstWorkspace.id)).toBe(true),
      );
      const oldReadIndex = reads.mock.calls.findIndex(([id]) => id === firstWorkspace.id);

      await selectWorkspace(secondWorkspace);

      if (listScreen) {
        fireEvent.click(await findResourceCard('Current two'));
        await expectResourceSelected('Current two');
      } else {
        await screen.findByDisplayValue(secondWorkspace.name);
        await screen.findByText(currentMember.email);
      }

      const workspaceReadCount = getWorkspaceReadCount();
      const preferenceKey = listScreen && `${listScreen.preferencePrefix}_${secondWorkspace.id}`;
      const storedPreference = preferenceKey && localStorage.getItem(preferenceKey);

      await releaseAndSettle(
        reads,
        () =>
          oldRead.respond(
            status === 200
              ? HttpResponse.json(obsoleteBody)
              : HttpResponse.json(buildProblem(status === 404 ? 'not_found' : 'internal', status), {
                  status,
                }),
          ),
        oldReadIndex,
      );

      if (listScreen) {
        await expectResourceSelected('Current two');
        expect(screen.getByRole('button', { name: /^Current one/ })).toBeTruthy();
        expect(localStorage.getItem(preferenceKey as string)).toBe(storedPreference);
      } else {
        expect(screen.getByDisplayValue(secondWorkspace.name)).toBeTruthy();
        expect(screen.getByText(currentMember.email)).toBeTruthy();
      }

      expect(screen.queryByRole('button', { name: new RegExp(obsoleteName) })).toBeNull();
      expect(screen.queryByDisplayValue(obsoleteName)).toBeNull();
      expect(screen.queryByText(obsoleteName)).toBeNull();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(document.querySelector('.ant-alert-error')).toBeNull();
      expect(getWorkspaceSelection().getByText(secondWorkspace.name)).toBeTruthy();
      expect(getWorkspaceReadCount()).toBe(workspaceReadCount);
    },
  );

  it.each(listScreens)(
    'resets the selection and keeps focus reachable after switching on $tab',
    async ({ tab, path, preferencePrefix, searchLabel, buildResource }) => {
      const resourceNumbers = searchLabel ? [1, 2, 3, 4, 5] : [1, 2];

      openApplication([firstWorkspace, secondWorkspace]);
      serveResources(path, [
        ...resourceNumbers.map((number) => buildResource(firstWorkspace, `First ${number}`)),
        ...resourceNumbers.map((number) => buildResource(secondWorkspace, `Second ${number}`)),
      ]);
      await openWorkspaceScreen(tab);

      fireEvent.click(await findResourceCard('First 2'));
      await expectResourceSelected('First 2');

      if (searchLabel) {
        fireEvent.change(screen.getByRole('textbox', { name: searchLabel }), {
          target: { value: 'First 2' },
        });
        await waitFor(() => expect(screen.queryByRole('button', { name: /^First 1/ })).toBeNull());
      }

      const workspaceSelector = screen.getByRole('combobox', { name: 'Selected workspace' });
      workspaceSelector.focus();

      await selectWorkspace(secondWorkspace);

      await expectResourceSelected('Second 1');
      expect(screen.queryByRole('button', { name: /^First/ })).toBeNull();
      expect(localStorage.getItem(`${preferencePrefix}_${secondWorkspace.id}`)).toBe(
        buildResource(secondWorkspace, 'Second 1').id,
      );
      expect(document.activeElement).toBe(workspaceSelector);

      if (searchLabel) {
        const search = screen.getByRole('textbox', { name: searchLabel }) as HTMLInputElement;

        expect(search.value).toBe('');
      }

      await selectWorkspace(firstWorkspace);

      await expectResourceSelected('First 2');
      expect(screen.getByRole('button', { name: /^First 1/ })).toBeTruthy();
      expect(screen.queryByRole('button', { name: /^Second/ })).toBeNull();
    },
  );
});
