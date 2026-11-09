// @vitest-environment happy-dom
import { screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, describe, expect, it } from 'vitest';

import type { SftpStorageSettings, Storage } from '@/entities/storages';
import { StoragesComponent } from '@/pages/storages/ui/StoragesComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

beforeEach(() => {
  mockServer.use(mockApi.get('/storages/{storageId}/backup-usage', 200, { totalUsageMb: 0 }));
});
const storage: Storage = {
  id: 'sftp-id',
  workspaceId: 'workspace-id',
  name: 'SFTP archive',
  settings: {
    type: 'SFTP',
    host: 'sftp.example.com',
    port: 22,
    username: 'backup',
    authMethods: ['PASSWORD', 'PRIVATE_KEY'],
    isSkipHostKeyVerify: false,
  },
};

const renderStoragesScreen = () =>
  renderScreen(
    <StoragesComponent
      workspace={{
        id: 'workspace-id',
        name: 'Workspace',
        currentUserRole: 'WORKSPACE_OWNER',
        createdAt: '2026-10-01T00:00:00Z',
      }}
      isCanManageStorages
    />,
  );

const openStorageSettings = async (user: ReturnType<typeof renderScreen>['user']) => {
  const settingsHeader = (await screen.findByText('Storage settings')).parentElement!;
  await user.click(settingsHeader.querySelector('img')!);
};

