// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { type Database, databaseApi } from '../../../entities/databases';
import type { Storage } from '../../../entities/storages';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { releaseAndSettle } from '../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { stubClipboard } from '../../../shared/lib/test/stubClipboard';
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

const reachPostgresBackupType = (backupType: 'Logical' | 'Physical') => async () => {
  enterBaseInfo('PostgreSQL');
  await choosePostgresBackupType(backupType);
};

const reachEngine = (engine: string) => async () => enterBaseInfo(engine);

const CONNECTION_VARIANTS = [
  {
    variant: 'PostgreSQL logical',
    reachConnectionStep: reachPostgresBackupType('Logical'),
    type: 'POSTGRES_LOGICAL',
    connectionString: 'postgresql://importer:imported-secret@imported.example.com:5433/imported',
    previousStepText: 'Physical',
  },
  {
    variant: 'PostgreSQL physical',
    reachConnectionStep: reachPostgresBackupType('Physical'),
    type: 'POSTGRES_PHYSICAL',
    connectionString: 'postgresql://importer:imported-secret@imported.example.com:5433/imported',
    previousStepText: 'Logical',
  },
  {
    variant: 'MySQL',
    reachConnectionStep: reachEngine('MySQL'),
    type: 'MYSQL',
    connectionString: 'mysql://importer:imported-secret@imported.example.com:3307/imported',
    previousStepText: 'MongoDB',
  },
  {
    variant: 'MariaDB',
    reachConnectionStep: reachEngine('MariaDB'),
    type: 'MARIADB',
    connectionString: 'mariadb://importer:imported-secret@imported.example.com:3307/imported',
    previousStepText: 'MongoDB',
  },
  {
    variant: 'MongoDB',
    reachConnectionStep: reachEngine('MongoDB'),
    type: 'MONGODB',
    connectionString: 'mongodb://importer:imported-secret@imported.example.com:27018/imported',
    previousStepText: 'MySQL',
  },
] as const;

const getInputByPlaceholder = (placeholder: RegExp) =>
  screen.getByPlaceholderText(placeholder) as HTMLInputElement;

const fillConnectionExceptPassword = async () => {
  fireEvent.change(await screen.findByPlaceholderText(/host$/), {
    target: { value: 'db.example.com' },
  });
  fireEvent.change(getInputByPlaceholder(/username$/), { target: { value: 'backup' } });

  if (screen.queryByPlaceholderText(/database name$/))
    fireEvent.change(getInputByPlaceholder(/database name$/), { target: { value: 'shop' } });
};

const getTestButton = () =>
  screen.getByRole('button', { name: /Test connection$/ }) as HTMLButtonElement;

