// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { EditDatabaseSpecificDataComponent } from './EditDatabaseSpecificDataComponent';

const mockServer = setUpMockApi();

// The API never returns a password, so a saved database reaches the form without one.
const savedMysqlDatabase: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: {
    ...createDefaultDatabaseSettings(DatabaseType.MYSQL),
    password: undefined,
    host: 'db.example.com',
    username: 'backup',
    database: 'shop',
  },
  notifierIds: [],
};

const savedPhysicalDatabase: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: {
    ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
    password: undefined,
    host: 'db.example.com',
    username: 'replicator',
  },
  notifierIds: [],
};

const renderSavedDatabaseForm = (database: Database, onSaved: (database: Database) => void) =>
  renderScreen(
    <EditDatabaseSpecificDataComponent
      database={database}
      isShowCancelButton
      onCancel={() => {}}
      isShowBackButton={false}
      onBack={() => {}}
      isSaveToApi
      onSaved={onSaved}
    />,
  );

describe('EditDatabaseSpecificDataComponent', () => {
  it('sends no password when a saved database is saved with the password left empty', async () => {
    const sentTests: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
    const sentUpdates: ApiComponents['schemas']['DatabaseUpdate'][] = [];
    mockServer.use(
      http.post(`*${API_BASE_PATH}/database-tests`, async ({ request }) => {
        sentTests.push(
          (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
        );
        return HttpResponse.json({ isConnected: true });
      }),
      http.put(`*${API_BASE_PATH}/databases/:databaseId`, async ({ request }) => {
        sentUpdates.push((await request.json()) as ApiComponents['schemas']['DatabaseUpdate']);
        return HttpResponse.json(savedMysqlDatabase);
      }),
      mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
    );
    const onSaved = vi.fn();
    const { container } = renderSavedDatabaseForm(savedMysqlDatabase, onSaved);

    const passwordInput = container.querySelector('input[type="password"]')!;
    fireEvent.change(passwordInput, { target: { value: 'typed by mistake' } });
    fireEvent.change(passwordInput, { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(sentTests[0].databaseId).toBe('database-id');
    expect(sentTests[0].settings).not.toHaveProperty('password');
    expect(sentUpdates).toHaveLength(1);
    expect(sentUpdates[0].settings).not.toHaveProperty('password');
    expect(sentUpdates[0].settings).toMatchObject({ host: 'db.example.com', username: 'backup' });
  });

  it('shows the reason and the server error when the connection test fails', async () => {
    mockServer.use(
      mockApi.post('/database-tests', 200, {
        isConnected: false,
        reason: 'PG_HBA_NO_ENTRY',
        error: 'no pg_hba.conf entry for replication connection from host "10.0.0.5"',
      }),
    );
    renderSavedDatabaseForm(savedPhysicalDatabase, () => {});

    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));

    expect(await screen.findByText('Replication is not allowed from this host')).toBeTruthy();
    expect(
      screen.getByText('no pg_hba.conf entry for replication connection from host "10.0.0.5"'),
    ).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });
});
