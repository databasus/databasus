import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, type RequestHandler, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  type BackupConfig,
  type BackupConfigInput,
  backupConfigApi,
} from '@/entities/backup-configs';
import {
  type Database,
  type DatabaseSettings,
  DatabaseType,
  PostgresSslMode,
  SshTunnelAuthType,
  createDefaultDatabaseSettings,
  databaseApi,
} from '@/entities/databases';
// @vitest-environment happy-dom
import { DatabaseConfigComponent } from '@/pages/databases/ui/DatabaseConfigComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { useLocale } from '@/shared/i18n';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { stubClipboard } from '@/shared/lib/test/stubClipboard';

const mockServer = setUpMockApi();

const savedDatabase: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: {
    ...createDefaultDatabaseSettings(DatabaseType.MYSQL),
    host: 'db.example.com',
    username: 'backup',
    database: 'shop',
  },
  notifierIds: [],
};

const LIMITED_USER_HINT_TITLE = 'Create a read-only user for Databasus?';

const renderDatabaseConfig = (isCanManageDBs: boolean, handlers: RequestHandler[] = []) => {
  mockServer.use(
    ...handlers,
    mockApi.get('/notifiers', 200, { items: [] }),
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/storages').handler,
    mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
  );

  return renderScreen(
    <DatabaseConfigComponent
      database={savedDatabase}
      onDatabaseChanged={() => {}}
      onDatabaseDeleted={() => {}}
      onDatabaseCopied={() => {}}
      onDatabaseTransferred={() => {}}
      isCanManageDBs={isCanManageDBs}
    />,
  );
};

const EDIT_ICON_SELECTOR = 'img[src="/icons/pen-gray.svg"]';

const recordLimitedUserChecks = () => {
  const limitedUserCheckCalls: unknown[] = [];

  const handlers = [
    mockApi.post('/database-tests', 200, { isConnected: true }),
    mockApi.put('/databases/{databaseId}', 200, savedDatabase),
    http.post(`*${API_BASE_PATH}/limited-user-checks`, async ({ request }) => {
      limitedUserCheckCalls.push(await request.json());
      return HttpResponse.json({ isAdvisable: true, privileges: ['ALL PRIVILEGES'] });
    }),
  ];

  return { handlers, limitedUserCheckCalls };
};

const buildSavedDatabase = (type: DatabaseType, change: Partial<DatabaseSettings> = {}): Database =>
  ({
    id: 'database-id',
    workspaceId: 'workspace-id',
    name: 'Shop',
    settings: {
      ...createDefaultDatabaseSettings(type),
      password: undefined,
      host: 'db.example.com',
      username: 'backup',
      ...(type === DatabaseType.POSTGRES_PHYSICAL ? {} : { database: 'shop' }),
      ...change,
    },
    notifierIds: [],
  }) as Database;

const SAVED_VARIANTS = [
  {
    variant: 'PostgreSQL logical',
    type: DatabaseType.POSTGRES_LOGICAL,
    connectionString: 'postgresql://importer:imported-secret@imported.example.com:5433/imported',
  },
  {
    variant: 'PostgreSQL physical',
    type: DatabaseType.POSTGRES_PHYSICAL,
    connectionString: 'postgresql://importer:imported-secret@imported.example.com:5433/imported',
  },
  {
    variant: 'MySQL',
    type: DatabaseType.MYSQL,
    connectionString: 'mysql://importer:imported-secret@imported.example.com:3307/imported',
  },
  {
    variant: 'MariaDB',
    type: DatabaseType.MARIADB,
    connectionString: 'mariadb://importer:imported-secret@imported.example.com:3307/imported',
  },
  {
    variant: 'MongoDB',
    type: DatabaseType.MONGODB,
    connectionString: 'mongodb://importer:imported-secret@imported.example.com:27018/imported',
  },
];

const openConnectionSection = async () => {
  const databaseSectionTitle = screen.getByText('Database settings').parentElement!;

  fireEvent.click(databaseSectionTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);

  return (await screen.findByPlaceholderText(/host$/)) as HTMLInputElement;
};

const renderSavedDatabaseConfig = (
  database: Database,
  handlers: RequestHandler[] = [],
  onDatabaseChanged: (database: Database) => void = () => {},
) => {
  mockServer.use(
    ...handlers,
    mockApi.get('/notifiers', 200, { items: [] }),
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/storages').handler,
    mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    mockApi.pending('get', '/verification-configs/{databaseId}').handler,
  );

  return renderScreen(
    <DatabaseConfigComponent
      database={database}
      onDatabaseChanged={onDatabaseChanged}
      onDatabaseDeleted={() => {}}
      onDatabaseCopied={() => {}}
      onDatabaseTransferred={() => {}}
      isCanManageDBs
    />,
  );
};

const getInputByPlaceholder = (placeholder: RegExp) =>
  screen.getByPlaceholderText(placeholder) as HTMLInputElement;

const clickTest = () => fireEvent.click(screen.getByRole('button', { name: /Test connection$/ }));

const clickSave = async () => fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

