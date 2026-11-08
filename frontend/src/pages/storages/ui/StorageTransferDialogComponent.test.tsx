// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '@/entities/backup-configs';
import { backupConfigApi } from '@/entities/backup-configs';
import type { Storage } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { StorageTransferDialogComponent } from '@/pages/storages/ui/StorageTransferDialogComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { inspectApiRequest } from '@/shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
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

const transferredStorage: Storage = {
  id: 'storage-id',
  workspaceId: sourceWorkspace.id,
  name: 'Nightly disk',
  settings: { type: 'LOCAL' },
};

const backupConfigOfStorage: BackupConfig = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: 'database-id',
  isBackupsEnabled: true,
  storageId: transferredStorage.id,
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
};

const mockStorageBackupConfigs = (
  storageBackupConfigs: BackupConfig[],
  backupConfigListQueries: URLSearchParams[],
) =>
  http.get(`*${API_BASE_PATH}/backup-configs`, ({ request }) => {
    backupConfigListQueries.push(new URL(request.url).searchParams);
    const page: ApiComponents['schemas']['BackupConfigPage'] = { items: storageBackupConfigs };
    return HttpResponse.json(page);
  });

const renderDialog = () =>
  renderScreen(
    <StorageTransferDialogComponent
      storage={transferredStorage}
      onClose={() => {}}
      onTransferred={() => {}}
    />,
  );

