// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import type { BackupConfig } from '../../../entity/backups/config';
import type { Storage } from '../../../entity/storages';
import type { Workspace } from '../../../entity/workspaces';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
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
});
