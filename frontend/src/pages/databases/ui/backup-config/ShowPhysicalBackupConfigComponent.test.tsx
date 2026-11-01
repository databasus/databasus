// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  type BackupConfig,
  toPhysicalBackupConfigInput,
} from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { ShowPhysicalBackupConfigComponent } from './ShowPhysicalBackupConfigComponent';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  notifierIds: [],
};
const config: BackupConfig = {
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
    type: 'PHYSICAL',
    fullBackupSchedule: { type: 'WEEKLY', timeOfDay: '12:00', weekday: 0 },
    incrementalBackupSchedule: null,
    retention: {
      policy: 'FULL_BACKUPS',
      chainsCount: 0,
      fullBackups: {
        policy: 'LAST_N',
        count: 5,
        gfsHours: 0,
        gfsDays: 0,
        gfsWeeks: 0,
        gfsMonths: 0,
        gfsYears: 0,
      },
    },
    walLagThresholdBytes: 0,
    notificationTriggers: [],
  },
};

describe('ShowPhysicalBackupConfigComponent', () => {
  beforeEach(() => {
    vi.stubEnv('TZ', 'UTC');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('shows a saved weekday 0 as Sunday', async () => {
    mockServer.use(mockApi.get('/storage-switches', 200, { items: [] }));

    renderScreen(
      <ShowPhysicalBackupConfigComponent
        database={database}
        backupConfig={toPhysicalBackupConfigInput(config)!}
      />,
    );

    expect(await screen.findByText('Sun')).toBeTruthy();
    expect(screen.getByText('Weekday')).toBeTruthy();
  });
});