const recordUpdates = (savedAnswer: Database) => {
  const sentUpdates: ApiComponents['schemas']['DatabaseUpdate'][] = [];
  const handler = http.put(`*${API_BASE_PATH}/databases/:databaseId`, async ({ request }) => {
    sentUpdates.push((await request.json()) as ApiComponents['schemas']['DatabaseUpdate']);

    return HttpResponse.json(savedAnswer);
  });

  return { handler, sentUpdates };
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

const countRequests = (pathSuffix: string, method: string) =>
  mockServer
    .getRequests()
    .filter((request) => request.method === method && request.url.endsWith(pathSuffix)).length;

const recordSettingsTests = () => {
  const sentTests: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
  const handler = http.post(`*${API_BASE_PATH}/database-tests`, async ({ request }) => {
    sentTests.push((await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget']);

    return HttpResponse.json({ isConnected: true });
  });

  return { handler, sentTests };
};

const passwordTunnel = {
  isEnabled: true,
  host: 'bastion.example.com',
  port: 22,
  username: 'tunnel-user',
  authType: SshTunnelAuthType.PASSWORD,
};

const renderConnectionSaveFlow = (database: Database) => {
  const { handler: updateHandler, sentUpdates } = recordUpdates(database);
  const { handler: testHandler, sentTests } = recordSettingsTests();

  renderSavedDatabaseConfig(database, [
    updateHandler,
    testHandler,
    mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
  ]);

  return { sentUpdates, sentTests };
};

const testAndSaveConnection = async (sentUpdates: unknown[]) => {
  clickTest();
  await clickSave();
  await waitFor(() => expect(sentUpdates).toHaveLength(1));
};

const chooseSshAuth = async (optionTitle: string) => {
  fireEvent.mouseDown(getRowOfLabel('SSH auth').querySelector('.ant-select-selector')!);
  const option = await waitFor(() => {
    const authOption = document.querySelector(`.ant-select-item-option[title="${optionTitle}"]`);
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

const savedWithCertificate = buildSavedDatabase(DatabaseType.POSTGRES_LOGICAL, {
  sslMode: PostgresSslMode.Require,
  sslClientCert: 'stored-client-cert',
  sslRootCert: 'stored-root-cert',
} as Partial<DatabaseSettings>);

describe('DatabaseConfigComponent', () => {
  it('shows the limited-user hint after an editor saves the connection settings', async () => {
    const { handlers, limitedUserCheckCalls } = recordLimitedUserChecks();
    renderDatabaseConfig(true, handlers);

    const databaseSectionTitle = screen.getByText('Database settings').parentElement!;

    fireEvent.click(databaseSectionTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);
    fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText(LIMITED_USER_HINT_TITLE)).toBeTruthy();
    expect(limitedUserCheckCalls.length).toBeGreaterThan(0);
  });

  it('neither checks the limited user nor shows its hint when the user is a viewer', async () => {
    const { handlers, limitedUserCheckCalls } = recordLimitedUserChecks();
    const { container } = renderDatabaseConfig(false, handlers);

    expect(await screen.findByText('db.example.com')).toBeTruthy();
    expect(container.querySelectorAll(EDIT_ICON_SELECTOR)).toHaveLength(0);
    expect(screen.queryByText(LIMITED_USER_HINT_TITLE)).toBeNull();
    expect(limitedUserCheckCalls).toHaveLength(0);
  });

  it('offers a retry when the backup configuration fails to load', async () => {
    const retriedBackupConfig = createPendingApiResponse();
    let backupConfigRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        ++backupConfigRequests <= 1
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : retriedBackupConfig.response,
      ),
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.get('/notifiers', 200, { items: [] }),
    );
    const { user } = renderScreen(
      <DatabaseConfigComponent
        database={savedDatabase}
        onDatabaseChanged={() => {}}
        onDatabaseDeleted={() => {}}
        onDatabaseCopied={() => {}}
        onDatabaseTransferred={() => {}}
        isCanManageDBs
      />,
    );

    await user.click(await screen.findByRole('button', { name: 'Retry' }));

    await waitFor(() => expect(backupConfigRequests).toBe(2));
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('ignores a late backup configuration failure from an earlier load of the same database', async () => {
    const configRead = vi.spyOn(backupConfigApi, 'getBackupConfig');
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const earlierBackupConfigLoads = [createPendingApiResponse(), createPendingApiResponse()];
    let backupConfigRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        backupConfigRequests < earlierBackupConfigLoads.length
          ? earlierBackupConfigLoads[backupConfigRequests++].response
          : createPendingApiResponse().response,
      ),
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.get('/notifiers', 200, { items: [] }),
    );

    const otherDatabase = { ...savedDatabase, id: 'other-database-id', name: 'Other' };
    const renderConfig = (database: Database) => (
      <DatabaseConfigComponent
        database={database}
        onDatabaseChanged={() => {}}
        onDatabaseDeleted={() => {}}
        onDatabaseCopied={() => {}}
        onDatabaseTransferred={() => {}}
        isCanManageDBs
      />
    );
    const view = renderScreen(renderConfig(savedDatabase));
    await waitFor(() => expect(backupConfigRequests).toBe(1));

    view.rerender(renderConfig(otherDatabase));
    await waitFor(() => expect(backupConfigRequests).toBe(earlierBackupConfigLoads.length));
    view.rerender(renderConfig(savedDatabase));
    await screen.findByText('Shop');

    await act(async () => {
      for (const earlierLoad of earlierBackupConfigLoads)
        earlierLoad.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 }));
      await Promise.allSettled(
        configRead.mock.results.slice(0, earlierBackupConfigLoads.length).map((read) => read.value),
      );
    });

    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();
  });

  describe.each(SAVED_VARIANTS)('connection section of $variant', ({ type, connectionString }) => {
    it('saves without a password when it stays empty and closes on Cancel without saving', async () => {
      const database = buildSavedDatabase(type);
      const { handler, sentUpdates } = recordUpdates(database);
      const sentTests: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
      const onDatabaseChanged = vi.fn();
      renderSavedDatabaseConfig(
        database,
        [
          handler,
          http.post(`*${API_BASE_PATH}/database-tests`, async ({ request }) => {
            sentTests.push(
              (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
            );

            return HttpResponse.json({ isConnected: true });
          }),
          mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
        ],
        onDatabaseChanged,
      );

      await openConnectionSection();
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

      expect(await screen.findByText('db.example.com')).toBeTruthy();
      expect(sentUpdates).toHaveLength(0);

      await openConnectionSection();
      clickTest();
      await clickSave();

      await waitFor(() => expect(onDatabaseChanged).toHaveBeenCalled());
      expect(sentTests[0].databaseId).toBe('database-id');
      expect(sentTests[0].settings).not.toHaveProperty('password');
      expect(sentUpdates[0].settings).not.toHaveProperty('password');
      expect(sentUpdates[0].settings).toMatchObject({ type, host: 'db.example.com' });
    });

    it('shows stored SSH credentials and offers editable ones after Replace', async () => {
      const database = buildSavedDatabase(type, {
        sshTunnel: {
          isEnabled: true,
          host: 'bastion.example.com',
          port: 22,
          username: 'tunnel-user',
          authType: SshTunnelAuthType.PASSWORD,
        },
      });
      renderSavedDatabaseConfig(database);
      await openConnectionSection();

      expect(screen.getByText('SSH credentials')).toBeTruthy();
      expect(
        (screen.getByRole('button', { name: /Test connection$/ }) as HTMLButtonElement).disabled,
      ).toBe(false);

      fireEvent.click(screen.getByRole('button', { name: 'Replace' }));

      expect(screen.queryByText('SSH credentials')).toBeNull();
      expect(screen.getByPlaceholderText('Enter SSH password')).toBeTruthy();
    });

    it('keeps a typed SSH replacement editable after the advanced settings collapse and reopen', async () => {
      renderSavedDatabaseConfig(
        buildSavedDatabase(type, {
          sshTunnel: {
            isEnabled: true,
            host: 'bastion.example.com',
            port: 22,
            username: 'tunnel-user',
            authType: SshTunnelAuthType.PASSWORD,
          },
        }),
      );
      await openConnectionSection();
      fireEvent.click(screen.getByRole('button', { name: 'Replace' }));
      fireEvent.change(screen.getByPlaceholderText('Enter SSH password'), {
        target: { value: 'typed-replacement' },
      });

      fireEvent.click(screen.getByText('Advanced settings'));
      fireEvent.click(screen.getByText('Advanced settings'));

      expect(screen.queryByText('SSH credentials')).toBeNull();
      expect((screen.getByPlaceholderText('Enter SSH password') as HTMLInputElement).value).toBe(
        'typed-replacement',
      );
    });

    it('shows the saved values and sends no update when Cancel follows an edit', async () => {
      const { sentUpdates } = renderConnectionSaveFlow(buildSavedDatabase(type));

      fireEvent.change(await openConnectionSection(), {
        target: { value: 'changed.example.com' },
      });
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'discarded' } });
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

      expect(await screen.findByText('db.example.com')).toBeTruthy();
      expect(screen.queryByText('changed.example.com')).toBeNull();

      expect((await openConnectionSection()).value).toBe('db.example.com');
      expect(getInputByPlaceholder(/password$/).value).toBe('');
      expect(sentUpdates).toHaveLength(0);
    });

    it('sends the entered password in the test and the update when a new password is entered', async () => {
      const { sentUpdates, sentTests } = renderConnectionSaveFlow(buildSavedDatabase(type));

      await openConnectionSection();
      fireEvent.change(getInputByPlaceholder(/password$/), {
        target: { value: 'replaced-secret' },
      });
      await testAndSaveConnection(sentUpdates);

      expect(sentTests[0].databaseId).toBe('database-id');
      expect(sentTests[0].settings).toMatchObject({ password: 'replaced-secret' });
      expect(sentUpdates[0].settings).toMatchObject({ password: 'replaced-secret' });
    });

    it('sends the new SSH password in the test and the update after Replace', async () => {
      const { sentUpdates, sentTests } = renderConnectionSaveFlow(
        buildSavedDatabase(type, { sshTunnel: passwordTunnel }),
      );
      const replacedTunnel = {
        ...passwordTunnel,
        password: 'replaced-tunnel-secret',
      };

      await openConnectionSection();
      fireEvent.click(screen.getByRole('button', { name: 'Replace' }));
      fireEvent.change(screen.getByPlaceholderText('Enter SSH password'), {
        target: { value: replacedTunnel.password },
      });
      await testAndSaveConnection(sentUpdates);

      expect(sentTests[0].settings.sshTunnel).toEqual(replacedTunnel);
      expect(sentUpdates[0].settings.sshTunnel).toEqual(replacedTunnel);
    });

    it('sends the key and passphrase without the typed password when key authentication replaces a password tunnel', async () => {
      const { sentUpdates, sentTests } = renderConnectionSaveFlow(
        buildSavedDatabase(type, { sshTunnel: passwordTunnel }),
      );
      const keyTunnel = {
        ...passwordTunnel,
        authType: SshTunnelAuthType.PRIVATE_KEY,
        privateKey: 'typed-private-key',
        privateKeyPassphrase: 'typed-passphrase',
      };

      await openConnectionSection();
      fireEvent.click(screen.getByRole('button', { name: 'Replace' }));
      fireEvent.change(screen.getByPlaceholderText('Enter SSH password'), {
        target: { value: 'abandoned-tunnel-secret' },
      });
      await chooseSshAuth('Private key');
      enterSshKey();
      await testAndSaveConnection(sentUpdates);

      expect(sentTests[0].settings.sshTunnel).toEqual(keyTunnel);
      expect(sentUpdates[0].settings.sshTunnel).toEqual(keyTunnel);
    });

    it('fills the fields from an imported connection string and asks for a new test', async () => {
      const database = buildSavedDatabase(type);
      renderSavedDatabaseConfig(database, [
        mockApi.post('/database-tests', 200, { isConnected: true }),
      ]);
      stubClipboard(async () => connectionString);
      await openConnectionSection();
      clickTest();
      await screen.findByRole('button', { name: /Save$/ });

      fireEvent.click(screen.getByText('Parse from clipboard'));

      await waitFor(() =>
        expect(getInputByPlaceholder(/host$/).value).toBe('imported.example.com'),
      );
      expect(screen.queryByRole('button', { name: /Save$/ })).toBeNull();
    });
  });

  it('shows a stored client certificate masked and clears it on Replace', async () => {
    renderSavedDatabaseConfig(
      buildSavedDatabase(DatabaseType.POSTGRES_PHYSICAL, {
        sslMode: PostgresSslMode.Require,
        sslClientCert: 'stored-client-cert',
      } as Partial<DatabaseSettings>),
    );
    await openConnectionSection();

    expect(screen.getByRole('button', { name: 'Replace' })).toBeTruthy();
    expect(screen.queryByDisplayValue('stored-client-cert')).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Replace' }));

    expect(screen.queryByRole('button', { name: 'Replace' })).toBeNull();
    expect(screen.getAllByPlaceholderText('-----BEGIN CERTIFICATE-----')[0]).toHaveProperty(
      'value',
      '',
    );
  });

  it('sends no stored certificate after Replace, a test and a save', async () => {
    const { handler, sentUpdates } = recordUpdates(savedWithCertificate);
    renderSavedDatabaseConfig(savedWithCertificate, [
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      handler,
    ]);
    await openConnectionSection();

    fireEvent.click(screen.getByRole('button', { name: 'Replace' }));
    clickTest();
    await clickSave();

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    const sentSettings = sentUpdates[0].settings as Record<string, unknown>;
    expect(sentSettings.sslClientCert).toBeFalsy();
    expect(sentSettings.sslClientKey).toBeFalsy();
    expect(sentSettings.sslRootCert).toBeFalsy();
  });

  it.each([
    { variant: 'PostgreSQL logical', type: DatabaseType.POSTGRES_LOGICAL },
    { variant: 'PostgreSQL physical', type: DatabaseType.POSTGRES_PHYSICAL },
  ])('sends no certificate after an import into $variant turns SSL off', async ({ type }) => {
    const savedDatabase = buildSavedDatabase(type, {
      sslMode: PostgresSslMode.Require,
      sslClientCert: 'stored-client-cert',
      sslRootCert: 'stored-root-cert',
    } as Partial<DatabaseSettings>);
    const { handler, sentUpdates } = recordUpdates(savedDatabase);
    renderSavedDatabaseConfig(savedDatabase, [
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      handler,
    ]);
    stubClipboard(
      async () =>
        'postgresql://importer:imported-secret@imported.example.com:5433/shop?sslmode=disable',
    );
    await openConnectionSection();

    fireEvent.click(screen.getByText('Parse from clipboard'));
    await waitFor(() => expect(getInputByPlaceholder(/host$/).value).toBe('imported.example.com'));
    clickTest();
    await clickSave();

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    const sentSettings = sentUpdates[0].settings as Record<string, unknown>;
    expect(sentSettings.sslMode).toBe(PostgresSslMode.Disable);
    expect(sentSettings.sslClientCert).toBeFalsy();
    expect(sentSettings.sslRootCert).toBeFalsy();
  });

  it.each([
    {
      variant: 'PostgreSQL logical',
      type: DatabaseType.POSTGRES_LOGICAL,
      field: 'Exclude whole tables',
    },
    { variant: 'MySQL', type: DatabaseType.MYSQL, field: 'Exclude tables' },
    { variant: 'MariaDB', type: DatabaseType.MARIADB, field: 'Exclude tables' },
    { variant: 'MongoDB', type: DatabaseType.MONGODB, field: 'Exclude collections' },
  ])(
    'keeps Save offered when an exclusion of $variant changes after a test',
    async ({ type, field }) => {
      renderSavedDatabaseConfig(buildSavedDatabase(type), [
        mockApi.post('/database-tests', 200, { isConnected: true }),
      ]);
      await openConnectionSection();
      clickTest();
      await screen.findByRole('button', { name: /Save$/ });

      fireEvent.click(screen.getByText('Advanced settings'));
      const exclusion = await screen.findByRole('combobox', { name: field });
      fireEvent.change(exclusion, { target: { value: 'audit_log' } });
      fireEvent.keyDown(exclusion, { key: 'Enter', code: 'Enter', keyCode: 13 });

      await waitFor(() => expect(screen.getAllByText('audit_log').length).toBeGreaterThan(0));
      expect(screen.getByRole('button', { name: /Save$/ })).toBeTruthy();
    },
  );

  describe('the defects of the connection section', () => {
    it('does not offer Save when the password changes during a successful test', async () => {
      const testCall = vi.spyOn(databaseApi, 'testDatabaseSettings');
      const test = mockApi.delayed({
        method: 'post',
        path: '/database-tests',
        status: 200,
        body: { isConnected: true },
      });
      renderSavedDatabaseConfig(buildSavedDatabase(DatabaseType.MYSQL), [test.handler]);
      await openConnectionSection();

      clickTest();
      await waitFor(() => expect(countRequests('/database-tests', 'POST')).toBe(1));
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'changed' } });
      await releaseAndSettle(testCall, test.release);

      expect(screen.queryByRole('button', { name: /Save$/ })).toBeNull();
    });

    it('shows no failure of a test whose settings changed while it was pending', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const testCall = vi.spyOn(databaseApi, 'testDatabaseSettings');
      const test = mockApi.delayed({
        method: 'post',
        path: '/database-tests',
        status: 200,
        body: { isConnected: false, error: 'late failure' },
      });
      renderSavedDatabaseConfig(buildSavedDatabase(DatabaseType.MYSQL), [test.handler]);
      await openConnectionSection();

      clickTest();
      await waitFor(() => expect(countRequests('/database-tests', 'POST')).toBe(1));
      fireEvent.change(getInputByPlaceholder(/password$/), { target: { value: 'changed' } });
      await releaseAndSettle(testCall, test.release);

      expect(screen.queryByText(/late failure/)).toBeNull();
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('reports the saved connection when the limited user suggestion is closed', async () => {
      const answeredDatabase = buildSavedDatabase(DatabaseType.MYSQL, {
        host: 'saved.example.com',
      });
      const onDatabaseChanged = vi.fn();
      renderSavedDatabaseConfig(
        buildSavedDatabase(DatabaseType.MYSQL),
        [
          mockApi.post('/database-tests', 200, { isConnected: true }),
          mockApi.put('/databases/{databaseId}', 200, answeredDatabase),
          mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: ['ALL'] }),
        ],
        onDatabaseChanged,
      );

      await openConnectionSection();
      clickTest();
      await clickSave();
      await screen.findByText('Create a read-only user for Databasus?');

      fireEvent.click(document.querySelector('.ant-modal-close')!);

      await waitFor(() => expect(onDatabaseChanged).toHaveBeenCalledWith(answeredDatabase));
      expect(getInputByPlaceholder(/host$/).value).toBe('saved.example.com');
    });

    it('asks for a new test when the user goes back from the limited user suggestion', async () => {
      const answeredDatabase = buildSavedDatabase(DatabaseType.MYSQL, {
        host: 'saved.example.com',
      });
      const onDatabaseChanged = vi.fn();
      renderSavedDatabaseConfig(
        buildSavedDatabase(DatabaseType.MYSQL),
        [
          mockApi.post('/database-tests', 200, { isConnected: true }),
          mockApi.put('/databases/{databaseId}', 200, answeredDatabase),
          mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: ['ALL'] }),
        ],
        onDatabaseChanged,
      );

      await openConnectionSection();
      clickTest();
      await clickSave();
      await screen.findByText('Create a read-only user for Databasus?');

      fireEvent.click(screen.getByRole('button', { name: 'Back' }));

      expect(((await screen.findByPlaceholderText(/host$/)) as HTMLInputElement).value).toBe(
        'saved.example.com',
      );
      expect(screen.getByRole('button', { name: /Test connection$/ })).toBeTruthy();
      expect(onDatabaseChanged).toHaveBeenCalledWith(answeredDatabase);
    });

    it('closes the section with the saved values and the reason when the check after a save fails', async () => {
      const answeredDatabase = buildSavedDatabase(DatabaseType.MYSQL, {
        host: 'saved.example.com',
      });
      const onDatabaseChanged = vi.fn();
      renderSavedDatabaseConfig(
        buildSavedDatabase(DatabaseType.MYSQL),
        [
          mockApi.post('/database-tests', 200, { isConnected: true }),
          mockApi.put('/databases/{databaseId}', 200, answeredDatabase),
          mockApi.post('/limited-user-checks', 403, buildProblem('permission_denied', 403)),
        ],
        onDatabaseChanged,
      );

      await openConnectionSection();
      clickTest();
      await clickSave();

      expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
      await waitFor(() => expect(screen.queryByPlaceholderText(/host$/)).toBeNull());
      expect(onDatabaseChanged).toHaveBeenCalledWith(answeredDatabase);
    });

    const reachLimitedUserSuggestion = async (handlers: RequestHandler[] = []) => {
      const onDatabaseChanged = vi.fn();
      const answeredDatabase = buildSavedDatabase(DatabaseType.MYSQL, {
        host: 'saved.example.com',
      });
      renderSavedDatabaseConfig(
        buildSavedDatabase(DatabaseType.MYSQL),
        [
          mockApi.post('/database-tests', 200, { isConnected: true }),
          mockApi.put('/databases/{databaseId}', 200, answeredDatabase),
          ...handlers,
        ],
        onDatabaseChanged,
      );

      await openConnectionSection();
      clickTest();
      await clickSave();
      await screen.findByText('Create a read-only user for Databasus?');

      return { onDatabaseChanged, answeredDatabase };
    };

    it('closes the section with the saved values when the user skips the suggestion', async () => {
      await reachLimitedUserSuggestion([
        mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: ['ALL'] }),
      ]);

      fireEvent.click(screen.getByRole('button', { name: 'Skip' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Yes, I accept risks' }));

      await waitFor(() => expect(screen.queryByPlaceholderText(/host$/)).toBeNull());
      expect(screen.queryByText('Create a read-only user for Databasus?')).toBeNull();
    });

    it('returns to the form with the generated user and asks for a new test and save', async () => {
      await reachLimitedUserSuggestion([
        mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: ['ALL'] }),
        mockApi.post('/limited-users', 201, {
          username: 'databasus_reader',
          password: 'generated-secret',
          isForcedWalRotationAvailable: true,
        }),
      ]);

      fireEvent.click(screen.getByRole('button', { name: 'Yes, create read-only user' }));

      expect(((await screen.findByPlaceholderText(/username$/)) as HTMLInputElement).value).toBe(
        'databasus_reader',
      );
      expect(getInputByPlaceholder(/password$/).value).toBe('generated-secret');
      expect(screen.getByRole('button', { name: /Test connection$/ })).toBeTruthy();
      expect(screen.queryByRole('button', { name: /Save$/ })).toBeNull();
    });

    it('keeps the suggestion with its choices when its own check fails', async () => {
      let limitedUserChecks = 0;
      await reachLimitedUserSuggestion([
        http.post(`*${API_BASE_PATH}/limited-user-checks`, () => {
          limitedUserChecks += 1;

          return limitedUserChecks === 1
            ? HttpResponse.json({ isAdvisable: true, privileges: [] })
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
        }),
      ]);

      await waitFor(() => expect(limitedUserChecks).toBe(2));

      expect(
        await screen.findByText('An internal server error occurred. Please try again later.'),
      ).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Skip' })).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Back' })).toBeTruthy();
    });

    it('opens no suggestion when another database is selected during a save', async () => {
      const updateCall = vi.spyOn(databaseApi, 'updateDatabase');
      const update = mockApi.delayed({
        method: 'put',
        path: '/databases/{databaseId}',
        status: 200,
        body: buildSavedDatabase(DatabaseType.MYSQL),
      });
      const onDatabaseChanged = vi.fn();
      const firstDatabase = buildSavedDatabase(DatabaseType.MYSQL);
      const otherDatabase = {
        ...buildSavedDatabase(DatabaseType.MYSQL),
        id: 'other-database-id',
        name: 'Other',
      };
      const renderConfig = (database: Database) => (
        <DatabaseConfigComponent
          database={database}
          onDatabaseChanged={onDatabaseChanged}
          onDatabaseDeleted={() => {}}
          onDatabaseCopied={() => {}}
          onDatabaseTransferred={() => {}}
          isCanManageDBs
        />
      );
      mockServer.use(
        mockApi.post('/database-tests', 200, { isConnected: true }),
        update.handler,
        mockApi.get('/notifiers', 200, { items: [] }),
        mockApi.pending('get', '/backup-configs/{databaseId}').handler,
        mockApi.pending('get', '/storages').handler,
        mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      );

      const view = renderScreen(renderConfig(firstDatabase));
      await openConnectionSection();
      clickTest();
      await clickSave();
      await waitFor(() => expect(countRequests('/databases/database-id', 'PUT')).toBe(1));

      view.rerender(renderConfig(otherDatabase));
      await releaseAndSettle(updateCall, update.release);

      expect(onDatabaseChanged).not.toHaveBeenCalled();
      expect(countRequests('/limited-user-checks', 'POST')).toBe(0);
      expect(screen.getByText('Other')).toBeTruthy();
    });

    it('sends no certificate after SSL is turned off and on and shows no mask', async () => {
      const { handler, sentUpdates } = recordUpdates(savedWithCertificate);
      renderSavedDatabaseConfig(savedWithCertificate, [
        handler,
        mockApi.post('/database-tests', 200, { isConnected: true }),
        mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      ]);
      await openConnectionSection();

      await chooseSslMode(/^Disable/);
      await chooseSslMode(/^Require/);

      expect(screen.queryByRole('button', { name: 'Replace' })).toBeNull();
      expect(screen.getAllByPlaceholderText('-----BEGIN CERTIFICATE-----')[0]).toHaveProperty(
        'value',
        '',
      );

      clickTest();
      await clickSave();

      await waitFor(() => expect(sentUpdates).toHaveLength(1));
      expect(sentUpdates[0].settings).not.toHaveProperty('sslClientCert');
      expect(sentUpdates[0].settings).not.toHaveProperty('sslClientKey');
      expect(sentUpdates[0].settings).not.toHaveProperty('sslRootCert');
    });

    it('removes the failure of a physical test when the SSH host changes', async () => {
      renderSavedDatabaseConfig(
        buildSavedDatabase(DatabaseType.POSTGRES_PHYSICAL, {
          sshTunnel: {
            isEnabled: true,
            host: 'bastion.example.com',
            port: 22,
            username: 'tunnel-user',
            authType: SshTunnelAuthType.PASSWORD,
          },
        }),
        [
          mockApi.post('/database-tests', 200, {
            isConnected: false,
            reason: 'PG_HBA_NO_ENTRY',
            error: 'no pg_hba.conf entry',
          }),
        ],
      );
      await openConnectionSection();

      clickTest();
      await screen.findByText('Replication is not allowed from this host');
      expect(screen.getByText(/^Add the line below to pg_hba\.conf/)).toBeTruthy();
      fireEvent.change(screen.getByDisplayValue('bastion.example.com'), {
        target: { value: 'other.example.com' },
      });

      expect(screen.queryByText('Replication is not allowed from this host')).toBeNull();
      expect(screen.queryByText(/^Add the line below to pg_hba\.conf/)).toBeNull();
    });

    it('shows a test that cannot reach the server inline', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      renderSavedDatabaseConfig(buildSavedDatabase(DatabaseType.MONGODB), [
        mockApi.post('/database-tests', 200, {
          isConnected: false,
          error: 'dial tcp: connection refused',
        }),
      ]);
      await openConnectionSection();

      clickTest();

      expect(
        await screen.findByText('The database could not be reached: dial tcp: connection refused'),
      ).toBeTruthy();
      expect(screen.getByText(/IP whitelist/)).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();
    });

    it('keeps the section open with every value and the reason when a save is refused', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      renderSavedDatabaseConfig(buildSavedDatabase(DatabaseType.MYSQL), [
        mockApi.post('/database-tests', 200, { isConnected: true }),
        mockApi.put('/databases/{databaseId}', 403, buildProblem('permission_denied', 403)),
      ]);
      await openConnectionSection();
      fireEvent.change(getInputByPlaceholder(/host$/), {
        target: { value: 'replica.example.com' },
      });
      fireEvent.change(screen.getByLabelText('Password'), {
        target: { value: 'replica-password' },
      });

      clickTest();
      await clickSave();

      expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();
      expect(getInputByPlaceholder(/host$/).value).toBe('replica.example.com');
    });

    it('sends one update when Save is clicked twice while pending', async () => {
      const updateCall = vi.spyOn(databaseApi, 'updateDatabase');
      const update = mockApi.delayed({
        method: 'put',
        path: '/databases/{databaseId}',
        status: 200,
        body: buildSavedDatabase(DatabaseType.MYSQL),
      });

      renderSavedDatabaseConfig(buildSavedDatabase(DatabaseType.MYSQL), [
        mockApi.post('/database-tests', 200, { isConnected: true }),
        update.handler,
        mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      ]);
      await openConnectionSection();
      clickTest();
      const saveButton = await screen.findByRole('button', { name: /Save$/ });

      fireEvent.click(saveButton);
      fireEvent.click(saveButton);
      await waitFor(() => expect(countRequests('/databases/database-id', 'PUT')).toBe(1));
      expect((getInputByPlaceholder(/host$/) as HTMLInputElement).disabled).toBe(true);
      await releaseAndSettle(updateCall, update.release);

      expect(countRequests('/databases/database-id', 'PUT')).toBe(1);
    });

    const STALE_ANSWER_CHANGES = [
      {
        change: 'the session changed',
        isSectionStillShown: true,
        applyChange: () =>
          act(() => {
            invalidateSession();
          }),
      },
      {
        change: 'another database was selected',
        isSectionStillShown: false,
        applyChange: (rerenderOtherDatabase: () => void) => rerenderOtherDatabase(),
      },
    ];

    const renderHeldSave = (handlers: RequestHandler[]) => {
      activateSession({ token: 'first-token', userId: 'first-user' });
      const onDatabaseChanged = vi.fn();
      const renderConfig = (database: Database) => (
        <DatabaseConfigComponent
          database={database}
          onDatabaseChanged={onDatabaseChanged}
          onDatabaseDeleted={() => {}}
          onDatabaseCopied={() => {}}
          onDatabaseTransferred={() => {}}
          isCanManageDBs
        />
      );

      mockServer.use(
        mockApi.post('/database-tests', 200, { isConnected: true }),
        ...handlers,
        mockApi.get('/notifiers', 200, { items: [] }),
        mockApi.pending('get', '/backup-configs/{databaseId}').handler,
        mockApi.pending('get', '/storages').handler,
        mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
        mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      );
      const view = renderScreen(renderConfig(buildSavedDatabase(DatabaseType.MYSQL)));
      const rerenderOtherDatabase = () =>
        view.rerender(
          renderConfig({
            ...buildSavedDatabase(DatabaseType.MYSQL),
            id: 'other-database-id',
            name: 'Other',
          }),
        );

      return { rerenderOtherDatabase };
    };

    describe.each(STALE_ANSWER_CHANGES)(
      'an answer after $change',
      ({ applyChange, isSectionStillShown }) => {
        const expectNoStaleOutcome = () => {
          expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
          expect(screen.queryByText(LIMITED_USER_HINT_TITLE)).toBeNull();
          if (isSectionStillShown) expect(screen.getByPlaceholderText(/host$/)).toBeTruthy();
        };

        it('shows nothing when the save was refused', async () => {
          const updateCall = vi.spyOn(databaseApi, 'updateDatabase');
          const refusedUpdate = mockApi.delayed({
            method: 'put',
            path: '/databases/{databaseId}',
            status: 403,
            body: buildProblem('permission_denied', 403),
          });

          const { rerenderOtherDatabase } = renderHeldSave([refusedUpdate.handler]);
          await openConnectionSection();
          clickTest();
          await clickSave();
          await waitFor(() => expect(updateCall).toHaveBeenCalled());

          applyChange(rerenderOtherDatabase);
          await releaseAndSettle(updateCall, refusedUpdate.release);

          expectNoStaleOutcome();
        });

        it.each([
          {
            outcome: 'advises a limited user',
            status: 200,
            body: { isAdvisable: true, privileges: ['ALL'] },
          },
          { outcome: 'fails', status: 403, body: buildProblem('permission_denied', 403) },
        ])('shows nothing when the check after the save $outcome', async ({ status, body }) => {
          const checkCall = vi.spyOn(databaseApi, 'checkLimitedUser');
          const limitedUserCheck = mockApi.delayed({
            method: 'post',
            path: '/limited-user-checks',
            status,
            body,
          } as Parameters<typeof mockApi.delayed>[0]);

          const { rerenderOtherDatabase } = renderHeldSave([
            mockApi.put('/databases/{databaseId}', 200, buildSavedDatabase(DatabaseType.MYSQL)),
            limitedUserCheck.handler,
          ]);
          await openConnectionSection();
          clickTest();
          await clickSave();
          await waitFor(() => expect(checkCall).toHaveBeenCalled());

          applyChange(rerenderOtherDatabase);
          await releaseAndSettle(checkCall, limitedUserCheck.release);

          expectNoStaleOutcome();
        });
      },
    );

    it('ignores a save answer after the session changed', async () => {
      activateSession({ token: 'first-token', userId: 'first-user' });
      const updateCall = vi.spyOn(databaseApi, 'updateDatabase');
      const update = mockApi.delayed({
        method: 'put',
        path: '/databases/{databaseId}',
        status: 200,
        body: buildSavedDatabase(DatabaseType.MYSQL),
      });
      const onDatabaseChanged = vi.fn();

      renderSavedDatabaseConfig(
        buildSavedDatabase(DatabaseType.MYSQL),
        [mockApi.post('/database-tests', 200, { isConnected: true }), update.handler],
        onDatabaseChanged,
      );
      await openConnectionSection();
      clickTest();
      await clickSave();
      await waitFor(() => expect(countRequests('/databases/database-id', 'PUT')).toBe(1));

      act(() => {
        invalidateSession();
      });
      await releaseAndSettle(updateCall, update.release);

      expect(onDatabaseChanged).not.toHaveBeenCalled();
      expect(countRequests('/limited-user-checks', 'POST')).toBe(0);
    });
  });
});

