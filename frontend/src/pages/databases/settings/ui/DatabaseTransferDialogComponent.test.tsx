// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { BackupConfig, DatabaseTransfer } from '@/entities/backup-configs';
import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import type { Storage } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { DatabaseTransferDialogComponent } from '@/pages/databases/settings/ui/DatabaseTransferDialogComponent';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const sourceWorkspace: Workspace = {
  id: 'source-workspace-id',
  name: 'Production',
  createdAt: '2026-01-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_ADMIN',
};

const targetWorkspace: Workspace = {
  ...sourceWorkspace,
  id: 'target-workspace-id',
  name: 'Staging',
};

const transferredDatabase: Database = {
  id: 'database-id',
  workspaceId: sourceWorkspace.id,
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
  notifierIds: [],
};

const currentStorage: Storage = {
  id: 'current-storage-id',
  workspaceId: sourceWorkspace.id,
  name: 'Nightly disk',
  settings: { type: 'LOCAL' },
};

const targetStorage: Storage = {
  ...currentStorage,
  id: 'target-storage-id',
  workspaceId: targetWorkspace.id,
  name: 'Staging disk',
};

const buildBackupConfig = (databaseId: string): BackupConfig => ({
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId,
  isBackupsEnabled: true,
  storageId: currentStorage.id,
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'HOURLY' },
    retention: {
      policy: 'TIME_PERIOD',
      period: 'WEEK',
      count: 0,
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
});

interface TransferRequests {
  storageBackupConfigListQueries: URLSearchParams[];
  sentTransfers: DatabaseTransfer[];
}

const mockTransferOperations = (storageBackupConfigs: BackupConfig[]): TransferRequests => {
  const transferRequests: TransferRequests = {
    storageBackupConfigListQueries: [],
    sentTransfers: [],
  };

  mockServer.use(
    mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
    mockApi.get('/storages', 200, { items: [targetStorage] }),
    mockApi.get('/notifiers', 200, { items: [] }),
    http.get(`*${API_BASE_PATH}/backup-configs`, ({ request }) => {
      transferRequests.storageBackupConfigListQueries.push(new URL(request.url).searchParams);
      return HttpResponse.json({ items: storageBackupConfigs });
    }),
    http.post(`*${API_BASE_PATH}/databases/:databaseId/transfer`, async ({ request, params }) => {
      expect(params.databaseId).toBe(transferredDatabase.id);
      transferRequests.sentTransfers.push((await request.json()) as DatabaseTransfer);
      return new HttpResponse(null, { status: 204 });
    }),
  );

  return transferRequests;
};

const renderDialog = () =>
  renderScreen(
    <DatabaseTransferDialogComponent
      database={transferredDatabase}
      currentStorageId={currentStorage.id}
      onClose={() => {}}
      onTransferred={() => {}}
    />,
  );

const chooseSelectOption = async (combobox: HTMLElement, optionLabel: string) => {
  if (combobox.getAttribute('aria-expanded') !== 'true') fireEvent.click(combobox);

  const option = await waitFor(() => {
    const matchingOption = Array.from(screen.queryAllByRole('option')).find(
      (candidate) => candidate.textContent === optionLabel,
    );

    expect(matchingOption).toBeTruthy();

    return matchingOption as HTMLElement;
  });

  fireEvent.click(option);
};

const chooseTargetWorkspace = async () => {
  await chooseSelectOption(await screen.findByRole('combobox'), targetWorkspace.name);
};

