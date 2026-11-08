import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, type RequestHandler, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { type BackupConfig, backupConfigApi } from '@/entities/backup-configs';
import {
  type Database,
  DatabaseType,
  type HealthStatus,
  createDefaultDatabaseSettings,
  databaseApi,
} from '@/entities/databases';
import { type Storage, storageApi } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
// @vitest-environment happy-dom
import { DatabasesTestScreen as DatabasesComponent } from '@/pages/databases/ui/test/DatabasesTestScreen';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { finishApiResponse } from '@/shared/lib/test/finishApiResponse';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

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
  id: `${name.replace(/ /g, '-')}-id`,
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
  id: `${name.replace(/ /g, '-')}-id`,
  workspaceId: workspace.id,
  name,
  settings: { type: 'LOCAL' },
});

const buildBackupConfig = (database: Database, storage: Storage): BackupConfig => ({
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
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
  const select = storageLabel.parentElement!.querySelector('[role="combobox"]')!;

  fireEvent.click(select);
  const option = await waitFor(() => {
    const storageOption = screen.queryByRole('option', { name: `${name}` });
    expect(storageOption).toBeTruthy();
    return storageOption!;
  });

  fireEvent.click(option);
};

const chooseSelectOption = async (combobox: HTMLElement, optionLabel: string) => {
  fireEvent.click(combobox);
  const option = await waitFor(() => {
    const matchingOption = Array.from(screen.queryAllByRole('option')).find(
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
  await user.click(await screen.findByRole('button', { name: 'Transfer' }));
  await chooseSelectOption(await screen.findByRole('combobox'), transferTargetWorkspace.name);
  await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(2));
  await chooseSelectOption(screen.getAllByRole('combobox')[1], transferTargetStorage.name);
  await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Transfer' }));
};

const findDatabaseCard = (name: string) =>
  waitFor(() => {
    const card = screen
      .getAllByRole('link', { name: (accessibleName) => accessibleName.startsWith(name) })
      .find((button) => button.hasAttribute('aria-current'));
    expect(card).toBeTruthy();
    return card!;
  });

const reachPhysicalWizardBackupStep = async (user: ReturnType<typeof renderScreen>['user']) => {
  await user.click(await screen.findByRole('button', { name: 'Add database' }));
  fireEvent.change(await screen.findByPlaceholderText('My favourite DB'), {
    target: { value: 'Created' },
  });
  fireEvent.click(screen.getByText('PostgreSQL'));
  fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
  fireEvent.click(await screen.findByText('Physical'));
  fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
  fireEvent.change(await screen.findByPlaceholderText(/host$/), {
    target: { value: 'db.example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'backup' } });
  fireEvent.change(screen.getByPlaceholderText(/password$/), { target: { value: 'secret' } });
  fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
  await screen.findByText('Storage');
};

const reachWizardBackupStep = async (user: ReturnType<typeof renderScreen>['user']) => {
  await user.click(await screen.findByRole('button', { name: 'Add database' }));
  fireEvent.change(await screen.findByPlaceholderText('My favourite DB'), {
    target: { value: 'Created' },
  });
  fireEvent.click(screen.getByText('MySQL'));
  fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
  fireEvent.change(await screen.findByPlaceholderText(/host$/), {
    target: { value: 'db.example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'backup' } });
  fireEvent.change(screen.getByPlaceholderText(/password$/), { target: { value: 'secret' } });
  fireEvent.change(screen.getByPlaceholderText(/database name$/), { target: { value: 'shop' } });
  fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));
  await screen.findByText('Storage');
};

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

    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('true');
    expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('false');
  });

  it('shows the list without a selected database on mobile', async () => {
    vi.stubGlobal('innerWidth', 500);
    declareListResponses([buildDatabase('Alpha'), buildDatabase('Bravo')]);
    declarePendingDetailRequests();

    renderDatabasesScreen();

    expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('false');
    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('false');
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
    await user.click(await screen.findByRole('button', { name: 'Delete' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );

    await waitForCondition(async () =>
      expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('true'),
    );
    expect(screen.queryByRole('link', { name: /^Bravo/ })).toBeNull();
  });

  it('keeps the database selected and listed and shows the reason when the delete is refused', async () => {
    const alphaDatabase = buildDatabase('Alpha');
    const bravoDatabase = buildDatabase('Bravo');
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, bravoDatabase.id);
    declareListResponses([alphaDatabase, bravoDatabase]);
    mockServer.use(
      mockApi.get('/databases/{databaseId}', 200, bravoDatabase),
      mockApi.delete('/databases/{databaseId}', 403, buildProblem('permission_denied', 403)),
      ...declareSettingsRequests(),
    );
    const { user } = renderDatabasesScreen();

    await user.click(await screen.findByText('Config'));
    await user.click(await screen.findByRole('button', { name: 'Delete' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('true');
    expect(await findDatabaseCard('Alpha')).toBeTruthy();
    expect(localStorage.getItem(SELECTED_DATABASE_PREFERENCE_KEY)).toBe(bravoDatabase.id);
    expect(screen.getByRole('button', { name: 'Delete' })).toBeTruthy();
    expect(countRequests('/databases')).toBe(1);
  });

  it('labels the search field', async () => {
    declareListResponses(
      ['Alpha', 'Bravo', 'Charlie', 'Delta', 'Echo'].map((name) => buildDatabase(name)),
    );
    declarePendingDetailRequests();

    renderDatabasesScreen();

    expect(await screen.findByLabelText('Search database')).toBeTruthy();
  });

  it('offers a retry instead of an empty list when the first list load fails', async () => {
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
  });

  it('keeps the list without feedback when a background refresh fails', async () => {
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

    expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('true');
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('keeps the selected database when a background network failure precedes a successful refresh', async () => {
    const alphaDatabase = buildDatabase('Alpha');
    const bravoDatabase = buildDatabase('Bravo');
    const databaseReads = vi.spyOn(databaseApi, 'listAllDatabases');
    declareListResponses([alphaDatabase]);
    declarePendingDetailRequests();

    renderDatabasesScreen();
    await findDatabaseCard('Alpha');

    const failedRead = mockApi.pending('get', '/databases');
    mockServer.use(failedRead.handler);

    fireEvent(document, new Event('visibilitychange'));
    await waitFor(() => expect(databaseReads).toHaveBeenCalledTimes(2));
    await releaseAndSettle(databaseReads, () => failedRead.respond(HttpResponse.error()));

    expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('true');
    expect(localStorage.getItem(SELECTED_DATABASE_PREFERENCE_KEY)).toBe(alphaDatabase.id);
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();

    declareListResponses([bravoDatabase]);

    fireEvent(document, new Event('visibilitychange'));

    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('true');
    expect(screen.queryByRole('link', { name: /^Alpha/ })).toBeNull();
    await waitFor(() =>
      expect(localStorage.getItem(SELECTED_DATABASE_PREFERENCE_KEY)).toBe(bravoDatabase.id),
    );
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('keeps showing the newly selected database when the previous one responds last', async () => {
    const alphaDatabase = buildDatabase('Alpha');
    const bravoDatabase = { ...buildDatabase('Bravo'), name: 'Bravo' };
    const pendingAlpha = createPendingApiResponse();
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, alphaDatabase.id);
    declareListResponses([alphaDatabase, bravoDatabase]);

    mockServer.use(
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
    await finishCancelledApiResponse(mockServer, `/api/v1/databases/${alphaDatabase.id}`, () =>
      pendingAlpha.respond(HttpResponse.json(alphaDatabase)),
    );

    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('true');
    expect(screen.getAllByText('Bravo')).toHaveLength(2);
    expect(screen.getAllByText('Alpha')).toHaveLength(1);
  });

  it('renames a database in its settings and its card while keeping the selection', async () => {
    const shopDatabase = buildDatabase('Shop');
    const renamedDatabase = { ...shopDatabase, name: 'Storefront' };
    declareListResponses([shopDatabase], [renamedDatabase]);
    mockServer.use(
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.put('/databases/{databaseId}', 200, renamedDatabase),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    const nameTitle = await screen.findByText('Shop', { selector: 'div.text-xl > span' });

    fireEvent.click(nameTitle.parentElement!.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);
    const nameInput = screen.getByPlaceholderText('Enter name...');
    await user.clear(nameInput);
    await user.type(nameInput, 'Storefront');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Storefront', { selector: 'div.text-xl > span' })).toBeTruthy();
    expect((await findDatabaseCard('Storefront')).getAttribute('aria-current')).toBe('true');
    expect(screen.queryByPlaceholderText('Enter name...')).toBeNull();
  });

  it('keeps the settings tab and the list after the connection settings are saved', async () => {
    const shopDatabase = buildDatabase('Shop', 'AVAILABLE', DatabaseType.MYSQL);
    declareListResponses([shopDatabase]);
    mockServer.use(
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
    const shopDatabase = buildDatabase('Shop', 'AVAILABLE', DatabaseType.MYSQL);
    declareListResponses([shopDatabase]);
    mockServer.use(
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
    await user.type(screen.getByLabelText('Password'), 'replica-password');
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

      fireEvent.click(document.querySelector('button[aria-label="Close"]')!);

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
      mockApi.get('/databases/{databaseId}', 200, ordersDatabase),
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
        HttpResponse.json(storedBackupConfig),
      ),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => {
        storedBackupConfig = { ...backupConfig, storageId: archiveStorage.id };
        return HttpResponse.json(storedBackupConfig);
      }),
      mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: [] }),
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
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        HttpResponse.json(params.databaseId === copiedDatabase.id ? copiedDatabase : shopDatabase),
      ),
      mockApi.post('/databases/{databaseId}/copy', 201, copiedDatabase),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));

    await user.click(await screen.findByRole('button', { name: 'Copy' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
    );

    await waitForCondition(async () =>
      expect((await findDatabaseCard('Copy of Shop')).getAttribute('aria-current')).toBe('true'),
    );
    expect((await findDatabaseCard('Shop')).getAttribute('aria-current')).toBe('false');
    expect(reload).not.toHaveBeenCalled();
  });

  it('keeps the list and the selection when copying is refused', async () => {
    const shopDatabase = buildDatabase('Shop');
    declareListResponses([shopDatabase]);
    mockServer.use(
      mockApi.get('/databases/{databaseId}', 200, shopDatabase),
      mockApi.post('/databases/{databaseId}/copy', 403, buildProblem('permission_denied', 403)),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));

    await user.click(await screen.findByRole('button', { name: 'Copy' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
    );

    await waitFor(() =>
      expect(countRequests(`/databases/${shopDatabase.id}/copy`, 'POST')).toBe(1),
    );
    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect((await findDatabaseCard('Shop')).getAttribute('aria-current')).toBe('true');
    expect(screen.getAllByRole('link', { name: /^Shop/ })).toHaveLength(1);
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
      expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('true'),
    );
    expect(screen.queryByRole('link', { name: /^Bravo/ })).toBeNull();
    expect(reload).not.toHaveBeenCalled();
  });

  it('leaves no database selected after the only database is transferred', async () => {
    const onlyDatabase = buildDatabase('Alpha', 'AVAILABLE', DatabaseType.MYSQL);
    declareTransferRequests([onlyDatabase], []);
    mockServer.use(mockApi.post('/databases/{databaseId}/transfer', 204, undefined));
    const { user } = renderDatabasesScreen();

    await transferSelectedDatabase(user);

    await waitFor(() => expect(screen.queryByRole('link', { name: /^Alpha/ })).toBeNull());
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
    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('true');
    expect((await findDatabaseCard('Alpha')).getAttribute('aria-current')).toBe('false');
    expect(countRequests('/databases')).toBe(1);
  });
});

describe('DatabasesComponent during the backup step of the wizard', () => {
  it.each([
    { kind: 'logical', reachBackupStep: reachWizardBackupStep },
    { kind: 'physical', reachBackupStep: reachPhysicalWizardBackupStep },
  ])(
    'keeps the entered $kind values when the page becomes visible and the list refreshes',
    async ({ reachBackupStep }) => {
      const shopDatabase = buildDatabase('Shop');
      declareListResponses([shopDatabase], [shopDatabase, buildDatabase('Refreshed')]);
      declarePendingDetailRequests();
      mockServer.use(
        mockApi.post('/database-tests', 200, { isConnected: true }),
        mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      );
      const { user } = renderDatabasesScreen();
      await reachBackupStep(user);
      const successTrigger = screen.getAllByRole('checkbox')[0] as HTMLInputElement;
      const isInitiallyChecked = successTrigger.checked;
      fireEvent.click(successTrigger);

      act(() => {
        document.dispatchEvent(new Event('visibilitychange'));
      });
      await findDatabaseCard('Refreshed');

      expect(document.querySelector('[role="dialog"] [role="status"]')).toBeNull();
      expect((screen.getAllByRole('checkbox')[0] as HTMLInputElement).checked).toBe(
        !isInitiallyChecked,
      );
    },
  );
});

describe('DatabasesComponent aggregate cancellation', () => {
  it.each([403, 404])(
    'keeps a recovered collection when a superseded sibling returns HTTP %i',
    async (status) => {
      const obsoleteStorageRead = mockApi.pending('get', '/storages');
      mockServer.use(
        mockApi.get('/databases', 500, buildProblem('internal', 500)),
        mockApi.get('/backup-configs', 200, { items: [] }),
        obsoleteStorageRead.handler,
      );
      const onUnavailable = vi.fn();
      renderScreen(
        <DatabasesComponent workspace={workspace} isCanManageDBs onUnavailable={onUnavailable} />,
      );
      await waitFor(() => expect(countRequests('/storages')).toBe(1));
      const retry = await screen.findByRole('button', { name: 'Retry' });
      const recoveredDatabase = buildDatabase('Recovered database');
      declarePendingDetailRequests();
      declareListResponses([recoveredDatabase]);

      fireEvent.click(retry);
      await findDatabaseCard(recoveredDatabase.name);

      const obsoleteRequest = mockServer
        .getRequests()
        .find((request) => new URL(request.url).pathname === '/api/v1/storages')!;
      expect(obsoleteRequest.signal.aborted).toBe(true);

      await finishCancelledApiResponse(mockServer, '/api/v1/storages', () =>
        obsoleteStorageRead.respond(
          HttpResponse.json(buildProblem('permission_denied', status), { status }),
        ),
      );

      expect(onUnavailable).not.toHaveBeenCalled();
      expect(await findDatabaseCard(recoveredDatabase.name)).toBeTruthy();
    },
  );

  it('reconciles once when concurrent siblings refuse access after a temporary failure', async () => {
    const configRead = vi.spyOn(backupConfigApi, 'listAllWorkspaceBackupConfigs');
    const storageRead = vi.spyOn(storageApi, 'listAllStorages');
    const refusedConfigs = mockApi.pending('get', '/backup-configs');
    const refusedStorages = mockApi.pending('get', '/storages');
    mockServer.use(
      mockApi.get('/databases', 500, buildProblem('internal', 500)),
      refusedConfigs.handler,
      refusedStorages.handler,
    );
    const onUnavailable = vi.fn();
    renderScreen(
      <DatabasesComponent workspace={workspace} isCanManageDBs onUnavailable={onUnavailable} />,
    );
    await screen.findByRole('button', { name: 'Retry' });
    await waitFor(() => expect(countRequests('/storages')).toBe(1));

    await act(async () => {
      refusedConfigs.respond(
        HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
      );
      refusedStorages.respond(HttpResponse.json(buildProblem('not_found', 404), { status: 404 }));
      await Promise.allSettled([
        configRead.mock.results[0].value,
        storageRead.mock.results[0].value,
      ]);
    });

    expect(onUnavailable).toHaveBeenCalledExactlyOnceWith(workspace.id);
  });

  it('reconciles access loss when a sibling of the current failed aggregate refuses access', async () => {
    const currentStorageRead = mockApi.pending('get', '/storages');
    mockServer.use(
      mockApi.get('/databases', 500, buildProblem('internal', 500)),
      mockApi.get('/backup-configs', 200, { items: [] }),
      currentStorageRead.handler,
    );
    const onUnavailable = vi.fn();
    renderScreen(
      <DatabasesComponent workspace={workspace} isCanManageDBs onUnavailable={onUnavailable} />,
    );
    await screen.findByRole('button', { name: 'Retry' });
    await waitFor(() => expect(countRequests('/storages')).toBe(1));
    expect(
      mockServer
        .getRequests()
        .find((request) => new URL(request.url).pathname === '/api/v1/storages')!.signal.aborted,
    ).toBe(false);

    await finishApiResponse(mockServer, '/api/v1/storages', () =>
      currentStorageRead.respond(
        HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
      ),
    );

    expect(onUnavailable).toHaveBeenCalledExactlyOnceWith(workspace.id);
  });

  it('cancels remaining collection reads after a failed aggregate unmounts', async () => {
    const obsoleteStorageRead = mockApi.pending('get', '/storages');
    mockServer.use(
      mockApi.get('/databases', 500, buildProblem('internal', 500)),
      mockApi.get('/backup-configs', 200, { items: [] }),
      obsoleteStorageRead.handler,
    );
    const view = renderDatabasesScreen();
    await screen.findByRole('button', { name: 'Retry' });
    await waitFor(() => expect(countRequests('/storages')).toBe(1));

    view.unmount();

    await finishCancelledApiResponse(mockServer, '/api/v1/storages', () =>
      obsoleteStorageRead.respond(HttpResponse.json({ items: [], nextCursor: 'obsolete-cursor' })),
    );

    expect(countRequests('/storages')).toBe(1);
  });
});

describe('DatabasesComponent with a late rename', () => {
  it('keeps the other selected database shown while the list shows the new name', async () => {
    const shopDatabase = buildDatabase('Shop');
    const bravoDatabase = buildDatabase('Bravo');
    const renamedDatabase = { ...shopDatabase, name: 'Storefront' };
    const pendingRename = createPendingApiResponse();
    let isRenamed = false;
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, shopDatabase.id);
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, () =>
        HttpResponse.json({ items: [isRenamed ? renamedDatabase : shopDatabase, bravoDatabase] }),
      ),
      mockApi.get('/backup-configs', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        HttpResponse.json(params.databaseId === bravoDatabase.id ? bravoDatabase : shopDatabase),
      ),
      http.put(`*${API_BASE_PATH}/databases/:databaseId`, () => pendingRename.response),
      ...declareSettingsRequests(),
    );

    const { user } = renderDatabasesScreen();
    await user.click(await screen.findByText('Config'));
    const nameTitle = await screen.findByText('Shop', { selector: 'div.text-xl > span' });
    fireEvent.click(nameTitle.parentElement!.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);
    const nameInput = screen.getByPlaceholderText('Enter name...');
    await user.clear(nameInput);
    await user.type(nameInput, 'Storefront');
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(countRequests(`/databases/${shopDatabase.id}`, 'PUT')).toBe(1));

    await user.click(await findDatabaseCard('Bravo'));
    await screen.findByText('Bravo', { selector: 'div.text-xl > span' });
    isRenamed = true;
    await act(async () => {
      pendingRename.respond(HttpResponse.json(renamedDatabase));
    });

    expect(await findDatabaseCard('Storefront')).toBeTruthy();
    expect((await findDatabaseCard('Bravo')).getAttribute('aria-current')).toBe('true');
    expect(screen.getByText('Bravo', { selector: 'div.text-xl > span' })).toBeTruthy();
    expect(screen.queryByText('Storefront', { selector: 'div.text-xl > span' })).toBeNull();
  });
});

describe('DatabasesComponent with late database actions', () => {
  const shopDatabase = buildDatabase('Shop');
  const bravoDatabase = buildDatabase('Bravo');
  const copiedDatabase = { ...buildDatabase('Copy of Shop'), id: 'shop-copy-id' };
  const renamedDatabase = { ...shopDatabase, name: 'Storefront' };
  const REFUSAL_MESSAGE = 'You do not have permission to do this.';
  const COPY_NOTICE = 'Database copied successfully!';
  const TEST_NOTICE = 'Connection test successful!';
  const refusal = () => HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });

  const declareActionServer = () => {
    const actionServer = {
      listedDatabases: [shopDatabase, bravoDatabase],
      pendingAction: createPendingApiResponse(),
    };
    vi.spyOn(databaseApi, 'listAllDatabases');
    vi.spyOn(databaseApi, 'copyDatabase');
    vi.spyOn(databaseApi, 'updateDatabase');
    vi.spyOn(databaseApi, 'deleteDatabase');
    vi.spyOn(databaseApi, 'testDatabase');
    localStorage.setItem(SELECTED_DATABASE_PREFERENCE_KEY, shopDatabase.id);
    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, () =>
        HttpResponse.json({ items: actionServer.listedDatabases }),
      ),
      mockApi.get('/backup-configs', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        HttpResponse.json(params.databaseId === bravoDatabase.id ? bravoDatabase : shopDatabase),
      ),
      http.post(
        `*${API_BASE_PATH}/databases/:databaseId/copy`,
        () => actionServer.pendingAction.response,
      ),
      http.post(
        `*${API_BASE_PATH}/databases/:databaseId/test`,
        () => actionServer.pendingAction.response,
      ),
      http.put(
        `*${API_BASE_PATH}/databases/:databaseId`,
        () => actionServer.pendingAction.response,
      ),
      http.delete(
        `*${API_BASE_PATH}/databases/:databaseId`,
        () => actionServer.pendingAction.response,
      ),
      ...declareSettingsRequests(),
    );

    return actionServer;
  };

  type ScreenUser = ReturnType<typeof renderScreen>['user'];

  const startCopy = async (user: ScreenUser) => {
    await user.click(await screen.findByRole('button', { name: 'Copy' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Copy' }),
    );
    await waitFor(() =>
      expect(countRequests(`/databases/${shopDatabase.id}/copy`, 'POST')).toBe(1),
    );
  };

  const startDelete = async (user: ScreenUser) => {
    await user.click(await screen.findByRole('button', { name: 'Delete' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );
    await waitFor(() => expect(countRequests(`/databases/${shopDatabase.id}`, 'DELETE')).toBe(1));
  };

  const startRename = async (user: ScreenUser) => {
    await user.click(await screen.findByRole('button', { name: 'Edit Shop' }));

    const nameInput = screen.getByPlaceholderText('Enter name...');
    await user.clear(nameInput);
    await user.type(nameInput, 'Storefront');

    await user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(countRequests(`/databases/${shopDatabase.id}`, 'PUT')).toBe(1));
  };

  const startTest = async (user: ScreenUser) => {
    await user.click(await screen.findByRole('button', { name: 'Test connection' }));
    await waitFor(() =>
      expect(countRequests(`/databases/${shopDatabase.id}/test`, 'POST')).toBe(1),
    );
  };

  const openConfigOfShop = async () => {
    const view = renderDatabasesScreen();
    await view.user.click(await screen.findByText('Config'));
    await screen.findByText('Shop', { selector: 'div.text-xl > span' });

    return view;
  };

  const selectBravo = async (user: ScreenUser) => {
    await user.click(await findDatabaseCard('Bravo'));
    await screen.findByText('Bravo', { selector: 'div.text-xl > span' });
  };

  const expectCardPressed = (name: string, isPressed: boolean) =>
    waitFor(async () =>
      expect((await findDatabaseCard(name)).getAttribute('aria-current')).toBe(String(isPressed)),
    );

  const expectNoActionFeedback = () => {
    expect(screen.queryByText(COPY_NOTICE)).toBeNull();
    expect(screen.queryByText(TEST_NOTICE)).toBeNull();
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  };

  const respondAndSettle = async (respond: () => void) => {
    const actionCalls = [
      vi.mocked(databaseApi.copyDatabase),
      vi.mocked(databaseApi.updateDatabase),
      vi.mocked(databaseApi.deleteDatabase),
      vi.mocked(databaseApi.testDatabase),
    ].flatMap((actionCall) => actionCall.mock.results.map((result) => result.value as unknown));

    await act(async () => {
      respond();
      await Promise.allSettled(actionCalls);
    });
  };

  describe('after another database was selected', () => {
    it('adds a late copy to the list unselected and keeps the selection', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startCopy(user);
      await selectBravo(user);

      actionServer.listedDatabases = [shopDatabase, bravoDatabase, copiedDatabase];
      await act(async () => {
        actionServer.pendingAction.respond(HttpResponse.json(copiedDatabase, { status: 201 }));
      });

      expect(await findDatabaseCard('Copy of Shop')).toBeTruthy();
      await expectCardPressed('Copy of Shop', false);
      await expectCardPressed('Bravo', true);
      expect(screen.getByText('Bravo', { selector: 'div.text-xl > span' })).toBeTruthy();
      expectNoActionFeedback();
    });

    it('removes a late deleted database and keeps the selection', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startDelete(user);
      await selectBravo(user);

      actionServer.listedDatabases = [bravoDatabase];
      await act(async () => {
        actionServer.pendingAction.respond(new HttpResponse(null, { status: 204 }));
      });

      await waitFor(() => expect(screen.queryByRole('link', { name: /^Shop/ })).toBeNull());
      await expectCardPressed('Bravo', true);
      expect(screen.getByText('Bravo', { selector: 'div.text-xl > span' })).toBeTruthy();
      expectNoActionFeedback();
    });

    it('shows no feedback for a late connection test', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startTest(user);
      await selectBravo(user);

      await respondAndSettle(() =>
        actionServer.pendingAction.respond(HttpResponse.json({ isConnected: true })),
      );

      expectNoActionFeedback();
      await expectCardPressed('Bravo', true);
    });

    it('shows no feedback for a late refusal', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startCopy(user);
      await selectBravo(user);

      await respondAndSettle(() => actionServer.pendingAction.respond(refusal()));

      expectNoActionFeedback();
      await expectCardPressed('Bravo', true);
    });
  });

  describe('after another tab was opened', () => {
    it('selects a late copy with no notice', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startCopy(user);
      await user.click(screen.getByText('Backups'));

      actionServer.listedDatabases = [shopDatabase, bravoDatabase, copiedDatabase];
      await act(async () => {
        actionServer.pendingAction.respond(HttpResponse.json(copiedDatabase, { status: 201 }));
      });

      await expectCardPressed('Copy of Shop', true);
      expectNoActionFeedback();
    });

    it('shows the new name of a late rename in the list', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startRename(user);
      await user.click(screen.getByText('Backups'));

      actionServer.listedDatabases = [renamedDatabase, bravoDatabase];
      await act(async () => {
        actionServer.pendingAction.respond(HttpResponse.json(renamedDatabase));
      });

      expect(await findDatabaseCard('Storefront')).toBeTruthy();
      await expectCardPressed('Storefront', true);
      expectNoActionFeedback();
    });

    it('moves the selection when a late delete answers', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startDelete(user);
      await user.click(screen.getByText('Backups'));

      actionServer.listedDatabases = [bravoDatabase];
      await act(async () => {
        actionServer.pendingAction.respond(new HttpResponse(null, { status: 204 }));
      });

      await waitFor(() => expect(screen.queryByRole('link', { name: /^Shop/ })).toBeNull());
      await expectCardPressed('Bravo', true);
      expectNoActionFeedback();
    });

    it('shows no feedback for a late connection test', async () => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await startTest(user);
      await user.click(screen.getByText('Backups'));

      await respondAndSettle(() =>
        actionServer.pendingAction.respond(HttpResponse.json({ isConnected: true })),
      );

      expectNoActionFeedback();
    });

    it.each([
      { action: 'copy', start: startCopy },
      { action: 'rename', start: startRename },
      { action: 'delete', start: startDelete },
      { action: 'connection test', start: startTest },
    ])('shows no reason for a late refused $action', async ({ start }) => {
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await start(user);
      await user.click(screen.getByText('Backups'));

      await respondAndSettle(() => actionServer.pendingAction.respond(refusal()));

      expectNoActionFeedback();
      await expectCardPressed('Shop', true);
      await user.click(screen.getByText('Config'));
      await screen.findByText('Shop', { selector: 'div.text-xl > span' });
      expectNoActionFeedback();
    });
  });

  const LATE_SUCCESSES = [
    {
      action: 'copy',
      start: startCopy,
      buildAnswer: () => HttpResponse.json(copiedDatabase, { status: 201 }),
    },
    {
      action: 'rename',
      start: startRename,
      buildAnswer: () => HttpResponse.json(renamedDatabase),
    },
    {
      action: 'delete',
      start: startDelete,
      buildAnswer: () => new HttpResponse(null, { status: 204 }),
    },
    {
      action: 'connection test',
      start: startTest,
      buildAnswer: () => HttpResponse.json({ isConnected: true }),
    },
  ];

  it.each(LATE_SUCCESSES)(
    'reports nothing and requests nothing when a $action answers after the screen left',
    async ({ start, buildAnswer }) => {
      const actionServer = declareActionServer();
      const { user, unmount } = await openConfigOfShop();
      await start(user);
      const listLoads = vi.mocked(databaseApi.listAllDatabases);
      const listLoadsBeforeLeaving = listLoads.mock.calls.length;
      unmount();

      await respondAndSettle(() => actionServer.pendingAction.respond(buildAnswer()));

      expectNoActionFeedback();
      expect(listLoads.mock.calls.length).toBe(listLoadsBeforeLeaving);
    },
  );

  it.each(LATE_SUCCESSES)(
    'changes nothing when a $action answers after the session ended',
    async ({ start, buildAnswer }) => {
      activateSession({ token: 'first-token', userId: 'first-user' });
      const actionServer = declareActionServer();
      const { user } = await openConfigOfShop();
      await start(user);
      const listLoads = vi.mocked(databaseApi.listAllDatabases);
      const listLoadsBeforeEnd = listLoads.mock.calls.length;
      act(() => {
        invalidateSession();
      });

      await respondAndSettle(() => actionServer.pendingAction.respond(buildAnswer()));

      expectNoActionFeedback();
      expect(listLoads.mock.calls.length).toBe(listLoadsBeforeEnd);
      expect(screen.queryByRole('link', { name: /^Copy of Shop/ })).toBeNull();
      expect(screen.queryByRole('link', { name: /^Storefront/ })).toBeNull();
      expect(screen.queryByRole('link', { name: /^Shop/ })).toBeNull();
      expect(screen.getByRole('status')).toBeTruthy();
      expect(screen.queryByText('Storefront', { selector: 'div.text-xl > span' })).toBeNull();
    },
  );

  it.each(
    LATE_SUCCESSES.filter(({ action }) => action !== 'connection test').map(
      ({ action, start }) => ({ action, start }),
    ),
  )('shows no reason when a refused $action answers after the session ended', async ({ start }) => {
    activateSession({ token: 'first-token', userId: 'first-user' });
    const actionServer = declareActionServer();
    const { user } = await openConfigOfShop();
    await start(user);
    act(() => {
      invalidateSession();
    });

    await respondAndSettle(() => actionServer.pendingAction.respond(refusal()));

    expectNoActionFeedback();
  });

  it('removes a shown reason after returning from another tab', async () => {
    const actionServer = declareActionServer();
    const { user } = await openConfigOfShop();
    await startCopy(user);
    await act(async () => {
      actionServer.pendingAction.respond(refusal());
    });
    await screen.findByText(REFUSAL_MESSAGE);

    await user.click(screen.getByText('Backups'));
    await user.click(screen.getByText('Config'));
    await screen.findByText('Shop', { selector: 'div.text-xl > span' });

    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('removes a shown reason after returning from another database', async () => {
    const actionServer = declareActionServer();
    const { user } = await openConfigOfShop();
    await startCopy(user);
    await act(async () => {
      actionServer.pendingAction.respond(refusal());
    });
    await screen.findByText(REFUSAL_MESSAGE);

    await selectBravo(user);
    await user.click(await findDatabaseCard('Shop'));
    await screen.findByText('Shop', { selector: 'div.text-xl > span' });

    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });
});

