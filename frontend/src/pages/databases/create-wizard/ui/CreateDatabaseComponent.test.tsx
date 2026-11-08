// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { backupConfigApi } from '@/entities/backup-configs';
import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
  databaseApi,
} from '@/entities/databases';
import type { Storage } from '@/entities/storages';
import { CreateDatabaseComponent } from '@/pages/databases/create-wizard/ui/CreateDatabaseComponent';
import { logicalBackupApi } from '@/pages/databases/logical-backups/api/logicalBackupApi';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { ru } from '@/shared/i18n/locales/ru';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { stubClipboard } from '@/shared/lib/test/stubClipboard';
import { Dialog } from '@/shared/ui/Dialog';

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
  const select = storageLabel.parentElement!.querySelector('[role="combobox"]')!;

  fireEvent.click(select);
  const option = await waitFor(() => {
    const storageOption = screen.queryByRole('option', { name: `${name}` });
    expect(storageOption).toBeTruthy();
    return storageOption!;
  });

  fireEvent.click(option);
};

const getChosenStorageText = async () => {
  const storageField = (await screen.findByText('Storage')).parentElement!;
  await waitFor(() => expect(storageField.querySelector('[role="status"]')).toBeNull());

  return storageField.querySelector('[role="combobox"]')!.textContent;
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
    expectedPort: 5432,
    expectedDatabase: 'shop',
    connectionString: 'postgresql://importer:imported-secret@imported.example.com:5433/imported',
    previousStepText: 'Physical',
  },
  {
    variant: 'PostgreSQL physical',
    reachConnectionStep: reachPostgresBackupType('Physical'),
    type: 'POSTGRES_PHYSICAL',
    expectedPort: 5432,
    expectedDatabase: undefined,
    connectionString: 'postgresql://importer:imported-secret@imported.example.com:5433/imported',
    previousStepText: 'Logical',
  },
  {
    variant: 'MySQL',
    reachConnectionStep: reachEngine('MySQL'),
    type: 'MYSQL',
    expectedPort: 3306,
    expectedDatabase: 'shop',
    connectionString: 'mysql://importer:imported-secret@imported.example.com:3307/imported',
    previousStepText: 'MongoDB',
  },
  {
    variant: 'MariaDB',
    reachConnectionStep: reachEngine('MariaDB'),
    type: 'MARIADB',
    expectedPort: 3306,
    expectedDatabase: 'shop',
    connectionString: 'mariadb://importer:imported-secret@imported.example.com:3307/imported',
    previousStepText: 'MongoDB',
  },
  {
    variant: 'MongoDB',
    reachConnectionStep: reachEngine('MongoDB'),
    type: 'MONGODB',
    expectedPort: 27017,
    expectedDatabase: 'shop',
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
  fireEvent.click(getRowOfLabel('SSL mode').querySelector('[role="combobox"]')!);
  const option = await waitFor(() => {
    const modeOption = Array.from(screen.queryAllByRole('option')).find((element) =>
      modeLabel.test(element.textContent ?? ''),
    );
    expect(modeOption).toBeTruthy();
    return modeOption!;
  });

  fireEvent.click(option);
};

const getSelectedSslMode = () =>
  getRowOfLabel('SSL mode').querySelector('button[role="combobox"]')!.textContent;

const openAdvancedWhenClosed = () => {
  if (!screen.queryByText('SSH tunnel')) fireEvent.click(screen.getByText('Advanced settings'));
};

const fillSshTunnelWithPassword = () => {
  openAdvancedWhenClosed();
  fireEvent.click(within(getRowOfLabel('SSH tunnel')).getByRole('checkbox'));
  fireEvent.change(screen.getByPlaceholderText('bastion.example.com'), {
    target: { value: 'bastion.internal' },
  });
  fireEvent.change(within(getRowOfLabel('SSH username')).getByRole('textbox'), {
    target: { value: 'tunnel-user' },
  });
  fireEvent.change(screen.getByPlaceholderText('Enter SSH password'), {
    target: { value: 'abandoned-tunnel-secret' },
  });
};

const chooseSshKeyAuth = async () => {
  fireEvent.click(getRowOfLabel('SSH auth').querySelector('[role="combobox"]')!);
  const option = await waitFor(() => {
    const authOption = screen.queryByRole('option', { name: 'Private key' });
    expect(authOption).toBeTruthy();
    return authOption!;
  });

  fireEvent.click(option);
};

const enterSshKey = () => {
  fireEvent.change(screen.getByPlaceholderText('-----BEGIN OPENSSH PRIVATE KEY-----'), {
    target: { value: 'typed-private-key' },
  });
  fireEvent.change(screen.getByPlaceholderText('Only for an encrypted key'), {
    target: { value: 'typed-passphrase' },
  });
};

