import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  physicalBackupsApi,
  type PhysicalRestoreTokenCreate,
} from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { API_BASE_PATH } from '@/shared/api';
import { setUpMockApi } from '@/shared/lib/test/mockApi';

const mockServer = setUpMockApi();

describe('physicalBackupsApi.createPhysicalRestoreToken', () => {
  for (const subject of [
    { type: 'BACKUP', backupId: 'backup-id' },
    { type: 'POINT_IN_TIME', databaseId: 'database-id', targetTime: '2026-10-28T12:00:00Z' },
  ] satisfies PhysicalRestoreTokenCreate[]) {
    it(`sends the ${subject.type} subject when a restore token is requested`, async () => {
      const requests: unknown[] = [];
      mockServer.use(
        http.post(`*${API_BASE_PATH}/physical-restore-tokens`, async ({ request }) => {
          requests.push(await request.json());
          return HttpResponse.json(
            { token: 'restore-token', expiresAt: '2026-10-28T12:15:00Z' },
            { status: 201 },
          );
        }),
      );

      const token = await physicalBackupsApi.createPhysicalRestoreToken(subject);

      expect(requests).toEqual([subject]);
      expect(token).toEqual({ token: 'restore-token', expiresAt: '2026-10-28T12:15:00Z' });
    });
  }
});
