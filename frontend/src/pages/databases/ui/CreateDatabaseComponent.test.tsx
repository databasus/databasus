// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Database } from '../../../entities/databases';
import type { Storage } from '../../../entities/storages';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { CreateDatabaseComponent } from './CreateDatabaseComponent';

dayjs.extend(utc);

const mockServer = setUpMockApi();

const archiveStorage: Storage = {
  id: 'archive-storage-id',
  workspaceId: 'workspace-id',
  name: 'Archive',
  settings: { type: 'LOCAL' },
};

const nightlyStorage: Storage = { ...archiveStorage, id: 'nightly-storage-id', name: 'Nightly' };

const renderWizard = (onCreated: (database: Database) => void = () => {}) =>
  renderScreen(
    <CreateDatabaseComponent workspaceId="workspace-id" onCreated={onCreated} onClose={() => {}} />,
  );

const choosePostgresBackupType = async (backupType: 'Logical' | 'Physical') => {
  fireEvent.click(await screen.findByText(backupType));
  fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
};

const enterBaseInfo = (engine: string) => {
  fireEvent.change(screen.getByPlaceholderText('My favourite DB'), { target: { value: 'Shop' } });
  fireEvent.click(screen.getByText(engine));
  fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
};

const enterConnection = async () => {
  fireEvent.change(await screen.findByPlaceholderText(/host$/), {
    target: { value: 'db.example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'backup' } });
  fireEvent.change(screen.getByPlaceholderText(/password$/), { target: { value: 'secret' } });
  if (screen.queryByPlaceholderText(/database name$/))
    fireEvent.change(screen.getByPlaceholderText(/database name$/), { target: { value: 'shop' } });
  fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
};

const chooseStorage = async (name: string) => {
  const storageLabel = await screen.findByText('Storage');
  const select = storageLabel.parentElement!.querySelector('.ant-select-selector')!;

  fireEvent.mouseDown(select);
  const option = await waitFor(() => {
    const storageOption = document.querySelector(`.ant-select-item-option[title="${name}"]`);
    expect(storageOption).toBeTruthy();
    return storageOption!;
  });

  fireEvent.click(option);
};

const getChosenStorageText = async () => {
  const storageField = (await screen.findByText('Storage')).parentElement!;
  await waitFor(() => expect(storageField.querySelector('.anticon-loading')).toBeNull());

  return storageField.querySelector('.ant-select-selector')!.textContent;
};

const declareWizardRequests = () =>
  mockServer.use(
    mockApi.post('/database-tests', 200, { isConnected: true }),
    mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
    mockApi.get('/storages', 200, { items: [archiveStorage, nightlyStorage] }),
    mockApi.get('/notifiers', 200, { items: [] }),
  );

const continueToNotifiers = async (storageName: string) => {
  await chooseStorage(storageName);
  fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
  await screen.findByRole('button', { name: 'Complete' });
};

const reachNotifierStep = async (storageName: string) => {
  enterBaseInfo('MySQL');
  await enterConnection();
  await continueToNotifiers(storageName);
};

// From the notifier step: 1 = backup configuration, 2 = connection, 3 = engine or backup type.
const goBackToStep = async (stepsBack: number) => {
  for (let step = 0; step < stepsBack; step++) {
    fireEvent.click(await screen.findByRole('button', { name: 'Back' }));
  }
};

describe('CreateDatabaseComponent', () => {
  it('advances to the read-only user step with the submitted connection settings', async () => {
    const limitedUserChecks: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
    mockServer.use(
      mockApi.post('/database-tests', 200, { isConnected: true }),
      http.post(`*${API_BASE_PATH}/limited-user-checks`, async ({ request }) => {
        limitedUserChecks.push(
          (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
        );

        return HttpResponse.json({ isAdvisable: true, privileges: [] });
      }),
    );
    renderWizard();

    enterBaseInfo('MySQL');
    await enterConnection();

    expect(await screen.findByText('Create a read-only user for Databasus?')).toBeTruthy();
    expect(limitedUserChecks).toHaveLength(1);
    expect(limitedUserChecks[0].settings).toMatchObject({
      type: 'MYSQL',
      host: 'db.example.com',
      username: 'backup',
      password: 'secret',
      database: 'shop',
    });
  });

  it('keeps the entered backup configuration after going back from the notifier step', async () => {
    declareWizardRequests();
    renderWizard();
    await reachNotifierStep('Archive');

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));

    expect(await getChosenStorageText()).toContain('Archive');
  });

  it('creates the database with the configuration changed after going back', async () => {
    const backupConfigUpdates: ApiComponents['schemas']['BackupConfigInput'][] = [];
    declareWizardRequests();
    mockServer.use(
      http.post(`*${API_BASE_PATH}/databases`, async ({ request }) =>
        HttpResponse.json(
          { ...((await request.json()) as Database), id: 'created-id' },
          {
            status: 201,
          },
        ),
      ),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
        backupConfigUpdates.push(
          (await request.json()) as ApiComponents['schemas']['BackupConfigInput'],
        );
        return HttpResponse.json({});
      }),
      mockApi.post('/logical-backups', 202, { id: 'backup-id' } as never),
    );
    const onCreated = vi.fn();
    renderWizard(onCreated);
    await reachNotifierStep('Archive');

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    await chooseStorage('Nightly');
    fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Complete' }));

    await waitFor(() => expect(onCreated).toHaveBeenCalled());
    expect(backupConfigUpdates).toHaveLength(1);
    expect(backupConfigUpdates[0].storageId).toBe(nightlyStorage.id);
  });

  it('starts the connection and backup configuration over when the engine changes', async () => {
    declareWizardRequests();
    renderWizard();
    await reachNotifierStep('Archive');

    await goBackToStep(3);
    enterBaseInfo('MariaDB');

    expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe('');

    await enterConnection();

    expect(await getChosenStorageText()).not.toContain('Archive');
  });

  it('sends one create request when Complete is clicked twice while pending', async () => {
    declareWizardRequests();
    const creation = mockApi.delayed({
      method: 'post',
      path: '/databases',
      status: 201,
      body: {
        id: 'created-id',
        workspaceId: 'workspace-id',
        name: 'Shop',
        settings: { type: 'MYSQL' },
        notifierIds: [],
      } as unknown as Database,
    });
    mockServer.use(
      creation.handler,
      mockApi.put('/backup-configs/{databaseId}', 200, {} as never),
      mockApi.post('/logical-backups', 202, { id: 'backup-id' } as never),
    );
    const onCreated = vi.fn();
    renderWizard(onCreated);
    await reachNotifierStep('Archive');
    const completeButton = screen.getByRole('button', { name: 'Complete' });

    fireEvent.click(completeButton);
    fireEvent.click(completeButton);
    await waitFor(() =>
      expect(
        mockServer
          .getRequests()
          .filter(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/databases'),
          ),
      ).toHaveLength(1),
    );
    creation.release();

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
  });

  it('keeps the entered physical backup configuration after going back from the notifier step', async () => {
    declareWizardRequests();
    renderWizard();
    enterBaseInfo('PostgreSQL');
    await choosePostgresBackupType('Physical');
    await enterConnection();
    await continueToNotifiers('Archive');

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));

    expect(await getChosenStorageText()).toContain('Archive');
  });

  it('starts the connection and both backup configurations over when the PostgreSQL backup type changes', async () => {
    declareWizardRequests();
    renderWizard();
    enterBaseInfo('PostgreSQL');
    await choosePostgresBackupType('Logical');
    await enterConnection();
    await continueToNotifiers('Archive');

    await goBackToStep(3);
    await choosePostgresBackupType('Physical');

    expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe('');

    await enterConnection();
    expect(await getChosenStorageText()).not.toContain('Archive');
    await continueToNotifiers('Nightly');
    await goBackToStep(3);
    await choosePostgresBackupType('Logical');

    expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe('');

    await enterConnection();

    expect(await getChosenStorageText()).not.toContain('Archive');
    expect(await getChosenStorageText()).not.toContain('Nightly');
  });
});
