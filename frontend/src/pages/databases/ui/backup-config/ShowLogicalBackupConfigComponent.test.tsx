// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type BackupConfig, toLogicalBackupConfigInput } from '@/entities/backup-configs';
import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import type { Schedule } from '@/entities/schedules';
import { ShowLogicalBackupConfigComponent } from '@/pages/databases/ui/backup-config/ShowLogicalBackupConfigComponent';
import { setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
};

const renderWeeklyConfig = (weekday: number) => {
  const schedule: Schedule = { type: 'WEEKLY', timeOfDay: '12:00', weekday };
  const config: BackupConfig = {
    cleanup: { pendingFileCount: 0, failedFileCount: 0 },
    databaseId: database.id,
    storageId: null,
    isBackupsEnabled: true,
    encryption: 'NONE',
    settings: {
      type: 'LOGICAL',
      schedule,
      retention: {
        policy: 'COUNT',
        count: 5,
        period: 'WEEK',
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

  mockServer.use();

  renderScreen(
    <ShowLogicalBackupConfigComponent
      database={database}
      backupConfig={toLogicalBackupConfigInput(config)!}
    />,
  );
};

describe('ShowLogicalBackupConfigComponent', () => {
  beforeEach(() => {
    vi.stubEnv('TZ', 'UTC');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('names the fields of a weekly schedule without the word Backup', async () => {
    renderWeeklyConfig(1);

    expect(await screen.findByText('Weekday')).toBeTruthy();
    expect(screen.getByText('Time of day')).toBeTruthy();
    expect(screen.getByText('Mon')).toBeTruthy();
    expect(screen.queryByText('Backup weekday')).toBeNull();
    expect(screen.queryByText('Backup time of day')).toBeNull();
  });

  it('shows a saved weekday 0 as Sunday', async () => {
    renderWeeklyConfig(0);

    expect(await screen.findByText('Sun')).toBeTruthy();
  });
});