describe('DatabasesComponent with the creation wizard', () => {
  const shopDatabase = buildDatabase('Shop');
  const createdDatabase = {
    ...buildDatabase('Created', 'AVAILABLE', DatabaseType.MYSQL),
    id: 'created-id',
  };
  const archiveStorage = buildStorage('Archive');
  const SCHEDULED_BACKUPS_DISABLED =
    'Scheduled backups are disabled (you can enable it back in the backup configuration)';

  const declareWizardServer = () => {
    const wizardServer = {
      isCreated: false,
      isBackupConfigRefused: false,
      storedBackupConfig: undefined as BackupConfig | undefined,
    };
    const getStoredBackupConfig = (databaseId: string): BackupConfig => {
      if (databaseId === shopDatabase.id) return buildBackupConfig(shopDatabase, archiveStorage);

      return (
        wizardServer.storedBackupConfig ?? {
          ...buildBackupConfig(createdDatabase, archiveStorage),
          isBackupsEnabled: false,
          storageId: null,
        }
      );
    };

    mockServer.use(
      http.get(`*${API_BASE_PATH}/databases`, ({ request }) => {
        if (new URL(request.url).searchParams.get('workspaceId') !== workspace.id)
          return HttpResponse.json({ items: [] });

        return HttpResponse.json({
          items: wizardServer.isCreated ? [shopDatabase, createdDatabase] : [shopDatabase],
        });
      }),
      http.get(`*${API_BASE_PATH}/backup-configs`, () =>
        HttpResponse.json({
          items: wizardServer.storedBackupConfig ? [wizardServer.storedBackupConfig] : [],
        }),
      ),
      mockApi.get('/storages', 200, { items: [archiveStorage] }),
      http.get(`*${API_BASE_PATH}/databases/:databaseId`, ({ params }) =>
        HttpResponse.json(
          params.databaseId === createdDatabase.id ? createdDatabase : shopDatabase,
        ),
      ),
      http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, ({ params }) =>
        HttpResponse.json(getStoredBackupConfig(params.databaseId as string)),
      ),
      mockApi.get('/logical-backups', 200, { totalUsageMb: 0, items: [] }),
      mockApi.get('/storages/{storageId}', 200, archiveStorage),
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      mockApi.get('/notifiers', 200, { items: [] }),
      mockApi.post('/database-tests', 200, { isConnected: true }),
      mockApi.post('/limited-user-checks', 200, { isAdvisable: false, privileges: [] }),
      http.post(`*${API_BASE_PATH}/databases`, () => {
        wizardServer.isCreated = true;

        return HttpResponse.json(createdDatabase, { status: 201 });
      }),
      http.put(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => {
        if (wizardServer.isBackupConfigRefused)
          return HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });

        wizardServer.storedBackupConfig = buildBackupConfig(createdDatabase, archiveStorage);

        return HttpResponse.json(wizardServer.storedBackupConfig);
      }),
      http.post(`*${API_BASE_PATH}/logical-backups`, () =>
        HttpResponse.json({ id: 'backup-id' }, { status: 202 }),
      ),
    );

    return wizardServer;
  };

  const reachWizardCompletion = async (user: ReturnType<typeof renderScreen>['user']) => {
    await reachWizardBackupStep(user);
    await chooseStorage('Archive');
    fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));

    return screen.findByRole('button', { name: 'Complete' });
  };

  const closeWizard = () => fireEvent.click(document.querySelector('button[aria-label="Close"]')!);

  const refreshInBackground = () =>
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

  const expectCardPressed = async (name: string, isPressed: boolean) =>
    waitFor(async () =>
      expect((await findDatabaseCard(name)).getAttribute('aria-current')).toBe(String(isPressed)),
    );

  it('does not select the created database while the wizard is open', async () => {
    declareWizardServer();
    const backupConfigSave = mockApi.pending('put', '/backup-configs/{databaseId}');
    mockServer.use(backupConfigSave.handler);
    const { user } = renderDatabasesScreen();

    fireEvent.click(await reachWizardCompletion(user));
    await waitFor(() => expect(countRequests('/backup-configs/created-id', 'PUT')).toBe(1));
    refreshInBackground();

    await expectCardPressed('Created', false);
    await expectCardPressed('Shop', true);
    expect(countRequests('/databases/created-id')).toBe(0);
    expect(countRequests('/backup-configs/created-id')).toBe(0);
  });

  it('selects the created database after the wizard finishes and shows what the server saved', async () => {
    declareWizardServer();
    const { user } = renderDatabasesScreen();
    await findDatabaseCard('Shop');

    fireEvent.click(await reachWizardCompletion(user));

    await expectCardPressed('Created', true);
    expect(await within(await findDatabaseCard('Created')).findByText('Archive')).toBeTruthy();
    expect(await within(screen.getByRole('table')).findByText('No backups yet')).toBeTruthy();
    expect(screen.queryByText(SCHEDULED_BACKUPS_DISABLED)).toBeNull();
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('lists and selects the created database when the user closes the wizard after a failure', async () => {
    const wizardServer = declareWizardServer();
    wizardServer.isBackupConfigRefused = true;
    const { user } = renderDatabasesScreen();

    fireEvent.click(await reachWizardCompletion(user));
    await screen.findByText(/The database was created/);
    closeWizard();

    await expectCardPressed('Created', true);
    await expectCardPressed('Shop', false);
  });

  const openAndCancelWizard = async (user: ReturnType<typeof renderScreen>['user']) => {
    await user.click(await screen.findByRole('button', { name: 'Add database' }));
    await screen.findByPlaceholderText('My favourite DB');
    closeWizard();
    await waitFor(() => expect(screen.queryByPlaceholderText('My favourite DB')).toBeNull());
  };

  it('keeps the chosen database selected when a later wizard is cancelled', async () => {
    const wizardServer = declareWizardServer();
    wizardServer.isBackupConfigRefused = true;
    const { user } = renderDatabasesScreen();

    fireEvent.click(await reachWizardCompletion(user));
    await screen.findByText(/The database was created/);
    closeWizard();
    await expectCardPressed('Created', true);
    await user.click(await findDatabaseCard('Shop'));
    await expectCardPressed('Shop', true);
    await openAndCancelWizard(user);

    await expectCardPressed('Shop', true);
    await expectCardPressed('Created', false);
  });

  it('leaves the selection alone and lists the database on a later refresh when its create answer comes after the wizard closed', async () => {
    const wizardServer = declareWizardServer();
    const creation = mockApi.pending('post', '/databases');
    mockServer.use(creation.handler);
    const createCall = vi.spyOn(databaseApi, 'createDatabase');
    const { user } = renderDatabasesScreen();

    fireEvent.click(await reachWizardCompletion(user));
    await waitFor(() => expect(createCall).toHaveBeenCalledTimes(1));
    closeWizard();
    wizardServer.isCreated = true;
    await releaseAndSettle(createCall, () =>
      creation.respond(HttpResponse.json(createdDatabase, { status: 201 })),
    );
    await openAndCancelWizard(user);

    await expectCardPressed('Shop', true);
    expect(
      screen
        .queryAllByRole('link', { name: /^Created/ })
        .filter((link) => link.hasAttribute('aria-current')),
    ).toHaveLength(0);

    refreshInBackground();

    await expectCardPressed('Created', false);
    await expectCardPressed('Shop', true);
  });
});

