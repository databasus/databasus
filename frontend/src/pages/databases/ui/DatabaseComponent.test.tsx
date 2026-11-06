import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { http, HttpResponse } from 'msw';
import { createRef } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { backupConfigApi } from '@/entities/backup-configs';
import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
  createDefaultDatabaseSettings,
} from '@/entities/databases';
import { verificationRunsApi } from '@/pages/databases/api/verificationRunsApi';
// @vitest-environment happy-dom
import { DatabaseComponent } from '@/pages/databases/ui/DatabaseComponent';
import { activateSession, invalidateSession } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { inspectApiRequest } from '@/shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

dayjs.extend(utc);

const server = setUpMockApi();
const DatabaseScreen = () => {
  const { setLocale } = useLocale();

  return (
    <>
      <button onClick={() => setLocale('fr')}>Change language</button>
      <DatabaseComponent
        databaseId="shop-id"
        scrollContainerRef={createRef()}
        isCanManageDBs
        onDatabaseChanged={() => {}}
        onDatabaseDeleted={() => {}}
        onDatabaseCopied={() => {}}
        onDatabaseTransferred={() => {}}
      />
    </>
  );
};

const databaseScreenProps = {
  scrollContainerRef: createRef<HTMLDivElement>(),
  isCanManageDBs: true,
  onDatabaseChanged: () => {},
  onDatabaseDeleted: () => {},
  onDatabaseCopied: () => {},
  onDatabaseTransferred: () => {},
};
const declareDatabaseDependencies = () =>
  server.use(
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/logical-backups').handler,
    mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
    mockApi.get('/notifiers', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
  );
const createDatabase = (id: string) => ({
  id,
  workspaceId: 'workspace-id',
  name: id,
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
  notifierIds: [],
});

const EDIT_ICON_SELECTOR = 'img[src="/icons/pen-gray.svg"]';

const shopDatabase: Database = {
  id: 'shop-id',
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

const healthcheckConfig = {
  databaseId: shopDatabase.id,
  isHealthcheckEnabled: true,
  isNotificationEnabled: true,
  intervalMinutes: 1,
  failedAttemptsBeforeUnavailable: 3,
  attemptRetentionDays: 7,
};

const countDatabaseRequests = () =>
  server
    .getRequests()
    .filter((request) => new URL(request.url).pathname === `/api/v1/databases/${shopDatabase.id}`)
    .length;

const openSettingsSection = (title: string) => {
  const sectionTitle = screen.getByText(title).parentElement!;
  fireEvent.click(sectionTitle.querySelector(EDIT_ICON_SELECTOR)!.parentElement!);
};

const renderShopSettings = async (database: Database = shopDatabase, isCanManageDBs = true) => {
  server.use(
    mockApi.get('/databases/{databaseId}', 200, database),
    mockApi.pending('get', '/backup-configs/{databaseId}').handler,
    mockApi.pending('get', '/logical-backups').handler,
    mockApi.pending('get', '/healthcheck-attempts').handler,
    mockApi.get('/notifiers', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
  );
  const view = renderScreen(
    <DatabaseComponent
      {...databaseScreenProps}
      databaseId={database.id}
      isCanManageDBs={isCanManageDBs}
    />,
  );
  await view.user.click(await screen.findByText('Config'));
  await screen.findByText(database.name);

  return view;
};

describe('DatabaseComponent', () => {
  it('keeps the selected database when an older selection responds after it', async () => {
    const oldDatabase = createPendingApiResponse();
    server.use(
      http.get('*/api/v1/databases/:databaseId', ({ params }) =>
        params.databaseId === 'old-shop'
          ? oldDatabase.response
          : HttpResponse.json(createDatabase('new-shop')),
      ),
    );
    declareDatabaseDependencies();
    const view = renderScreen(<DatabaseComponent {...databaseScreenProps} databaseId="old-shop" />);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(1));

    view.rerender(<DatabaseComponent {...databaseScreenProps} databaseId="new-shop" />);
    await view.user.click(await screen.findByText('Config'));

    expect(await screen.findByText('new-shop')).toBeTruthy();

    const oldResponseProcessed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', ({ request }) => {
        if (new URL(request.url).pathname.endsWith('/old-shop')) resolve();
      }),
    );
    await act(async () => {
      oldDatabase.respond(HttpResponse.json(createDatabase('old-shop')));
      await oldResponseProcessed;
    });

    expect(screen.getByText('new-shop')).toBeTruthy();
    expect(screen.queryByText('old-shop')).toBeNull();
  });

  it('shows no late error when the database request fails after unmount', async () => {
    const pendingDatabase = createPendingApiResponse();
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    server.use(http.get('*/api/v1/databases/:databaseId', () => pendingDatabase.response));

    const view = renderScreen(<DatabaseComponent {...databaseScreenProps} databaseId="shop-id" />);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(1));
    const responseProcessed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );

    view.unmount();
    await act(async () => {
      pendingDatabase.respond(HttpResponse.json(buildProblem('internal', 500), { status: 500 }));
      await responseProcessed;
    });

    expect(showAlert).not.toHaveBeenCalled();
  });

  it('keeps the configuration tab and unsaved database edits when the language changes', async () => {
    declareDatabaseDependencies();
    server.use(
      mockApi.get('/databases/{databaseId}', 200, {
        id: 'shop-id',
        workspaceId: 'workspace-id',
        name: 'Shop',
        settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
        notifierIds: [],
      }),
    );
    const { user } = renderScreen(<DatabaseScreen />);

    await user.click(await screen.findByText('Config'));
    await user.click(await screen.findByRole('button', { name: 'Edit Shop' }));
    const nameInput = screen.getByPlaceholderText(DICTIONARIES.en.databases.config.namePlaceholder);

    await user.clear(nameInput);
    await user.type(nameInput, 'Unsaved database name');
    await user.click(screen.getByRole('button', { name: 'Change language' }));

    expect(await screen.findByDisplayValue('Unsaved database name')).toBeTruthy();
    expect(screen.getByPlaceholderText(DICTIONARIES.fr.databases.config.namePlaceholder)).toBe(
      nameInput,
    );
    for (const actionName of [
      DICTIONARIES.fr.common.actions.transfer,
      DICTIONARIES.fr.common.actions.copy,
      DICTIONARIES.fr.common.actions.delete,
      DICTIONARIES.fr.common.actions.close,
    ]) {
      expect(screen.getByRole('button', { name: actionName })).toBeTruthy();
    }
    expect(
      server
        .getRequests()
        .filter((request) => new URL(request.url).pathname === '/api/v1/databases/shop-id'),
    ).toHaveLength(1);
  });

  it('changes the tab from the keyboard and keeps the shown database', async () => {
    declareDatabaseDependencies();
    server.use(
      mockApi.get('/databases/{databaseId}', 200, {
        id: 'shop-id',
        workspaceId: 'workspace-id',
        name: 'Shop',
        settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
        notifierIds: [],
      }),
    );
    const { user } = renderScreen(<DatabaseScreen />);

    const backupsTab = await screen.findByRole('tab', { name: 'Backups' });
    expect(backupsTab.getAttribute('aria-selected')).toBe('true');

    backupsTab.focus();
    await user.keyboard('{ArrowLeft}{Enter}');

    await waitFor(() =>
      expect(screen.getByRole('tab', { name: /Config$/ }).getAttribute('aria-selected')).toBe(
        'true',
      ),
    );
    expect(await screen.findByText('Shop')).toBeTruthy();
    expect(
      server
        .getRequests()
        .filter((request) => new URL(request.url).pathname === '/api/v1/databases/shop-id'),
    ).toHaveLength(1);
  });

  it('offers a retry that reloads the same database when its request fails', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    const requestedDatabaseIds: string[] = [];
    server.use(
      http.get('*/api/v1/databases/:databaseId', ({ params }) => {
        requestedDatabaseIds.push(String(params.databaseId));
        return requestedDatabaseIds.length === 1
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json(createDatabase('shop-id'));
      }),
      mockApi.pending('get', '/healthcheck-attempts').handler,
    );
    declareDatabaseDependencies();

    const { user } = renderScreen(
      <DatabaseComponent {...databaseScreenProps} databaseId="shop-id" />,
    );

    await user.click(await screen.findByRole('button', { name: 'Retry' }));

    expect(await screen.findByText('Config')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(requestedDatabaseIds).toEqual(['shop-id', 'shop-id']);
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('offers no edit control in other sections while one section is open', async () => {
    server.use(mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler);
    const { container } = await renderShopSettings();
    expect(container.querySelectorAll(EDIT_ICON_SELECTOR).length).toBeGreaterThan(1);

    openSettingsSection('Database settings');

    expect(await screen.findByPlaceholderText(/host$/)).toBeTruthy();
    expect(container.querySelectorAll(EDIT_ICON_SELECTOR)).toHaveLength(0);
  });

  it('closes a section on cancel without requesting the database again', async () => {
    server.use(mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler);
    const { user } = await renderShopSettings();
    openSettingsSection('Database settings');
    const hostInput = await screen.findByPlaceholderText(/host$/);

    await user.clear(hostInput);
    await user.type(hostInput, 'unsaved.example.com');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(await screen.findByText('db.example.com')).toBeTruthy();
    expect(screen.queryByText('unsaved.example.com')).toBeNull();
    expect(screen.getByText('Shop')).toBeTruthy();
    expect(countDatabaseRequests()).toBe(1);
  });

  it('shows the stored healthcheck values after saving and requests no database', async () => {
    let storedHealthcheckConfig = healthcheckConfig;
    server.use(
      http.get('*/api/v1/healthcheck-configs/:databaseId', () =>
        HttpResponse.json(storedHealthcheckConfig),
      ),
      http.put('*/api/v1/healthcheck-configs/:databaseId', () => {
        storedHealthcheckConfig = { ...healthcheckConfig, intervalMinutes: 5 };
        return HttpResponse.json(storedHealthcheckConfig);
      }),
    );

    const { user } = await renderShopSettings();
    await screen.findByText('1');
    openSettingsSection('Healthcheck settings');
    await screen.findAllByRole('switch');

    await user.click(screen.getAllByRole('switch')[1]);
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText('5')).toBeTruthy();
    expect(countDatabaseRequests()).toBe(1);
  });

  it('keeps the last backup error after a successful connection test', async () => {
    server.use(
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.post('/databases/{databaseId}/test', 200, { isConnected: true }),
    );
    const { user } = await renderShopSettings({
      ...shopDatabase,
      lastBackupErrorMessage: 'connection refused',
    });

    await user.click(screen.getByRole('button', { name: 'Test connection' }));

    await waitFor(() =>
      expect(
        server
          .getRequests()
          .some((request) => request.url.endsWith(`/databases/${shopDatabase.id}/test`)),
      ).toBe(true),
    );
    await waitFor(() =>
      expect(
        // happy-dom never finishes AntD's loading icon exit, so the name keeps a "loading" prefix.
        screen.getByRole('button', { name: /Test connection$/ }).hasAttribute('disabled'),
      ).toBe(false),
    );

    expect(screen.getByText(/connection refused/)).toBeTruthy();
    expect(
      screen.getByText(DICTIONARIES.en.databases.config.lastBackupError.clearedByNextBackup),
    ).toBeTruthy();
    expect(screen.queryByText(/test connection via button below/)).toBeNull();
  });

  it('offers a viewer no edit, copy, transfer or delete control but keeps the connection test', async () => {
    server.use(mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler);
    const { container } = await renderShopSettings(shopDatabase, false);

    expect(container.querySelectorAll(EDIT_ICON_SELECTOR)).toHaveLength(0);
    expect(screen.getByRole('button', { name: 'Test connection' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Copy' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Transfer' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Delete' })).toBeNull();
  });

  it('shows the stored restore verification values after saving and requests no database', async () => {
    const verificationConfig = {
      databaseId: shopDatabase.id,
      isAutomaticVerificationEnabled: false,
      scheduleType: 'AFTER_BACKUP' as const,
      schedule: { type: 'DAILY' as const, timeOfDay: '00:00' },
      notificationTriggers: [],
    };
    let storedVerificationConfig = verificationConfig;
    server.use(
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      http.get('*/api/v1/verification-configs/:databaseId', () =>
        HttpResponse.json(storedVerificationConfig),
      ),
      http.put('*/api/v1/verification-configs/:databaseId', () => {
        storedVerificationConfig = { ...verificationConfig, isAutomaticVerificationEnabled: true };
        return HttpResponse.json(storedVerificationConfig);
      }),
    );

    const { user } = await renderShopSettings({
      ...shopDatabase,
      settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
    });
    await screen.findByText('Scheduled verification');
    expect(screen.queryByText('After backup')).toBeNull();
    openSettingsSection('Restore verification');

    await user.click((await screen.findAllByRole('switch'))[0]);
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText('After backup')).toBeTruthy();
    expect(countDatabaseRequests()).toBe(1);
  });

  it('returns to the verification view when Cancel closes an editor that could not load', async () => {
    const verificationConfig = {
      databaseId: shopDatabase.id,
      isAutomaticVerificationEnabled: false,
      scheduleType: 'AFTER_BACKUP' as const,
      schedule: { type: 'DAILY' as const, timeOfDay: '00:00' },
      notificationTriggers: [],
    };
    let verificationConfigRequests = 0;
    server.use(
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      http.get('*/api/v1/verification-configs/:databaseId', () =>
        ++verificationConfigRequests === 2
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json(verificationConfig),
      ),
    );

    const { user } = await renderShopSettings({
      ...shopDatabase,
      settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
    });
    await screen.findByText('Scheduled verification');
    openSettingsSection('Restore verification');

    await screen.findByRole('button', { name: 'Retry' });
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    await screen.findByText('Scheduled verification');
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(verificationConfigRequests).toBe(3);
  });

  it('shows the latest load of a database when an earlier load of it responds last', async () => {
    const earlierShopLoad = createPendingApiResponse();
    let shopRequests = 0;
    server.use(
      http.get('*/api/v1/databases/:databaseId', ({ params }) => {
        if (params.databaseId !== shopDatabase.id)
          return HttpResponse.json(createDatabase('other'));

        return ++shopRequests === 1
          ? earlierShopLoad.response
          : HttpResponse.json({ ...shopDatabase, name: 'Latest shop' });
      }),
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/healthcheck-attempts').handler,
    );
    declareDatabaseDependencies();

    const view = renderScreen(
      <DatabaseComponent {...databaseScreenProps} databaseId={shopDatabase.id} />,
    );
    await waitForCondition(() => expect(shopRequests).toBe(1));

    view.rerender(<DatabaseComponent {...databaseScreenProps} databaseId="other" />);
    view.rerender(<DatabaseComponent {...databaseScreenProps} databaseId={shopDatabase.id} />);
    await view.user.click(await screen.findByText('Config'));
    expect(await screen.findByText('Latest shop')).toBeTruthy();
    const earlierLoadProcessed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', ({ request }) => {
        if (request.url.endsWith(`/databases/${shopDatabase.id}`)) resolve();
      }),
    );
    await act(async () => {
      earlierShopLoad.respond(HttpResponse.json({ ...shopDatabase, name: 'Earlier shop' }));
      await earlierLoadProcessed;
    });

    expect(screen.getByText('Latest shop')).toBeTruthy();
    expect(screen.queryByText('Earlier shop')).toBeNull();
  });
});

