// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useRef } from 'react';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  PostgresSslMode,
  SshTunnelAuthType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { type Restore, restoreApi } from '../../../../entities/restores';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
import { activateSession, invalidateSession } from '../../../../shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '../../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { stubClipboard } from '../../../../shared/lib/test/stubClipboard';
import type { LogicalBackup } from '../../model/LogicalBackup';
import { LogicalBackupsComponent } from '../logical-backups/LogicalBackupsComponent';
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

const tunneledPostgresqlDatabase: Database = {
  id: 'postgresql-database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: {
    ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
    version: '16',
    host: '10.0.0.5',
    port: 6432,
    username: 'backup',
    password: undefined,
    database: 'orders',
    sslMode: PostgresSslMode.VerifyFull,
    sslClientCert: 'source-client-cert',
    sslClientKey: 'source-client-key',
    sslRootCert: 'source-root-cert',
    sshTunnel: {
      isEnabled: true,
      host: 'bastion.example.com',
      port: 2222,
      username: 'tunnel-user',
      authType: SshTunnelAuthType.PASSWORD,
    },
  },
  notifierIds: [],
};

const createSourceDatabase = (type: 'MYSQL' | 'MARIADB' | 'MONGODB'): Database => ({
  id: `${type}-database-id`,
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: {
    ...createDefaultDatabaseSettings(type),
    host: 'db.example.com',
    port: 4000,
    username: 'backup',
    password: undefined,
    database: 'shop',
  },
  notifierIds: [],
});

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

const postgresqlBackup: LogicalBackup = {
  ...completedBackup,
  databaseId: tunneledPostgresqlDatabase.id,
};

const startedRestore: Restore = {
  id: 'restore-id',
  backupId: completedBackup.id,
  status: 'IN_PROGRESS',
  durationMs: 0,
  createdAt: '2026-10-27T11:00:00Z',
};

// The screen polls the list and shows what it answers, so a started restore has to stay listed.
const declareRestoreList = () => {
  const listedRestores: Restore[] = [];
  mockServer.use(
    http.get(`*${API_BASE_PATH}/restores`, () => HttpResponse.json({ items: listedRestores })),
  );

  return listedRestores;
};

const renderRestoreFlow = (database: Database, backup: LogicalBackup) =>
  renderScreen(<RestoresComponent database={database} backup={backup} />);