describe('DatabaseConfigComponent backup configuration', () => {
  const BACKUP_SECTION_TITLE = 'Backup config';
  const REFUSAL_REASON = 'You do not have permission to do this.';

  const logicalDatabase = buildSavedDatabase(DatabaseType.POSTGRES_LOGICAL);
  const physicalDatabase = buildSavedDatabase(DatabaseType.POSTGRES_PHYSICAL);

  const logicalSettings: Extract<BackupConfig['settings'], { type: 'LOGICAL' }> = {
    type: 'LOGICAL',
    schedule: { type: 'DAILY', timeOfDay: '04:00' },
    retention: {
      policy: 'TIME_PERIOD',
      period: 'MONTH',
      count: 0,
      gfsHours: 0,
      gfsDays: 0,
      gfsWeeks: 0,
      gfsMonths: 0,
      gfsYears: 0,
    },
    isRetryIfFailed: true,
    maxFailedTriesCount: 3,
    notificationTriggers: ['BACKUP_FAILED'],
  };

  const buildLogicalConfig = (
    settings: Partial<typeof logicalSettings> = {},
    retention: Partial<typeof logicalSettings.retention> = {},
  ): BackupConfig => ({
    cleanup: { pendingFileCount: 0, failedFileCount: 0 },
    databaseId: logicalDatabase.id,
    storageId: null,
    isBackupsEnabled: true,
    encryption: 'NONE',
    settings: {
      ...logicalSettings,
      ...settings,
      retention: { ...logicalSettings.retention, ...retention },
    },
  });

  const physicalConfig: BackupConfig = {
    cleanup: { pendingFileCount: 0, failedFileCount: 0 },
    databaseId: physicalDatabase.id,
    storageId: null,
    isBackupsEnabled: true,
    encryption: 'NONE',
    settings: {
      type: 'PHYSICAL',
      fullBackupSchedule: { type: 'HOURLY' },
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

  const BACKUP_EDITORS = [
    {
      kind: 'logical',
      database: logicalDatabase,
      config: buildLogicalConfig(),
      backupHistory: mockApi.get('/logical-backups', 200, {
        items: [
          {
            id: 'old-backup',
            databaseId: logicalDatabase.id,
            storageId: 'old-storage',
            status: 'COMPLETED',
            sizeMb: 1,
            rawDatabaseSizeMb: 1,
            durationMs: 1,
            encryption: 'NONE',
            verificationStatus: 'NOT_VERIFIED',
            createdAt: '2026-10-29T10:00:00Z',
          },
        ],
      }),
    },
    {
      kind: 'physical',
      database: physicalDatabase,
      config: physicalConfig,
      backupHistory: mockApi.get('/physical-backups', 200, {
        items: [
          {
            id: 'old-backup',
            type: 'FULL',
            status: 'COMPLETED',
            timelineId: 1,
            startLsn: '0/1000000',
            stopLsn: '0/2000000',
            hasGapBefore: false,
            sizeMb: 1,
            createdAt: '2026-10-29T10:00:00Z',
          },
        ],
        totalUsageMb: 1,
      }),
    },
  ];

  const declareBackupConfigRequests = () => [
    mockApi.get('/logical-backups', 200, { items: [] }),
    mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
    mockApi.get('/storages', 200, { items: [] }),
  ];

  const openBackupSection = async () => {
    const sectionTitle = screen.getByText(BACKUP_SECTION_TITLE).parentElement!;

    fireEvent.click(sectionTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);

    return screen.findByRole('button', { name: /Save$/ });
  };

  const renderLogicalSection = async (config: BackupConfig) => {
    renderSavedDatabaseConfig(logicalDatabase, [
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      ...declareBackupConfigRequests(),
    ]);

    return openBackupSection();
  };

  const toggleSuccessTrigger = () => fireEvent.click(screen.getAllByRole('checkbox')[0]);

  const chooseRetentionPolicy = async (optionTitle: RegExp) => {
    const policyLabel = screen.getByText('Retention policy');

    fireEvent.mouseDown(policyLabel.parentElement!.querySelector('.ant-select-selector')!);
    const option = await waitFor(() => {
      const policyOption = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
        (element) => optionTitle.test(element.textContent ?? ''),
      );
      expect(policyOption).toBeTruthy();
      return policyOption!;
    });

    fireEvent.click(option);
  };

  const getRetentionInput = (label: string) =>
    screen.getByText(label).parentElement!.querySelector('input') as HTMLInputElement;

  const setRetentionValue = (label: string, value: string) =>
    fireEvent.change(getRetentionInput(label), { target: { value } });

  const GFS_POLICY = /^GFS/;
  const COUNT_POLICY = /^Count/;
  const TIME_PERIOD_POLICY = /^Time period/;

  describe.each(BACKUP_EDITORS)('of the $kind editor', ({ database, config }) => {
    it('disables Cancel during a pending save', async () => {
      const saveCall = vi.spyOn(backupConfigApi, 'updateBackupConfig');
      const save = mockApi.delayed({
        method: 'put',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: config,
      });
      const { container } = renderSavedDatabaseConfig(database, [
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        save.handler,
        ...declareBackupConfigRequests(),
      ]);
      const saveButton = await openBackupSection();
      toggleSuccessTrigger();

      fireEvent.click(saveButton);
      await waitFor(() => expect(saveCall).toHaveBeenCalled());

      expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', true);
      expect(container.querySelectorAll(EDIT_ICON_SELECTOR)).toHaveLength(0);

      await releaseAndSettle(saveCall, save.release);
    });

    it('keeps the form, shows the reason and reports no save when the server refuses', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const saveCall = vi.spyOn(backupConfigApi, 'updateBackupConfig');
      const onDatabaseChanged = vi.fn();
      renderSavedDatabaseConfig(
        database,
        [
          mockApi.get('/backup-configs/{databaseId}', 200, config),
          mockApi.put('/backup-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
          ...declareBackupConfigRequests(),
        ],
        onDatabaseChanged,
      );
      const saveButton = await openBackupSection();
      toggleSuccessTrigger();
      const isEnteredTriggerChecked = (screen.getAllByRole('checkbox')[0] as HTMLInputElement)
        .checked;

      fireEvent.click(saveButton);
      await waitFor(() => expect(saveCall).toHaveBeenCalled());
      await act(async () => {
        await Promise.allSettled([saveCall.mock.results[0].value]);
      });

      expect(screen.getByText(REFUSAL_REASON)).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', false);
      expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(
        isEnteredTriggerChecked,
      );
      expect(onDatabaseChanged).not.toHaveBeenCalled();
    });
  });

  describe.each(BACKUP_EDITORS)('when the $kind section cannot load', ({ database, config }) => {
    const openFailedSection = async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const configLoad = { isFailing: false };
      renderSavedDatabaseConfig(database, [
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
          configLoad.isFailing
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json(config),
        ),
        ...declareBackupConfigRequests(),
      ]);
      await screen.findByText('Backups enabled');

      configLoad.isFailing = true;
      const sectionTitle = screen.getByText(BACKUP_SECTION_TITLE).parentElement!;
      fireEvent.click(sectionTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);

      expect(
        await screen.findByText('An internal server error occurred. Please try again later.'),
      ).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();

      return configLoad;
    };

    it('shows the form after a successful retry', async () => {
      const configLoad = await openFailedSection();

      configLoad.isFailing = false;
      fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

      expect(await screen.findByRole('button', { name: /Save$/ })).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    });

    it('returns to the configuration view on Cancel', async () => {
      await openFailedSection();

      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

      expect(await screen.findByText('Backups enabled')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull();
    });
  });

  it.each(BACKUP_EDITORS)(
    'shows one error and one retry when the settings tab cannot load a $kind config',
    async ({ database, config }) => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isConfigFailing = true;
      renderSavedDatabaseConfig(database, [
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
          isConfigFailing
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json(config),
        ),
        ...declareBackupConfigRequests(),
      ]);

      const retryButton = await screen.findByRole('button', { name: 'Retry' });
      await waitFor(() =>
        expect(
          screen.getAllByText('An internal server error occurred. Please try again later.'),
        ).toHaveLength(1),
      );

      expect(screen.getAllByRole('button', { name: 'Retry' })).toHaveLength(1);
      expect(showAlert).not.toHaveBeenCalled();

      isConfigFailing = false;
      fireEvent.click(retryButton);

      expect(await screen.findByText('Backups enabled')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      expect(showAlert).not.toHaveBeenCalled();
    },
  );

  it('shows one error and one retry when the section opens after the settings tab could not load', async () => {
    renderSavedDatabaseConfig(logicalDatabase, [
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
      ),
      ...declareBackupConfigRequests(),
    ]);
    await screen.findByRole('button', { name: 'Retry' });
    const sectionTitle = screen.getByText(BACKUP_SECTION_TITLE).parentElement!;

    fireEvent.click(sectionTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);
    await screen.findByRole('button', { name: 'Cancel' });

    await waitFor(() =>
      expect(
        screen.getAllByText('An internal server error occurred. Please try again later.'),
      ).toHaveLength(1),
    );
    expect(screen.getAllByRole('button', { name: 'Retry' })).toHaveLength(1);
  });

  it.each(BACKUP_EDITORS)(
    'shows the reason with a retry when the $kind configuration view cannot load its storage',
    async ({ database, config }) => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isStorageFailing = true;
      renderSavedDatabaseConfig(database, [
        mockApi.get('/backup-configs/{databaseId}', 200, {
          ...config,
          storageId: 'archive-storage-id',
        }),
        http.get(`*${API_BASE_PATH}/storages/:storageId`, () =>
          isStorageFailing
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({
                id: 'archive-storage-id',
                workspaceId: database.workspaceId,
                name: 'Archive',
                settings: { type: 'LOCAL' },
              }),
        ),
        ...declareBackupConfigRequests(),
      ]);

      const retryButton = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.getByText('Backups enabled')).toBeTruthy();
      expect(
        screen.getByText('An internal server error occurred. Please try again later.'),
      ).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();

      isStorageFailing = false;
      fireEvent.click(retryButton);

      expect(await screen.findByText('Archive')).toBeTruthy();
    },
  );

  it.each(BACKUP_EDITORS)(
    'shows the saved values with Save disabled and sends no update when Cancel follows an edit in the $kind section',
    async ({ database, config }) => {
      const chooseEncryption = async (optionTitle: string) => {
        const control = await screen.findByRole('combobox', { name: 'Encryption' });

        fireEvent.mouseDown(control.closest('.ant-select')!.querySelector('.ant-select-selector')!);
        const option = await waitFor(() => {
          const encryptionOption = document.querySelector(
            `.ant-select-item-option[title="${optionTitle}"]`,
          );
          expect(encryptionOption).toBeTruthy();
          return encryptionOption!;
        });

        fireEvent.click(option);
      };
      const getShownEncryption = async () =>
        (await screen.findByText('Encryption')).parentElement!.textContent;

      renderSavedDatabaseConfig(database, [
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        ...declareBackupConfigRequests(),
      ]);
      await screen.findByText('Backups enabled');

      const saveButton = await openBackupSection();
      await chooseEncryption('Encrypt backup files');

      expect(saveButton).toHaveProperty('disabled', false);

      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      await waitFor(() =>
        expect(screen.queryByRole('combobox', { name: 'Encryption' })).toBeNull(),
      );

      expect(await getShownEncryption()).toBe('EncryptionNone');

      const reopenedSaveButton = await openBackupSection();

      expect(reopenedSaveButton).toHaveProperty('disabled', true);
      expect(
        screen
          .getByRole('combobox', { name: 'Encryption' })
          .closest('.ant-select')!
          .querySelector('.ant-select-selection-item')?.textContent,
      ).toBe('None');
      expect(countRequests(`/backup-configs/${database.id}`, 'PUT')).toBe(0);
    },
  );

  it('keeps entered GFS values through a switch to another policy and back', async () => {
    await renderLogicalSection(buildLogicalConfig({}, { policy: 'GFS', gfsDays: 7 }));
    setRetentionValue('Daily backups', '11');

    await chooseRetentionPolicy(COUNT_POLICY);
    await chooseRetentionPolicy(GFS_POLICY);

    expect(getRetentionInput('Daily backups').value).toBe('11');
  });

  it('fills the GFS defaults when a config with no GFS periods switches to GFS', async () => {
    const saveButton = await renderLogicalSection(buildLogicalConfig());

    await chooseRetentionPolicy(GFS_POLICY);

    expect(getRetentionInput('Daily backups').value).toBe('7');
    expect(getRetentionInput('Yearly backups').value).toBe('3');
    expect(saveButton).toHaveProperty('disabled', false);
  });

  it('allows saving a GFS policy that keeps only hourly backups on an hourly schedule', async () => {
    const saveButton = await renderLogicalSection(
      buildLogicalConfig({ schedule: { type: 'HOURLY' } }, { policy: 'GFS', gfsHours: 5 }),
    );

    toggleSuccessTrigger();

    expect(saveButton).toHaveProperty('disabled', false);
  });

  it('keeps Save disabled for a GFS policy with no shown period and backups off', async () => {
    const saveButton = await renderLogicalSection(
      buildLogicalConfig({}, { policy: 'GFS', gfsDays: 7 }),
    );

    fireEvent.click(screen.getAllByRole('switch')[0]);
    setRetentionValue('Daily backups', '0');

    expect(saveButton).toHaveProperty('disabled', true);
  });

  it('disables Save again when the retention period is changed and changed back', async () => {
    const saveButton = await renderLogicalSection(buildLogicalConfig());
    const choosePeriod = async (periodTitle: string) => {
      const periodSelect = screen
        .getByText('Retention policy')
        .parentElement!.querySelectorAll('.ant-select-selector')[1];

      fireEvent.mouseDown(periodSelect);
      const option = await waitFor(() => {
        const periodOption = document.querySelector(
          `.ant-select-item-option[title="${periodTitle}"]`,
        );
        expect(periodOption).toBeTruthy();
        return periodOption!;
      });

      fireEvent.click(option);
    };

    await choosePeriod('1 week');

    expect(saveButton).toHaveProperty('disabled', false);

    await choosePeriod('1 month');

    expect(saveButton).toHaveProperty('disabled', true);
  });

  it('keeps Save disabled after a policy round trip on a config with no GFS periods', async () => {
    const saveButton = await renderLogicalSection(buildLogicalConfig());

    await chooseRetentionPolicy(GFS_POLICY);
    await chooseRetentionPolicy(TIME_PERIOD_POLICY);

    expect(saveButton).toHaveProperty('disabled', true);
  });

  it('opens the advanced settings for a config with retry turned off', async () => {
    await renderLogicalSection(buildLogicalConfig({ isRetryIfFailed: false }));

    expect(screen.getByText('Retry backup if failed')).toBeTruthy();
  });

  it('keeps the advanced settings closed for default values until the toggle opens them', async () => {
    await renderLogicalSection(buildLogicalConfig());

    expect(screen.queryByText('Retry backup if failed')).toBeNull();

    fireEvent.click(screen.getByText('Advanced settings'));

    expect(screen.getByText('Retry backup if failed')).toBeTruthy();
  });

  describe.each(BACKUP_EDITORS)(
    'after a storage was cleared in the $kind section',
    ({ database, config, backupHistory }) => {
      const renderConfigWithStoredStorage = (targetStorageId: string) => {
        let storedConfig: BackupConfig = { ...config, storageId: 'old-storage' };
        const submitted: BackupConfigInput[] = [];
        const buildStorage = ({ id, name }: { id: string; name: string }) => ({
          id,
          workspaceId: database.workspaceId,
          name,
          settings: { type: 'LOCAL' as const },
        });
        renderSavedDatabaseConfig(database, [
          http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
            HttpResponse.json(storedConfig),
          ),
          http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
            const saved = (await request.json()) as BackupConfigInput;
            submitted.push(saved);
            storedConfig = { ...storedConfig, storageId: saved.storageId };

            return HttpResponse.json(storedConfig);
          }),
          mockApi.get('/storages', 200, {
            items: [
              buildStorage({ id: 'old-storage', name: 'Previous archive' }),
              buildStorage({ id: 'target-storage', name: 'Archive' }),
            ],
          }),
          http.get(`*${API_BASE_PATH}/storages/:storageId`, ({ params }) =>
            HttpResponse.json(
              buildStorage({ id: String(params.storageId), name: 'Shown archive' }),
            ),
          ),
          backupHistory,
        ]);

        return {
          submitted,
          targetName: targetStorageId === 'old-storage' ? 'Previous archive' : 'Archive',
        };
      };

      it.each(['old-storage', 'target-storage'])(
        'warns before reassignment to %s when the section opens again',
        async (storageId) => {
          const { submitted, targetName } = renderConfigWithStoredStorage(storageId);
          await screen.findByText('Backups enabled');

          await openBackupSection();
          const storageField = (await screen.findByText('Storage')).parentElement!;
          await waitFor(() => expect(storageField.querySelector('.ant-select-clear')).toBeTruthy());

          fireEvent.mouseDown(storageField.querySelector('.ant-select-clear')!);
          fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));
          await waitFor(() => expect(submitted).toHaveLength(1));

          expect(submitted[0].storageId).toBeNull();

          await waitFor(() => expect(screen.queryByRole('button', { name: /Save$/ })).toBeNull());
          await screen.findByText('Backups enabled');

          await openBackupSection();
          const reopenedField = (await screen.findByText('Storage')).parentElement!;
          fireEvent.mouseDown(reopenedField.querySelector('.ant-select-selector')!);
          const option = await waitFor(() => {
            const storageOption = document.querySelector(
              `.ant-select-item-option[title="${targetName}"]`,
            );
            expect(storageOption).toBeTruthy();
            return storageOption!;
          });
          fireEvent.click(option);

          fireEvent.click(await screen.findByRole('button', { name: /Save$/ }));

          expect(
            await screen.findByText(/Switching storage immediately removes the old backup history/),
          ).toBeTruthy();
          expect(submitted).toHaveLength(1);

          fireEvent.click(
            screen.getByRole('button', { name: 'Switch storage and remove history' }),
          );

          await waitFor(() => expect(submitted).toHaveLength(2));
          expect(submitted[1].storageId).toBe(storageId);
        },
      );
    },
  );

  it.each(BACKUP_EDITORS)(
    'shows an accepted storage switch in the $kind configuration view after a save',
    async ({ database, config }) => {
      const acknowledged = { ...config, cleanup: { pendingFileCount: 2, failedFileCount: 0 } };
      let isSwitchAccepted = false;
      renderSavedDatabaseConfig(database, [
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
          HttpResponse.json(isSwitchAccepted ? acknowledged : config),
        ),
        http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => {
          isSwitchAccepted = true;
          return HttpResponse.json(acknowledged);
        }),
        mockApi.get('/logical-backups', 200, { items: [] }),
        mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
        mockApi.get('/storages', 200, { items: [] }),
      ]);
      await screen.findByText('Backups enabled');
      const saveButton = await openBackupSection();
      toggleSuccessTrigger();

      fireEvent.click(saveButton);

      expect(await screen.findByText('Deleting old files. Files remaining: 2.')).toBeTruthy();
    },
  );

  it('keeps earlier cleanup visible when a second reassignment adds no files', async () => {
    let currentConfig = { ...buildLogicalConfig(), storageId: 'archive-a' };
    const storages = ['a', 'b', 'c'].map((archive) => ({
      id: `archive-${archive}`,
      workspaceId: logicalDatabase.workspaceId,
      name: `Archive ${archive.toUpperCase()}`,
      settings: { type: 'LOCAL' as const },
    }));
    const savedSelections: string[] = [];
    renderSavedDatabaseConfig(logicalDatabase, [
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        HttpResponse.json(currentConfig),
      ),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
        const config = (await request.json()) as BackupConfigInput;
        savedSelections.push(config.storageId!);
        currentConfig = {
          ...config,
          databaseId: logicalDatabase.id,
          storageId: config.storageId!,
          cleanup: { pendingFileCount: 4, failedFileCount: 1 },
        };
        return HttpResponse.json(currentConfig);
      }),
      mockApi.get('/storages', 200, { items: storages }),
      http.get(`*${API_BASE_PATH}/storages/:storageId`, ({ params }) =>
        HttpResponse.json(storages.find((storage) => storage.id === params.storageId)),
      ),
      mockApi.get('/logical-backups', 200, { items: [] }),
    ]);
    await screen.findByText('Archive A');

    for (const archive of ['B', 'C']) {
      await openBackupSection();
      const storageField = screen.getByText('Storage').parentElement!;
      fireEvent.mouseDown(storageField.querySelector('.ant-select-selector')!);
      const option = await waitFor(() => {
        const storageOption = document.querySelector(
          `.ant-select-item-option[title="Archive ${archive}"]`,
        );
        expect(storageOption).toBeTruthy();
        return storageOption!;
      });
      fireEvent.click(option);
      fireEvent.click(screen.getByRole('button', { name: /Save$/ }));
      fireEvent.click(
        await screen.findByRole('button', { name: 'Switch storage and remove history' }),
      );
      await screen.findByText('Deleting old files. Files remaining: 4.');
    }

    expect(savedSelections).toEqual(['archive-b', 'archive-c']);
    expect(screen.getByText('Deletion will be retried for files: 1.')).toBeTruthy();
    mockServer.use(
      mockApi.get('/backup-configs/{databaseId}', 200, {
        ...currentConfig,
        cleanup: { pendingFileCount: 0, failedFileCount: 0 },
      }),
    );
    await waitFor(() =>
      expect(screen.queryByText('Deleting old files. Files remaining: 4.')).toBeNull(),
    );
    expect(screen.getByText('Archive C')).toBeTruthy();
  });

  it.each(BACKUP_EDITORS)(
    'keeps the last cleanup count on failure and retries only reads in the $kind view',
    async ({ database, config }) => {
      const pendingConfig = {
        ...config,
        isBackupsEnabled: false,
        cleanup: { pendingFileCount: 4, failedFileCount: 2 },
      };
      let configReads = 0;
      renderSavedDatabaseConfig(database, [
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => {
          configReads++;
          return configReads === 1
            ? HttpResponse.json(pendingConfig)
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 });
        }),
        mockApi.get('/storages', 200, { items: [] }),
      ]);
      await screen.findByText('Deleting old files. Files remaining: 4.');
      expect(screen.getByText('Deletion will be retried for files: 2.')).toBeTruthy();

      await screen.findByRole('button', { name: 'Retry status' });
      vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(6000);
      });
      expect(configReads).toBe(2);
      expect(screen.getByText('Deleting old files. Files remaining: 4.')).toBeTruthy();
      mockServer.use(mockApi.get('/backup-configs/{databaseId}', 200, config));
      fireEvent.click(screen.getByRole('button', { name: 'Retry status' }));
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });
      await waitForCondition(() =>
        expect(screen.queryByText('Deleting old files. Files remaining: 4.')).toBeNull(),
      );
      const readsAfterCompletion = countRequests('/backup-configs/database-id', 'GET');
      await act(async () => {
        await vi.advanceTimersByTimeAsync(6000);
      });
      expect(countRequests('/backup-configs/database-id', 'GET')).toBe(readsAfterCompletion);
      expect(mockServer.getRequests().filter((request) => request.method !== 'GET')).toHaveLength(
        0,
      );
    },
  );

  it.each(BACKUP_EDITORS)(
    'retains cleanup and offers read retry when polling fails during a $kind draft',
    async ({ database, config }) => {
      const configRead = vi.spyOn(backupConfigApi, 'getBackupConfig');
      const pendingConfig = { ...config, cleanup: { pendingFileCount: 4, failedFileCount: 1 } };
      renderSavedDatabaseConfig(database, [
        mockApi.get('/backup-configs/{databaseId}', 200, pendingConfig),
        ...declareBackupConfigRequests(),
      ]);

      await screen.findByText('Deleting old files. Files remaining: 4.');
      await openBackupSection();
      toggleSuccessTrigger();
      const isDraftChecked = (screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked;
      const polling = mockApi.pending('get', '/backup-configs/{databaseId}');
      mockServer.use(polling.handler);
      const readsBeforePolling = configRead.mock.calls.length;

      await waitFor(() => expect(configRead.mock.calls.length).toBeGreaterThan(readsBeforePolling));
      await releaseAndSettle(configRead, () =>
        polling.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 })),
      );

      expect(screen.getByText('Deleting old files. Files remaining: 4.')).toBeTruthy();
      expect(screen.getByText('Deletion will be retried for files: 1.')).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Retry status' })).toBeTruthy();

      mockServer.use(mockApi.get('/backup-configs/{databaseId}', 200, config));
      fireEvent.click(screen.getByRole('button', { name: 'Retry status' }));
      await waitFor(() =>
        expect(screen.queryByText('Deleting old files. Files remaining: 4.')).toBeNull(),
      );

      expect(screen.getAllByRole('checkbox')[0]).toHaveProperty('checked', isDraftChecked);
      expect(screen.getByRole('button', { name: /Save$/ })).toHaveProperty('disabled', false);
      expect(mockServer.getRequests().filter((request) => request.method !== 'GET')).toHaveLength(
        0,
      );
    },
  );

  it.each(BACKUP_EDITORS)(
    'retains acknowledged cleanup when a read fails after saving the $kind config',
    async ({ database, config }) => {
      const onDatabaseChanged = vi.fn();
      const acknowledgedConfig = {
        ...config,
        cleanup: { pendingFileCount: 7, failedFileCount: 2 },
      };
      renderSavedDatabaseConfig(
        database,
        [
          mockApi.get('/backup-configs/{databaseId}', 200, config),
          mockApi.put('/backup-configs/{databaseId}', 200, acknowledgedConfig),
          ...declareBackupConfigRequests(),
        ],
        onDatabaseChanged,
      );

      await screen.findByText('Backups enabled');
      const saveButton = await openBackupSection();
      toggleSuccessTrigger();
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 500, buildProblem('internal', 500)),
      );

      fireEvent.click(saveButton);

      await waitFor(() => expect(onDatabaseChanged).toHaveBeenCalledOnce());

      expect(screen.getByText('Deleting old files. Files remaining: 7.')).toBeTruthy();
      await screen.findByRole('button', { name: 'Retry status' });
      expect(screen.getByText('Deletion will be retried for files: 2.')).toBeTruthy();
      expect(screen.queryByRole('button', { name: /Save$/ })).toBeNull();
      expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(
        1,
      );
    },
  );

  it.each(BACKUP_EDITORS)(
    'preserves the active draft when cleanup completes in the $kind view',
    async ({ database, config }) => {
      const configRead = vi.spyOn(backupConfigApi, 'getBackupConfig');
      const archive = {
        id: 'draft-archive',
        workspaceId: database.workspaceId,
        name: 'Draft archive',
        settings: { type: 'LOCAL' as const },
      };
      const view = renderSavedDatabaseConfig(database, [
        mockApi.get('/backup-configs/{databaseId}', 200, {
          ...config,
          cleanup: { pendingFileCount: 2, failedFileCount: 0 },
        }),
        mockApi.get('/storages', 200, { items: [archive] }),
        mockApi.get('/logical-backups', 200, { items: [] }),
        mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
      ]);
      await screen.findByText('Deleting old files. Files remaining: 2.');
      await openBackupSection();
      const polling = mockApi.pending('get', '/backup-configs/{databaseId}');
      mockServer.use(polling.handler);
      const readsBeforePolling = configRead.mock.calls.length;
      toggleSuccessTrigger();
      const successTrigger = screen.getAllByRole('checkbox')[0];
      const isDraftChecked = (successTrigger as HTMLInputElement).checked;
      if (config.settings.type === 'LOGICAL') await chooseRetentionPolicy(COUNT_POLICY);
      const retentionLabel =
        config.settings.type === 'LOGICAL' ? 'Most recent backups' : 'Most recent full backups';
      const retentionInput = screen.getByRole('spinbutton', { name: retentionLabel });
      fireEvent.change(retentionInput, { target: { value: '37' } });
      fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Storage' }));
      fireEvent.click(await screen.findByText('Draft archive'));
      retentionInput.focus();
      view.container.style.height = '200px';
      view.container.style.overflowY = 'auto';
      view.container.scrollTop = 180;

      await waitFor(() => expect(configRead.mock.calls.length).toBeGreaterThan(readsBeforePolling));
      await releaseAndSettle(configRead, () =>
        polling.respond(
          HttpResponse.json({
            ...config,
            cleanup: { pendingFileCount: 0, failedFileCount: 0 },
          }),
        ),
      );

      expect(screen.getAllByRole('checkbox')[0]).toHaveProperty('checked', isDraftChecked);
      expect(screen.getByRole('spinbutton', { name: retentionLabel })).toHaveProperty(
        'value',
        '37',
      );
      expect(
        screen
          .getByRole('combobox', { name: 'Storage' })
          .closest('.ant-select')
          ?.querySelector('.ant-select-selection-item')?.textContent,
      ).toBe('Draft archive');
      expect(document.activeElement).toBe(retentionInput);
      expect(view.container.scrollTop).toBe(180);
      expect(screen.getByRole('button', { name: /Save$/ })).toBeTruthy();
      expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(
        0,
      );
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      expect(screen.queryByText('Deleting old files. Files remaining: 2.')).toBeNull();
    },
  );

  it.each(['database', 'workspace', 'session', 'unmount'] as const)(
    'cancels a scheduled cleanup poll before it starts after a %s change',
    async (contextChange) => {
      const initialRead = createPendingApiResponse();
      const configRead = vi.spyOn(backupConfigApi, 'getBackupConfig');
      mockServer.use(
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, async () =>
          (await initialRead.response).clone(),
        ),
        mockApi.get('/storages', 200, { items: [] }),
        mockApi.get('/notifiers', 200, { items: [] }),
        mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
        mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      );
      const renderConfig = (database: Database) => (
        <DatabaseConfigComponent
          database={database}
          onDatabaseChanged={() => {}}
          onDatabaseDeleted={() => {}}
          onDatabaseCopied={() => {}}
          onDatabaseTransferred={() => {}}
          isCanManageDBs
        />
      );
      const view = renderScreen(renderConfig(logicalDatabase), { isStrictMode: true });
      await waitFor(() => expect(configRead.mock.calls.length).toBeGreaterThan(0));
      vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
      await releaseAndSettle(configRead, () =>
        initialRead.respond(
          HttpResponse.json({
            ...buildLogicalConfig(),
            cleanup: { pendingFileCount: 3, failedFileCount: 0 },
          }),
        ),
      );
      expect(screen.getByText('Deleting old files. Files remaining: 3.')).toBeTruthy();
      const initialReads = configRead.mock.calls.length;
      mockServer.use(mockApi.get('/backup-configs/{databaseId}', 200, buildLogicalConfig()));

      if (contextChange === 'database' || contextChange === 'workspace') {
        view.rerender(
          renderConfig({
            ...logicalDatabase,
            id: contextChange === 'database' ? 'new-database' : logicalDatabase.id,
            workspaceId: 'new-workspace',
          }),
        );
        await waitForCondition(() =>
          expect(configRead.mock.calls.length).toBeGreaterThan(initialReads),
        );
        await act(async () => {
          await Promise.allSettled(
            configRead.mock.results.slice(initialReads).map((read) => read.value),
          );
        });
      } else if (contextChange === 'session') {
        act(() => activateSession({ token: 'replacement-session', userId: 'replacement-user' }));
      } else {
        view.unmount();
      }
      const readsAfterScopeChange = configRead.mock.calls.length;
      const requestsAfterScopeChange = mockServer
        .getRequests()
        .filter((request) => request.url.includes('/backup-configs/')).length;

      await act(async () => {
        await vi.advanceTimersByTimeAsync(6000);
      });

      expect(configRead.mock.calls.length).toBe(readsAfterScopeChange);
      expect(
        mockServer.getRequests().filter((request) => request.url.includes('/backup-configs/')),
      ).toHaveLength(requestsAfterScopeChange);
      view.unmount();
    },
  );

  it.each(['database', 'workspace', 'session', 'unmount'] as const)(
    'ignores obsolete cleanup after a %s change',
    async (contextChange) => {
      const configRead = vi.spyOn(backupConfigApi, 'getBackupConfig');
      const oldRead = mockApi.pending('get', '/backup-configs/{databaseId}');
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, {
          ...buildLogicalConfig(),
          cleanup: { pendingFileCount: 3, failedFileCount: 0 },
        }),
        mockApi.get('/storages', 200, { items: [] }),
        mockApi.get('/notifiers', 200, { items: [] }),
        mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      );
      mockServer.use(mockApi.pending('get', '/verification-configs/{databaseId}').handler);
      const renderConfig = (database: Database) => (
        <DatabaseConfigComponent
          database={database}
          onDatabaseChanged={() => {}}
          onDatabaseDeleted={() => {}}
          onDatabaseCopied={() => {}}
          onDatabaseTransferred={() => {}}
          isCanManageDBs
        />
      );
      const view = renderScreen(renderConfig(logicalDatabase), { isStrictMode: true });
      await screen.findByText('Deleting old files. Files remaining: 3.');
      const readsBeforePolling = configRead.mock.calls.length;
      mockServer.use(oldRead.handler);
      await waitFor(() => expect(configRead.mock.calls.length).toBeGreaterThan(readsBeforePolling));
      const obsoleteReadIndex = configRead.mock.calls.length - 1;
      mockServer.use(
        mockApi.get('/backup-configs/{databaseId}', 200, {
          ...buildLogicalConfig(),
          databaseId: 'new-database',
        }),
      );
      if (contextChange === 'database' || contextChange === 'workspace') {
        view.rerender(
          renderConfig({
            ...logicalDatabase,
            id: contextChange === 'database' ? 'new-database' : logicalDatabase.id,
            workspaceId: 'new-workspace',
          }),
        );
        if (contextChange === 'database')
          await waitFor(() =>
            expect(screen.queryByText('Deleting old files. Files remaining: 3.')).toBeNull(),
          );
      } else if (contextChange === 'session') {
        act(() => activateSession({ token: 'new-session', userId: 'new-user' }));
      } else view.unmount();

      await releaseAndSettle(
        configRead,
        () => {
          oldRead.respond(
            contextChange === 'database' || contextChange === 'unmount'
              ? HttpResponse.json({
                  ...buildLogicalConfig(),
                  cleanup: { pendingFileCount: 99, failedFileCount: 1 },
                })
              : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
          );
        },
        obsoleteReadIndex,
      );

      expect(screen.queryByText('Deleting old files. Files remaining: 99.')).toBeNull();
      expect(screen.queryByRole('button', { name: 'Retry status' })).toBeNull();
    },
  );

  describe('of the physical section', () => {
    const incrementalDatabase = buildSavedDatabase(DatabaseType.POSTGRES_PHYSICAL, {
      backupType: 'FULL_INCREMENTAL',
    });
    const walStreamDatabase = buildSavedDatabase(DatabaseType.POSTGRES_PHYSICAL, {
      backupType: 'FULL_INCREMENTAL_WAL_STREAM',
    });
    const gfsRetention = {
      policy: 'GFS' as const,
      count: 0,
      gfsHours: 0,
      gfsDays: 7,
      gfsWeeks: 4,
      gfsMonths: 0,
      gfsYears: 0,
    };

    type PhysicalSettings = Extract<BackupConfig['settings'], { type: 'PHYSICAL' }>;

    const buildPhysicalConfig = (settings: Partial<PhysicalSettings> = {}): BackupConfig => ({
      cleanup: { pendingFileCount: 0, failedFileCount: 0 },
      databaseId: incrementalDatabase.id,
      storageId: null,
      isBackupsEnabled: true,
      encryption: 'NONE',
      settings: {
        type: 'PHYSICAL',
        fullBackupSchedule: { type: 'HOURLY' },
        incrementalBackupSchedule: { type: 'HOURLY' },
        retention: {
          policy: 'CHAINS_AND_FULL_BACKUPS',
          chainsCount: 3,
          fullBackups: gfsRetention,
        },
        walLagThresholdBytes: 0,
        notificationTriggers: ['BACKUP_FAILED'],
        ...settings,
      },
    });

    const renderPhysicalSection = async (
      config: BackupConfig,
      database: Database = incrementalDatabase,
    ) => {
      const submitted: BackupConfigInput[] = [];
      renderSavedDatabaseConfig(database, [
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, async ({ request }) => {
          submitted.push((await request.json()) as BackupConfigInput);

          return HttpResponse.json(config);
        }),
        ...declareBackupConfigRequests(),
      ]);
      await screen.findByText('Backups enabled');

      return { saveButton: await openBackupSection(), submitted };
    };

    const chooseOptionOf = async ({
      fieldLabel,
      optionText,
    }: {
      fieldLabel: string;
      optionText: string;
    }) => {
      fireEvent.mouseDown(
        screen.getByText(fieldLabel).parentElement!.querySelector('.ant-select-selector')!,
      );
      const option = await waitFor(() => {
        const matchingOption = Array.from(
          document.querySelectorAll('.ant-select-item-option'),
        ).find((element) => element.textContent === optionText);
        expect(matchingOption).toBeTruthy();
        return matchingOption!;
      });

      fireEvent.click(option);
    };

    const chooseFullBackupsPolicy = (optionText: string) =>
      chooseOptionOf({ fieldLabel: 'Full backups', optionText });
    const chooseRetention = (optionText: string) =>
      chooseOptionOf({ fieldLabel: 'Retention', optionText });

    it('disables Save again when an edit is reverted', async () => {
      const { saveButton } = await renderPhysicalSection(buildPhysicalConfig());

      setRetentionValue('Daily', '8');
      expect(saveButton).toHaveProperty('disabled', false);

      setRetentionValue('Daily', '7');
      expect(saveButton).toHaveProperty('disabled', true);
    });

    it('keeps Save disabled when a policy is tried and abandoned on a chains-only config', async () => {
      const { saveButton } = await renderPhysicalSection(
        buildPhysicalConfig({
          retention: { policy: 'CHAINS', chainsCount: 3, fullBackups: null },
        }),
      );

      await chooseRetention('Keep last N chains and full backups');
      expect(saveButton).toHaveProperty('disabled', false);

      await chooseRetention('Keep last N chains');
      expect(saveButton).toHaveProperty('disabled', true);
    });

    it('disables Save for all-zero GFS periods with backups off', async () => {
      const { saveButton } = await renderPhysicalSection(buildPhysicalConfig());

      fireEvent.click(screen.getByRole('switch'));
      setRetentionValue('Daily', '0');
      setRetentionValue('Weekly', '0');

      expect(saveButton).toHaveProperty('disabled', true);

      setRetentionValue('Yearly', '1');

      expect(saveButton).toHaveProperty('disabled', false);
    });

    it('shows the entered values after a switch of the full-backup policy and back', async () => {
      await renderPhysicalSection(buildPhysicalConfig());

      setRetentionValue('Daily', '9');
      await chooseFullBackupsPolicy('Count');
      setRetentionValue('Most recent full backups', '5');
      await chooseFullBackupsPolicy('GFS');

      expect(getRetentionInput('Daily').value).toBe('9');

      await chooseFullBackupsPolicy('Count');

      expect(getRetentionInput('Most recent full backups').value).toBe('5');
    });

    it('sends no full-backup retention for chains only', async () => {
      const config = buildPhysicalConfig();
      const { saveButton, submitted } = await renderPhysicalSection(config);

      await chooseRetention('Keep last N chains');
      fireEvent.click(saveButton);
      await waitFor(() => expect(submitted).toHaveLength(1));

      expect(submitted[0]).toEqual({
        isBackupsEnabled: true,
        storageId: null,
        encryption: 'NONE',
        settings: {
          ...config.settings,
          retention: { policy: 'CHAINS', chainsCount: 3, fullBackups: null },
        },
      });
    });

    it('sends the count and zero GFS periods for a last-N retention over stored GFS periods', async () => {
      const config = buildPhysicalConfig();
      const { saveButton, submitted } = await renderPhysicalSection(config);

      await chooseFullBackupsPolicy('Count');
      setRetentionValue('Most recent full backups', '5');
      fireEvent.click(saveButton);
      await waitFor(() => expect(submitted).toHaveLength(1));

      expect(submitted[0]).toEqual({
        isBackupsEnabled: true,
        storageId: null,
        encryption: 'NONE',
        settings: {
          ...config.settings,
          retention: {
            policy: 'CHAINS_AND_FULL_BACKUPS',
            chainsCount: 3,
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
        },
      });
    });

    it('sends the stored WAL lag threshold of a WAL streaming database unchanged', async () => {
      const { saveButton, submitted } = await renderPhysicalSection(
        buildPhysicalConfig({ walLagThresholdBytes: 4096 }),
        walStreamDatabase,
      );

      setRetentionValue('Daily', '8');
      fireEvent.click(saveButton);
      await waitFor(() => expect(submitted).toHaveLength(1));

      expect(submitted[0].settings).toMatchObject({ walLagThresholdBytes: 4096 });
    });
  });

  it('removes the advanced settings toggle when backups are turned off', async () => {
    await renderLogicalSection(buildLogicalConfig());

    fireEvent.click(screen.getAllByRole('switch')[0]);

    expect(screen.queryByText('Advanced settings')).toBeNull();
  });

  it('removes the open advanced settings and their toggle when backups are turned off', async () => {
    await renderLogicalSection(buildLogicalConfig({ isRetryIfFailed: false }));
    expect(screen.getByText('Retry backup if failed')).toBeTruthy();

    fireEvent.click(screen.getAllByRole('switch')[0]);

    expect(screen.queryByText('Retry backup if failed')).toBeNull();
    expect(screen.queryByText('Advanced settings')).toBeNull();
  });
});