describe('DatabaseComponent backup configuration answers', () => {
  afterEach(() => invalidateSession());

  const logicalSettings = {
    type: 'LOGICAL' as const,
    schedule: { type: 'DAILY' as const, timeOfDay: '04:00' },
    retention: {
      policy: 'TIME_PERIOD' as const,
      period: 'MONTH' as const,
      count: 0,
      gfsHours: 0,
      gfsDays: 0,
      gfsWeeks: 0,
      gfsMonths: 0,
      gfsYears: 0,
    },
    isRetryIfFailed: true,
    maxFailedTriesCount: 3,
    notificationTriggers: ['BACKUP_FAILED' as const],
  };
  const physicalSettings = {
    type: 'PHYSICAL' as const,
    fullBackupSchedule: { type: 'HOURLY' as const },
    incrementalBackupSchedule: null,
    retention: {
      policy: 'FULL_BACKUPS' as const,
      chainsCount: 0,
      fullBackups: {
        policy: 'LAST_N' as const,
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
  };

  const buildScreenDatabase = (id: string, type: DatabaseType): Database => ({
    id,
    workspaceId: 'workspace-id',
    name: id,
    settings: createDefaultDatabaseSettings(type),
    notifierIds: [],
  });

  const getBackupsTab = () => screen.getByRole('tab', { name: 'Backups' });

  const EDITOR_KINDS = [
    { kind: 'logical', type: DatabaseType.POSTGRES_LOGICAL, settings: logicalSettings },
    { kind: 'physical', type: DatabaseType.POSTGRES_PHYSICAL, settings: physicalSettings },
  ];

  const declareSavedScreen = (
    databases: Database[],
    settings: typeof logicalSettings | typeof physicalSettings,
  ) => {
    server.use(
      http.get('*/api/v1/databases/:databaseId', ({ params }) =>
        HttpResponse.json(databases.find((database) => database.id === params.databaseId)),
      ),
      http.get('*/api/v1/backup-configs/:databaseId', ({ params }) =>
        HttpResponse.json({
          cleanup: { pendingFileCount: 0, failedFileCount: 0 },
          databaseId: params.databaseId,
          storageId: null,
          isBackupsEnabled: true,
          encryption: 'NONE',
          settings,
        }),
      ),
      mockApi.get('/logical-backups', 200, { items: [] }),
      mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
      mockApi.pending('get', '/healthcheck-attempts').handler,
      mockApi.pending('get', '/healthcheck-configs/{databaseId}').handler,
      mockApi.pending('get', '/verification-configs/{databaseId}').handler,
      mockApi.get('/notifiers', 200, { items: [] }),
      mockApi.get('/storages', 200, { items: [] }),
    );
  };

  const startHeldSave = async (
    databases: Database[],
    settings: typeof logicalSettings | typeof physicalSettings,
    answer: { status: 200 | 403 },
  ) => {
    const saveCall = vi.spyOn(backupConfigApi, 'updateBackupConfig');
    const historyCall = vi.spyOn(backupConfigApi, 'hasBackupHistory');
    const save = createPendingApiResponse();
    const showAlert = vi.fn();
    const onDatabaseChanged = vi.fn();
    vi.stubGlobal('alert', showAlert);
    declareSavedScreen(databases, settings);
    server.use(http.put('*/api/v1/backup-configs/:databaseId', () => save.response));

    const renderedScreen = (databaseId: string) => (
      <DatabaseComponent
        {...databaseScreenProps}
        databaseId={databaseId}
        onDatabaseChanged={onDatabaseChanged}
      />
    );
    const view = renderScreen(renderedScreen(databases[0].id));

    await view.user.click(await screen.findByText('Config'));
    await screen.findByText(databases[0].name);
    openSettingsSection('Backup config');
    const saveButton = await screen.findByRole('button', { name: /Save$/ });
    fireEvent.click(screen.getAllByRole('checkbox')[0]);
    fireEvent.click(saveButton);
    await waitFor(() => expect(saveCall).toHaveBeenCalled());

    const settleAnswer = async () => {
      await act(async () => {
        save.respond(
          answer.status === 200
            ? HttpResponse.json({
                cleanup: { pendingFileCount: 0, failedFileCount: 0 },
                databaseId: databases[0].id,
                storageId: null,
                isBackupsEnabled: true,
                encryption: 'NONE',
                settings,
              })
            : HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
        );
        await Promise.allSettled([saveCall.mock.results[0].value]);
        await Promise.allSettled(historyCall.mock.results.map((call) => call.value));
      });
    };

    return { view, renderedScreen, settleAnswer, showAlert, onDatabaseChanged };
  };

  describe.each(EDITOR_KINDS)('of the $kind editor', ({ type, settings }) => {
    it.each([
      { outcome: 'successful', status: 200 as const },
      { outcome: 'refused', status: 403 as const },
    ])(
      'keeps the new database and reports nothing for a $outcome answer after another database is selected',
      async ({ status }) => {
        const firstDatabase = buildScreenDatabase('first-database', type);
        const secondDatabase = buildScreenDatabase('second-database', type);
        const { view, renderedScreen, settleAnswer, showAlert, onDatabaseChanged } =
          await startHeldSave([firstDatabase, secondDatabase], settings, { status });

        view.rerender(renderedScreen(secondDatabase.id));
        await screen.findByText(secondDatabase.name);
        await settleAnswer();

        expect(screen.getByText(secondDatabase.name)).toBeTruthy();
        expect(screen.queryByText(firstDatabase.name)).toBeNull();
        expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
        expect(onDatabaseChanged).not.toHaveBeenCalled();
        expect(showAlert).not.toHaveBeenCalled();
      },
    );

    it.each([
      { outcome: 'successful', status: 200 as const },
      { outcome: 'refused', status: 403 as const },
    ])('shows no feedback for a $outcome answer after a tab switch', async ({ status }) => {
      const database = buildScreenDatabase('first-database', type);
      const { view, settleAnswer, showAlert, onDatabaseChanged } = await startHeldSave(
        [database],
        settings,
        { status },
      );

      await view.user.click(screen.getByText('Backups'));
      await screen.findByRole('heading', { name: 'Backups' });
      await settleAnswer();

      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
      expect(getBackupsTab().getAttribute('aria-selected')).toBe('true');
      expect(onDatabaseChanged).not.toHaveBeenCalled();
    });

    it.each([
      { outcome: 'successful', status: 200 as const },
      { outcome: 'refused', status: 403 as const },
    ])('shows no feedback for a $outcome answer after a session change', async ({ status }) => {
      activateSession({ token: 'first-token', userId: 'first-user' });
      const database = buildScreenDatabase('first-database', type);
      const { settleAnswer, showAlert, onDatabaseChanged } = await startHeldSave(
        [database],
        settings,
        { status },
      );

      act(() => {
        invalidateSession();
      });
      await settleAnswer();

      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
      expect(onDatabaseChanged).not.toHaveBeenCalled();
      expect(screen.getByRole('button', { name: /Save$/ })).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeTruthy();
    });
  });

  const buildCompletedBackup = (databaseId: string) => ({
    id: 'completed-backup',
    databaseId,
    storageId: 'storage-id',
    status: 'COMPLETED' as const,
    sizeMb: 12,
    rawDatabaseSizeMb: 40,
    durationMs: 3_000,
    encryption: 'NONE' as const,
    verificationStatus: 'NOT_VERIFIED' as const,
    createdAt: '2026-10-27T10:00:00Z',
  });

  const buildStartedVerification = (databaseId: string) => ({
    id: 'verification-id',
    databaseId,
    backupId: 'completed-backup',
    trigger: 'MANUAL' as const,
    status: 'COMPLETED' as const,
    attemptCount: 1,
    createdAt: '2026-10-27T10:00:00Z',
  });

  it('starts the check of the chosen backup, reports it and opens the Verifications tab after Queue check is accepted', async () => {
    const shownDatabase = buildScreenDatabase('shown-database', DatabaseType.POSTGRES_LOGICAL);
    declareSavedScreen([shownDatabase], logicalSettings);
    server.use(
      mockApi.get('/logical-backups', 200, { items: [buildCompletedBackup(shownDatabase.id)] }),
      mockApi.post('/verifications', 202, buildStartedVerification(shownDatabase.id)),
      mockApi.get('/verifications', 200, { items: [] }),
      mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
    );
    renderScreen(<DatabaseComponent {...databaseScreenProps} databaseId={shownDatabase.id} />);
    await screen.findAllByText('Successful');

    fireEvent.click(screen.getAllByRole('button', { name: /^Verify restore/ })[0]);
    fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));

    expect(await screen.findByText('Restore check queued')).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Verifications' }).getAttribute('aria-selected')).toBe(
      'true',
    );
    expect(
      await screen.findByRole('heading', { name: DICTIONARIES.en.verification.runs.title }),
    ).toBeTruthy();

    const startRequests = server
      .getRequests()
      .filter(
        (request) =>
          request.method === 'POST' && new URL(request.url).pathname === '/api/v1/verifications',
      );
    expect(startRequests).toHaveLength(1);
    const startRequest = await inspectApiRequest(startRequests[0], {
      method: 'post',
      path: '/verifications',
    });
    expect(startRequest.body).toEqual({ backupId: 'completed-backup' });
  });

  it('keeps the open tab and shows nothing when a verification start answers after a switch', async () => {
    const firstDatabase = buildScreenDatabase('first-database', DatabaseType.POSTGRES_LOGICAL);
    const secondDatabase = buildScreenDatabase('second-database', DatabaseType.POSTGRES_LOGICAL);
    const startCall = vi.spyOn(verificationRunsApi, 'startVerification');
    const start = mockApi.delayed({
      method: 'post',
      path: '/verifications',
      status: 202,
      body: buildStartedVerification(firstDatabase.id),
    });
    declareSavedScreen([firstDatabase, secondDatabase], logicalSettings);
    server.use(
      http.get('*/api/v1/logical-backups', ({ request }) =>
        HttpResponse.json({
          items:
            new URL(request.url).searchParams.get('databaseId') === firstDatabase.id
              ? [buildCompletedBackup(firstDatabase.id)]
              : [],
        }),
      ),
      start.handler,
    );
    const renderedScreen = (databaseId: string) => (
      <DatabaseComponent {...databaseScreenProps} databaseId={databaseId} />
    );
    const view = renderScreen(renderedScreen(firstDatabase.id));
    await screen.findAllByText('Successful');

    fireEvent.click(screen.getAllByRole('button', { name: /^Verify restore/ })[0]);
    fireEvent.click(await screen.findByRole('button', { name: 'Queue check' }));
    await waitFor(() => expect(startCall).toHaveBeenCalled());

    view.rerender(renderedScreen(secondDatabase.id));
    await screen.findAllByText('No backups yet');
    await releaseAndSettle(startCall, start.release);

    expect(getBackupsTab().getAttribute('aria-selected')).toBe('true');
    expect(screen.queryByText('Restore check queued')).toBeNull();
    expect(screen.getByRole('heading', { name: 'Backups' })).toBeTruthy();
  });

  describe('when the selected database lacks the open tab', () => {
    const declareTabRequests = (databases: Database[], settings = logicalSettings) => {
      declareSavedScreen(databases, settings);
      server.use(
        mockApi.pending('get', '/verifications').handler,
        mockApi.pending('get', '/verification-agent-counts').handler,
        mockApi.pending('get', '/wal-streams/{databaseId}').handler,
        mockApi.pending('get', '/wal-streams/{databaseId}/statistics').handler,
      );
    };

    const renderSelectedDatabase = (databaseId: string) => (
      <DatabaseComponent {...databaseScreenProps} databaseId={databaseId} />
    );

    const expectBackupsTabShown = async () => {
      expect(await screen.findByRole('button', { name: /Make backup right now/ })).toBeTruthy();
      expect(getBackupsTab().getAttribute('aria-selected')).toBe('true');
    };

    it('shows the backups of a MySQL database selected on the verifications tab', async () => {
      const logicalDatabase = buildScreenDatabase('logical', DatabaseType.POSTGRES_LOGICAL);
      const mysqlDatabase = buildScreenDatabase('mysql', DatabaseType.MYSQL);
      declareTabRequests([logicalDatabase, mysqlDatabase]);
      const view = renderScreen(renderSelectedDatabase(logicalDatabase.id));
      await view.user.click(await screen.findByText('Verifications'));

      view.rerender(renderSelectedDatabase(mysqlDatabase.id));

      await expectBackupsTabShown();
    });

    it('shows the backups of a database without monitoring selected on the monitoring tab', async () => {
      const walDatabase: Database = {
        ...buildScreenDatabase('wal', DatabaseType.POSTGRES_PHYSICAL),
        settings: {
          ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
          backupType: PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM,
        } as Database['settings'],
      };
      const mysqlDatabase = buildScreenDatabase('mysql', DatabaseType.MYSQL);
      declareTabRequests([walDatabase, mysqlDatabase]);
      const view = renderScreen(renderSelectedDatabase(walDatabase.id));
      await view.user.click(await screen.findByText('Monitoring'));

      view.rerender(renderSelectedDatabase(mysqlDatabase.id));

      await expectBackupsTabShown();
    });

    it('reopens verifications when a logical database is selected again', async () => {
      const logicalDatabase = buildScreenDatabase('logical', DatabaseType.POSTGRES_LOGICAL);
      const mysqlDatabase = buildScreenDatabase('mysql', DatabaseType.MYSQL);
      declareTabRequests([logicalDatabase, mysqlDatabase]);
      const view = renderScreen(renderSelectedDatabase(logicalDatabase.id));
      await view.user.click(await screen.findByText('Verifications'));

      view.rerender(renderSelectedDatabase(mysqlDatabase.id));
      await screen.findByText('Config');
      view.rerender(renderSelectedDatabase(logicalDatabase.id));

      expect(
        (await screen.findByRole('tab', { name: 'Verifications' })).getAttribute('aria-selected'),
      ).toBe('true');
    });
  });
});