const renderBackupListWithRestore = () => {
  let backupListRequests = 0;
  mockServer.use(
    mockApi.get('/backup-configs/{databaseId}', 200, {
      cleanup: { pendingFileCount: 0, failedFileCount: 0 },
      databaseId: backedUpDatabase.id,
      isBackupsEnabled: true,
      storageId: 'storage-id',
      encryption: 'NONE',
      settings: {
        type: 'LOGICAL',
        schedule: { type: 'HOURLY' },
        retention: {
          policy: 'TIME_PERIOD',
          period: 'WEEK',
          count: 0,
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
    }),
    http.get(`*${API_BASE_PATH}/logical-backups`, () =>
      backupListRequests++ === 0
        ? HttpResponse.json({ items: [completedBackup] })
        : createPendingApiResponse().response,
    ),
    mockApi.get('/restores', 200, { items: [] }),
  );

  const ScrollingBackupList = () => {
    const scrollContainerRef = useRef<HTMLDivElement>(null);

    return (
      <div ref={scrollContainerRef}>
        <LogicalBackupsComponent
          database={backedUpDatabase}
          isCanManageDBs
          scrollContainerRef={scrollContainerRef}
        />
      </div>
    );
  };

  renderScreen(<ScrollingBackupList />);
};

const openRestoreDialog = async () => {
  const restoreIcon = await waitFor(() => {
    const icon = screen.getAllByRole('button', { name: 'Restore from backup' })[0];
    expect(icon).toBeTruthy();

    return icon!;
  });

  fireEvent.click(restoreIcon);
  await screen.findByRole('button', { name: 'Select database to restore to' });
};

const closeRestoreDialog = () => fireEvent.click(document.querySelector('.ant-modal-close')!);

const openTargetForm = async () =>
  fireEvent.click(await screen.findByRole('button', { name: 'Select database to restore to' }));

const enterTarget = async ({ password = 'target-secret' }: { password?: string } = {}) => {
  await openTargetForm();
  fireEvent.change(screen.getByPlaceholderText(/host$/), {
    target: { value: 'restore-target.example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'restorer' } });
  fireEvent.change(screen.getByPlaceholderText(/password$/), { target: { value: password } });
  fireEvent.change(screen.getByPlaceholderText(/database name$/), {
    target: { value: 'restored' },
  });
};

// happy-dom never finishes the exit animation of a loading icon, so a button that was loading
// keeps "loading" at the start of its accessible name.
const startEnteredRestore = () =>
  fireEvent.click(screen.getByRole('button', { name: /Restore to this DB$/ }));

const chooseRequireSslMode = async () => {
  fireEvent.mouseDown(
    screen.getByText('SSL mode').parentElement!.querySelector('.ant-select-selector')!,
  );
  fireEvent.click(
    await waitFor(() => {
      const requireOption = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
        (element) => /^Require/.test(element.textContent ?? ''),
      );
      expect(requireOption).toBeTruthy();
      return requireOption!;
    }),
  );
};

// A request that names the source database would let the server fill omitted settings from it.
const findRequestsNamingDatabase = async (requests: Request[], databaseId: string) => {
  const requestTexts = await Promise.all(
    requests.map(async (request) => `${request.url} ${await request.text()}`),
  );

  return requestTexts.filter((requestText) => requestText.includes(databaseId));
};

describe('Restore target composition', () => {
  it.each([
    {
      sourceDatabase: createSourceDatabase(DatabaseType.MYSQL),
      target: {
        type: DatabaseType.MYSQL,
        host: 'restore-target.example.com',
        port: 3306,
        username: 'restorer',
        password: 'target-secret',
        database: 'restored',
        isHttps: false,
        sshTunnel: createDefaultDatabaseSettings(DatabaseType.MYSQL).sshTunnel,
      },
    },
    {
      sourceDatabase: createSourceDatabase(DatabaseType.MARIADB),
      target: {
        type: DatabaseType.MARIADB,
        host: 'restore-target.example.com',
        port: 3306,
        username: 'restorer',
        password: 'target-secret',
        database: 'restored',
        isHttps: false,
        isSkipGaleraDisable: false,
        sshTunnel: createDefaultDatabaseSettings(DatabaseType.MARIADB).sshTunnel,
      },
    },
    {
      sourceDatabase: createSourceDatabase(DatabaseType.MONGODB),
      target: {
        type: DatabaseType.MONGODB,
        host: 'restore-target.example.com',
        port: 27017,
        username: 'restorer',
        password: 'target-secret',
        database: 'restored',
        authDatabase: '',
        isHttps: false,
        isSrv: false,
        isDirectConnection: false,
        cpuCount: 1,
        sshTunnel: createDefaultDatabaseSettings(DatabaseType.MONGODB).sshTunnel,
      },
    },
  ])(
    'starts the restore into the entered $target.type target without a connection test',
    async ({ sourceDatabase, target }) => {
      const restoreStarts: ApiComponents['schemas']['RestoreStart'][] = [];
      const listedRestores = declareRestoreList();
      mockServer.use(
        http.post(`*${API_BASE_PATH}/restores`, async ({ request }) => {
          restoreStarts.push((await request.json()) as ApiComponents['schemas']['RestoreStart']);
          listedRestores.push(startedRestore);

          return HttpResponse.json(startedRestore, { status: 202 });
        }),
      );
      renderRestoreFlow(sourceDatabase, { ...completedBackup, databaseId: sourceDatabase.id });

      await enterTarget();
      startEnteredRestore();

      expect(await screen.findByText('In progress')).toBeTruthy();
      expect(restoreStarts).toEqual([{ backupId: completedBackup.id, target }]);
    },
  );

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
  ])(
    'shows $code inline and keeps the entered target when the restore is refused',
    async (refusal) => {
      mockServer.use(
        mockApi.get('/restores', 200, { items: [] }),
        mockApi.post('/restores', refusal.status, buildProblem(refusal.code, refusal.status)),
      );
      renderRestoreFlow(backedUpDatabase, completedBackup);

      await enterTarget();
      startEnteredRestore();

      expect(await screen.findByText(refusal.shownText)).toBeTruthy();
      for (const enteredValue of [
        'restore-target.example.com',
        'restorer',
        'target-secret',
        'restored',
      ]) {
        expect(screen.getByDisplayValue(enteredValue)).toBeTruthy();
      }
      expect(screen.queryByText('In progress')).toBeNull();
    },
  );

  it('starts the restore with the corrected password after a refused start', async () => {
    const sentPasswords: string[] = [];
    const listedRestores = declareRestoreList();
    mockServer.use(
      http.post(`*${API_BASE_PATH}/restores`, async ({ request }) => {
        const restoreStart = (await request.json()) as ApiComponents['schemas']['RestoreStart'];
        sentPasswords.push(restoreStart.target.password);

        if (sentPasswords.length === 1) {
          return HttpResponse.json(buildProblem('database_connection_failed', 400), {
            status: 400,
          });
        }

        listedRestores.push(startedRestore);

        return HttpResponse.json(startedRestore, { status: 202 });
      }),
    );
    renderRestoreFlow(backedUpDatabase, completedBackup);

    await enterTarget({ password: 'wrong-secret' });
    startEnteredRestore();
    await screen.findByText('Could not connect to the database: database_connection_failed');
    fireEvent.change(screen.getByPlaceholderText(/password$/), {
      target: { value: 'target-secret' },
    });
    startEnteredRestore();

    expect(await screen.findByText('In progress')).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Restore to this DB$/ })).toBeNull();
    expect(sentPasswords).toEqual(['wrong-secret', 'target-secret']);
  });

  it('sends one restore start when the start is clicked twice while pending', async () => {
    const restoreStart = mockApi.delayed({
      method: 'post',
      path: '/restores',
      status: 202,
      body: startedRestore,
    });
    const listedRestores = declareRestoreList();
    mockServer.use(restoreStart.handler);
    renderRestoreFlow(backedUpDatabase, completedBackup);

    await enterTarget();
    const startButton = screen.getByRole('button', { name: 'Restore to this DB' });
    fireEvent.click(startButton);
    fireEvent.click(startButton);
    await waitFor(() => expect(startButton.className).toContain('ant-btn-loading'));
    listedRestores.push(startedRestore);
    restoreStart.release();

    expect(await screen.findByText('In progress')).toBeTruthy();
    expect(
      mockServer
        .getRequests()
        .filter((request) => request.method === 'POST' && request.url.endsWith('/restores')),
    ).toHaveLength(1);
  });

  it('opens the target form as a new PostgreSQL database without the source settings', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);

    await openTargetForm();
    fireEvent.click(screen.getByText('Advanced settings'));

    expect((screen.getByPlaceholderText('Enter PG port') as HTMLInputElement).value).toBe('5432');
    for (const placeholder of [
      'Enter PG host',
      'Enter PG username',
      'Enter PG password',
      'Enter PG database name',
    ]) {
      expect((screen.getByPlaceholderText(placeholder) as HTMLInputElement).value).toBe('');
    }
    const sshTunnelRow = screen.getByText('SSH tunnel').parentElement!;
    expect((within(sshTunnelRow).getByRole('checkbox') as HTMLInputElement).checked).toBe(false);
    expect(screen.queryByDisplayValue('bastion.example.com')).toBeNull();
    expect(screen.queryByText('SSH credentials')).toBeNull();
    expect(screen.queryByText('Client certificate')).toBeNull();
  });

  it('keeps the start disabled when the target password holds only spaces', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);

    await enterTarget({ password: '   ' });

    expect(
      (screen.getByRole('button', { name: 'Restore to this DB' }) as HTMLButtonElement).disabled,
    ).toBe(true);
  });

  it('keeps the start disabled without any request when the target password is empty', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);

    await enterTarget({ password: '' });
    const startButton = screen.getByRole('button', { name: 'Restore to this DB' });
    fireEvent.click(startButton);

    expect((startButton as HTMLButtonElement).disabled).toBe(true);
    expect(mockServer.getRequests().filter((request) => request.method === 'POST')).toEqual([]);
  });

  it('starts the restore with only entered values and without the member-only connection test', async () => {
    const restoreStarts: ApiComponents['schemas']['RestoreStart'][] = [];
    const listedRestores = declareRestoreList();
    mockServer.use(
      http.post(`*${API_BASE_PATH}/restores`, async ({ request }) => {
        restoreStarts.push((await request.json()) as ApiComponents['schemas']['RestoreStart']);
        listedRestores.push(startedRestore);

        return HttpResponse.json(startedRestore, { status: 202 });
      }),
    );
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);

    await enterTarget();
    startEnteredRestore();

    expect(await screen.findByText('In progress')).toBeTruthy();
    expect(restoreStarts).toEqual([
      {
        backupId: postgresqlBackup.id,
        target: {
          type: 'POSTGRES_LOGICAL',
          host: 'restore-target.example.com',
          port: 5432,
          username: 'restorer',
          password: 'target-secret',
          database: 'restored',
          sslMode: PostgresSslMode.Disable,
          sshTunnel: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL).sshTunnel,
          cpuCount: 1,
          isExcludeExtensions: false,
          isRestoreOwnership: false,
          isRestorePrivileges: false,
        },
      },
    ]);
    const sentRequests = mockServer.getRequests();
    expect(sentRequests.filter((request) => request.url.includes('/database-tests'))).toEqual([]);
    expect(await findRequestsNamingDatabase(sentRequests, tunneledPostgresqlDatabase.id)).toEqual(
      [],
    );
  });

  describe.each([
    {
      variant: 'PostgreSQL',
      type: DatabaseType.POSTGRES_LOGICAL,
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
  ] as const)('target form of $variant', ({ type, connectionString }) => {
    const buildSource = (): Database => ({
      id: 'source-id',
      workspaceId: 'workspace-id',
      name: 'Source',
      settings: createDefaultDatabaseSettings(type),
      notifierIds: [],
    });

    it('requires the SSH credentials once a tunnel is enabled', async () => {
      mockServer.use(mockApi.get('/restores', 200, { items: [] }));
      renderRestoreFlow(buildSource(), completedBackup);
      await enterTarget();

      if (!screen.queryByText('SSH tunnel')) fireEvent.click(screen.getByText('Advanced settings'));

      fireEvent.click(within(screen.getByText('SSH tunnel').parentElement!).getByRole('checkbox'));

      expect(
        (screen.getByRole('button', { name: /Restore to this DB$/ }) as HTMLButtonElement).disabled,
      ).toBe(true);
    });

    it('fills the target from an imported connection string', async () => {
      mockServer.use(mockApi.get('/restores', 200, { items: [] }));
      renderRestoreFlow(buildSource(), completedBackup);
      stubClipboard(async () => connectionString);
      await openTargetForm();

      fireEvent.click(screen.getByText('Parse from clipboard'));

      await waitFor(() =>
        expect((screen.getByPlaceholderText(/host$/) as HTMLInputElement).value).toBe(
          'imported.example.com',
        ),
      );
      expect((screen.getByPlaceholderText(/password$/) as HTMLInputElement).value).toBe(
        'imported-secret',
      );
    });

    it('offers no exclusion setting a restore ignores', async () => {
      mockServer.use(mockApi.get('/restores', 200, { items: [] }));
      renderRestoreFlow(buildSource(), completedBackup);
      await openTargetForm();

      fireEvent.click(screen.getByText('Advanced settings'));

      for (const backupOnlyLabel of [
        'Exclude tables',
        'Exclude events',
        'Exclude collections',
        'Exclude whole tables',
        'Exclude table data',
      ]) {
        expect(screen.queryByText(backupOnlyLabel)).toBeNull();
      }

      if (type === DatabaseType.MARIADB)
        expect(screen.getByText('Galera replication')).toBeTruthy();
    });
  });

  it('derives the SSL mode of a target from its host until the user chooses one', async () => {
    mockServer.use(mockApi.get('/restores', 200, { items: [] }));
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);
    await openTargetForm();
    const getSelectedSslMode = () =>
      screen.getByText('SSL mode').parentElement!.querySelector('.ant-select-selection-item')!
        .textContent;

    fireEvent.change(screen.getByPlaceholderText(/host$/), {
      target: { value: 'https://target.example.com' },
    });

    expect(getSelectedSslMode()).toMatch(/^Require/);

    fireEvent.change(screen.getByPlaceholderText(/host$/), { target: { value: 'localhost' } });

    expect(getSelectedSslMode()).toMatch(/^Disable/);
  });

  it('shows a typed certificate as typed after reopening a target', async () => {
    mockServer.use(
      mockApi.get('/restores', 200, { items: [] }),
      mockApi.post('/restores', 202, { ...startedRestore, status: 'FAILED' }),
    );
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);
    await enterTarget();
    await chooseRequireSslMode();
    fireEvent.click(screen.getByText('Advanced settings'));
    fireEvent.change(screen.getAllByPlaceholderText('-----BEGIN CERTIFICATE-----')[0], {
      target: { value: 'typed-client-cert' },
    });
    startEnteredRestore();
    await openTargetForm();

    expect(await screen.findByDisplayValue('typed-client-cert')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Replace' })).toBeNull();
  });

  it('reopens with the target and the restore options after a failed start of the previous target', async () => {
    const restoreStarts: ApiComponents['schemas']['RestoreStart'][] = [];
    mockServer.use(
      mockApi.get('/restores', 200, { items: [] }),
      http.post(`*${API_BASE_PATH}/restores`, async ({ request }) => {
        restoreStarts.push((await request.json()) as ApiComponents['schemas']['RestoreStart']);

        return HttpResponse.json({ ...startedRestore, status: 'FAILED' }, { status: 202 });
      }),
    );
    renderRestoreFlow(tunneledPostgresqlDatabase, postgresqlBackup);

    await enterTarget();
    fireEvent.click(screen.getByText('Advanced settings'));
    fireEvent.click(
      within(screen.getByText('Exclude extensions').parentElement!).getByRole('checkbox'),
    );
    startEnteredRestore();

    await openTargetForm();

    expect(restoreStarts[0].target).toMatchObject({ isExcludeExtensions: true });
    expect(screen.getByDisplayValue('restore-target.example.com') as HTMLInputElement).toBeTruthy();
    const excludeExtensionsRow = screen.getByText('Exclude extensions').parentElement!;
    expect((within(excludeExtensionsRow).getByRole('checkbox') as HTMLInputElement).checked).toBe(
      true,
    );
  });

  it('discards the target when the dialog closes and opens again', async () => {
    renderBackupListWithRestore();
    await openRestoreDialog();
    await enterTarget();

    closeRestoreDialog();
    await waitFor(() => expect(screen.queryByPlaceholderText(/host$/)).toBeNull());
    await openRestoreDialog();
    await openTargetForm();

    expect((screen.getByPlaceholderText(/host$/) as HTMLInputElement).value).toBe('');
    expect((screen.getByPlaceholderText(/password$/) as HTMLInputElement).value).toBe('');
  });

  it('shows nothing when a start answers after the dialog closed', async () => {
    const startCall = vi.spyOn(restoreApi, 'startRestore');
    const restoreStart = mockApi.delayed({
      method: 'post',
      path: '/restores',
      status: 400,
      body: buildProblem('database_connection_failed', 400),
    });
    renderBackupListWithRestore();
    mockServer.use(restoreStart.handler);

    await openRestoreDialog();
    await enterTarget();
    startEnteredRestore();
    await waitFor(() => expect(startCall).toHaveBeenCalled());

    closeRestoreDialog();
    await waitFor(() => expect(screen.queryByPlaceholderText(/host$/)).toBeNull());
    await releaseAndSettle(startCall, restoreStart.release);

    expect(screen.queryByPlaceholderText(/host$/)).toBeNull();
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.queryByText(/Could not connect to the database/)).toBeNull();
  });

  it('shows nothing when a start answers after the session changed', async () => {
    activateSession({ token: 'first-token', userId: 'first-user' });
    const startCall = vi.spyOn(restoreApi, 'startRestore');
    const restoreStart = mockApi.delayed({
      method: 'post',
      path: '/restores',
      status: 202,
      body: startedRestore,
    });
    mockServer.use(mockApi.get('/restores', 200, { items: [] }), restoreStart.handler);
    renderRestoreFlow(backedUpDatabase, completedBackup);

    await enterTarget();
    startEnteredRestore();
    await waitFor(() =>
      expect(mockServer.getRequests().filter((request) => request.method === 'POST')).toHaveLength(
        1,
      ),
    );

    act(() => {
      invalidateSession();
    });
    await releaseAndSettle(startCall, restoreStart.release);

    expect(screen.queryByText('In progress')).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('keeps every entered value when a start is refused as not found', async () => {
    declareRestoreList();
    mockServer.use(mockApi.post('/restores', 404, buildProblem('not_found', 404)));
    renderRestoreFlow(backedUpDatabase, completedBackup);

    await enterTarget();
    startEnteredRestore();

    expect(
      await screen.findByText('The requested item was not found. It may have been deleted.'),
    ).toBeTruthy();
    expect((screen.getByPlaceholderText(/host$/) as HTMLInputElement).value).toBe(
      'restore-target.example.com',
    );
    expect((screen.getByPlaceholderText(/username$/) as HTMLInputElement).value).toBe('restorer');
    expect((screen.getByPlaceholderText(/password$/) as HTMLInputElement).value).toBe(
      'target-secret',
    );
    expect((screen.getByPlaceholderText(/database name$/) as HTMLInputElement).value).toBe(
      'restored',
    );
    expect(screen.getByRole('button', { name: /Restore to this DB$/ })).toBeTruthy();
  });

  it('shows a started restore at once while the list does not name it yet', async () => {
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/restores`, () =>
        ++listRequests === 1
          ? HttpResponse.json({ items: [] })
          : createPendingApiResponse().response,
      ),
      mockApi.post('/restores', 202, startedRestore),
    );
    renderRestoreFlow(backedUpDatabase, completedBackup);

    await enterTarget();
    startEnteredRestore();

    expect(await screen.findByText('In progress')).toBeTruthy();
  });
});