describe('StorageTransferDialogComponent', () => {
  it('blocks the transfer when a backup config of one database uses the storage', async () => {
    const backupConfigListQueries: URLSearchParams[] = [];
    mockServer.use(mockStorageBackupConfigs([backupConfigOfStorage], backupConfigListQueries));
    renderDialog();

    expect(
      await screen.findByText(
        'This storage is used by some databases. Please transfer or remove related databases first.',
      ),
    ).toBeTruthy();
    expect(backupConfigListQueries).toHaveLength(1);
    expect(backupConfigListQueries[0].get('storageId')).toBe(transferredStorage.id);
    expect(backupConfigListQueries[0].get('limit')).toBe('1');
    expect(backupConfigListQueries[0].has('workspaceId')).toBe(false);
  });

  it('offers the other workspaces when no backup config uses the storage', async () => {
    const backupConfigListQueries: URLSearchParams[] = [];
    mockServer.use(
      mockStorageBackupConfigs([], backupConfigListQueries),
      mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
    );
    renderDialog();

    expect(await screen.findByText('Select a workspace to transfer this storage to.')).toBeTruthy();
    expect(
      screen.queryByText(
        'This storage is used by some databases. Please transfer or remove related databases first.',
      ),
    ).toBeNull();
  });

  it('reloads the dialog data through Retry when the data fails to load', async () => {
    mockServer.use(mockApi.get('/backup-configs', 500, buildProblem('internal', 500)));
    const view = renderDialog();

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    mockServer.use(
      mockStorageBackupConfigs([], []),
      mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
    );

    await view.user.click(retryButton);

    expect(await screen.findByRole('combobox', { name: 'Target workspace' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('keeps the dialog and the chosen workspace and shows the translated reason when the transfer is refused', async () => {
    mockServer.use(
      mockStorageBackupConfigs([], []),
      mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
      mockApi.post('/storages/{storageId}/transfer', 409, buildProblem('in_use', 409)),
    );
    const onClose = vi.fn();
    const onTransferred = vi.fn();
    const view = renderScreen(
      <StorageTransferDialogComponent
        storage={transferredStorage}
        onClose={onClose}
        onTransferred={onTransferred}
      />,
    );

    fireEvent.click(await screen.findByRole('combobox', { name: 'Target workspace' }));
    fireEvent.click(await screen.findByRole('option', { name: targetWorkspace.name }));
    await view.user.click(screen.getByRole('button', { name: /Transfer/ }));

    expect(
      await screen.findByText(
        'It is still in use by a database or by a running backup or restore. Try again once nothing uses it.',
      ),
    ).toBeTruthy();
    expect(screen.getByRole('dialog')).toBeTruthy();
    expect(screen.queryByText('in_use')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(
      within(screen.getByRole('combobox', { name: 'Target workspace' })).getByText(
        targetWorkspace.name,
      ),
    ).toBeTruthy();
    expect(onTransferred).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();

    mockServer.use(
      http.post(
        `*${API_BASE_PATH}/storages/:storageId/transfer`,
        () => new HttpResponse(null, { status: 204 }),
      ),
    );

    await view.user.click(screen.getByRole('button', { name: /Transfer/ }));

    await waitFor(() => expect(onTransferred).toHaveBeenCalledTimes(1));

    const transfers = mockServer.getRequests().filter((request) => request.method === 'POST');

    expect(transfers).toHaveLength(2);

    for (const transfer of transfers) {
      const inspectedTransfer = await inspectApiRequest(transfer, {
        method: 'post',
        path: '/storages/{storageId}/transfer',
      });

      expect(inspectedTransfer.params.storageId).toBe(transferredStorage.id);
      expect(inspectedTransfer.body).toEqual({ targetWorkspaceId: targetWorkspace.id });
    }
  });
});

const TransferDialog = ({
  resourceId = transferredStorage.id,
  onClose,
  onTransferred,
}: {
  resourceId?: string;
  onClose: () => void;
  onTransferred: () => void;
}) => (
  <StorageTransferDialogComponent
    storage={{ ...transferredStorage, id: resourceId }}
    onClose={onClose}
    onTransferred={onTransferred}
  />
);
const usagePath = '/backup-configs';
const availableTargets = () =>
  mockServer.use(
    mockApi.get(usagePath, 200, { items: [] }),
    mockApi.get('/workspaces', 200, { items: [sourceWorkspace, targetWorkspace] }),
  );
const chooseTarget = async () => {
  fireEvent.click(await screen.findByRole('combobox', { name: 'Target workspace' }));

  expect(screen.queryByRole('option', { name: sourceWorkspace.name })).toBeNull();

  fireEvent.click(await screen.findByRole('option', { name: targetWorkspace.name }));
};

describe('Storage transfer lifecycle', () => {
  it('loads current targets when Strict Mode cancels its first setup', async () => {
    const usageRead = vi.spyOn(backupConfigApi, 'listStorageBackupConfigs');
    availableTargets();
    renderScreen(<TransferDialog onClose={() => {}} onTransferred={() => {}} />, {
      isStrictMode: true,
    });
    await screen.findByRole('combobox', { name: 'Target workspace' });
    expect(usageRead.mock.calls.length).toBeGreaterThanOrEqual(2);
    expect(usageRead.mock.calls[0][2]?.signal?.aborted).toBe(true);
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('submits once and calls the current completion handler when transfer options change while pending', async () => {
    availableTargets();
    const transfer = createPendingApiResponse();
    let transfers = 0;
    mockServer.use(
      http.post(`*${API_BASE_PATH}/storages/:resourceId/transfer`, () => {
        transfers++;
        return transfer.response;
      }),
    );
    const oldTransferred = vi.fn();
    const currentTransferred = vi.fn();
    const view = renderScreen(<TransferDialog onClose={() => {}} onTransferred={oldTransferred} />);

    await chooseTarget();
    const submit = screen.getByRole('button', { name: 'Transfer' });
    act(() => {
      submit.click();
      submit.click();
    });
    await waitFor(() => expect(transfers).toBe(1));

    view.rerender(<TransferDialog onClose={() => {}} onTransferred={currentTransferred} />);
    transfer.respond(new HttpResponse(null, { status: 204 }));
    await waitFor(() => expect(currentTransferred).toHaveBeenCalledTimes(1));

    expect(oldTransferred).not.toHaveBeenCalled();
    expect(transfers).toBe(1);
  });

  it.each([204, 409])(
    'resets the target and ignores an obsolete HTTP %i transfer after the resource changes',
    async (status) => {
      availableTargets();
      const transfer = createPendingApiResponse();
      let transfers = 0;
      mockServer.use(
        http.post(`*${API_BASE_PATH}/storages/:resourceId/transfer`, () => {
          transfers++;
          return transfer.response;
        }),
      );
      const onTransferred = vi.fn();
      const view = renderScreen(
        <TransferDialog onClose={() => {}} onTransferred={onTransferred} />,
      );

      await chooseTarget();
      await view.user.click(screen.getByRole('button', { name: 'Transfer' }));
      await waitFor(() => expect(transfers).toBe(1));

      view.rerender(
        <TransferDialog
          resourceId="replacement-id"
          onClose={() => {}}
          onTransferred={onTransferred}
        />,
      );
      await screen.findByRole('combobox', { name: 'Target workspace' });

      expect(
        within(screen.getByRole('combobox', { name: 'Target workspace' })).queryByText(
          targetWorkspace.name,
        ),
      ).toBeNull();
      expect((screen.getByRole('button', { name: 'Transfer' }) as HTMLButtonElement).disabled).toBe(
        true,
      );

      await finishApiResponse(
        mockServer,
        `/api/v1/storages/${transferredStorage.id}/transfer`,
        () =>
          transfer.respond(
            status === 204
              ? new HttpResponse(null, { status: 204 })
              : HttpResponse.json(buildProblem('in_use', 409), { status: 409 }),
          ),
      );

      expect(onTransferred).not.toHaveBeenCalled();
      expect(screen.queryByRole('alert')).toBeNull();

      await chooseTarget();

      expect((screen.getByRole('button', { name: 'Transfer' }) as HTMLButtonElement).disabled).toBe(
        false,
      );
    },
  );

  it.each([200, 500])(
    'stops the target read and ignores HTTP %i usage results after closing',
    async (status) => {
      const usageRead = mockApi.pending('get', usagePath);
      mockServer.use(usageRead.handler);
      const onTransferred = vi.fn();
      const onClose = vi.fn(() => view.rerender(<></>));
      const view = renderScreen(<TransferDialog onClose={onClose} onTransferred={onTransferred} />);
      await waitFor(() => expect(mockServer.getRequests()).toHaveLength(1));
      await view.user.click(screen.getByRole('button', { name: 'Close' }));
      await finishCancelledApiResponse(mockServer, `/api/v1${usagePath}`, () =>
        usageRead.respond(
          status === 200
            ? HttpResponse.json({ items: [] })
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
        ),
      );

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(onTransferred).not.toHaveBeenCalled();
      expect(mockServer.getRequests()).toHaveLength(1);
    },
  );

  it.each([204, 409])(
    'ignores a pending HTTP %i transfer after closing the dialog',
    async (status) => {
      availableTargets();
      const transfer = createPendingApiResponse();
      mockServer.use(
        http.post(`*${API_BASE_PATH}/storages/:resourceId/transfer`, () => transfer.response),
      );
      const onTransferred = vi.fn();
      const onClose = vi.fn(() => view.rerender(<></>));
      const view = renderScreen(<TransferDialog onClose={onClose} onTransferred={onTransferred} />);
      await chooseTarget();
      await view.user.click(screen.getByRole('button', { name: 'Transfer' }));
      await waitFor(() =>
        expect(
          mockServer.getRequests().filter((request) => request.method === 'POST'),
        ).toHaveLength(1),
      );
      await view.user.click(screen.getByRole('button', { name: 'Cancel' }));
      await finishApiResponse(
        mockServer,
        `/api/v1/storages/${transferredStorage.id}/transfer`,
        () =>
          transfer.respond(
            status === 204
              ? new HttpResponse(null, { status: 204 })
              : HttpResponse.json(buildProblem('in_use', 409), { status: 409 }),
          ),
      );

      expect(screen.queryByRole('dialog')).toBeNull();
      expect(onTransferred).not.toHaveBeenCalled();
      expect(onClose).toHaveBeenCalledTimes(1);
    },
  );
});
