import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, type RequestHandler, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import { API_BASE_PATH } from '../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
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
      ).toHaveLength(3),
    );

    view.unmount();
    const requestsFinished = new Promise<void>((resolve) => {
      let completedRequests = 0;
      mockServer.events.on('request:end', ({ request }) => {
        if (
          /\/(backup-configs|healthcheck-configs)\//.test(request.url) &&
          ++completedRequests === 3
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
        ++backupConfigRequests <= 2
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

    await waitFor(() => expect(backupConfigRequests).toBe(3));
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
    await waitFor(() => expect(backupConfigRequests).toBe(earlierBackupConfigLoads.length));

    view.rerender(renderConfig(otherDatabase));
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
});
