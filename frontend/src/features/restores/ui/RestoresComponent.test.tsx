// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { LogicalBackup } from '../../../entities/backups/logical';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import type { Restore } from '../../../entities/restores';
import { API_BASE_PATH } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { RestoresComponent } from './RestoresComponent';
import type { RestoreTargetFormOptions } from './RestoreTargetFormOptions';

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

const startedRestore: Restore = {
  id: 'restore-id',
  backupId: completedBackup.id,
  status: 'IN_PROGRESS',
  durationMs: 0,
  createdAt: '2026-10-27T11:00:00Z',
};

describe('RestoresComponent', () => {
  it('offers no target form for a physical database', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    const renderRestoreTargetForm = vi.fn(() => null);
    renderScreen(
      <RestoresComponent
        database={buildDatabase(DatabaseType.POSTGRES_PHYSICAL)}
        backup={completedBackup}
        renderRestoreTargetForm={renderRestoreTargetForm}
      />,
    );

    expect(await screen.findByText('No restores yet')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Select database to restore to' })).toBeNull();
    expect(renderRestoreTargetForm).not.toHaveBeenCalled();
  });

  it('gives the form a restore draft with no database identity and lists a started restore', async () => {
    // The screen polls the list and shows what it answers, so a started restore has to stay listed.
    const listedRestores: Restore[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/restores`, () => HttpResponse.json({ items: listedRestores })),
      http.post(`*${API_BASE_PATH}/restores`, () => {
        listedRestores.push(startedRestore);

        return HttpResponse.json(startedRestore, { status: 202 });
      }),
    );
    const receivedOptions: RestoreTargetFormOptions[] = [];
    renderScreen(
      <RestoresComponent
        database={buildDatabase(DatabaseType.MYSQL)}
        backup={completedBackup}
        renderRestoreTargetForm={(options) => {
          receivedOptions.push(options);

          return <button onClick={() => void options.onSubmit(options.draft)}>Start target</button>;
        }}
      />,
    );

    fireEvent.click(await screen.findByRole('button', { name: 'Select database to restore to' }));
    fireEvent.click(screen.getByRole('button', { name: 'Start target' }));

    expect(await screen.findByText('In progress')).toBeTruthy();
    expect(receivedOptions[0].draft.scenario).toBe('restore');
    expect(receivedOptions[0].draft).not.toHaveProperty('databaseId');
    expect(receivedOptions[0].draft).not.toHaveProperty('workspaceId');
    expect(receivedOptions[0].draft.settings.host).toBe('');
  });
});
