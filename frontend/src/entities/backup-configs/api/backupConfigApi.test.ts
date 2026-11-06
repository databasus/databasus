import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { backupConfigApi } from '@/entities/backup-configs/api/backupConfigApi';
import type { DatabaseTransfer } from '@/entities/backup-configs/model/DatabaseTransfer';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';

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
  it('completes without operation data when the transfer returns 204', async () => {
    server.use(mockApi.post('/databases/{databaseId}/transfer', 204, undefined));
    await expect(backupConfigApi.transferDatabase('shop-id', transfer)).resolves.toBeUndefined();
    expect(server.getRequests()).toHaveLength(1);
  });
});