const recordSettingsTests = () => {
  const sentTests: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
  mockServer.use(
    http.post(`*${API_BASE_PATH}/database-tests`, async ({ request }) => {
      sentTests.push((await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget']);

      return HttpResponse.json({ isConnected: true });
    }),
  );

  return sentTests;
};

const getRowOfLabel = (label: string) => screen.getByText(label).parentElement!;

const chooseSslMode = async (modeLabel: RegExp) => {
  fireEvent.mouseDown(getRowOfLabel('SSL mode').querySelector('.ant-select-selector')!);
  const option = await waitFor(() => {
    const modeOption = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
      (element) => modeLabel.test(element.textContent ?? ''),
    );
    expect(modeOption).toBeTruthy();
    return modeOption!;
  });

  fireEvent.click(option);
};

const getSelectedSslMode = () =>
  getRowOfLabel('SSL mode').querySelector('.ant-select-selection-item')!.textContent;

const openAdvancedWhenClosed = () => {
  if (!screen.queryByText('SSH tunnel')) fireEvent.click(screen.getByText('Advanced settings'));
};

const expectDatabaseTestRequests = (count: number) =>
  waitFor(() =>
    expect(
      mockServer.getRequests().filter((request) => request.url.endsWith('/database-tests')),
    ).toHaveLength(count),
  );

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
    enterBaseInfo('MongoDB');

    expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe('');
    expect((screen.getByPlaceholderText(/password$/) as HTMLInputElement).value).toBe('');
    expect((screen.getByLabelText('Port') as HTMLInputElement).value).toBe('27017');

    await enterConnection();

    expect(await getChosenStorageText()).not.toContain('Archive');
  });

  it('keeps the selected notifier after going back from the notifier step', async () => {
    declareWizardRequests();
    mockServer.use(
      mockApi.get('/notifiers', 200, {
        items: [
          {
            id: 'alerts-id',
            workspaceId: 'workspace-id',
            name: 'Alerts',
            lastSendError: null,
            settings: { type: 'DISCORD' as const },
          },
        ],
      }),
    );
    renderWizard();
    await reachNotifierStep('Archive');
    const notifiers = await screen.findByRole('combobox', { name: 'Notifiers' });
    fireEvent.mouseDown(notifiers);
    fireEvent.click(
      await waitFor(() => {
        const notifierOption = document.querySelector('.ant-select-item-option[title="Alerts"]');
        expect(notifierOption).toBeTruthy();
        return notifierOption!;
      }),
    );

    await goBackToStep(1);
    await continueToNotifiers('Nightly');

    const reopenedNotifiers = await screen.findByRole('combobox', { name: 'Notifiers' });
    expect(reopenedNotifiers.closest('.ant-select')!.textContent).toContain('Alerts');
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
    expect((screen.getByPlaceholderText(/password$/) as HTMLInputElement).value).toBe('');
    expect((screen.getByLabelText('Port') as HTMLInputElement).value).toBe('5432');

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

  describe.each(CONNECTION_VARIANTS)(
    'connection step of $variant',
    ({ reachConnectionStep, type, connectionString, previousStepText }) => {
      it('requires the password before a test and sends the entered settings without a database', async () => {
        const sentTests = recordSettingsTests();
        renderWizard();
        await reachConnectionStep();

        await fillConnectionExceptPassword();

        expect(getTestButton().disabled).toBe(true);

        fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
        fireEvent.click(getTestButton());

        await screen.findByRole('button', { name: 'Continue' });
        expect(sentTests).toHaveLength(1);
        expect(sentTests[0].databaseId).toBeUndefined();
        expect(sentTests[0].settings).toMatchObject({
          type,
          host: 'db.example.com',
          username: 'backup',
          password: 'secret',
        });
      });

      it('requires the SSH credentials once a tunnel is enabled and sends them', async () => {
        const sentTests = recordSettingsTests();
        renderWizard();
        await reachConnectionStep();
        await fillConnectionExceptPassword();
        fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });

        openAdvancedWhenClosed();
        fireEvent.click(within(getRowOfLabel('SSH tunnel')).getByRole('checkbox'));

        expect(getTestButton().disabled).toBe(true);

        fireEvent.change(screen.getByPlaceholderText('bastion.example.com'), {
          target: { value: 'bastion.internal' },
        });
        fireEvent.change(within(getRowOfLabel('SSH username')).getByRole('textbox'), {
          target: { value: 'tunnel-user' },
        });
        fireEvent.change(getRowOfLabel('SSH password').querySelector('input')!, {
          target: { value: 'tunnel-secret' },
        });
        fireEvent.click(getTestButton());

        await screen.findByRole('button', { name: 'Continue' });
        expect(sentTests[0].settings.sshTunnel).toMatchObject({
          isEnabled: true,
          host: 'bastion.internal',
          username: 'tunnel-user',
          password: 'tunnel-secret',
        });
      });

      it('fills the fields from an imported connection string and asks for a test', async () => {
        mockServer.use(mockApi.post('/database-tests', 200, { isConnected: true }));
        renderWizard();
        stubClipboard(async () => connectionString);
        await reachConnectionStep();
        await fillConnectionExceptPassword();
        fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
        fireEvent.click(getTestButton());
        await screen.findByRole('button', { name: 'Continue' });

        fireEvent.click(screen.getByText('Parse from clipboard'));

        await waitFor(() =>
          expect(getInputByPlaceholder(/host$/).value).toBe('imported.example.com'),
        );
        expect(getInputByPlaceholder(/username$/).value).toBe('importer');
        expect(getInputByPlaceholder(/password$/).value).toBe('imported-secret');
        expect(getTestButton()).toBeTruthy();
        expect(screen.queryByRole('button', { name: 'Continue' })).toBeNull();
      });

      it('goes back to the previous step', async () => {
        renderWizard();
        await reachConnectionStep();
        await fillConnectionExceptPassword();

        fireEvent.click(screen.getByRole('button', { name: 'Back' }));

        expect(await screen.findByText(previousStepText)).toBeTruthy();
      });

      it('keeps the connection input after going back from the connection step', async () => {
        renderWizard();
        await reachConnectionStep();
        await fillConnectionExceptPassword();
        fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });

        fireEvent.click(screen.getByRole('button', { name: 'Back' }));
        await screen.findByText(previousStepText);
        fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
        expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe(
          'db.example.com',
        );
        expect(getInputByPlaceholder(/password$/).value).toBe('secret');
        expect(getTestButton()).toBeTruthy();
      });
    },
  );

  it('sends the chosen PostgreSQL SSL mode and certificates in the test', async () => {
    const sentTests = recordSettingsTests();
    renderWizard();
    await reachPostgresBackupType('Logical')();
    await fillConnectionExceptPassword();
    fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });

    await chooseSslMode(/^Require/);
    openAdvancedWhenClosed();
    fireEvent.change(screen.getAllByPlaceholderText('-----BEGIN CERTIFICATE-----')[0], {
      target: { value: 'typed-client-cert' },
    });
    fireEvent.change(screen.getByPlaceholderText('-----BEGIN PRIVATE KEY-----'), {
      target: { value: 'typed-client-key' },
    });
    fireEvent.click(getTestButton());

    await screen.findByRole('button', { name: 'Continue' });
    expect(sentTests[0].settings).toMatchObject({
      sslMode: 'require',
      sslClientCert: 'typed-client-cert',
      sslClientKey: 'typed-client-key',
    });
  });

  it.each(['MySQL', 'MariaDB', 'MongoDB'])('sends the HTTPS choice of %s', async (engine) => {
    const sentTests = recordSettingsTests();
    renderWizard();
    enterBaseInfo(engine);
    await fillConnectionExceptPassword();
    fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });

    fireEvent.click(within(getRowOfLabel('Use HTTPS')).getByRole('switch'));
    fireEvent.click(getTestButton());

    await screen.findByRole('button', { name: 'Continue' });
    expect(sentTests[0].settings).toMatchObject({ isHttps: true });
  });

  describe('the defects of the connection step', () => {
    it('does not offer Continue when the password changes during a successful test', async () => {
      const testCall = vi.spyOn(databaseApi, 'testDatabaseSettings');
      const test = mockApi.delayed({
        method: 'post',
        path: '/database-tests',
        status: 200,
        body: { isConnected: true },
      });
      mockServer.use(test.handler);
      renderWizard();
      enterBaseInfo('MySQL');
      await fillConnectionExceptPassword();
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'tested' } });

      fireEvent.click(getTestButton());
      await expectDatabaseTestRequests(1);
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'changed' } });
      await releaseAndSettle(testCall, test.release);

      expect(screen.queryByRole('button', { name: 'Continue' })).toBeNull();
      expect(getTestButton()).toBeTruthy();
    });

    it('keeps the chosen SSL mode and certificates after going back and entering localhost', async () => {
      declareWizardRequests();
      renderWizard();
      await reachPostgresBackupType('Logical')();
      await fillConnectionExceptPassword();
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
      await chooseSslMode(/^Require/);
      openAdvancedWhenClosed();
      fireEvent.change(screen.getAllByPlaceholderText('-----BEGIN CERTIFICATE-----')[0], {
        target: { value: 'typed-client-cert' },
      });
      fireEvent.click(getTestButton());
      fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
      await screen.findByText('Storage');

      fireEvent.click(screen.getByRole('button', { name: 'Back' }));
      fireEvent.change(await screen.findByPlaceholderText(/host$/), {
        target: { value: 'localhost' },
      });

      expect(getSelectedSslMode()).toMatch(/^Require/);
      expect(screen.getByDisplayValue('typed-client-cert')).toBeTruthy();
    });

    it('shows a typed certificate as editable text after going back', async () => {
      declareWizardRequests();
      renderWizard();
      await reachPostgresBackupType('Physical')();
      await fillConnectionExceptPassword();
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
      await chooseSslMode(/^Require/);
      fireEvent.change(screen.getAllByPlaceholderText('-----BEGIN CERTIFICATE-----')[0], {
        target: { value: 'typed-client-cert' },
      });
      fireEvent.click(getTestButton());
      fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
      await screen.findByText('Storage');

      fireEvent.click(screen.getByRole('button', { name: 'Back' }));

      expect(await screen.findByDisplayValue('typed-client-cert')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Replace' })).toBeNull();
    });

    it('keeps the input when another engine is chosen and the original one is chosen again', async () => {
      renderWizard();
      enterBaseInfo('MySQL');
      await fillConnectionExceptPassword();
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
      fireEvent.click(screen.getByRole('button', { name: 'Back' }));

      fireEvent.click(await screen.findByText('MongoDB'));
      fireEvent.click(screen.getByText('MySQL'));
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe(
        'db.example.com',
      );
      expect(getInputByPlaceholder(/password$/).value).toBe('secret');
    });

    it('keeps the physical type and input when the highlighted PostgreSQL engine is clicked', async () => {
      renderWizard();
      await reachPostgresBackupType('Physical')();
      await fillConnectionExceptPassword();
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
      fireEvent.click(screen.getByRole('button', { name: 'Back' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Back' }));

      fireEvent.click(await screen.findByText('PostgreSQL'));
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
      await screen.findByText('Physical');
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe(
        'db.example.com',
      );
      expect(getInputByPlaceholder(/password$/).value).toBe('secret');
      expect(screen.getByText('Backup type')).toBeTruthy();
    });

    it('shows a test that cannot reach the server inline and keeps every value', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      mockServer.use(
        mockApi.post('/database-tests', 200, {
          isConnected: false,
          error: 'dial tcp: connection refused',
        }),
      );
      renderWizard();
      enterBaseInfo('MySQL');
      await fillConnectionExceptPassword();
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });

      fireEvent.click(getTestButton());

      expect(
        await screen.findByText('The database could not be reached: dial tcp: connection refused'),
      ).toBeTruthy();
      expect(screen.getByText(/IP whitelist/)).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();
      expect(getInputByPlaceholder(/password$/).value).toBe('secret');
    });
  });
});
