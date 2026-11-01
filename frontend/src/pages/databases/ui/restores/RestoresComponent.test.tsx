// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { describe, expect, it } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import type { LogicalBackup } from '../../model/LogicalBackup';
import { RestoresComponent } from './RestoresComponent';

dayjs.extend(utc);

const mockServer = setUpMockApi();

const buildDatabase = (type: DatabaseType): Database => ({
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(type),
  notifierIds: [],
});

const completedBackup: LogicalBackup = {
  id: 'backup-id',
  databaseId: 'database-id',
  storageId: 'storage-id',
  status: 'COMPLETED',
  sizeMb: 12,
  rawDatabaseSizeMb: 40,
  durationMs: 3_000,
  encryption: 'NONE',
  verificationStatus: 'NOT_VERIFIED',
  createdAt: '2026-10-27T10:00:00Z',
};

describe('RestoresComponent', () => {
  it('offers no target form for a physical database', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    renderScreen(
      <RestoresComponent
        database={buildDatabase(DatabaseType.POSTGRES_PHYSICAL)}
        backup={completedBackup}
      />,
    );

    expect(await screen.findByText('No restores yet')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Select database to restore to' })).toBeNull();
  });
});
