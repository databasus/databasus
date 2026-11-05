// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '../../../entities/backup-configs';
import type { Storage } from '../../../entities/storages';
import type { Workspace } from '../../../entities/workspaces';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { inspectApiRequest } from '../../../shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { StorageTransferDialogComponent } from './StorageTransferDialogComponent';

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

    fireEvent.mouseDown(await screen.findByRole('combobox', { name: 'Target workspace' }));
    fireEvent.click(
      await screen.findByText(targetWorkspace.name, {
        selector: '.ant-select-item-option-content',
      }),
    );
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
      screen.getByText(targetWorkspace.name, { selector: '.ant-select-selection-item' }),
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
