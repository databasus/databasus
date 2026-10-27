// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import type { BackupConfig } from '../../../entity/backups/config';
import {
  type Database,
  DatabaseType,
  type HealthStatus,
  createDefaultDatabaseSettings,
} from '../../../entity/databases';
import type { Storage } from '../../../entity/storages';
import type { Workspace } from '../../../entity/workspaces';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { DatabasesComponent } from './DatabasesComponent';

const mockServer = setUpMockApi();

const workspace: Workspace = {
  id: 'workspace-id',
  name: 'Production',
  createdAt: '2026-01-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_VIEWER',
};

const buildDatabase = (name: string, healthStatus: HealthStatus = 'AVAILABLE'): Database => ({
  id: `${name}-id`,
  workspaceId: workspace.id,
  name,
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
  healthStatus,
});

const buildStorage = (name: string): Storage => ({
  id: `${name}-id`,
  workspaceId: workspace.id,
  name,
  settings: { type: 'LOCAL' },
});

const buildBackupConfig = (database: Database, storage: Storage): BackupConfig => ({
  databaseId: database.id,
  isBackupsEnabled: true,
  storageId: storage.id,
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'DAILY', timeOfDay: '04:00' },
    retention: {
      policy: 'COUNT',
      period: 'MONTH',
      count: 10,
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

// The selected database's details load through operations these tests do not cover.
const leaveOtherRequestsPending = () =>
  http.all(`*${API_BASE_PATH}/*`, () => new Promise<never>(() => {}));

describe('DatabasesComponent', () => {
  it('lists unavailable databases first when the server lists them after available ones', async () => {
    mockServer.use(
      mockApi.get('/databases', 200, {
        items: [
          buildDatabase('Alpha', 'AVAILABLE'),
          buildDatabase('Bravo', 'UNAVAILABLE'),
          buildDatabase('Charlie', 'AVAILABLE'),
          buildDatabase('Delta', 'UNAVAILABLE'),
        ],
      }),
      mockApi.get('/backup-configs', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
      leaveOtherRequestsPending(),
    );

    renderScreen(<DatabasesComponent contentHeight={800} workspace={workspace} isCanManageDBs />);

    await screen.findByText('Alpha');
    const listedNames = screen
      .getAllByText(/^(Alpha|Bravo|Charlie|Delta)$/)
      .map((element) => element.textContent);
    expect(listedNames).toEqual(['Bravo', 'Delta', 'Alpha', 'Charlie']);
  });

  it('shows each card its storage name from one backup config list request', async () => {
    const ordersDatabase = buildDatabase('Orders');
    const usersDatabase = buildDatabase('Users');
    const draftDatabase = buildDatabase('Drafts');
    const nightlyStorage = buildStorage('Nightly disk');
    const archiveStorage = buildStorage('Archive disk');
    const backupConfigListQueries: URLSearchParams[] = [];
    const workspaceBackupConfigPage: ApiComponents['schemas']['BackupConfigPage'] = {
      items: [
        buildBackupConfig(ordersDatabase, nightlyStorage),
        buildBackupConfig(usersDatabase, archiveStorage),
      ],
    };
    mockServer.use(
      mockApi.get('/databases', 200, { items: [ordersDatabase, usersDatabase, draftDatabase] }),
      mockApi.get('/storages', 200, { items: [nightlyStorage, archiveStorage] }),
      http.get(`*${API_BASE_PATH}/backup-configs`, ({ request }) => {
        backupConfigListQueries.push(new URL(request.url).searchParams);
        return HttpResponse.json(workspaceBackupConfigPage);
      }),
      leaveOtherRequestsPending(),
    );

    renderScreen(<DatabasesComponent contentHeight={800} workspace={workspace} isCanManageDBs />);

    expect(await screen.findByText('Nightly disk')).toBeTruthy();
    expect(screen.getByText('Archive disk')).toBeTruthy();
    expect(backupConfigListQueries).toHaveLength(1);
    expect(backupConfigListQueries[0].get('workspaceId')).toBe(workspace.id);
    expect(backupConfigListQueries[0].has('storageId')).toBe(false);
  });
});