const KEY_TUNNEL_REQUEST = {
  isEnabled: true,
  host: 'bastion.internal',
  port: 22,
  username: 'tunnel-user',
  authType: 'PRIVATE_KEY',
  privateKey: 'typed-private-key',
  privateKeyPassphrase: 'typed-passphrase',
};

const expectDatabaseTestRequests = (count: number) =>
  waitFor(() =>
    expect(
      mockServer.getRequests().filter((request) => request.url.endsWith('/database-tests')),
    ).toHaveLength(count),
  );

describe('CreateDatabaseComponent', () => {
  it('preserves the wizard draft when Escape dismisses notifier search', async () => {
    declareWizardRequests();
    const onClose = vi.fn();
    const { user } = renderScreen(
      <Dialog title="Create database" opened onClose={onClose}>
        <CreateDatabaseComponent
          workspaceId="workspace-id"
          onCreated={() => {}}
          onClose={onClose}
        />
      </Dialog>,
    );
    await reachNotifierStep('Archive');
    const picker = screen.getByRole('combobox', { name: 'Notifiers' });

    await user.click(picker);
    await user.type(screen.getByRole('textbox', { name: 'Notifiers' }), 'operations');
    await user.keyboard('{Escape}');

    expect(picker.getAttribute('aria-expanded')).toBe('false');
    expect(onClose).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(picker);
    expect(screen.getByRole('dialog', { name: 'Create database' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Complete' })).toBeTruthy();

    await user.keyboard('{Escape}');

    expect(onClose).toHaveBeenCalledTimes(1);
  });

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
    fireEvent.click(notifiers);
    fireEvent.click(
      await waitFor(() => {
        const notifierOption = screen.queryByRole('option', { name: 'Alerts' });
        expect(notifierOption).toBeTruthy();
        return notifierOption!;
      }),
    );

    await goBackToStep(1);
    await continueToNotifiers('Nightly');

    const reopenedNotifiers = await screen.findByRole('combobox', { name: 'Notifiers' });
    expect(reopenedNotifiers.closest('div')!.textContent).toContain('Alerts');
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

      it('requires the key after key authentication is chosen and sends it without the typed password', async () => {
        const sentTests = recordSettingsTests();
        renderWizard();
        await reachConnectionStep();
        await fillConnectionExceptPassword();
        fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
        fillSshTunnelWithPassword();

        await chooseSshKeyAuth();

        expect(getTestButton().disabled).toBe(true);

        enterSshKey();
        fireEvent.click(getTestButton());

        await screen.findByRole('button', { name: 'Continue' });
        expect(sentTests[0].settings.sshTunnel).toEqual(KEY_TUNNEL_REQUEST);
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

describe('the backup configuration step of the wizard', () => {
  const reachLogicalBackupStep = async () => {
    enterBaseInfo('MySQL');
    await enterConnection();
    await screen.findByText('Storage');
  };

  const reachPhysicalBackupStep = async () => {
    enterBaseInfo('PostgreSQL');
    await choosePostgresBackupType('Physical');
    await enterConnection();
    await screen.findByText('Storage');
  };

  const renderRerenderableWizard = () => {
    const wizard = () => (
      <CreateDatabaseComponent workspaceId="workspace-id" onCreated={() => {}} onClose={() => {}} />
    );
    const view = renderScreen(wizard());

    return { ...view, rerenderWizard: () => view.rerender(wizard()) };
  };

  const getFirstCheckbox = () => screen.getAllByRole('checkbox')[0] as HTMLInputElement;

  describe.each([
    { kind: 'logical', reachBackupStep: reachLogicalBackupStep },
    { kind: 'physical', reachBackupStep: reachPhysicalBackupStep },
  ])('of a $kind database', ({ reachBackupStep }) => {
    it('keeps the input without a spinner when the wizard rerenders', async () => {
      declareWizardRequests();
      const { rerenderWizard } = renderRerenderableWizard();
      await reachBackupStep();
      const isInitiallyChecked = getFirstCheckbox().checked;
      fireEvent.click(getFirstCheckbox());

      rerenderWizard();

      expect(document.querySelector('[role="status"]')).toBeNull();
      expect(getFirstCheckbox().checked).toBe(!isInitiallyChecked);
    });

    it('keeps storage creation open when the wizard rerenders', async () => {
      declareWizardRequests();
      const { rerenderWizard, user } = renderRerenderableWizard();
      await reachBackupStep();
      await user.click(await screen.findByRole('combobox', { name: 'Storage' }));
      await user.click(await screen.findByRole('button', { name: 'Create new storage' }));
      const dialog = within(await screen.findByRole('dialog', { name: 'Add storage' }));
      await user.type(dialog.getByRole('textbox', { name: 'Name' }), 'Nested archive');

      rerenderWizard();

      expect(
        (
          within(screen.getByRole('dialog', { name: 'Add storage' })).getByRole('textbox', {
            name: 'Name',
          }) as HTMLInputElement
        ).value,
      ).toBe('Nested archive');
    });

    it('offers Continue without a change after going back from the notifier step', async () => {
      declareWizardRequests();
      renderWizard();
      await reachBackupStep();
      await continueToNotifiers('Archive');

      fireEvent.click(screen.getByRole('button', { name: 'Back' }));
      await getChosenStorageText();

      expect(screen.getByRole('button', { name: 'Continue' })).toHaveProperty('disabled', false);
    });
  });

  it('keeps logical input after going back to the connection step and returning', async () => {
    declareWizardRequests();
    renderWizard();
    await reachLogicalBackupStep();
    const isInitiallyChecked = getFirstCheckbox().checked;
    fireEvent.click(getFirstCheckbox());

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    await enterConnection();
    await screen.findByText('Storage');

    expect(getFirstCheckbox().checked).toBe(!isInitiallyChecked);
  });
});

describe('completing the wizard', () => {
  const REFUSAL_REASON = 'You do not have permission to do this.';
  const CREATED_DATABASE_MESSAGE = /The database was created/;

  const buildCreatedDatabase = (type: DatabaseType, notifierIds: string[] = []): Database => ({
    id: 'created-id',
    workspaceId: 'workspace-id',
    name: 'Shop',
    settings: {
      ...createDefaultDatabaseSettings(type),
      host: 'db.example.com',
      username: 'backup',
      database: 'shop',
    } as Database['settings'],
    notifierIds,
  });

  const refuseRequest = () =>
    HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });

  const declareCompletionServer = (type: DatabaseType) => {
    const completionServer = {
      isCreationRefused: false,
      isBackupConfigRefused: false,
      isBackupStartRefused: false,
      sentBackupConfigs: [] as ApiComponents['schemas']['BackupConfigInput'][],
      sentCreations: [] as ApiComponents['schemas']['DatabaseCreate'][],
      sentNotifierIds: [] as string[][],
    };
    const startBackup = () =>
      completionServer.isBackupStartRefused
        ? refuseRequest()
        : HttpResponse.json({ id: 'backup-id' }, { status: 202 });

    declareWizardRequests();
    mockServer.use(
      http.post(`*${API_BASE_PATH}/databases`, async ({ request }) => {
        if (completionServer.isCreationRefused) return refuseRequest();

        const creation = (await request.json()) as ApiComponents['schemas']['DatabaseCreate'];
        completionServer.sentCreations.push(creation);

        return HttpResponse.json(buildCreatedDatabase(type, creation.notifierIds), {
          status: 201,
        });
      }),
      http.put(`*${API_BASE_PATH}/databases/:databaseId`, async ({ request }) => {
        const { notifierIds } = (await request.json()) as { notifierIds: string[] };
        completionServer.sentNotifierIds.push(notifierIds);

        return HttpResponse.json(buildCreatedDatabase(type, notifierIds));
      }),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
        completionServer.sentBackupConfigs.push(
          (await request.json()) as ApiComponents['schemas']['BackupConfigInput'],
        );

        return completionServer.isBackupConfigRefused ? refuseRequest() : HttpResponse.json({});
      }),
      http.post(`*${API_BASE_PATH}/logical-backups`, startBackup),
      http.post(`*${API_BASE_PATH}/physical-backups`, startBackup),
    );

    return completionServer;
  };

  const countRequests = ({ method, path }: { method: string; path: string }) =>
    mockServer
      .getRequests()
      .filter(
        (request) => request.method === method && new URL(request.url).pathname.endsWith(path),
      ).length;

  const renderCompletableWizard = () => {
    const onCreated = vi.fn();
    const onClose = vi.fn();
    const view = renderScreen(
      <CreateDatabaseComponent
        workspaceId="workspace-id"
        onCreated={onCreated}
        onClose={onClose}
      />,
    );

    return { ...view, onCreated, onClose };
  };

  const clickComplete = async () =>
    fireEvent.click(await screen.findByRole('button', { name: 'Complete' }));

  const reachPhysicalNotifierStep = async () => {
    enterBaseInfo('PostgreSQL');
    await choosePostgresBackupType('Physical');
    await enterConnection();
    await continueToNotifiers('Archive');
  };

  const buildNotifier = (name: string) => ({
    id: `${name.toLowerCase()}-id`,
    workspaceId: 'workspace-id',
    name,
    lastSendError: null,
    settings: { type: 'DISCORD' as const },
  });

  const declareNotifiers = () =>
    mockServer.use(
      mockApi.get('/notifiers', 200, { items: [buildNotifier('Alerts'), buildNotifier('Pager')] }),
    );

  const toggleNotifier = async (name: string) => {
    fireEvent.click(await screen.findByRole('combobox', { name: 'Notifiers' }));
    fireEvent.click(
      await waitFor(() => {
        const notifierOption = screen.queryByRole('option', { name: `${name}` });
        expect(notifierOption).toBeTruthy();
        return notifierOption!;
      }),
    );
  };

  const findCreatedDatabaseMessage = () => screen.findByText(CREATED_DATABASE_MESSAGE);

  it('closes after physical configuration acknowledgement without queuing a manual backup', async () => {
    declareCompletionServer(DatabaseType.POSTGRES_PHYSICAL);
    const backupConfigSave = mockApi.pending('put', '/backup-configs/{databaseId}');
    mockServer.use(backupConfigSave.handler);
    const { onClose } = renderCompletableWizard();
    await reachPhysicalNotifierStep();

    await clickComplete();
    await waitFor(() =>
      expect(countRequests({ method: 'PUT', path: '/backup-configs/created-id' })).toBe(1),
    );
    expect(onClose).not.toHaveBeenCalled();

    backupConfigSave.respond(HttpResponse.json({}));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(countRequests({ method: 'POST', path: '/physical-backups' })).toBe(0);
  });

  describe.each([
    {
      kind: 'logical',
      type: DatabaseType.MYSQL,
      backupStartPath: '/logical-backups',
      reachCompletionStep: () => reachNotifierStep('Archive'),
    },
    {
      kind: 'physical',
      type: DatabaseType.POSTGRES_PHYSICAL,
      backupStartPath: '/physical-backups',
      reachCompletionStep: reachPhysicalNotifierStep,
    },
  ])('of a $kind database', ({ type, backupStartPath, reachCompletionStep }) => {
    const hasManualBackupStart = type !== DatabaseType.POSTGRES_PHYSICAL;

    it('sends each request once and closes after the last one when everything succeeds', async () => {
      const completionServer = declareCompletionServer(type);
      const lastRequest = hasManualBackupStart
        ? mockApi.pending('post', '/logical-backups')
        : mockApi.pending('put', '/backup-configs/{databaseId}');
      mockServer.use(lastRequest.handler);
      const { onCreated, onClose } = renderCompletableWizard();
      await reachCompletionStep();

      await clickComplete();
      await waitFor(() =>
        expect(
          countRequests({
            method: hasManualBackupStart ? 'POST' : 'PUT',
            path: hasManualBackupStart ? backupStartPath : '/backup-configs/created-id',
          }),
        ).toBe(1),
      );

      expect(onClose).not.toHaveBeenCalled();

      lastRequest.respond(
        hasManualBackupStart
          ? HttpResponse.json({ id: 'backup-id' }, { status: 202 })
          : HttpResponse.json({}),
      );

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(onCreated).toHaveBeenCalledTimes(1);
      expect(countRequests({ method: 'POST', path: '/databases' })).toBe(1);
      expect(countRequests({ method: 'PUT', path: '/backup-configs/created-id' })).toBe(1);
      expect(completionServer.sentNotifierIds).toHaveLength(0);
    });

    it('creates once and sends the configuration again when Complete follows a refused configuration', async () => {
      const completionServer = declareCompletionServer(type);
      completionServer.isBackupConfigRefused = true;
      const { onClose } = renderCompletableWizard();
      await reachCompletionStep();

      await clickComplete();
      await findCreatedDatabaseMessage();
      completionServer.isBackupConfigRefused = false;
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(countRequests({ method: 'POST', path: '/databases' })).toBe(1);
      expect(completionServer.sentBackupConfigs).toHaveLength(2);
      expect(countRequests({ method: 'POST', path: backupStartPath })).toBe(
        hasManualBackupStart ? 1 : 0,
      );
    });

    if (hasManualBackupStart) {
      it('sends only the backup start again when Complete follows a failed backup start', async () => {
        const completionServer = declareCompletionServer(type);
        completionServer.isBackupStartRefused = true;
        const { onClose } = renderCompletableWizard();
        await reachCompletionStep();

        await clickComplete();
        await findCreatedDatabaseMessage();
        completionServer.isBackupStartRefused = false;
        await clickComplete();

        await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
        expect(countRequests({ method: 'POST', path: '/databases' })).toBe(1);
        expect(completionServer.sentBackupConfigs).toHaveLength(1);
        expect(countRequests({ method: 'POST', path: backupStartPath })).toBe(2);
      });
    }

    it('offers no way to the connection step and saves a changed configuration for the created database after a refused configuration', async () => {
      const completionServer = declareCompletionServer(type);
      completionServer.isBackupConfigRefused = true;
      const { onClose } = renderCompletableWizard();
      await reachCompletionStep();

      await clickComplete();
      await findCreatedDatabaseMessage();
      await goBackToStep(1);
      await getChosenStorageText();

      expect(screen.queryByRole('button', { name: 'Back' })).toBeNull();

      completionServer.isBackupConfigRefused = false;
      await continueToNotifiers('Nightly');
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(countRequests({ method: 'POST', path: '/databases' })).toBe(1);
      expect(countRequests({ method: 'PUT', path: '/backup-configs/created-id' })).toBe(2);
      expect(completionServer.sentBackupConfigs[1].storageId).toBe(nightlyStorage.id);
    });

    if (hasManualBackupStart) {
      it('offers no way to the configuration step after a failed backup start', async () => {
        const completionServer = declareCompletionServer(type);
        completionServer.isBackupStartRefused = true;
        renderCompletableWizard();
        await reachCompletionStep();

        await clickComplete();
        await findCreatedDatabaseMessage();

        expect(await screen.findByRole('button', { name: 'Complete' })).toBeTruthy();
        expect(screen.queryByRole('button', { name: 'Back' })).toBeNull();
      });
    }

    it('sends changed notifiers to the created database through an update with no second create', async () => {
      const completionServer = declareCompletionServer(type);
      declareNotifiers();
      completionServer.isBackupConfigRefused = true;
      const { onClose } = renderCompletableWizard();
      await reachCompletionStep();

      await clickComplete();
      await findCreatedDatabaseMessage();
      await toggleNotifier('Alerts');
      completionServer.isBackupConfigRefused = false;
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(countRequests({ method: 'POST', path: '/databases' })).toBe(1);
      expect(completionServer.sentNotifierIds).toEqual([['alerts-id']]);
      expect(completionServer.sentBackupConfigs).toHaveLength(2);
    });
  });

  describe.each(CONNECTION_VARIANTS)(
    'of $variant',
    ({ reachConnectionStep, type, expectedPort, expectedDatabase }) => {
      it('sends the name, the entered settings and the chosen notifiers in the create request', async () => {
        const completionServer = declareCompletionServer(type);
        declareNotifiers();
        const { onClose } = renderCompletableWizard();
        await reachConnectionStep();
        await enterConnection();
        await continueToNotifiers('Archive');
        await toggleNotifier('Alerts');

        await clickComplete();

        await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
        expect(completionServer.sentCreations).toHaveLength(1);

        const [sentCreation] = completionServer.sentCreations;

        expect(sentCreation.workspaceId).toBe('workspace-id');
        expect(sentCreation.name).toBe('Shop');
        expect(sentCreation.notifierIds).toEqual(['alerts-id']);
        expect(sentCreation.settings).toMatchObject({
          type,
          host: 'db.example.com',
          port: expectedPort,
          username: 'backup',
          password: 'secret',
          sshTunnel: { isEnabled: false },
        });
        expect((sentCreation.settings as { database?: string }).database).toBe(expectedDatabase);
      });
    },
  );

  it('sends the key and passphrase in the create request when the SSH tunnel uses key authentication', async () => {
    const completionServer = declareCompletionServer(DatabaseType.MYSQL);
    const { onClose } = renderCompletableWizard();
    enterBaseInfo('MySQL');
    await fillConnectionExceptPassword();
    fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'secret' } });
    fillSshTunnelWithPassword();
    await chooseSshKeyAuth();
    enterSshKey();
    fireEvent.click(getTestButton());
    fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
    await continueToNotifiers('Archive');

    await clickComplete();

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(completionServer.sentCreations[0].settings.sshTunnel).toEqual(KEY_TUNNEL_REQUEST);
  });

  describe('when a limited user is suggested', () => {
    const SUGGESTION_TITLE = 'Create a read-only user for Databasus?';
    const CREATE_READ_ONLY_USER = 'Yes, create read-only user';

    const reachSuggestion = async () => {
      mockServer.use(
        mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: [] }),
      );
      enterBaseInfo('MySQL');
      await enterConnection();
      await screen.findByText(SUGGESTION_TITLE);
    };

    const acceptSkipRisks = async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Skip' }));
      fireEvent.click(
        within(await screen.findByRole('dialog')).getByRole('button', {
          name: 'Yes, I accept risks',
        }),
      );
    };

    it('creates the database with the generated user when the suggestion is accepted', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      const sentLimitedUsers: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
      mockServer.use(
        http.post(`*${API_BASE_PATH}/limited-users`, async ({ request }) => {
          sentLimitedUsers.push(
            (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
          );

          return HttpResponse.json(
            {
              username: 'databasus_reader',
              password: 'generated-secret',
              isForcedWalRotationAvailable: false,
            },
            { status: 201 },
          );
        }),
      );
      const { onClose } = renderCompletableWizard();
      await reachSuggestion();

      fireEvent.click(screen.getByRole('button', { name: CREATE_READ_ONLY_USER }));
      await continueToNotifiers('Archive');
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(sentLimitedUsers).toHaveLength(1);
      expect(sentLimitedUsers[0].databaseId).toBeUndefined();
      expect(sentLimitedUsers[0].settings).toMatchObject({
        type: 'MYSQL',
        host: 'db.example.com',
        username: 'backup',
        password: 'secret',
      });
      expect(completionServer.sentCreations[0].settings).toMatchObject({
        username: 'databasus_reader',
        password: 'generated-secret',
      });
    });

    it('creates the database with the entered user and creates no limited user when the suggestion is skipped', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      const { onClose } = renderCompletableWizard();
      await reachSuggestion();

      await acceptSkipRisks();
      await continueToNotifiers('Archive');
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(countRequests({ method: 'POST', path: '/limited-users' })).toBe(0);
      expect(completionServer.sentCreations[0].settings).toMatchObject({
        username: 'backup',
        password: 'secret',
      });
    });

    it('stays on the suggestion with the reason and still allows a skip when the create request is refused', async () => {
      declareCompletionServer(DatabaseType.MYSQL);
      mockServer.use(mockApi.post('/limited-users', 403, buildProblem('permission_denied', 403)));
      renderCompletableWizard();
      await reachSuggestion();

      fireEvent.click(screen.getByRole('button', { name: CREATE_READ_ONLY_USER }));

      expect(await screen.findByText(REFUSAL_REASON)).toBeTruthy();
      expect(screen.getByText(SUGGESTION_TITLE)).toBeTruthy();
      expect(screen.queryByText('Storage')).toBeNull();

      await acceptSkipRisks();

      expect(await screen.findByText('Storage')).toBeTruthy();
    });
  });

  describe('when notifiers are chosen', () => {
    it('sends no update on a retry when the selection made before the first Complete is unchanged', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      declareNotifiers();
      completionServer.isBackupConfigRefused = true;
      const { onClose } = renderCompletableWizard();
      await reachNotifierStep('Archive');
      await toggleNotifier('Alerts');

      await clickComplete();
      await findCreatedDatabaseMessage();
      completionServer.isBackupConfigRefused = false;
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(completionServer.sentNotifierIds).toHaveLength(0);
    });

    it('sends the new selection when one notifier is swapped for another before a retry', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      declareNotifiers();
      completionServer.isBackupStartRefused = true;
      const { onClose } = renderCompletableWizard();
      await reachNotifierStep('Archive');
      await toggleNotifier('Alerts');

      await clickComplete();
      await findCreatedDatabaseMessage();
      await toggleNotifier('Alerts');
      await toggleNotifier('Pager');
      completionServer.isBackupStartRefused = false;
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(completionServer.sentNotifierIds).toEqual([['pager-id']]);
    });

    it('does not repeat an update that succeeded before a refused configuration', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      declareNotifiers();
      completionServer.isBackupConfigRefused = true;
      const { onClose } = renderCompletableWizard();
      await reachNotifierStep('Archive');

      await clickComplete();
      await findCreatedDatabaseMessage();
      await toggleNotifier('Alerts');
      await clickComplete();
      await waitFor(() => expect(completionServer.sentBackupConfigs).toHaveLength(2));
      await findCreatedDatabaseMessage();
      completionServer.isBackupConfigRefused = false;
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(completionServer.sentNotifierIds).toEqual([['alerts-id']]);
    });
  });

  describe('when the wizard closes during its requests', () => {
    it('sends no later request and reports nothing when the create request answers after the wizard closed', async () => {
      declareCompletionServer(DatabaseType.MYSQL);
      const creation = mockApi.pending('post', '/databases');
      mockServer.use(creation.handler);
      const createCall = vi.spyOn(databaseApi, 'createDatabase');
      const backupConfigCall = vi.spyOn(backupConfigApi, 'updateBackupConfig');
      const { onCreated, onClose, unmount } = renderCompletableWizard();
      await reachNotifierStep('Archive');

      await clickComplete();
      await waitFor(() => expect(createCall).toHaveBeenCalledTimes(1));
      unmount();
      await releaseAndSettle(createCall, () =>
        creation.respond(
          HttpResponse.json(buildCreatedDatabase(DatabaseType.MYSQL), { status: 201 }),
        ),
      );

      expect(backupConfigCall).not.toHaveBeenCalled();
      expect(onCreated).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();
    });

    it('sends no later request and calls nothing when the save request answers after the wizard closed', async () => {
      declareCompletionServer(DatabaseType.MYSQL);
      const backupConfigSave = mockApi.pending('put', '/backup-configs/{databaseId}');
      mockServer.use(backupConfigSave.handler);
      const backupConfigCall = vi.spyOn(backupConfigApi, 'updateBackupConfig');
      const backupStartCall = vi.spyOn(logicalBackupApi, 'startLogicalBackup');
      const { onClose, unmount } = renderCompletableWizard();
      await reachNotifierStep('Archive');

      await clickComplete();
      await waitFor(() => expect(backupConfigCall).toHaveBeenCalledTimes(1));
      unmount();
      await releaseAndSettle(backupConfigCall, () =>
        backupConfigSave.respond(HttpResponse.json({})),
      );

      expect(backupStartCall).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();
    });

    it.each([
      {
        answer: 'accepted',
        buildAnswer: () => HttpResponse.json({}),
      },
      {
        answer: 'refused',
        buildAnswer: refuseRequest,
      },
    ])(
      'leaves a newly opened wizard on its first step with no feedback when the save request is $answer late',
      async ({ buildAnswer }) => {
        declareCompletionServer(DatabaseType.MYSQL);
        const backupConfigSave = mockApi.pending('put', '/backup-configs/{databaseId}');
        mockServer.use(backupConfigSave.handler);
        const backupConfigCall = vi.spyOn(backupConfigApi, 'updateBackupConfig');
        const backupStartCall = vi.spyOn(logicalBackupApi, 'startLogicalBackup');
        const onClose = vi.fn();
        const { rerender } = renderScreen(
          <CreateDatabaseComponent
            key="first-wizard"
            workspaceId="workspace-id"
            onCreated={() => {}}
            onClose={onClose}
          />,
        );
        await reachNotifierStep('Archive');

        await clickComplete();
        await waitFor(() => expect(backupConfigCall).toHaveBeenCalledTimes(1));
        rerender(
          <CreateDatabaseComponent
            key="second-wizard"
            workspaceId="workspace-id"
            onCreated={() => {}}
            onClose={onClose}
          />,
        );
        await releaseAndSettle(backupConfigCall, () => backupConfigSave.respond(buildAnswer()));

        expect(backupStartCall).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.queryByText(REFUSAL_REASON)).toBeNull();
        expect(screen.queryByText(CREATED_DATABASE_MESSAGE)).toBeNull();
        expect((screen.getByPlaceholderText('My favourite DB') as HTMLInputElement).value).toBe('');
      },
    );

    it('shows no error when the wizard closes before the create request is refused', async () => {
      declareCompletionServer(DatabaseType.MYSQL);
      const creation = mockApi.pending('post', '/databases');
      mockServer.use(creation.handler);
      const createCall = vi.spyOn(databaseApi, 'createDatabase');
      const { rerender } = renderCompletableWizard();
      await reachNotifierStep('Archive');

      await clickComplete();
      await waitFor(() => expect(createCall).toHaveBeenCalledTimes(1));
      rerender(<div />);
      await releaseAndSettle(createCall, () => creation.respond(refuseRequest()));

      expect(screen.queryByText(REFUSAL_REASON)).toBeNull();
    });
  });

  describe('when a request fails', () => {
    it('shows the reason on the notifier step and keeps the message after going to the configuration step and returning', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      declareNotifiers();
      completionServer.isBackupConfigRefused = true;
      renderCompletableWizard();
      await reachNotifierStep('Archive');
      await toggleNotifier('Alerts');

      await clickComplete();

      const message = await findCreatedDatabaseMessage();
      expect(message.textContent).toContain(REFUSAL_REASON);
      const notifiers = await screen.findByRole('combobox', { name: 'Notifiers' });
      await waitFor(() => expect(notifiers.closest('div')!.textContent).toContain('Alerts'));

      await goBackToStep(1);

      expect(await getChosenStorageText()).toContain('Archive');
      expect(screen.getByText(CREATED_DATABASE_MESSAGE)).toBeTruthy();

      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      expect(await findCreatedDatabaseMessage()).toBeTruthy();
    });

    it('says that a retry is finishing the setup rather than creating a database', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      completionServer.isBackupStartRefused = true;
      renderCompletableWizard();
      await reachNotifierStep('Archive');

      await clickComplete();
      await findCreatedDatabaseMessage();
      mockServer.use(mockApi.pending('post', '/logical-backups').handler);
      await clickComplete();

      expect(await screen.findByText('Finishing database setup...')).toBeTruthy();
      expect(screen.queryByText('Creating database...')).toBeNull();
      expect(screen.queryByText(CREATED_DATABASE_MESSAGE)).toBeNull();
    });

    it('translates the kept message when the language changes', async () => {
      const LanguageSwitch = () => {
        const { setLocale } = useLocale();

        return <button onClick={() => setLocale('ru')}>Switch language</button>;
      };
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      completionServer.isBackupConfigRefused = true;
      renderScreen(
        <>
          <LanguageSwitch />
          <CreateDatabaseComponent
            workspaceId="workspace-id"
            onCreated={() => {}}
            onClose={() => {}}
          />
        </>,
      );
      await reachNotifierStep('Archive');

      await clickComplete();
      await findCreatedDatabaseMessage();
      fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

      const [russianMessageStart] = ru.databases.create.createdButNotFinished.split('{{reason}}');
      expect(
        await screen.findByText((content) => content.startsWith(russianMessageStart)),
      ).toBeTruthy();
      expect(screen.queryByText(CREATED_DATABASE_MESSAGE)).toBeNull();
    });

    it('shows the reason, mentions no created database and leaves every earlier step reachable when the creation is refused', async () => {
      const completionServer = declareCompletionServer(DatabaseType.MYSQL);
      completionServer.isCreationRefused = true;
      renderCompletableWizard();
      await reachNotifierStep('Archive');

      await clickComplete();

      expect(await screen.findByText(REFUSAL_REASON)).toBeTruthy();
      expect(screen.queryByText(CREATED_DATABASE_MESSAGE)).toBeNull();
      expect(completionServer.sentBackupConfigs).toHaveLength(0);

      await goBackToStep(1);
      expect(await getChosenStorageText()).toContain('Archive');

      await goBackToStep(1);
      expect(await screen.findByPlaceholderText(/host$/)).toBeTruthy();

      await goBackToStep(1);
      expect(await screen.findByPlaceholderText('My favourite DB')).toBeTruthy();
    });
  });

  describe('when the physical backup type changes', () => {
    const chooseBackupType = async (backupType: RegExp) =>
      fireEvent.click(await screen.findByRole('radio', { name: backupType }));

    const FULL_ONLY = /^Full backups only/;
    const FULL_AND_INCREMENTAL = /^Full \+ incremental(?! \+)/;

    const reachPhysicalBackupStepWithIncrementalBackups = async () => {
      enterBaseInfo('PostgreSQL');
      await choosePostgresBackupType('Physical');
      await chooseBackupType(FULL_AND_INCREMENTAL);
      await enterConnection();
    };

    const reachNotifierStepWithIncrementalBackups = async () => {
      await reachPhysicalBackupStepWithIncrementalBackups();
      await continueToNotifiers('Archive');
    };

    it('starts the configuration over with the defaults of the new type and sends them', async () => {
      const completionServer = declareCompletionServer(DatabaseType.POSTGRES_PHYSICAL);
      const { onClose } = renderCompletableWizard();
      await reachNotifierStepWithIncrementalBackups();

      await goBackToStep(2);
      await chooseBackupType(FULL_ONLY);
      await enterConnection();

      expect(await getChosenStorageText()).not.toContain('Archive');
      expect(screen.queryByText('Incremental backup cadence')).toBeNull();

      await continueToNotifiers('Nightly');
      await clickComplete();

      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(completionServer.sentBackupConfigs[0].settings).toMatchObject({
        retention: { policy: 'FULL_BACKUPS', chainsCount: 0 },
        incrementalBackupSchedule: null,
        walLagThresholdBytes: 0,
      });
    });

    it('shows the entered values after Back from the step and returning', async () => {
      declareWizardRequests();
      renderWizard();
      await reachPhysicalBackupStepWithIncrementalBackups();
      await chooseStorage('Archive');

      fireEvent.click(screen.getByRole('button', { name: 'Back' }));
      await enterConnection();

      expect(await getChosenStorageText()).toContain('Archive');
    });

    it('shows the defaults of the new type after Back without continuing and another type', async () => {
      declareWizardRequests();
      renderWizard();
      await reachPhysicalBackupStepWithIncrementalBackups();
      await chooseStorage('Archive');

      await goBackToStep(1);
      await chooseBackupType(FULL_ONLY);
      await enterConnection();

      expect(await getChosenStorageText()).not.toContain('Archive');
      expect(screen.queryByText('Incremental backup cadence')).toBeNull();
    });

    it('keeps the entered configuration when the original type is chosen again before continuing', async () => {
      declareCompletionServer(DatabaseType.POSTGRES_PHYSICAL);
      renderCompletableWizard();
      await reachNotifierStepWithIncrementalBackups();

      await goBackToStep(2);
      await chooseBackupType(FULL_ONLY);
      await chooseBackupType(FULL_AND_INCREMENTAL);
      await enterConnection();

      expect(await getChosenStorageText()).toContain('Archive');
    });
  });
});
