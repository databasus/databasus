import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, type RequestHandler, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { BackupConfig } from '../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  type HealthStatus,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import type { Storage } from '../../../entities/storages';
import type { Workspace } from '../../../entities/workspaces';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { waitForCondition } from '../../../shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
// @vitest-environment happy-dom
import { DatabasesComponent } from './DatabasesComponent';

dayjs.extend(utc);

const mockServer = setUpMockApi();

const workspace: Workspace = {
  id: 'workspace-id',
  name: 'Production',
  createdAt: '2026-01-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_VIEWER',
};

const buildDatabase = (
  name: string,
  healthStatus: HealthStatus = 'AVAILABLE',
  type: DatabaseType = DatabaseType.POSTGRES_LOGICAL,
): Database => ({
  id: `${name}-id`,
  workspaceId: workspace.id,
  name,
  settings: {
    ...createDefaultDatabaseSettings(type),
    host: 'db.example.com',
    username: 'backup',
    database: 'shop',
  } as Database['settings'],
  notifierIds: [],
  healthStatus,
});

const buildStorage = (name: string): Storage => ({
  id: `${name}-id`,
  workspaceId: workspace.id,
  name,
  settings: { type: 'LOCAL' },
});

const buildBackupConfig = (database: Database, storage: Storage): BackupConfig => ({
  databaseId: database.id,
  isBackupsEnabled: true,
  storageId: storage.id,
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'DAILY', timeOfDay: '04:00' },
    retention: {
      policy: 'COUNT',
      period: 'MONTH',
      count: 10,
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
});

const SELECTED_DATABASE_PREFERENCE_KEY = `selectedDatabaseId_${workspace.id}`;

const declarePendingDetailRequests = () =>
  mockServer.use(
    mockApi.get('/storage-switches', 200, { items: [] }),
    mockApi.pending('get', '/databases/{databaseId}').handler,
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/logical-backups').handler,
    mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    mockApi.pending('get', '/healthcheck-attempts').handler,
    mockApi.pending('get', '/notifiers').handler,
    mockApi.pending('get', '/storages/{storageId}').handler,
  );

const declareListResponses = (...listedDatabasePages: Database[][]) => {
  let listRequests = 0;
  mockServer.use(
    http.get(`*${API_BASE_PATH}/databases`, () => {
      const listedDatabases =
        listedDatabasePages[Math.min(listRequests++, listedDatabasePages.length - 1)];
      return HttpResponse.json({ items: listedDatabases });
    }),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
  );
};

const renderDatabasesScreen = (isCanManageDBs = true) =>
  renderScreen(<DatabasesComponent workspace={workspace} isCanManageDBs={isCanManageDBs} />);

const EDIT_ICON_SELECTOR = 'img[src="/icons/pen-gray.svg"]';

const declareSettingsRequests = () => [
  mockApi.pending('get', '/backup-configs/{databaseId}').handler,
  mockApi.pending('get', '/logical-backups').handler,
  mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
  mockApi.pending('get', '/healthcheck-attempts').handler,
  mockApi.pending('get', '/verification-configs/{databaseId}').handler,
  mockApi.get('/notifiers', 200, { items: [] }),
];

const openSettingsSection = (sectionTitle: HTMLElement) =>
  fireEvent.click(sectionTitle.parentElement!.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);

const countRequests = (pathSuffix: string, method = 'GET') =>
  mockServer
    .getRequests()
    .filter(
      (request) => request.method === method && new URL(request.url).pathname.endsWith(pathSuffix),
    ).length;

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

const chooseSelectOption = async (combobox: HTMLElement, optionLabel: string) => {
  fireEvent.mouseDown(combobox);
  const option = await waitFor(() => {
    const matchingOption = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
      (candidate) => candidate.textContent === optionLabel,
    );

    expect(matchingOption).toBeTruthy();

    return matchingOption as HTMLElement;
  });

  fireEvent.click(option);
};

const transferTargetWorkspace: Workspace = {
  ...workspace,
  id: 'target-workspace-id',
  name: 'Staging',
};

const transferTargetStorage = {
  ...buildStorage('Staging disk'),
  workspaceId: transferTargetWorkspace.id,
};