it('restores the new workspace preference when the database page changes scope without remounting', async () => {
  const nextWorkspace = { ...workspace, id: 'next-workspace', name: 'Next team' };
  const shopDatabase = buildDatabase('Shop');
  const firstDatabase = { ...buildDatabase('First'), workspaceId: nextWorkspace.id };
  const rememberedDatabase = { ...buildDatabase('Remembered'), workspaceId: nextWorkspace.id };

  localStorage.setItem(`selectedDatabaseId_${nextWorkspace.id}`, rememberedDatabase.id);
  mockServer.use(
    http.get(`*${API_BASE_PATH}/databases`, ({ request }) =>
      HttpResponse.json({
        items:
          new URL(request.url).searchParams.get('workspaceId') === nextWorkspace.id
            ? [firstDatabase, rememberedDatabase]
            : [shopDatabase],
      }),
    ),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
  );
  declarePendingDetailRequests();

  const view = renderDatabasesScreen();

  await waitFor(async () =>
    expect((await findDatabaseCard('Shop')).getAttribute('aria-current')).toBe('true'),
  );

  view.rerender(<DatabasesComponent workspace={nextWorkspace} isCanManageDBs />);

  await waitFor(async () =>
    expect((await findDatabaseCard('Remembered')).getAttribute('aria-current')).toBe('true'),
  );
  expect((await findDatabaseCard('First')).getAttribute('aria-current')).toBe('false');
});