describe('DatabaseTransferDialogComponent', () => {
  it.each(['unmount', 'retry', 'session'] as const)(
    'cancels unfinished target reads after a failure and %s',
    async (action) => {
      mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
      const failedStorage = mockApi.pending('get', '/storages');
      const obsoleteNotifiers = mockApi.pending('get', '/notifiers');
      mockServer.use(failedStorage.handler, obsoleteNotifiers.handler);
      const view = renderDialog();

      await chooseTargetWorkspace();
      const getNotifierRequests = () =>
        mockServer
          .getRequests()
          .filter((request) => new URL(request.url).pathname === '/api/v1/notifiers');
      await waitFor(() => expect(getNotifierRequests()).toHaveLength(1));

      await finishApiResponse(mockServer, '/api/v1/storages', () =>
        failedStorage.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
      );
      await screen.findByRole('button', { name: 'Retry' });

      if (action === 'unmount') view.unmount();
      else if (action === 'retry') {
        mockServer.use(
          mockApi.get('/storages', 200, { items: [targetStorage] }),
          mockApi.get('/notifiers', 200, { items: [] }),
        );
        fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
        await waitFor(() => expect(getNotifierRequests()).toHaveLength(2));
      } else
        act(() => activateSession({ token: 'replacement-session', userId: 'replacement-user' }));

      expect(getNotifierRequests()[0].signal.aborted).toBe(true);

      const requestsBeforeRelease = getNotifierRequests().length;
      await finishCancelledApiResponse(mockServer, '/api/v1/notifiers', () =>
        obsoleteNotifiers.respond(HttpResponse.json({ items: [], nextCursor: 'obsolete-page' })),
      );

      expect(getNotifierRequests()).toHaveLength(requestsBeforeRelease);
    },
  );

  it('cancels unfinished notifier usage listings after a parallel usage read fails', async () => {
    mockTransferOperations([]);
    const database = {
      ...transferredDatabase,
      notifierIds: ['failed-notifier', 'pending-notifier'],
    };
    const failedDatabases = mockApi.pending('get', '/databases');
    const obsoleteDatabases = mockApi.pending('get', '/databases');
    mockServer.use(
      mockApi.get('/notifiers', 200, {
        items: database.notifierIds.map((id) => ({
          id,
          name: id,
          lastSendError: null,
          workspaceId: sourceWorkspace.id,
          settings: { type: 'DISCORD' as const },
        })),
      }),
      http.get(`*${API_BASE_PATH}/databases`, ({ request }) =>
        new URL(request.url).searchParams.get('notifierId') === 'failed-notifier'
          ? failedDatabases.response
          : obsoleteDatabases.response,
      ),
    );
    const view = renderScreen(
      <DatabaseTransferDialogComponent
        database={database}
        onClose={() => {}}
        onTransferred={() => {}}
      />,
    );
    const getUsageRequests = () =>
      mockServer
        .getRequests()
        .filter((request) => new URL(request.url).pathname === '/api/v1/databases');
    await waitFor(() => expect(getUsageRequests()).toHaveLength(2));

    await finishApiResponse(mockServer, '/api/v1/databases', () =>
      failedDatabases.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
    );
    await screen.findByRole('button', { name: 'Retry' });
    view.unmount();

    expect(
      getUsageRequests().find(
        (request) => new URL(request.url).searchParams.get('notifierId') === 'pending-notifier',
      )!.signal.aborted,
    ).toBe(true);

    await finishCancelledApiResponse(mockServer, '/api/v1/databases', () =>
      obsoleteDatabases.respond(HttpResponse.json({ items: [], nextCursor: 'obsolete-page' })),
    );

    expect(getUsageRequests()).toHaveLength(2);
  });

  it('automatically selects completed storage when creation finishes inside transfer', async () => {
    const transferRequests = mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
    const database = { ...transferredDatabase, notifierIds: ['source-alerts'] };
    const createdStorage = { ...targetStorage, id: 'created-storage', name: 'Created target disk' };
    const creation = mockApi.delayed({
      method: 'post',
      path: '/storages',
      status: 201,
      body: createdStorage,
    });
    mockServer.use(
      creation.handler,
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      mockApi.get('/databases', 200, { items: [database] }),
      http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) =>
        HttpResponse.json({
          items:
            new URL(request.url).searchParams.get('workspaceId') === sourceWorkspace.id
              ? [
                  {
                    id: 'source-alerts',
                    workspaceId: sourceWorkspace.id,
                    name: 'Source alerts',
                    lastSendError: null,
                    settings: { type: 'DISCORD' },
                  },
                ]
              : [],
        }),
      ),
    );

    const onTransferred = vi.fn();
    renderScreen(
      <DatabaseTransferDialogComponent
        database={database}
        currentStorageId={currentStorage.id}
        onClose={() => {}}
        onTransferred={onTransferred}
      />,
      { isStrictMode: true },
    );

    await chooseTargetWorkspace();
    await chooseSelectOption(
      await screen.findByRole('combobox', { name: 'Storage' }),
      targetStorage.name,
    );
    expect(
      screen.getByRole('radiogroup', { name: DICTIONARIES.en.databases.transfer.storage.title }),
    ).toBeTruthy();
    expect(
      screen.getByRole('radiogroup', { name: DICTIONARIES.en.databases.transfer.notifiers.title }),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole('radio', { name: 'Transfer notifiers with database' }));
    await screen.findByText('Source alerts');

    fireEvent.click(await screen.findByRole('combobox', { name: 'Storage' }));
    fireEvent.click(screen.getByRole('button', { name: 'Create new storage' }));
    const editor = within(await screen.findByRole('region', { name: 'Storage settings' }));

    fireEvent.change(editor.getByRole('textbox', { name: 'Name' }), {
      target: { value: createdStorage.name },
    });
    fireEvent.click(editor.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await editor.findByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(
        mockServer
          .getRequests()
          .some(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
          ),
      ).toBe(true),
    );
    expect(screen.getByRole('combobox', { name: 'Storage' }).closest('div')!.textContent).toContain(
      targetStorage.name,
    );

    creation.release();

    await waitFor(() =>
      expect(screen.queryByRole('region', { name: 'Storage settings' })).toBeNull(),
    );
    expect(screen.getByRole('combobox', { name: 'Storage' }).closest('div')!.textContent).toContain(
      createdStorage.name,
    );
    expect(
      screen.getByRole('combobox', { name: 'Target workspace' }).closest('div')!.textContent,
    ).toContain(targetWorkspace.name);
    expect(screen.getByRole('radio', { name: 'Transfer notifiers with database' })).toHaveProperty(
      'checked',
      true,
    );
    expect(database.workspaceId).toBe(sourceWorkspace.id);

    const createRequest = mockServer
      .getRequests()
      .find(
        (request) =>
          request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
      )!;

    expect(await createRequest.json()).toMatchObject({ workspaceId: targetWorkspace.id });

    fireEvent.click(screen.getByRole('button', { name: 'Transfer' }));

    await waitFor(() => expect(onTransferred).toHaveBeenCalledOnce());
    expect(transferRequests.sentTransfers).toEqual([
      {
        targetWorkspaceId: targetWorkspace.id,
        storage: { type: 'SWITCH_TO_STORAGE', storageId: createdStorage.id },
        isMovingNotifiers: true,
        targetNotifierIds: [],
      },
    ]);
  });

  it('sends a move of the storage with the database when no other database uses it', async () => {
    const transferRequests = mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
    renderDialog();

    await chooseTargetWorkspace();

    fireEvent.click(await screen.findByText('Transfer with existing storage'));
    await screen.findByText('Storage can be transferred');

    fireEvent.click(screen.getByRole('button', { name: 'Transfer' }));

    await waitFor(() => expect(transferRequests.sentTransfers).toHaveLength(1));
    expect(transferRequests.sentTransfers[0]).toEqual({
      targetWorkspaceId: targetWorkspace.id,
      storage: { type: 'MOVE_WITH_DATABASE' },
      isMovingNotifiers: false,
      targetNotifierIds: [],
    });
    expect(transferRequests.storageBackupConfigListQueries[0].get('storageId')).toBe(
      currentStorage.id,
    );
    expect(transferRequests.storageBackupConfigListQueries[0].get('limit')).toBe('2');
  });

  it('sends a switch to the chosen storage of the target workspace', async () => {
    const transferRequests = mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
    renderDialog();

    await chooseTargetWorkspace();

    await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));

    await chooseSelectOption(screen.getAllByRole('combobox')[1], targetStorage.name);

    expect(
      screen.getByText(/Switching storage immediately removes the old backup history/),
    ).toBeTruthy();

    const warning = screen.getByText(
      /Switching storage immediately removes the old backup history/,
    );

    expect(warning.textContent).toContain('activates the new storage.');
    expect(warning.textContent).toContain('Unreachable storage delays cleanup.');
    expect(transferRequests.sentTransfers).toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: 'Transfer' }));

    await waitFor(() => expect(transferRequests.sentTransfers).toHaveLength(1));
    expect(transferRequests.sentTransfers[0]).toEqual({
      targetWorkspaceId: targetWorkspace.id,
      storage: { type: 'SWITCH_TO_STORAGE', storageId: targetStorage.id },
      isMovingNotifiers: false,
      targetNotifierIds: [],
    });
  });

  it('keeps the dialog and its choices open when the transfer is refused', async () => {
    mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
    mockServer.use(
      mockApi.post('/databases/{databaseId}/transfer', 403, buildProblem('permission_denied', 403)),
    );
    const onTransferred = vi.fn();
    renderScreen(
      <DatabaseTransferDialogComponent
        database={transferredDatabase}
        currentStorageId={currentStorage.id}
        onClose={() => {}}
        onTransferred={onTransferred}
      />,
    );

    await chooseTargetWorkspace();
    await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));
    await chooseSelectOption(screen.getAllByRole('combobox')[1], targetStorage.name);

    fireEvent.click(screen.getByRole('button', { name: 'Transfer' }));

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(screen.getAllByRole('combobox')[0].closest('div')!.textContent).toContain(
      targetWorkspace.name,
    );
    expect(screen.getAllByRole('combobox')[1].closest('div')!.textContent).toContain(
      targetStorage.name,
    );
    expect(onTransferred).not.toHaveBeenCalled();
  });

  it('acknowledges a committed transfer immediately without showing history', async () => {
    mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
    const onTransferred = vi.fn();
    renderScreen(
      <DatabaseTransferDialogComponent
        database={transferredDatabase}
        currentStorageId={currentStorage.id}
        onClose={() => {}}
        onTransferred={onTransferred}
      />,
    );
    await chooseTargetWorkspace();
    await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));
    await chooseSelectOption(screen.getAllByRole('combobox')[1], targetStorage.name);
    fireEvent.click(screen.getByRole('button', { name: 'Transfer' }));
    await waitFor(() => expect(onTransferred).toHaveBeenCalledTimes(1));
    expect(screen.queryByText('Storage switch history')).toBeNull();
  });

  it('blocks a move of the storage when another database backs up into it', async () => {
    mockTransferOperations([
      buildBackupConfig(transferredDatabase.id),
      buildBackupConfig('other-database-id'),
    ]);
    renderDialog();

    await chooseTargetWorkspace();

    fireEvent.click(await screen.findByText('Transfer with existing storage'));

    expect(
      await screen.findByText(
        'Other databases back up into this storage, so it cannot move with this database.',
      ),
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Transfer' }).hasAttribute('disabled')).toBe(true);
  });

  it('discards a nested storage creation after the transfer target workspace changes', async () => {
    mockTransferOperations([]);
    const nextWorkspace = { ...targetWorkspace, id: 'next-workspace', name: 'Next workspace' };
    const createdStorage: Storage = {
      ...targetStorage,
      id: 'created-storage',
      name: 'Created for staging',
    };
    const creation = mockApi.pending('post', '/storages');
    mockServer.use(
      mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace, nextWorkspace] }),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      creation.handler,
    );
    renderDialog();

    await chooseTargetWorkspace();
    const workspaceInput = screen.getAllByRole('combobox')[0];

    fireEvent.click(await screen.findByRole('combobox', { name: 'Storage' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Create new storage' }));
    const editor = within(await screen.findByRole('region', { name: 'Storage settings' }));

    fireEvent.change(editor.getByRole('textbox', { name: 'Name' }), {
      target: { value: createdStorage.name },
    });
    fireEvent.click(editor.getByRole('button', { name: /Test connection/ }));
    fireEvent.click(await editor.findByRole('button', { name: /Save/ }));

    await waitFor(() =>
      expect(
        mockServer
          .getRequests()
          .some(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
          ),
      ).toBe(true),
    );

    mockServer.use(mockApi.get('/storages', 200, { items: [] }));

    await chooseSelectOption(workspaceInput, nextWorkspace.name);

    await waitFor(() =>
      expect(screen.queryByRole('region', { name: 'Storage settings' })).toBeNull(),
    );

    const response = new Promise<void>((resolve) => {
      const onResponse = () => {
        mockServer.events.removeListener('response:mocked', onResponse);
        resolve();
      };
      mockServer.events.on('response:mocked', onResponse);
    });
    await act(async () => {
      creation.respond(HttpResponse.json(createdStorage, { status: 201 }));
      await response;
    });

    expect(screen.queryByText(createdStorage.name)).toBeNull();
    expect(screen.getByRole('button', { name: 'Transfer' })).toHaveProperty('disabled', true);
  });

  it.each([false, true])(
    'retains transfer context until a delayed response completes with failure=%s',
    async (hasFailed) => {
      mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
      const nextWorkspace = { ...targetWorkspace, id: 'next-workspace', name: 'Next workspace' };
      const pendingTransfer = mockApi.pending('post', '/databases/{databaseId}/transfer');
      mockServer.use(
        mockApi.get('/workspaces', 200, {
          items: [sourceWorkspace, targetWorkspace, nextWorkspace],
        }),
        pendingTransfer.handler,
      );

      const onTransferred = vi.fn();
      const onClose = vi.fn();
      const { user: interaction } = renderScreen(
        <DatabaseTransferDialogComponent
          database={transferredDatabase}
          currentStorageId={currentStorage.id}
          onClose={onClose}
          onTransferred={onTransferred}
        />,
      );

      await chooseTargetWorkspace();

      await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));

      await chooseSelectOption(screen.getAllByRole('combobox')[1], targetStorage.name);
      const transferButton = screen.getByRole('button', { name: 'Transfer' });

      await interaction.click(transferButton);
      const transfers = () =>
        mockServer
          .getRequests()
          .filter(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/transfer'),
          );

      await waitFor(() => expect(transfers()).toHaveLength(1));

      const workspaceControl = screen.getByRole('combobox', { name: 'Target workspace' });

      expect(workspaceControl).toHaveProperty('disabled', true);
      expect(screen.getByRole('combobox', { name: 'Storage' })).toHaveProperty('disabled', true);
      expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', true);

      await interaction.click(workspaceControl);

      expect(workspaceControl.getAttribute('aria-expanded')).toBe('false');
      expect(
        within(workspaceControl.closest('div') as HTMLElement).getByText(targetWorkspace.name),
      ).toBeTruthy();

      await interaction.click(transferButton);

      expect(transfers()).toHaveLength(1);
      expect(onClose).not.toHaveBeenCalled();
      expect(onTransferred).not.toHaveBeenCalled();

      await act(async () =>
        pendingTransfer.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : new HttpResponse(null, { status: 204 }),
        ),
      );

      if (hasFailed) {
        await screen.findByText('An internal server error occurred. Please try again later.');

        expect(screen.getByRole('combobox', { name: 'Target workspace' })).toHaveProperty(
          'disabled',
          false,
        );
        expect(screen.getByRole('button', { name: 'Transfer' })).toHaveProperty('disabled', false);
      } else {
        await waitFor(() => expect(onTransferred).toHaveBeenCalledTimes(1));
      }

      expect(transfers()).toHaveLength(1);
      expect(await transfers()[0].json()).toMatchObject({
        targetWorkspaceId: targetWorkspace.id,
        storage: { type: 'SWITCH_TO_STORAGE', storageId: targetStorage.id },
      });
      expect(onTransferred).toHaveBeenCalledTimes(hasFailed ? 0 : 1);
    },
  );

  it.each([false, true])(
    'retains persisted nested storage and refreshed choices after an older target read fails=%s',
    async (hasFailed) => {
      mockTransferOperations([]);
      const initialList = mockApi.pending('get', '/storages');
      const creation = mockApi.pending('post', '/storages');
      const createdStorage = { ...targetStorage, id: 'created-storage', name: 'Created archive' };
      mockServer.use(
        initialList.handler,
        creation.handler,
        mockApi.post('/storage-tests', 200, { isConnected: true }),
      );
      renderDialog();

      await chooseTargetWorkspace();

      await waitFor(() =>
        expect(
          mockServer
            .getRequests()
            .some((request) => new URL(request.url).pathname.endsWith('/storages')),
        ).toBe(true),
      );

      fireEvent.click(await screen.findByRole('combobox', { name: 'Storage' }));
      fireEvent.click(screen.getByRole('button', { name: 'Create new storage' }));
      const editorRegion = await screen.findByRole('region', { name: 'Storage settings' });
      const editor = within(editorRegion);

      fireEvent.change(editor.getByRole('textbox', { name: 'Name' }), {
        target: { value: createdStorage.name },
      });
      fireEvent.click(editor.getByRole('button', { name: 'Test connection' }));
      fireEvent.click(await editor.findByRole('button', { name: 'Save' }));

      await waitFor(() =>
        expect(
          mockServer
            .getRequests()
            .some(
              (request) =>
                request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
            ),
        ).toBe(true),
      );

      fireEvent.change(editor.getByRole('textbox', { name: 'Name' }), {
        target: { value: 'Newer draft' },
      });
      mockServer.use(mockApi.get('/storages', 200, { items: [targetStorage] }));

      await act(async () => creation.respond(HttpResponse.json(createdStorage, { status: 201 })));

      await waitFor(() => expect(editor.queryByRole('combobox', { name: 'Type' })).toBeNull());

      expect(editor.getByDisplayValue('Newer draft')).toBeTruthy();

      await finishCancelledApiResponse(mockServer, '/api/v1/storages', () => {
        initialList.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [targetStorage] }),
        );
      });

      fireEvent.click(
        within(editorRegion.closest('[role=dialog]') as HTMLElement).getByRole('button', {
          name: 'Close',
        }),
      );

      await chooseSelectOption(
        screen.getByRole('combobox', { name: 'Storage' }),
        createdStorage.name,
      );

      expect(screen.getByRole('button', { name: 'Transfer' })).toHaveProperty('disabled', false);

      await chooseSelectOption(
        screen.getByRole('combobox', { name: 'Storage' }),
        targetStorage.name,
      );

      expect(screen.getByRole('button', { name: 'Transfer' })).toHaveProperty('disabled', false);
      expect(screen.getAllByRole('alert').map((alert) => alert.textContent)).toEqual([
        DICTIONARIES.en.backups.storageSwitch.warning,
      ]);
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull();
    },
  );
});
