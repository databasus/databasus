// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import { HttpResponse, type RequestHandler, http } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entity/databases';
import { API_BASE_PATH } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
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
    http.all(`*${API_BASE_PATH}/*`, () => new Promise<never>(() => {})),
  );

  return renderScreen(
    <DatabaseConfigComponent
      database={savedDatabase}
      setDatabase={() => {}}
      onDatabaseChanged={() => {}}
      onDatabaseDeleted={() => {}}
      editDatabase={undefined}
      setEditDatabase={() => {}}
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
});