describe('DatabaseConfigComponent action feedback', () => {
  const REFUSAL_MESSAGE = 'You do not have permission to do this.';
  const refusal = buildProblem('permission_denied', 403);

  const renderActions = (handlers: RequestHandler[] = []) => {
    const callbacks = {
      onDatabaseChanged: vi.fn(),
      onDatabaseDeleted: vi.fn(),
      onDatabaseCopied: vi.fn(),
      onDatabaseTransferred: vi.fn(),
    };
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const LanguageSwitch = () => {
      const { setLocale } = useLocale();

      return <button onClick={() => setLocale('ru')}>Switch language</button>;
    };

    mockServer.use(
      ...handlers,
      mockApi.get('/notifiers', 200, { items: [] }),
      mockApi.pending('get', '/backup-configs/{databaseId}').handler,
      mockApi.pending('get', '/storages').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    );

    const view = renderScreen(
      <>
        <LanguageSwitch />
        <DatabaseConfigComponent database={savedDatabase} isCanManageDBs {...callbacks} />
      </>,
    );

    return { ...view, ...callbacks, showAlert };
  };

  const confirmCopy = async () => {
    fireEvent.click(await screen.findByRole('button', { name: 'Copy' }));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
    );
  };

  const confirmDelete = async () => {
    fireEvent.click(await screen.findByRole('button', { name: 'Delete' }));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );
  };

  const startNameEdit = async (user: ReturnType<typeof renderScreen>['user']) => {
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Shop' }));

    const nameInput = screen.getByPlaceholderText('Enter name...');
    await user.clear(nameInput);
    await user.type(nameInput, 'Storefront');

    return nameInput as HTMLInputElement;
  };

  it('shows the reason on the screen when the connection test reports no connection', async () => {
    const { showAlert } = renderActions([
      mockApi.post('/databases/{databaseId}/test', 200, {
        isConnected: false,
        error: 'connection refused',
      }),
    ]);

    fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));

    expect(
      await screen.findByText('The database could not be reached: connection refused'),
    ).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('shows the reason and lets the test start again when the test request fails', async () => {
    const { showAlert } = renderActions([
      mockApi.post('/databases/{databaseId}/test', 403, refusal),
    ]);

    fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(
        screen.getByRole<HTMLButtonElement>('button', { name: /Test connection/ }).disabled,
      ).toBe(false),
    );

    mockServer.use(mockApi.post('/databases/{databaseId}/test', 200, { isConnected: true }));
    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));

    expect(await screen.findByText('Connection test successful!')).toBeTruthy();
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('shows a notice and reports the source database when a copy succeeds', async () => {
    const copiedDatabase = { ...savedDatabase, id: 'copied-id', name: 'Copy of Shop' };
    const { onDatabaseCopied, showAlert } = renderActions([
      mockApi.post('/databases/{databaseId}/copy', 201, copiedDatabase),
    ]);

    await confirmCopy();

    expect(await screen.findByText('Database copied successfully!')).toBeTruthy();
    expect(screen.getByText('"Copy of Shop" has been created successfully')).toBeTruthy();
    expect(onDatabaseCopied).toHaveBeenCalledWith(copiedDatabase, savedDatabase.id);
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('keeps the database and offers the copy again when a copy is refused', async () => {
    const { onDatabaseCopied } = renderActions([
      mockApi.post('/databases/{databaseId}/copy', 403, refusal),
    ]);

    await confirmCopy();

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(onDatabaseCopied).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole<HTMLButtonElement>('button', { name: /copy/i }).disabled).toBe(false),
    );
  });

  it('keeps the name editor open with the entered name when a rename is refused', async () => {
    const { user, onDatabaseChanged } = renderActions([
      mockApi.put('/databases/{databaseId}', 403, refusal),
    ]);
    const nameInput = await startNameEdit(user);

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(nameInput.value).toBe('Storefront');
    expect(screen.getByPlaceholderText('Enter name...')).toBe(nameInput);
    expect(onDatabaseChanged).not.toHaveBeenCalled();
  });

  it('ends the removing indicator and offers delete again when a delete is refused', async () => {
    const { onDatabaseDeleted } = renderActions([
      mockApi.delete('/databases/{databaseId}', 403, refusal),
    ]);

    await confirmDelete();

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(screen.queryByText('Removing...')).toBeNull();
    expect(onDatabaseDeleted).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole<HTMLButtonElement>('button', { name: /delete/i }).disabled).toBe(
        false,
      ),
    );
  });

  it('reports the deleted database when a delete succeeds', async () => {
    const { onDatabaseDeleted } = renderActions([
      mockApi.delete('/databases/{databaseId}', 204, undefined),
    ]);

    await confirmDelete();

    await waitFor(() => expect(onDatabaseDeleted).toHaveBeenCalledWith(savedDatabase.id));
  });

  it('translates a shown refusal reason when the language changes', async () => {
    renderActions([mockApi.post('/databases/{databaseId}/copy', 403, refusal)]);

    await confirmCopy();
    await screen.findByText(REFUSAL_MESSAGE);
    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

    await waitFor(() => expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull());
    expect(screen.getByRole('alert').textContent).toContain(
      DICTIONARIES.ru.errors.permissionDenied,
    );
  });

  it('translates a shown connection reason when the language changes', async () => {
    renderActions([
      mockApi.post('/databases/{databaseId}/test', 200, {
        isConnected: false,
        error: 'connection refused',
      }),
    ]);

    fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));
    await screen.findByText('The database could not be reached: connection refused');
    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

    await waitFor(() =>
      expect(
        screen.queryByText('The database could not be reached: connection refused'),
      ).toBeNull(),
    );
    expect(screen.getByRole('alert').textContent).toContain(
      DICTIONARIES.ru.databases.edit.connectionTestFailed.replace(
        '{{error}}',
        'connection refused',
      ),
    );
  });

  it('removes a shown reason when the user closes it', async () => {
    renderActions([mockApi.post('/databases/{databaseId}/copy', 403, refusal)]);

    await confirmCopy();
    await screen.findByText(REFUSAL_MESSAGE);
    fireEvent.click(within(screen.getByRole('alert')).getByRole('button', { name: 'Close' }));

    await waitFor(() => expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull());
  });

  it('removes a shown reason when a section edit starts', async () => {
    renderActions([mockApi.post('/databases/{databaseId}/copy', 403, refusal)]);

    await confirmCopy();
    await screen.findByText(REFUSAL_MESSAGE);
    fireEvent.click(screen.getByRole('button', { name: 'Edit Database settings' }));

    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('removes a reason that arrived during an edit when the connection save succeeds', async () => {
    const heldCopy = mockApi.pending('post', '/databases/{databaseId}/copy');
    renderActions([
      heldCopy.handler,
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      mockApi.put('/databases/{databaseId}', 200, savedDatabase),
    ]);

    await confirmCopy();
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Database settings' }));
    await screen.findByPlaceholderText(/host$/);
    await act(async () => {
      heldCopy.respond(HttpResponse.json(refusal, { status: 403 }));
    });
    await screen.findByText(REFUSAL_MESSAGE);

    fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull());
  });

  it('removes a shown reason when the next action of the database starts', async () => {
    const heldTest = mockApi.pending('post', '/databases/{databaseId}/test');
    renderActions([mockApi.post('/databases/{databaseId}/copy', 403, refusal), heldTest.handler]);

    await confirmCopy();
    await screen.findByText(REFUSAL_MESSAGE);
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));

    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
    heldTest.cancel();
  });
});