const declareTransferRequests = (...listedDatabasePages: Database[][]) => {
  let databaseListRequests = 0;
  mockServer.use(
    http.get(`*${API_BASE_PATH}/databases`, () =>
      HttpResponse.json({
        items:
          listedDatabasePages[Math.min(databaseListRequests++, listedDatabasePages.length - 1)],
      }),
    ),
    mockApi.get('/backup-configs', 200, { items: [] }),
    http.get(`*${API_BASE_PATH}/storages`, ({ request }) =>
      HttpResponse.json({
        items:
          new URL(request.url).searchParams.get('workspaceId') === transferTargetWorkspace.id
            ? [transferTargetStorage]
            : [],
      }),
    ),
    mockApi.get('/storage-switches', 200, { items: [] }),
    mockApi.get('/workspaces', 200, { items: [workspace, transferTargetWorkspace] }),
    http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
      HttpResponse.json(
        listedDatabasePages[0].find((database) => database.id === params.databaseId),
      ),
    ),
    ...declareSettingsRequests(),
  );
};

const transferSelectedDatabase = async (user: ReturnType<typeof renderScreen>['user']) => {
  await user.click(await screen.findByText('Config'));
  await user.click(await screen.findByRole('button', { name: 'arrow-right' }));
  await chooseSelectOption(await screen.findByRole('combobox'), transferTargetWorkspace.name);
  await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));
  await chooseSelectOption(screen.getAllByRole('combobox')[1], transferTargetStorage.name);
  await user.click(screen.getByRole('button', { name: 'Transfer' }));
};

const findDatabaseCard = (name: string) =>
  screen.findByRole('button', { name: new RegExp(`^${name}`) });

