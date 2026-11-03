import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import type { DatabaseTransfer } from '../model/DatabaseTransfer';
import type { StorageSwitch } from '../model/StorageSwitch';
import { backupConfigApi } from './backupConfigApi';

const server = setUpMockApi();
const transfer: DatabaseTransfer = {
  targetWorkspaceId: 'target-workspace',
  storage: { type: 'SWITCH_TO_STORAGE', storageId: 'target-storage' },
  isMovingNotifiers: false,
};

describe('backupConfigApi.listAllWorkspaceBackupConfigs', () => {
  it('returns the configs of every page and sends the workspace, limit and cursor', async () => {
    const requestedQueries: URLSearchParams[] = [];
    server.use(
      http.get('*/api/v1/backup-configs', ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        return HttpResponse.json(
          query.get('after') === null
            ? { items: [{ databaseId: 'first-database' }], nextCursor: 'second-page' }
            : { items: [{ databaseId: 'second-database' }] },
        );
      }),
    );

    const listedConfigs = await backupConfigApi.listAllWorkspaceBackupConfigs('workspace-id');

    expect(listedConfigs.map((config) => config.databaseId)).toEqual([
      'first-database',
      'second-database',
    ]);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(requestedQueries.map((query) => query.get('workspaceId'))).toEqual([
      'workspace-id',
      'workspace-id',
    ]);
    expect(requestedQueries.map((query) => query.get('limit'))).toEqual(['100', '100']);
  });
});

describe('backupConfigApi.transferDatabase', () => {
  it('rejects missing operation data when the transfer is accepted', async () => {
    server.use(
      http.post(
        '*/api/v1/databases/:databaseId/transfer',
        () => new HttpResponse(null, { status: 202 }),
      ),
    );
    await expect(backupConfigApi.transferDatabase('shop-id', transfer)).rejects.toMatchObject({
      code: 'request_failed',
      status: 202,
    });
    expect(server.getRequests()).toHaveLength(1);
  });

  it('returns the operation when the transfer is accepted with data', async () => {
    const storageSwitch: StorageSwitch = {
      id: 'switch-id',
      databaseId: 'shop-id',
      workspaceId: 'target-workspace',
      targetStorageId: 'target-storage',
      previousStorageId: 'old-storage',
      status: 'CLEANUP_PENDING',
      createdAt: '2026-10-30T10:00:00Z',
      cleanupDeadline: '2026-10-31T10:00:00Z',
      total: 4,
      pending: 4,
      deleted: 0,
      remaining: 0,
    };
    server.use(mockApi.post('/databases/{databaseId}/transfer', 202, storageSwitch));

    await expect(backupConfigApi.transferDatabase('shop-id', transfer)).resolves.toEqual(
      storageSwitch,
    );
    expect(server.getRequests()).toHaveLength(1);
  });

  it('completes without operation data when the transfer returns 204', async () => {
    server.use(mockApi.post('/databases/{databaseId}/transfer', 204, undefined));
    await expect(backupConfigApi.transferDatabase('shop-id', transfer)).resolves.toBeUndefined();
    expect(server.getRequests()).toHaveLength(1);
  });
});
