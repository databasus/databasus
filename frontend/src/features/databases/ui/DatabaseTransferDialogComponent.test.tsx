// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import type { BackupConfig, DatabaseTransfer, StorageSwitch } from '../../../entity/backups/config';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entity/databases';
import type { Storage } from '../../../entity/storages';
import type { Workspace } from '../../../entity/workspaces';
import { API_BASE_PATH } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { DatabaseTransferDialogComponent } from './DatabaseTransferDialogComponent';

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
  fireEvent.mouseDown(combobox);

  const option = await waitFor(() => {
    const matchingOption = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
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
      screen.getByText(/Switching storage removes the old backup history immediately/),
    ).toBeTruthy();
    const warning = screen.getByText(
      /Switching storage removes the old backup history immediately/,
    );
    expect(warning.textContent).toContain('The new storage becomes active immediately.');
    expect(warning.textContent).toContain('If the old storage is unreachable, files may remain.');
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

  it('shows cleanup progress in the target workspace after an accepted switch', async () => {
    mockTransferOperations([buildBackupConfig(transferredDatabase.id)]);
    const acceptedSwitch: StorageSwitch = {
      id: 'switch-id',
      databaseId: transferredDatabase.id,
      workspaceId: targetWorkspace.id,
      targetStorageId: targetStorage.id,
      previousStorageId: currentStorage.id,
      status: 'CLEANUP_PENDING',
      createdAt: '2026-10-30T10:00:00Z',
      cleanupDeadline: '2026-10-31T10:00:00Z',
      total: 4,
      pending: 4,
      deleted: 0,
      remaining: 0,
    };
    mockServer.use(
      mockApi.post('/databases/{databaseId}/transfer', 202, acceptedSwitch),
      http.get(`*${API_BASE_PATH}/storage-switches`, ({ request }) => {
        expect(new URL(request.url).searchParams.get('workspaceId')).toBe(targetWorkspace.id);
        return HttpResponse.json({ items: [acceptedSwitch] });
      }),
    );
    renderDialog();
    await chooseTargetWorkspace();
    await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));
    await chooseSelectOption(screen.getAllByRole('combobox')[1], targetStorage.name);
    fireEvent.click(screen.getByRole('button', { name: 'Transfer' }));
    expect(
      await screen.findByText('Storage switched. Old files are being cleaned up.'),
    ).toBeTruthy();
    expect(screen.getByText('Files: 4. Pending: 4. Deleted: 0. Remaining: 0.')).toBeTruthy();
    expect(screen.getByText(targetStorage.name)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Transfer' })).toBeNull();
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
});
