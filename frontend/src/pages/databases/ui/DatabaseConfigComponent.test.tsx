import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, type RequestHandler, http } from 'msw';
import { type Mock, describe, expect, it, vi } from 'vitest';

import {
  type BackupConfig,
  type BackupConfigInput,
  backupConfigApi,
} from '../../../entities/backup-configs';
import {
  type Database,
  type DatabaseSettings,
  DatabaseType,
  PostgresSslMode,
  SshTunnelAuthType,
  createDefaultDatabaseSettings,
  databaseApi,
} from '../../../entities/databases';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { activateSession, invalidateSession } from '../../../shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { stubClipboard } from '../../../shared/lib/test/stubClipboard';
// @vitest-environment happy-dom
import { DatabaseConfigComponent } from './DatabaseConfigComponent';

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

const savedWithCertificate = buildSavedDatabase(DatabaseType.POSTGRES_LOGICAL, {
  sslMode: PostgresSslMode.Require,
  sslClientCert: 'stored-client-cert',
  sslRootCert: 'stored-root-cert',
} as Partial<DatabaseSettings>);

describe('DatabaseConfigComponent', () => {
  it('ignores pending configuration errors after the screen unmounts', async () => {
    const alert = vi.fn();
    vi.stubGlobal('alert', alert);
    const backupConfig = mockApi.pending('get', '/backup-configs/{databaseId}');
    const healthcheckConfig = mockApi.pending('get', '/healthcheck-configs/{databaseId}');
    const view = renderDatabaseConfig(false, [backupConfig.handler, healthcheckConfig.handler]);

    await waitFor(() =>
      expect(
        mockServer
          .getRequests()
          .filter((request) => /\/(backup-configs|healthcheck-configs)\//.test(request.url)),
      ).toHaveLength(2),
    );

    view.unmount();
    const requestsFinished = new Promise<void>((resolve) => {
      let completedRequests = 0;
      mockServer.events.on('request:end', ({ request }) => {
        if (
          /\/(backup-configs|healthcheck-configs)\//.test(request.url) &&
          ++completedRequests === 2
        )
          resolve();
      });
    });
    await act(async () => {
      backupConfig.cancel();
      healthcheckConfig.cancel();
      await requestsFinished;
    });

    expect(alert).not.toHaveBeenCalled();
  });

  it('shows no notifier or verification error after the screen unmounts', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const notifiers = mockApi.pending('get', '/notifiers');
    const verificationConfig = mockApi.pending('get', '/verification-configs/{databaseId}');
    mockServer.use(
      notifiers.handler,
      verificationConfig.handler,
      mockApi.pending('get', '/backup-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    );

    const view = renderScreen(
      <DatabaseConfigComponent
        database={{
          ...savedDatabase,
          settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
        }}
        onDatabaseChanged={() => {}}
        onDatabaseDeleted={() => {}}
        onDatabaseCopied={() => {}}
        onDatabaseTransferred={() => {}}
        isCanManageDBs={false}
      />,
    );
    await waitFor(() =>
      expect(
        mockServer
          .getRequests()
          .filter((request) => /\/(notifiers|verification-configs\/)/.test(request.url)),
      ).toHaveLength(2),
    );

    view.unmount();
    const requestsFinished = new Promise<void>((resolve) => {
      let completedRequests = 0;
      mockServer.events.on('request:end', ({ request }) => {
        if (/\/(notifiers|verification-configs\/)/.test(request.url) && ++completedRequests === 2)
          resolve();
      });
    });
    await act(async () => {
      notifiers.cancel();
      verificationConfig.cancel();
      await requestsFinished;
    });

    expect(showAlert).not.toHaveBeenCalled();
  });

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
    vi.stubGlobal('alert', vi.fn());
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

    const earlierLoadsFailed = new Promise<void>((resolve) => {
      let failedLoads = 0;
      mockServer.events.on('response:mocked', ({ response }) => {
        if (response.status === 500 && ++failedLoads === earlierBackupConfigLoads.length) resolve();
      });
    });
    await act(async () => {
      for (const earlierLoad of earlierBackupConfigLoads)
        earlierLoad.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 }));
      await earlierLoadsFailed;
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

  it('shows no failure of a physical backup configuration after the screen unmounts', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const configCall = vi.spyOn(backupConfigApi, 'getBackupConfig');
    const backupConfigLoad = mockApi.delayed({
      method: 'get',
      path: '/backup-configs/{databaseId}',
      status: 500,
      body: buildProblem('internal', 500),
    });
    mockServer.use(
      backupConfigLoad.handler,
      mockApi.get('/notifiers', 200, { items: [] }),
      mockApi.pending('get', '/storages').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/verification-configs/{databaseId}').handler,
    );

    const view = renderScreen(
      <DatabaseConfigComponent
        database={buildSavedDatabase(DatabaseType.POSTGRES_PHYSICAL)}
        onDatabaseChanged={() => {}}
        onDatabaseDeleted={() => {}}
        onDatabaseCopied={() => {}}
        onDatabaseTransferred={() => {}}
        isCanManageDBs
      />,
    );
    await waitFor(() => expect(configCall).toHaveBeenCalled());

    view.unmount();
    await releaseAndSettle(configCall, backupConfigLoad.release);

    expect(showAlert).not.toHaveBeenCalled();
  });

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
      expectRefusalShown: () => expect(screen.getByText(REFUSAL_REASON)).toBeTruthy(),
    },
    {
      kind: 'physical',
      database: physicalDatabase,
      config: physicalConfig,
      expectRefusalShown: (showAlert: Mock) =>
        expect(showAlert).toHaveBeenCalledWith(REFUSAL_REASON),
    },
  ];

  const declareBackupConfigRequests = () => [
    mockApi.get('/storage-switches', 200, { items: [] }),
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

  describe.each(BACKUP_EDITORS)(
    'of the $kind editor',
    ({ database, config, expectRefusalShown }) => {
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
            mockApi.put(
              '/backup-configs/{databaseId}',
              403,
              buildProblem('permission_denied', 403),
            ),
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

        expectRefusalShown(showAlert);
        expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', false);
        expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(
          isEnteredTriggerChecked,
        );
        expect(onDatabaseChanged).not.toHaveBeenCalled();
      });
    },
  );

  describe('when the logical section cannot load', () => {
    const openFailedSection = async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const configLoad = { isFailing: false };
      renderSavedDatabaseConfig(logicalDatabase, [
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
          configLoad.isFailing
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json(buildLogicalConfig()),
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

  it('shows one error and one retry when the settings tab cannot load a logical config', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    let isConfigFailing = true;
    renderSavedDatabaseConfig(logicalDatabase, [
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        isConfigFailing
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json(buildLogicalConfig()),
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
  });

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

  it('shows the reason with a retry when the configuration view cannot load its storage', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    let isStorageFailing = true;
    renderSavedDatabaseConfig(logicalDatabase, [
      mockApi.get('/backup-configs/{databaseId}', 200, {
        ...buildLogicalConfig(),
        storageId: 'archive-storage-id',
      }),
      http.get(`*${API_BASE_PATH}/storages/:storageId`, () =>
        isStorageFailing
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json({
              id: 'archive-storage-id',
              workspaceId: logicalDatabase.workspaceId,
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
  });

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

  describe('after a storage was cleared', () => {
    const renderConfigWithStoredStorage = (targetStorageId: string) => {
      let storedConfig: BackupConfig = { ...buildLogicalConfig(), storageId: 'old-storage' };
      const submitted: BackupConfigInput[] = [];
      const buildStorage = (id: string, name: string) => ({
        id,
        workspaceId: logicalDatabase.workspaceId,
        name,
        settings: { type: 'LOCAL' as const },
      });
      renderSavedDatabaseConfig(logicalDatabase, [
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
            buildStorage('old-storage', 'Previous archive'),
            buildStorage('target-storage', 'Archive'),
          ],
        }),
        http.get(`*${API_BASE_PATH}/storages/:storageId`, ({ params }) =>
          HttpResponse.json(buildStorage(String(params.storageId), 'Shown archive')),
        ),
        mockApi.get('/storage-switches', 200, { items: [] }),
        mockApi.get('/logical-backups', 200, {
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
          await screen.findByText(/Switching storage removes the old backup history immediately/),
        ).toBeTruthy();
        expect(submitted).toHaveLength(1);

        fireEvent.click(screen.getByRole('button', { name: 'Switch storage and remove history' }));

        await waitFor(() => expect(submitted).toHaveLength(2));
        expect(submitted[1].storageId).toBe(storageId);
      },
    );
  });

  it('shows an accepted storage switch in the configuration view after a save', async () => {
    const acceptedSwitch = {
      id: 'switch-id',
      databaseId: logicalDatabase.id,
      workspaceId: logicalDatabase.workspaceId,
      targetStorageId: 'target-storage',
      status: 'CLEANUP_PENDING' as const,
      createdAt: '2026-10-30T10:00:00Z',
      cleanupDeadline: '2026-10-31T10:00:00Z',
      total: 2,
      pending: 2,
      deleted: 0,
      remaining: 0,
    };
    let isSwitchAccepted = false;
    renderSavedDatabaseConfig(logicalDatabase, [
      mockApi.get('/backup-configs/{databaseId}', 200, buildLogicalConfig()),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => {
        isSwitchAccepted = true;
        return HttpResponse.json(acceptedSwitch, { status: 202 });
      }),
      http.get(`*${API_BASE_PATH}/storage-switches`, () =>
        HttpResponse.json({ items: isSwitchAccepted ? [acceptedSwitch] : [] }),
      ),
      mockApi.get('/storage-switches/{storageSwitchId}', 200, acceptedSwitch),
      mockApi.get('/logical-backups', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
    ]);
    await screen.findByText('Backups enabled');
    const saveButton = await openBackupSection();
    toggleSuccessTrigger();

    fireEvent.click(saveButton);

    expect(
      await screen.findByText('Storage switched. Old files are being cleaned up.'),
    ).toBeTruthy();
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