describe('StoragesComponent', () => {
  it('offers every supported provider when creating storage', async () => {
    mockServer.use(mockApi.get('/storages', 200, { items: [] }));
    const { user } = renderStoragesScreen();

    await user.click(await screen.findByRole('button', { name: 'Add storage' }));
    await user.click(screen.getByRole('combobox', { name: 'Type' }));

    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'Local storage',
      'S3',
      'NAS',
      'Azure Blob Storage',
      'FTP',
      'SFTP',
    ]);
  });

  it('searches storages like notifiers when a name substring uses different casing', async () => {
    const storages = [
      storage,
      ...Array.from({ length: 4 }, (_, index) => ({
        ...storage,
        id: `archive-${index}`,
        name: `Other archive ${index}`,
      })),
    ];
    mockServer.use(
      mockApi.get('/storages', 200, { items: storages }),
      mockApi.get('/storages/{storageId}', 200, storage),
    );
    const { user } = renderStoragesScreen();
    await screen.findByRole('button', { name: storage.name });

    await user.type(screen.getByRole('textbox', { name: 'Search storages...' }), 'SfTp');

    expect(screen.getByRole('button', { name: storage.name })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Other archive 0' })).toBeNull();
    expect(
      mockServer
        .getRequests()
        .filter((request) => new URL(request.url).pathname.endsWith('/storages')),
    ).toHaveLength(1);
  });

  it('retains newer credentials when saving settings refreshes the storage list', async () => {
    const pendingSave = createPendingApiResponse();
    const pendingList = createPendingApiResponse();
    const sentUpdates: ApiComponents['schemas']['StorageUpdate'][] = [];
    let listRequests = 0;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/storages`, () => {
        listRequests++;
        return listRequests === 2 ? pendingList.response : HttpResponse.json({ items: [storage] });
      }),
      mockApi.get('/storages/{storageId}', 200, storage),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      http.put(`*${API_BASE_PATH}/storages/sftp-id`, async ({ request }) => {
        sentUpdates.push((await request.json()) as ApiComponents['schemas']['StorageUpdate']);
        return sentUpdates.length === 1 ? pendingSave.response : HttpResponse.json(storage);
      }),
    );

    const { user } = renderStoragesScreen();
    await openStorageSettings(user);
    const passwordInput = screen.getByLabelText('Password', { exact: true });
    const privateKeyInput = screen.getByRole('textbox', { name: 'Private key' });

    await user.type(passwordInput, 'submitted-password');
    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(1));

    await user.clear(passwordInput);
    await user.type(passwordInput, 'newer-password');
    await user.type(privateKeyInput, 'newer-private-key');
    pendingSave.respond(HttpResponse.json(storage));

    await waitFor(() => expect(listRequests).toBe(2));

    expect(screen.getByLabelText('Password', { exact: true })).toBe(passwordInput);
    expect((passwordInput as HTMLInputElement).value).toBe('newer-password');
    expect((privateKeyInput as HTMLTextAreaElement).value).toBe('newer-private-key');

    pendingList.respond(
      HttpResponse.json({
        items: [storage, { ...storage, id: 'other-sftp-id', name: 'Other SFTP archive' }],
      }),
    );
    await screen.findByText('Other SFTP archive');

    expect(screen.getByLabelText('Password', { exact: true })).toBe(passwordInput);
    expect((passwordInput as HTMLInputElement).value).toBe('newer-password');
    expect((privateKeyInput as HTMLTextAreaElement).value).toBe('newer-private-key');

    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(2));

    expect(sentUpdates[1].settings).toMatchObject({
      password: 'newer-password',
      privateKey: 'newer-private-key',
      authMethods: ['PASSWORD', 'PRIVATE_KEY'],
    });
  });

  it.each([
    {
      isSwitchingMethods: true,
      scenario: 'renames with the saved method set after switching from password to private key',
    },
    {
      isSwitchingMethods: false,
      scenario: 'retains the saved path when renaming after a path change',
    },
  ])('$scenario', async ({ isSwitchingMethods }) => {
    const initialSettings: SftpStorageSettings = {
      type: 'SFTP',
      host: 'sftp.example.com',
      port: 22,
      username: 'backup',
      authMethods: ['PASSWORD'],
      path: 'original-path',
      isSkipHostKeyVerify: false,
    };
    const initialStorage: Storage = { ...storage, settings: initialSettings };
    const savedStorage: Storage = {
      ...initialStorage,
      settings: {
        ...initialSettings,
        authMethods: isSwitchingMethods ? ['PRIVATE_KEY'] : ['PASSWORD'],
        path: isSwitchingMethods ? 'original-path' : 'saved-path',
      },
    };
    const sentUpdates: ApiComponents['schemas']['StorageUpdate'][] = [];
    let listRequests = 0;
    let persistedStorage = initialStorage;

    mockServer.use(
      http.get(`*${API_BASE_PATH}/storages`, () => {
        listRequests++;
        return HttpResponse.json({ items: [persistedStorage] });
      }),
      http.get(`*${API_BASE_PATH}/storages/:storageId`, () => HttpResponse.json(persistedStorage)),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      http.put(`*${API_BASE_PATH}/storages/sftp-id`, async ({ request }) => {
        const sentUpdate = (await request.json()) as ApiComponents['schemas']['StorageUpdate'];
        sentUpdates.push(sentUpdate);
        persistedStorage = { ...savedStorage, name: sentUpdate.name };
        return HttpResponse.json(persistedStorage);
      }),
    );
    const { user } = renderStoragesScreen();
    await openStorageSettings(user);

    if (isSwitchingMethods) {
      await user.click(screen.getByRole('combobox', { name: 'Auth method' }));
      await user.click(await screen.findByRole('option', { name: 'Private key' }));
      await user.type(screen.getByRole('textbox', { name: 'Private key' }), 'replacement-key');
    } else {
      await user.clear(screen.getByRole('textbox', { name: 'Path' }));
      await user.type(screen.getByRole('textbox', { name: 'Path' }), 'saved-path');
    }

    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(listRequests).toBe(2));

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await screen.findByRole('button', { name: 'Test connection' });

    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
    await user.clear(screen.getByPlaceholderText('Enter name...'));
    await user.type(screen.getByPlaceholderText('Enter name...'), 'Renamed archive');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(2));
    expect(sentUpdates[1]).toEqual({ name: 'Renamed archive', settings: savedStorage.settings });
    await waitFor(() => expect(screen.getAllByText('Renamed archive')).toHaveLength(2));
  });

  it('closes the old editing session when another storage is selected', async () => {
    const otherStorage: Storage = {
      ...storage,
      id: 'other-sftp-id',
      name: 'Other SFTP archive',
      settings: {
        type: 'SFTP',
        host: 'other.example.com',
        port: 22,
        username: 'other-backup',
        authMethods: ['PRIVATE_KEY'],
        isSkipHostKeyVerify: false,
      },
    };
    mockServer.use(
      mockApi.get('/storages', 200, { items: [storage, otherStorage] }),
      http.get(`*${API_BASE_PATH}/storages/:storageId`, ({ params }) =>
        HttpResponse.json(params.storageId === otherStorage.id ? otherStorage : storage),
      ),
    );
    const { user } = renderStoragesScreen();
    await openStorageSettings(user);

    await user.type(screen.getByLabelText('Password', { exact: true }), 'unsaved-password');
    await user.click(screen.getByText('Other SFTP archive', { exact: true }));
    await screen.findByText('other.example.com');

    expect(screen.queryByLabelText('Password', { exact: true })).toBeNull();
    expect(screen.queryByRole('textbox', { name: 'Private key' })).toBeNull();

    await openStorageSettings(user);

    expect(screen.queryByLabelText('Password', { exact: true })).toBeNull();
    expect(
      (screen.getByRole('textbox', { name: 'Private key' }) as HTMLTextAreaElement).value,
    ).toBe('');
    expect((screen.getByRole('textbox', { name: 'Host' }) as HTMLInputElement).value).toBe(
      'other.example.com',
    );
  });
});
