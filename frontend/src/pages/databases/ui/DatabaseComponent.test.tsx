import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { http, HttpResponse } from 'msw';
import { createRef } from 'react';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import { DICTIONARIES, useLocale } from '../../../shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
// @vitest-environment happy-dom
import { DatabaseComponent } from './DatabaseComponent';

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
    await vi.waitFor(() => expect(server.getRequests()).toHaveLength(1));

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
    await vi.waitFor(() => expect(server.getRequests()).toHaveLength(1));
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
    const nameTitle = await screen.findByText('Shop');

    fireEvent.click(nameTitle.querySelector('img[src="/icons/pen-gray.svg"]')!.parentElement!);
    const nameInput = screen.getByPlaceholderText(DICTIONARIES.en.databases.config.namePlaceholder);

    await user.clear(nameInput);
    await user.type(nameInput, 'Unsaved database name');
    await user.click(screen.getByRole('button', { name: 'Change language' }));

    expect(await screen.findByDisplayValue('Unsaved database name')).toBeTruthy();
    expect(screen.getByPlaceholderText(DICTIONARIES.fr.databases.config.namePlaceholder)).toBe(
      nameInput,
    );
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
    expect(screen.queryByRole('button', { name: 'copy' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'arrow-right' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'delete' })).toBeNull();
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
    await vi.waitFor(() => expect(shopRequests).toBe(1));

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
