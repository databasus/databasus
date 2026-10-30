// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { DatabaseType, createDefaultDatabaseSettings } from '../../../entities/databases';
import { DICTIONARIES, useLocale } from '../../../shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../shared/lib/test/pendingApiResponse';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { DatabaseComponent } from './DatabaseComponent';

const server = setUpMockApi();
const DatabaseScreen = () => {
  const { setLocale } = useLocale();
  return (
    <>
      <button onClick={() => setLocale('fr')}>Change language</button>
      <DatabaseComponent
        databaseId="shop-id"
        contentHeight={800}
        isCanManageDBs
        onDatabaseChanged={() => {}}
        onDatabaseDeleted={() => {}}
      />
    </>
  );
};

const databaseScreenProps = {
  contentHeight: 800,
  isCanManageDBs: true,
  onDatabaseChanged: () => {},
  onDatabaseDeleted: () => {},
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
});