describe('DatabasesComponent', () => {
  it('lists unavailable databases first when the server lists them after available ones', async () => {
    mockServer.use(
      mockApi.get('/databases', 200, {
        items: [
          buildDatabase('Alpha', 'AVAILABLE'),
          buildDatabase('Bravo', 'UNAVAILABLE'),
          buildDatabase('Charlie', 'AVAILABLE'),
          buildDatabase('Delta', 'UNAVAILABLE'),
        ],
      }),
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/backup-configs', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
      mockApi.pending('get', '/databases/{databaseId}').handler,
      mockApi.pending('get', '/backup-configs/{databaseId}').handler,
      mockApi.pending('get', '/logical-backups').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/notifiers').handler,
      mockApi.pending('get', '/storages/{storageId}').handler,
    );

    renderScreen(<DatabasesComponent workspace={workspace} isCanManageDBs />);

    await screen.findByText('Alpha');
    const listedNames = screen
      .getAllByText(/^(Alpha|Bravo|Charlie|Delta)$/)
      .map((element) => element.textContent);

    expect(listedNames).toEqual(['Bravo', 'Delta', 'Alpha', 'Charlie']);
  });

  it('shows each card its storage name from one backup config list request', async () => {
    const ordersDatabase = buildDatabase('Orders');
    const usersDatabase = buildDatabase('Users');
    const draftDatabase = buildDatabase('Drafts');
    const nightlyStorage = buildStorage('Nightly disk');
    const archiveStorage = buildStorage('Archive disk');
    const backupConfigListQueries: URLSearchParams[] = [];
    const workspaceBackupConfigPage: ApiComponents['schemas']['BackupConfigPage'] = {
      items: [
        buildBackupConfig(ordersDatabase, nightlyStorage),
        buildBackupConfig(usersDatabase, archiveStorage),
      ],
    };

    mockServer.use(
      mockApi.get('/databases', 200, { items: [ordersDatabase, usersDatabase, draftDatabase] }),
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [nightlyStorage, archiveStorage] }),
      http.get(`*${API_BASE_PATH}/backup-configs`, ({ request }) => {
        backupConfigListQueries.push(new URL(request.url).searchParams);
        return HttpResponse.json(workspaceBackupConfigPage);
      }),
      mockApi.pending('get', '/databases/{databaseId}').handler,
      mockApi.pending('get', '/backup-configs/{databaseId}').handler,
      mockApi.pending('get', '/logical-backups').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/notifiers').handler,
      mockApi.pending('get', '/storages/{storageId}').handler,
    );

    renderScreen(<DatabasesComponent workspace={workspace} isCanManageDBs />);

    expect(await screen.findByText('Nightly disk')).toBeTruthy();
    expect(screen.getByText('Archive disk')).toBeTruthy();
    expect(backupConfigListQueries).toHaveLength(1);
    expect(backupConfigListQueries[0].get('workspaceId')).toBe(workspace.id);
    expect(backupConfigListQueries[0].has('storageId')).toBe(false);
  });

  it('selects the remembered database on desktop', async () => {
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, 'Bravo-id');
    declareListResponses([buildDatabase('Alpha'), buildDatabase('Bravo')]);
    declarePendingDetailRequests();

    renderDatabasesScreen();

    expect((await findDatabaseCard('Bravo')).getAttribute('aria-pressed')).toBe('true');
    expect((await findDatabaseCard('Alpha')).getAttribute('aria-pressed')).toBe('false');
  });

  it('shows the list without a selected database on mobile', async () => {
    vi.stubGlobal('innerWidth', 500);
    declareListResponses([buildDatabase('Alpha'), buildDatabase('Bravo')]);
    declarePendingDetailRequests();

    renderDatabasesScreen();

    expect((await findDatabaseCard('Alpha')).getAttribute('aria-pressed')).toBe('false');
    expect((await findDatabaseCard('Bravo')).getAttribute('aria-pressed')).toBe('false');
    expect(
      mockServer
        .getRequests()
        .filter((request) => /\/databases\/[^/]+$/.test(new URL(request.url).pathname)),
    ).toHaveLength(0);
  });

  it('selects the first remaining database after the selected one is deleted', async () => {
    const alphaDatabase = buildDatabase('Alpha');
    const bravoDatabase = buildDatabase('Bravo');
    const charlieDatabase = buildDatabase('Charlie');
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, bravoDatabase.id);
    declareListResponses(
      [alphaDatabase, bravoDatabase, charlieDatabase],
      [alphaDatabase, charlieDatabase],
    );

    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        HttpResponse.json(params.databaseId === bravoDatabase.id ? bravoDatabase : alphaDatabase),
      ),
      mockApi.delete('/databases/{databaseId}', 204, undefined),
      mockApi.pending('get', '/backup-configs/{databaseId}').handler,
      mockApi.pending('get', '/logical-backups').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      mockApi.pending('get', '/notifiers').handler,
    );
    const { user } = renderDatabasesScreen();

    await user.click(await screen.findByText('Config'));
    await user.click(await screen.findByRole('button', { name: 'delete' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );

    await waitForCondition(async () =>
      expect((await findDatabaseCard('Alpha')).getAttribute('aria-pressed')).toBe('true'),
    );
    expect(screen.queryByRole('button', { name: /^Bravo/ })).toBeNull();
  });

  it('offers a retry instead of an empty list when the first list load fails', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, () =>
        ++listRequests === 1
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json({ items: [buildDatabase('Alpha')] }),
      ),
      mockApi.get('/backup-configs', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
    );
    declarePendingDetailRequests();
    const { user } = renderDatabasesScreen();

    await user.click(await screen.findByRole('button', { name: 'Retry' }));

    expect(await findDatabaseCard('Alpha')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('keeps the list without feedback when a background refresh fails', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, () =>
        ++listRequests === 1
          ? HttpResponse.json({ items: [buildDatabase('Alpha')] })
          : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
      ),
      mockApi.get('/backup-configs', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
    );
    declarePendingDetailRequests();

    renderDatabasesScreen();
    await findDatabaseCard('Alpha');
    const refreshFailed = new Promise<void>((resolve) =>
      mockServer.events.on('response:mocked', ({ request, response }) => {
        if (new URL(request.url).pathname.endsWith('/databases') && response.status === 500)
          resolve();
      }),
    );

    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
      await refreshFailed;
    });

    expect((await findDatabaseCard('Alpha')).getAttribute('aria-pressed')).toBe('true');
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('keeps showing the newly selected database when the previous one responds last', async () => {
    const alphaDatabase = buildDatabase('Alpha');
    const bravoDatabase = { ...buildDatabase('Bravo'), name: 'Bravo' };
    const pendingAlpha = createPendingApiResponse();
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, alphaDatabase.id);
    declareListResponses([alphaDatabase, bravoDatabase]);

    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        params.databaseId === alphaDatabase.id
          ? pendingAlpha.response
          : HttpResponse.json(bravoDatabase),
      ),
      mockApi.pending('get', '/backup-configs/{databaseId}').handler,
      mockApi.pending('get', '/logical-backups').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      mockApi.pending('get', '/notifiers').handler,
    );
    const { user } = renderDatabasesScreen();
    await waitForCondition(() =>
      expect(
        mockServer
          .getRequests()
          .some((request) => request.url.endsWith(`/databases/${alphaDatabase.id}`)),
      ).toBe(true),
    );

    await user.click(await findDatabaseCard('Bravo'));
    await user.click(await screen.findByText('Config'));
    const alphaProcessed = new Promise<void>((resolve) =>
      mockServer.events.on('response:mocked', ({ request }) => {
        if (request.url.endsWith(`/databases/${alphaDatabase.id}`)) resolve();
      }),
    );
    await act(async () => {
      pendingAlpha.respond(HttpResponse.json(alphaDatabase));
      await alphaProcessed;
    });

    expect((await findDatabaseCard('Bravo')).getAttribute('aria-pressed')).toBe('true');
    expect(screen.getAllByText('Bravo')).toHaveLength(2);
    expect(screen.getAllByText('Alpha')).toHaveLength(1);
  });

  it('renames a database in its settings and its card while keeping the selection', async () => {
    const shopDatabase = buildDatabase('Shop');
    const renamedDatabase = { ...shopDatabase, name: 'Storefront' };
    declareListResponses([shopDatabase], [renamedDatabase]);
    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.put('/databases/{databaseId}', 200, renamedDatabase),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    const nameTitle = await screen.findByText('Shop', { selector: 'div.text-xl' });

    fireEvent.click(nameTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);
    const nameInput = screen.getByPlaceholderText('Enter name...');
    await user.clear(nameInput);
    await user.type(nameInput, 'Storefront');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Storefront', { selector: 'div.text-xl' })).toBeTruthy();
    expect((await findDatabaseCard('Storefront')).getAttribute('aria-pressed')).toBe('true');
    expect(screen.queryByPlaceholderText('Enter name...')).toBeNull();
  });

  it('keeps the settings tab and the list after the connection settings are saved', async () => {
    const shopDatabase = buildDatabase('Shop', 'AVAILABLE', DatabaseType.MYSQL);
    declareListResponses([shopDatabase]);
    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.put('/databases/{databaseId}', 200, shopDatabase),
      mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    openSettingsSection(await screen.findByText('Database settings'));

    await user.click(await screen.findByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByPlaceholderText(/host$/)).toBeNull());
    expect(screen.getByText('Database settings')).toBeTruthy();
    expect(await findDatabaseCard('Shop')).toBeTruthy();
    expect(countRequests(`/databases/${shopDatabase.id}`)).toBe(1);
  });

  it('keeps the connection section open with the entered values when saving is refused', async () => {
    vi.stubGlobal('alert', vi.fn());
    const shopDatabase = buildDatabase('Shop', 'AVAILABLE', DatabaseType.MYSQL);
    declareListResponses([shopDatabase]);
    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.put('/databases/{databaseId}', 403, buildProblem('permission_denied', 403)),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    openSettingsSection(await screen.findByText('Database settings'));
    const hostInput = await screen.findByPlaceholderText(/host$/);

    await user.clear(hostInput);
    await user.type(hostInput, 'replica.example.com');
    await user.click(await screen.findByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(countRequests(`/databases/${shopDatabase.id}`, 'PUT')).toBe(1));
    expect(screen.getByDisplayValue('replica.example.com')).toBeTruthy();
  });

  describe('a connection saved on the path of the limited user suggestion', () => {
    const SAVED_HOST = 'saved.example.com';

    const saveWithLimitedUserCheck = async (limitedUserCheck: RequestHandler) => {
      const shopDatabase = buildDatabase('Shop', 'AVAILABLE', DatabaseType.MYSQL);
      const savedDatabase = {
        ...shopDatabase,
        name: 'Renamed Shop',
        settings: { ...shopDatabase.settings, host: SAVED_HOST },
      } as Database;
      declareListResponses([shopDatabase], [savedDatabase]);
      mockServer.use(
        mockApi.get('/storage-switches', 200, { items: [] }),
        mockApi.get('/databases/{databaseId}', 200, shopDatabase),
        mockApi.post('/database-tests', 200, { isConnected: true }),
        mockApi.put('/databases/{databaseId}', 200, savedDatabase),
        limitedUserCheck,
        ...declareSettingsRequests(),
      );

      const { user } = renderDatabasesScreen();
      await user.click(await screen.findByText('Config'));
      openSettingsSection(await screen.findByText('Database settings'));
      await user.click(await screen.findByRole('button', { name: 'Test connection' }));
      await user.click(await screen.findByRole('button', { name: 'Save' }));

      return { user };
    };

    const advisingCheck = () =>
      mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: ['ALL'] });

    it('shows the saved values on the card and after Cancel when the suggestion is closed', async () => {
      const { user } = await saveWithLimitedUserCheck(advisingCheck());
      await screen.findByText('Create a read-only user for Databasus?');

      fireEvent.click(document.querySelector('.ant-modal-close')!);

      expect(await findDatabaseCard('Renamed Shop')).toBeTruthy();
      await user.click(await screen.findByRole('button', { name: 'Cancel' }));
      expect(await screen.findByText(SAVED_HOST)).toBeTruthy();
    });

    it('shows the saved values when the suggestion is skipped', async () => {
      const { user } = await saveWithLimitedUserCheck(advisingCheck());

      await user.click(await screen.findByRole('button', { name: 'Skip' }));
      await user.click(await screen.findByRole('button', { name: 'Yes, I accept risks' }));

      expect(await findDatabaseCard('Renamed Shop')).toBeTruthy();
      expect(await screen.findByText(SAVED_HOST)).toBeTruthy();
    });

    it('shows the reason and the saved values when the check fails', async () => {
      await saveWithLimitedUserCheck(
        mockApi.post('/limited-user-checks', 403, buildProblem('permission_denied', 403)),
      );

      expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
      expect(await findDatabaseCard('Renamed Shop')).toBeTruthy();
      expect(await screen.findByText(SAVED_HOST)).toBeTruthy();
    });
  });

  it('shows the newly chosen storage on the card after the backup configuration is saved', async () => {
    const ordersDatabase = buildDatabase('Orders');
    const archiveStorage = buildStorage('Archive');
    const backupConfig = { ...buildBackupConfig(ordersDatabase, archiveStorage), storageId: null };
    let storedBackupConfig: BackupConfig = backupConfig;

    mockServer.use(
      mockApi.get('/databases', 200, { items: [ordersDatabase] }),
      http.get(`*${API_BASE_PATH}/backup-configs`, () =>
        HttpResponse.json({ items: [storedBackupConfig] }),
      ),
      mockApi.get('/storages', 200, { items: [archiveStorage] }),
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/databases/{databaseId}', 200, ordersDatabase),
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        HttpResponse.json(storedBackupConfig),
      ),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => {
        storedBackupConfig = { ...backupConfig, storageId: archiveStorage.id };
        return HttpResponse.json(storedBackupConfig);
      }),
      mockApi.get('/logical-backups', 200, { items: [] }),
      mockApi.get('/storages/{storageId}', 200, archiveStorage),
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      mockApi.get('/notifiers', 200, { items: [] }),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    openSettingsSection(await screen.findByText('Backup config'));

    await chooseStorage('Archive');
    await user.click(await screen.findByRole('button', { name: /Save$/ }));

    expect(await within(await findDatabaseCard('Orders')).findByText('Archive')).toBeTruthy();
  });

  it('keeps the settings tab and the list after the notifier settings are saved', async () => {
    const shopDatabase = buildDatabase('Shop', 'AVAILABLE', DatabaseType.MYSQL);
    const alertsNotifier = {
      id: 'alerts-id',
      workspaceId: workspace.id,
      name: 'Alerts',
      lastSendError: null,
      settings: { type: 'DISCORD' as const },
    };
    declareListResponses([shopDatabase]);
    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.put('/databases/{databaseId}', 200, {
        ...shopDatabase,
        notifierIds: [alertsNotifier.id],
      }),
      mockApi.get('/notifiers', 200, { items: [alertsNotifier] }),
      ...declareSettingsRequests().slice(0, -1),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    openSettingsSection(await screen.findByText('Notifiers settings'));

    await chooseSelectOption(await screen.findByRole('combobox', { name: 'Notifiers' }), 'Alerts');
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByRole('combobox', { name: 'Notifiers' })).toBeNull());
    expect(await screen.findByText(/Alerts/)).toBeTruthy();
    expect(screen.getByText('Notifiers settings')).toBeTruthy();
    expect(await findDatabaseCard('Shop')).toBeTruthy();
    expect(countRequests(`/databases/${shopDatabase.id}`)).toBe(1);
  });

  it('adds and selects the copy after a successful copy', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const shopDatabase = buildDatabase('Shop');
    const copiedDatabase = { ...buildDatabase('Copy of Shop'), id: 'shop-copy-id' };
    declareListResponses([shopDatabase], [shopDatabase, copiedDatabase]);
    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        HttpResponse.json(params.databaseId === copiedDatabase.id ? copiedDatabase : shopDatabase),
      ),
      mockApi.post('/databases/{databaseId}/copy', 201, copiedDatabase),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));

    await user.click(await screen.findByRole('button', { name: 'copy' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
    );

    await waitForCondition(async () =>
      expect((await findDatabaseCard('Copy of Shop')).getAttribute('aria-pressed')).toBe('true'),
    );
    expect((await findDatabaseCard('Shop')).getAttribute('aria-pressed')).toBe('false');
    expect(reload).not.toHaveBeenCalled();
  });

  it('keeps the list and the selection when copying is refused', async () => {
    vi.stubGlobal('alert', vi.fn());
    const shopDatabase = buildDatabase('Shop');
    declareListResponses([shopDatabase]);
    mockServer.use(
      mockApi.get('/storage-switches', 200, { items: [] }),
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.post('/databases/{databaseId}/copy', 403, buildProblem('permission_denied', 403)),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));

    await user.click(await screen.findByRole('button', { name: 'copy' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
    );

    await waitFor(() =>
      expect(countRequests(`/databases/${shopDatabase.id}/copy`, 'POST')).toBe(1),
    );
    expect(window.alert).toHaveBeenCalled();
    expect((await findDatabaseCard('Shop')).getAttribute('aria-pressed')).toBe('true');
    expect(screen.getAllByRole('button', { name: /^Shop/ })).toHaveLength(1);
    expect(countRequests('/databases')).toBe(1);
  });

  it('removes the transferred database and selects the first remaining one', async () => {
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {});
    const alphaDatabase = buildDatabase('Alpha', 'AVAILABLE', DatabaseType.MYSQL);
    const bravoDatabase = buildDatabase('Bravo', 'AVAILABLE', DatabaseType.MYSQL);
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, bravoDatabase.id);
    declareTransferRequests([alphaDatabase, bravoDatabase], [alphaDatabase]);
    mockServer.use(mockApi.post('/databases/{databaseId}/transfer', 204, undefined));
    const { user } = renderDatabasesScreen();

    await transferSelectedDatabase(user);

    await waitForCondition(async () =>
      expect((await findDatabaseCard('Alpha')).getAttribute('aria-pressed')).toBe('true'),
    );
    expect(screen.queryByRole('button', { name: /^Bravo/ })).toBeNull();
    expect(reload).not.toHaveBeenCalled();
  });

  it('leaves no database selected after the only database is transferred', async () => {
    const onlyDatabase = buildDatabase('Alpha', 'AVAILABLE', DatabaseType.MYSQL);
    declareTransferRequests([onlyDatabase], []);
    mockServer.use(mockApi.post('/databases/{databaseId}/transfer', 204, undefined));
    const { user } = renderDatabasesScreen();

    await transferSelectedDatabase(user);

    await waitFor(() => expect(screen.queryByRole('button', { name: /^Alpha/ })).toBeNull());
    expect(screen.queryByText('Config')).toBeNull();
    expect(screen.queryAllByRole('button', { pressed: true })).toHaveLength(0);
    expect(localStorage.getItem(SELECTED_DATABASE_PREFERENCE_KEY)).toBeNull();
  });

  it('keeps the list and the selection when the transfer is refused', async () => {
    const alphaDatabase = buildDatabase('Alpha', 'AVAILABLE', DatabaseType.MYSQL);
    const bravoDatabase = buildDatabase('Bravo', 'AVAILABLE', DatabaseType.MYSQL);
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, bravoDatabase.id);
    declareTransferRequests([alphaDatabase, bravoDatabase]);
    mockServer.use(
      mockApi.post('/databases/{databaseId}/transfer', 403, buildProblem('permission_denied', 403)),
    );
    const { user } = renderDatabasesScreen();

    await transferSelectedDatabase(user);

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect((await findDatabaseCard('Bravo')).getAttribute('aria-pressed')).toBe('true');
    expect((await findDatabaseCard('Alpha')).getAttribute('aria-pressed')).toBe('false');
    expect(countRequests('/databases')).toBe(1);
  });
});
