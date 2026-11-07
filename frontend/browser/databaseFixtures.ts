import type { components } from '../src/shared/api/schema.gen';
import { applicationFixtures, workspace } from './applicationFixtures';
import { apiResponse } from './HttpFixture';

export const database: components['schemas']['Database'] = {
  id: 'orders-id',
  workspaceId: workspace.id,
  name: 'Orders',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_LOGICAL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    database: 'orders',
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
    includeSchemas: [],
    excludeTables: [],
    excludeTableData: [],
    cpuCount: 1,
    isSkipUserMappings: false,
  },
};
export const backup: components['schemas']['LogicalBackup'] = {
  id: 'completed-backup',
  databaseId: database.id,
  storageId: 'storage-id',
  status: 'COMPLETED',
  sizeMb: 101,
  rawDatabaseSizeMb: 500,
  durationMs: 3000,
  encryption: 'NONE',
  verificationStatus: 'NOT_VERIFIED',
  createdAt: '2026-10-18T10:00:00Z',
};
export const backupConfig: components['schemas']['BackupConfig'] = {
  databaseId: database.id,
  storageId: 'storage-id',
  isBackupsEnabled: true,
  encryption: 'NONE',
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'DAILY', timeOfDay: '04:00' },
    retention: {
      policy: 'COUNT',
      count: 10,
      period: 'MONTH',
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
export const databaseFixtures = () =>
  applicationFixtures()
    .filter((fixture) => !['/databases', '/backup-configs', '/storages'].includes(fixture.path))
    .concat([
      apiResponse({
        method: 'get',
        path: '/storages/{storageId}',
        status: 200,
        body: {
          id: 'storage-id',
          workspaceId: workspace.id,
          name: 'Archive storage with a long descriptive label',
          settings: { type: 'LOCAL' },
        },
      }),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
      apiResponse({
        method: 'get',
        path: '/backup-configs',
        status: 200,
        body: { items: [backupConfig] },
      }),
      apiResponse({
        method: 'get',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: backupConfig,
      }),
      apiResponse({
        method: 'get',
        path: '/storages',
        status: 200,
        body: {
          items: [
            {
              id: 'storage-id',
              workspaceId: workspace.id,
              name: 'Archive storage with a long descriptive label',
              settings: { type: 'LOCAL' },
            },
          ],
        },
      }),
      apiResponse({
        method: 'get',
        path: '/logical-backups',
        status: 200,
        body: {
          items: [
            backup,
            {
              ...backup,
              id: 'failed-backup',
              status: 'FAILED',
              failMessage: 'pg_dump: permission denied <literal>\n' + 'long_identifier_'.repeat(60),
            },
          ],
          totalUsageMb: 101,
        },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isHealthcheckEnabled: false,
          isNotificationEnabled: false,
          intervalMinutes: 1,
          failedAttemptsBeforeUnavailable: 3,
          attemptRetentionDays: 7,
        },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-attempts',
        status: 200,
        body: { items: [] },
      }),
      apiResponse({
        method: 'get',
        path: '/verification-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'DAILY' },
          notificationTriggers: [],
        },
      }),
      apiResponse({ method: 'get', path: '/restores', status: 200, body: { items: [] } }),
    ]);
