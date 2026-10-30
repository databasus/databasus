// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import type { LogicalBackup } from '../../../entities/backups/logical';
import type { Database } from '../../../entities/databases';
import type { Restore } from '../../../entities/restores';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { RestoresComponent } from './RestoresComponent';

// main.tsx registers the plugin for the application; the restore cards render times through it.
dayjs.extend(utc);

const mockServer = setUpMockApi();

const backedUpDatabase: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: {
    type: 'MYSQL',
    version: '8.4',
    host: 'db.example.com',
    port: 3306,
    username: 'backup',
    database: 'shop',
    isHttps: false,
    sshTunnel: { isEnabled: false },
    excludeTables: [],
  },
  notifierIds: [],
};

const completedBackup: LogicalBackup = {
  id: 'backup-id',
  databaseId: backedUpDatabase.id,
  storageId: 'storage-id',
  status: 'COMPLETED',
  sizeMb: 12,
  rawDatabaseSizeMb: 40,
  durationMs: 3_000,
  encryption: 'NONE',
  verificationStatus: 'NOT_VERIFIED',
  createdAt: '2026-10-27T10:00:00Z',
};

const startedRestore: Restore = {
  id: 'restore-id',
  backupId: completedBackup.id,
  status: 'IN_PROGRESS',
  durationMs: 0,
  createdAt: '2026-10-27T11:00:00Z',
};

const restoreIntoTarget = async () => {
  fireEvent.click(await screen.findByRole('button', { name: 'Select database to restore to' }));
  fireEvent.change(screen.getByPlaceholderText(/host$/), {
    target: { value: 'restore-target.example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'restorer' } });
  fireEvent.change(screen.getByPlaceholderText(/password$/), {
    target: { value: 'target-secret' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Restore to this DB' }));
};

describe('RestoresComponent', () => {
  it('starts the restore into the MySQL target and shows the returned restore', async () => {
    const restoreStarts: ApiComponents['schemas']['RestoreStart'][] = [];
    mockServer.use(
      mockApi.get('/restores', 200, { items: [] }),
      mockApi.post('/database-tests', 200, { isConnected: true }),
      http.post(`*${API_BASE_PATH}/restores`, async ({ request }) => {
        restoreStarts.push((await request.json()) as ApiComponents['schemas']['RestoreStart']);

        return HttpResponse.json(startedRestore, { status: 202 });
      }),
    );
    renderScreen(<RestoresComponent database={backedUpDatabase} backup={completedBackup} />);

    await restoreIntoTarget();

    expect(await screen.findByText('In progress')).toBeTruthy();
    expect(restoreStarts).toEqual([
      {
        backupId: completedBackup.id,
        target: {
          type: 'MYSQL',
          host: 'restore-target.example.com',
          port: 3306,
          username: 'restorer',
          password: 'target-secret',
          database: 'shop',
          isHttps: false,
          sshTunnel: { isEnabled: false },
        },
      },
    ]);
  });

  it.each([
    {
      code: 'database_connection_failed' as const,
      status: 400,
      shownText: 'Could not connect to the database: database_connection_failed',
    },
    {
      code: 'already_running' as const,
      status: 409,
      shownText: 'This is already running. Wait for it to finish and try again.',
    },
    {
      code: 'insufficient_disk_space' as const,
      status: 409,
      shownText:
        'The Databasus server does not have enough free disk space: insufficient_disk_space',
    },
  ])('shows $code inline in the target form when the restore is refused', async (refusal) => {
    mockServer.use(
      mockApi.get('/restores', 200, { items: [] }),
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.post('/restores', refusal.status, buildProblem(refusal.code, refusal.status)),
    );
    renderScreen(<RestoresComponent database={backedUpDatabase} backup={completedBackup} />);

    await restoreIntoTarget();

    expect(await screen.findByText(refusal.shownText)).toBeTruthy();
    expect(screen.getByDisplayValue('restore-target.example.com')).toBeTruthy();
    expect(screen.queryByText('No restores yet')).toBeNull();
  });
});
