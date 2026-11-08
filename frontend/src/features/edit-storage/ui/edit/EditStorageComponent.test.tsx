// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Storage, StorageSettings } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage/ui/edit/EditStorageComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { useLocale } from '@/shared/i18n';
import { en } from '@/shared/i18n/locales/en';
import { ru } from '@/shared/i18n/locales/ru';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { watchNoticeTimer } from '@/shared/lib/test/noticeTimer';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

afterEach(() => {
  vi.restoreAllMocks();
});

const savedS3Storage = {
  id: 'storage-id',
  workspaceId: 'workspace-id',
  name: 'Offsite backups',
  settings: {
    type: 'S3',
    s3Bucket: 'databasus-backups',
    s3Region: 'eu-central-1',
    isVirtualHostedStyle: false,
    isSkipTlsVerify: false,
  },
} satisfies Storage;

const savedStorageEditor = (storage: Storage, workspaceId = 'workspace-id') => (
  <EditStorageComponent
    workspaceId={workspaceId}
    isShowClose={false}
    onClose={() => {}}
    isShowName
    editingStorage={storage}
    onChanged={() => {}}
  />
);

const renderSavedStorageForm = (storage: Storage) => renderScreen(savedStorageEditor(storage));

// Records every update the form sends and answers it with the storage the update describes.
const mockUpdate = () => {
  const sentUpdates: ApiComponents['schemas']['StorageUpdate'][] = [];
  mockServer.use(
    http.put(`*${API_BASE_PATH}/storages/:storageId`, async ({ request }) => {
      const sentUpdate = (await request.json()) as ApiComponents['schemas']['StorageUpdate'];
      sentUpdates.push(sentUpdate);

      return HttpResponse.json({
        ...savedS3Storage,
        name: sentUpdate.name,
        settings: sentUpdate.settings,
      });
    }),
  );

  return sentUpdates;
};

describe('EditStorageComponent', () => {
  it('shows the type of a saved storage as text instead of a type selector', () => {
    renderSavedStorageForm(savedS3Storage);
    expect(screen.getByText('S3')).toBeTruthy();
    expect(screen.queryByRole('combobox')).toBeNull();
  });

  it('leaves no notice timer running after the form unmounts', async () => {
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    const noticeTimer = watchNoticeTimer(4_000);
    const view = renderSavedStorageForm(savedS3Storage);

    fireEvent.change(screen.getByDisplayValue('Offsite backups'), {
      target: { value: 'Renamed backups' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    await waitFor(() => expect(noticeTimer.getStartedCount()).toBeGreaterThan(0));
    view.unmount();

    expect(noticeTimer.getLeftRunningCount()).toBe(0);
  });

  it('sends no keys when a renamed S3 storage is saved', async () => {
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    const sentUpdates = mockUpdate();
    renderSavedStorageForm(savedS3Storage);

    fireEvent.change(screen.getByDisplayValue('Offsite backups'), {
      target: { value: 'Renamed backups' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0]).toEqual({
      name: 'Renamed backups',
      settings: {
        type: 'S3',
        s3Bucket: 'databasus-backups',
        s3Region: 'eu-central-1',
        isVirtualHostedStyle: false,
        isSkipTlsVerify: false,
      },
    });
  });

  it.each(['COLD', 'DEEP_ARCHIVE'])(
    'preserves %s when only the storage name changes',
    async (storageClass) => {
      mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
      const sentUpdates = mockUpdate();
      renderSavedStorageForm({
        ...savedS3Storage,
        settings: { ...savedS3Storage.settings, s3StorageClass: storageClass },
      });

      expect(screen.getByDisplayValue(storageClass)).toBeTruthy();

      fireEvent.change(screen.getByDisplayValue('Offsite backups'), {
        target: { value: 'Renamed backups' },
      });
      fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(sentUpdates).toHaveLength(1));
      expect(sentUpdates[0].settings).toMatchObject({ s3StorageClass: storageClass });
    },
  );

  it('saves a custom class when it is entered in the storage editor', async () => {
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    const sentUpdates = mockUpdate();
    renderSavedStorageForm(savedS3Storage);

    fireEvent.click(screen.getByText('Advanced settings'));
    fireEvent.change(screen.getByRole('combobox', { name: 'Storage class' }), {
      target: { value: 'CUSTOM_ARCHIVE' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0].settings).toMatchObject({ s3StorageClass: 'CUSTOM_ARCHIVE' });
  });

  it("shows the storage's error when the connection test fails", async () => {
    mockServer.use(
      mockApi.post('/storage-tests', 200, {
        isConnected: false,
        error: 'the bucket databasus-backups does not exist',
      }),
    );
    renderSavedStorageForm(savedS3Storage);

    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));

    expect(
      await screen.findByText(
        'The storage could not be reached: the bucket databasus-backups does not exist',
      ),
    ).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });
});

it('keeps credentials and the draft out of Google authorization navigation', () => {
  const popup = vi.spyOn(window, 'open').mockReturnValue({} as Window);
  renderScreen(
    <EditStorageComponent
      workspaceId="workspace-id"
      isShowClose={false}
      onClose={() => {}}
      isShowName
      onChanged={() => {}}
      draft={{
        name: 'Private draft',
        settings: {
          type: 'GOOGLE_DRIVE',
          clientId: 'public-client',
          clientSecret: 'synthetic-client-secret',
        },
      }}
    />,
  );

  fireEvent.click(screen.getByRole('button', { name: 'Authorize' }));
  const authorizationUrl = new URL(String(popup.mock.calls[0][0]));

  expect(authorizationUrl.searchParams.get('state')).toMatch(/^[a-f0-9]{64}$/);
  expect(decodeURIComponent(authorizationUrl.href)).not.toContain('synthetic-client-secret');
  expect(decodeURIComponent(authorizationUrl.href)).not.toContain('Private draft');
});

it.each(
  ['settings', 'workspace', 'storage', 'session', 'unmount'].flatMap((changedContext) =>
    ['success', 'network failure'].map((outcome) => ({ changedContext, outcome })),
  ),
)(
  'ignores a connection test $outcome after the $changedContext changes',
  async ({ changedContext, outcome }) => {
    activateSession({ token: 'original-session', userId: 'owner-id' });
    const previousTest = createPendingApiResponse();
    const currentTest = createPendingApiResponse();
    const fetchObservation = vi.spyOn(globalThis, 'fetch');
    const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    let testRequests = 0;

    mockServer.use(
      http.post(`*${API_BASE_PATH}/storage-tests`, () => {
        testRequests++;
        return testRequests === 1 ? previousTest.response : currentTest.response;
      }),
    );

    const screenView = renderSavedStorageForm(savedS3Storage);

    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));

    await waitFor(() => expect(testRequests).toBe(1));

    if (changedContext === 'settings')
      fireEvent.change(screen.getByDisplayValue('databasus-backups'), {
        target: { value: 'replacement-bucket' },
      });

    if (changedContext === 'workspace')
      screenView.rerender(savedStorageEditor(savedS3Storage, 'replacement-workspace'));

    if (changedContext === 'storage')
      screenView.rerender(savedStorageEditor({ ...savedS3Storage, id: 'replacement-storage' }));

    if (changedContext === 'session')
      act(() => activateSession({ token: 'replacement-session', userId: 'owner-id' }));

    if (changedContext === 'unmount') screenView.unmount();
    else {
      fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
        target: { value: 'Current draft' },
      });
      fireEvent.click(await screen.findByRole('button', { name: /Test connection/ }));

      await waitFor(() => expect(testRequests).toBe(2));
    }
    await act(async () => {
      previousTest.respond(
        outcome === 'success' ? HttpResponse.json({ isConnected: true }) : HttpResponse.error(),
      );
      await fetchObservation.mock.results[0].value.catch(() => undefined);
    });

    expect(alert).not.toHaveBeenCalled();

    const currentEditor = screen.queryByRole('region', { name: 'Storage settings' });

    expect(currentEditor && within(currentEditor).queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();

    if (changedContext !== 'unmount') {
      expect(
        (screen.getByRole('button', { name: /Test connection/ }) as HTMLButtonElement).disabled,
      ).toBe(true);

      currentTest.respond(HttpResponse.json({ isConnected: true }));

      expect(await screen.findByRole('button', { name: 'Save' })).toBeTruthy();
    }
  },
);

const savedSftpStorage = (authMethods: ('PASSWORD' | 'PRIVATE_KEY')[]): Storage => ({
  id: 'sftp-id',
  workspaceId: 'workspace-id',
  name: 'SFTP archive',
  settings: {
    type: 'SFTP',
    host: 'sftp.example.com',
    port: 22,
    username: 'backup',
    authMethods,
    isSkipHostKeyVerify: false,
  },
});

const selectSftpMethods = async (user: ReturnType<typeof renderScreen>['user'], label: string) => {
  await user.click(screen.getByRole('combobox', { name: 'Auth method' }));
  await user.click(await screen.findByRole('option', { name: label }));
};

describe('SFTP credentials', () => {
  it('requires another test when a completed overlapping test used the previous saved credentials', async () => {
    const storage = savedSftpStorage(['PASSWORD']);
    const pendingSave = mockApi.delayed({
      method: 'put',
      path: '/storages/{storageId}',
      status: 200,
      body: storage,
    });
    mockServer.use(pendingSave.handler, mockApi.post('/storage-tests', 200, { isConnected: true }));
    const { user } = renderSavedStorageForm(storage);
    const passwordInput = screen.getByLabelText('Password', { exact: true });

    fireEvent.change(passwordInput, { target: { value: 'replacement-password' } });
    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(
        1,
      ),
    );

    fireEvent.change(passwordInput, { target: { value: '' } });
    fireEvent.change(screen.getByLabelText('Path'), { target: { value: '/new-path' } });
    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await screen.findByRole('button', { name: 'Save' });

    pendingSave.release();

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Test connection' })).toBeTruthy(),
    );
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    expect((passwordInput as HTMLInputElement).value).toBe('');
    expect(screen.getByDisplayValue('/new-path')).toBeTruthy();

    await user.click(screen.getByRole('button', { name: 'Test connection' }));

    expect(await screen.findByRole('button', { name: 'Save' })).toBeTruthy();
    expect(mockServer.getRequests().filter((request) => request.method === 'POST')).toHaveLength(3);
  });

  it.each([
    {
      outcome: 'success',
      scenario: 'allows retesting a newer draft after a save invalidates a pending successful test',
    },
    {
      outcome: 'network failure',
      scenario: 'allows retesting a newer draft after a save invalidates a pending failed test',
    },
  ])('$scenario', async ({ outcome }) => {
    const pendingSave = createPendingApiResponse();
    const pendingTest = createPendingApiResponse();
    const fetchObservation = vi.spyOn(globalThis, 'fetch');
    const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const sentUpdates: ApiComponents['schemas']['StorageUpdate'][] = [];
    let testRequests = 0;
    const storage = savedSftpStorage(['PASSWORD']);

    mockServer.use(
      http.post(`*${API_BASE_PATH}/storage-tests`, () => {
        testRequests++;
        return testRequests === 2 ? pendingTest.response : HttpResponse.json({ isConnected: true });
      }),
      http.put(`*${API_BASE_PATH}/storages/sftp-id`, async ({ request }) => {
        sentUpdates.push((await request.json()) as ApiComponents['schemas']['StorageUpdate']);
        return sentUpdates.length === 1 ? pendingSave.response : HttpResponse.json(storage);
      }),
    );

    const { user } = renderSavedStorageForm(storage);
    const passwordInput = screen.getByLabelText('Password', { exact: true });

    await user.type(passwordInput, 'submitted-password');
    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(1));

    const savingFetch =
      fetchObservation.mock.results[fetchObservation.mock.results.length - 1].value;

    await user.clear(passwordInput);
    await user.type(passwordInput, 'newer-password');
    await user.click(screen.getByRole('button', { name: 'Test connection' }));

    await waitFor(() => expect(testRequests).toBe(2));

    const testingFetch =
      fetchObservation.mock.results[fetchObservation.mock.results.length - 1].value;

    await act(async () => {
      pendingSave.respond(HttpResponse.json(storage));
      await savingFetch;
    });

    await waitFor(() =>
      expect(
        (screen.getByRole('button', { name: /Test connection/ }) as HTMLButtonElement).disabled,
      ).toBe(false),
    );
    expect((passwordInput as HTMLInputElement).value).toBe('newer-password');

    await act(async () => {
      pendingTest.respond(
        outcome === 'success' ? HttpResponse.json({ isConnected: true }) : HttpResponse.error(),
      );
      await testingFetch.catch(() => undefined);
    });

    expect(alert).not.toHaveBeenCalled();

    const currentEditor = screen.queryByRole('region', { name: 'Storage settings' });

    expect(currentEditor && within(currentEditor).queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();

    await user.click(screen.getByRole('button', { name: /Test connection/ }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(2));

    expect(sentUpdates[1].settings).toMatchObject({ password: 'newer-password' });
  });

  it.each([
    { methods: ['PASSWORD'] as ('PASSWORD' | 'PRIVATE_KEY')[], hasPassword: true, hasKey: false },
    {
      methods: ['PRIVATE_KEY'] as ('PASSWORD' | 'PRIVATE_KEY')[],
      hasPassword: false,
      hasKey: true,
    },
    {
      methods: ['PASSWORD', 'PRIVATE_KEY'] as ('PASSWORD' | 'PRIVATE_KEY')[],
      hasPassword: true,
      hasKey: true,
    },
  ])(
    'preserves selected saved credentials when renaming $methods',
    async ({ methods, hasPassword, hasKey }) => {
      const storage = savedSftpStorage(methods);
      const sentUpdates: ApiComponents['schemas']['StorageUpdate'][] = [];
      mockServer.use(
        mockApi.post('/storage-tests', 200, { isConnected: true }),
        http.put(`*${API_BASE_PATH}/storages/sftp-id`, async ({ request }) => {
          sentUpdates.push((await request.json()) as ApiComponents['schemas']['StorageUpdate']);
          return HttpResponse.json({ ...storage, name: 'Renamed archive' });
        }),
      );
      const { user } = renderSavedStorageForm(storage);

      expect(!!screen.queryByLabelText('Password', { exact: true })).toBe(hasPassword);
      expect(!!screen.queryByRole('textbox', { name: 'Private key' })).toBe(hasKey);

      if (hasPassword)
        expect(screen.getByText('A password is saved. Leave blank to keep it.')).toBeTruthy();

      if (hasKey)
        expect(screen.getByText('A private key is saved. Leave blank to keep it.')).toBeTruthy();

      await user.clear(screen.getByRole('textbox', { name: 'Name' }));
      await user.type(screen.getByRole('textbox', { name: 'Name' }), 'Renamed archive');
      await user.click(screen.getByRole('button', { name: 'Test connection' }));
      await user.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(sentUpdates).toHaveLength(1));

      expect(sentUpdates[0].settings).toEqual(storage.settings);
    },
  );

  it('requires a new credential and invalidates approval when methods change', async () => {
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    const { user } = renderSavedStorageForm(savedSftpStorage(['PASSWORD']));

    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: 'Renamed SFTP' },
    });

    await user.click(screen.getByRole('button', { name: 'Test connection' }));

    expect(await screen.findByRole('button', { name: 'Save' })).toBeTruthy();

    await selectSftpMethods(user, 'Private key');

    expect(screen.queryByLabelText('Password', { exact: true })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    expect(screen.getByText('Enter a private key for this method.')).toBeTruthy();
    expect(
      (screen.getByRole('button', { name: 'Test connection' }) as HTMLButtonElement).disabled,
    ).toBe(true);

    await user.type(screen.getByRole('textbox', { name: 'Private key' }), 'replacement-key');

    expect(
      (screen.getByRole('button', { name: 'Test connection' }) as HTMLButtonElement).disabled,
    ).toBe(false);

    await selectSftpMethods(user, 'Password');
    await selectSftpMethods(user, 'Private key');

    expect(
      (screen.getByRole('textbox', { name: 'Private key' }) as HTMLTextAreaElement).value,
    ).toBe('');
    expect(
      (screen.getByRole('button', { name: 'Test connection' }) as HTMLButtonElement).disabled,
    ).toBe(true);
  });

  it('retains persisted credentials when an unsaved method change is undone', async () => {
    const { user } = renderSavedStorageForm(savedSftpStorage(['PASSWORD', 'PRIVATE_KEY']));

    await user.type(screen.getByLabelText('Password', { exact: true }), 'discarded-password');
    await selectSftpMethods(user, 'Private key');
    await selectSftpMethods(user, 'Password and private key');

    expect((screen.getByLabelText('Password', { exact: true }) as HTMLInputElement).value).toBe('');
    expect(
      (screen.getByRole('button', { name: 'Test connection' }) as HTMLButtonElement).disabled,
    ).toBe(false);
  });

  it.each([
    {
      hasNewerInput: false,
      scenario: 'clears submitted credentials when the input is unchanged during saving',
    },
    {
      hasNewerInput: true,
      scenario: 'preserves newer credentials when the input changes during saving',
    },
  ])('$scenario', async ({ hasNewerInput }) => {
    const pendingSave = createPendingApiResponse();
    const requests: ApiComponents['schemas']['StorageUpdate'][] = [];

    mockServer.use(
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      http.put(`*${API_BASE_PATH}/storages/sftp-id`, async ({ request }) => {
        requests.push((await request.json()) as ApiComponents['schemas']['StorageUpdate']);
        return requests.length === 1
          ? pendingSave.response
          : HttpResponse.json(savedSftpStorage(['PASSWORD', 'PRIVATE_KEY']));
      }),
    );

    const { user } = renderSavedStorageForm(savedSftpStorage(['PASSWORD', 'PRIVATE_KEY']));

    await user.type(screen.getByLabelText('Password', { exact: true }), 'submitted-password');
    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(requests).toHaveLength(1));

    if (hasNewerInput) {
      await user.clear(screen.getByLabelText('Password', { exact: true }));
      await user.type(screen.getByLabelText('Password', { exact: true }), 'newer-password');
    }

    pendingSave.respond(HttpResponse.json(savedSftpStorage(['PASSWORD', 'PRIVATE_KEY'])));

    await waitFor(() =>
      expect((screen.getByLabelText('Password', { exact: true }) as HTMLInputElement).value).toBe(
        hasNewerInput ? 'newer-password' : '',
      ),
    );

    await user.type(screen.getByRole('textbox', { name: 'Path' }), 'other-path');
    await user.click(screen.getByRole('button', { name: 'Test connection' }));
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(requests).toHaveLength(2));
    expect(requests[0].settings).toMatchObject({
      password: 'submitted-password',
      authMethods: ['PASSWORD', 'PRIVATE_KEY'],
    });
    expect(requests[0].settings).not.toHaveProperty('privateKey');

    if (hasNewerInput) expect(requests[1].settings).toHaveProperty('password', 'newer-password');
    else expect(requests[1].settings).not.toHaveProperty('password');

    expect(requests[1].settings).not.toHaveProperty('privateKey');
  });
});

function SftpLocaleControl() {
  const { setLocale } = useLocale();
  return <button onClick={() => setLocale('ru')}>Russian</button>;
}

it('translates guidance and validation while preserving the SFTP draft when the locale changes', async () => {
  const { user } = renderScreen(
    <>
      <SftpLocaleControl />
      {savedStorageEditor(savedSftpStorage(['PASSWORD']))}
    </>,
  );

  await user.type(screen.getByRole('textbox', { name: 'Path' }), 'draft-path');
  await selectSftpMethods(user, en.storages.sftpAuthMethods.both);

  expect(screen.getByText(en.storages.edit.sftp.savedPassword)).toBeTruthy();
  expect(screen.getByText(en.storages.edit.sftp.requiredPrivateKey)).toBeTruthy();

  await user.click(screen.getByRole('button', { name: 'Russian' }));

  expect(screen.getByText(ru.storages.edit.sftp.savedPassword)).toBeTruthy();
  expect(screen.getByText(ru.storages.edit.sftp.requiredPrivateKey)).toBeTruthy();
  expect(
    (
      screen.getByRole('textbox', {
        name: ru.storages.fields.path,
      }) as HTMLInputElement
    ).value,
  ).toBe('draft-path');
  expect(
    (screen.getByRole('button', { name: ru.storages.actions.testConnection }) as HTMLButtonElement)
      .disabled,
  ).toBe(true);
});

it.each(
  ['workspace', 'storage', 'session', 'unmount'].flatMap((changedContext) =>
    ['success', 'failure'].map((outcome) => ({ changedContext, outcome })),
  ),
)(
  'ignores a pending SFTP save $outcome after the $changedContext changes',
  async ({ changedContext, outcome }) => {
    activateSession({ token: 'save-session', userId: 'owner-id' });
    const pendingSave = createPendingApiResponse();
    const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const onChanged = vi.fn();
    const fetchObservation = vi.spyOn(globalThis, 'fetch');
    let saveRequests = 0;

    const editor = (storage: Storage, workspaceId = 'workspace-id') => (
      <EditStorageComponent
        workspaceId={workspaceId}
        editingStorage={storage}
        onChanged={onChanged}
        isShowName
        isShowClose={false}
        onClose={() => {}}
      />
    );

    mockServer.use(
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      http.put(`*${API_BASE_PATH}/storages/sftp-id`, () => {
        saveRequests++;
        return pendingSave.response;
      }),
    );

    const storage = savedSftpStorage(['PRIVATE_KEY']);
    const view = renderScreen(editor(storage));

    await view.user.type(screen.getByRole('textbox', { name: 'Private key' }), 'submitted-key');
    await view.user.click(screen.getByRole('button', { name: 'Test connection' }));
    await view.user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(saveRequests).toBe(1));

    const savingFetch =
      fetchObservation.mock.results[fetchObservation.mock.results.length - 1].value;

    if (changedContext === 'workspace') view.rerender(editor(storage, 'other-workspace'));

    if (changedContext === 'storage') view.rerender(editor({ ...storage, id: 'other-storage' }));

    if (changedContext === 'session')
      act(() => activateSession({ token: 'new-session', userId: 'owner-id' }));

    if (changedContext === 'unmount') view.unmount();
    await act(async () => {
      pendingSave.respond(
        outcome === 'success'
          ? HttpResponse.json(storage)
          : HttpResponse.json({ code: 'invalid_request' }, { status: 400 }),
      );
      await savingFetch;
    });

    expect(onChanged).not.toHaveBeenCalled();
    expect(alert).not.toHaveBeenCalled();

    const currentEditor = screen.queryByRole('region', { name: 'Storage settings' });

    expect(currentEditor && within(currentEditor).queryByRole('alert')).toBeNull();
  },
);

it('requires the removed key when it is selected again after a successful method switch', async () => {
  mockServer.use(
    mockApi.post('/storage-tests', 200, { isConnected: true }),
    mockApi.put('/storages/{storageId}', 200, savedSftpStorage(['PASSWORD'])),
  );
  const { user } = renderSavedStorageForm(savedSftpStorage(['PRIVATE_KEY']));

  await selectSftpMethods(user, 'Password');

  await user.type(screen.getByLabelText('Password'), 'new-password');
  await user.click(screen.getByRole('button', { name: 'Test connection' }));
  await user.click(await screen.findByRole('button', { name: 'Save' }));

  await waitFor(() =>
    expect((screen.getByLabelText('Password') as HTMLInputElement).value).toBe(''),
  );

  await selectSftpMethods(user, 'Private key');

  expect(screen.queryByText('A private key is saved. Leave blank to keep it.')).toBeNull();
  expect(screen.getByText('Enter a private key for this method.')).toBeTruthy();
  expect(
    (screen.getByRole('button', { name: 'Test connection' }) as HTMLButtonElement).disabled,
  ).toBe(true);
});

const providerStorages: {
  provider: string;
  typeLabel: string;
  authMethodLabel?: string;
  entered: Record<string, string>;
  secretLabels: Record<string, string>;
  sentOnCreate: StorageSettings;
  saved: StorageSettings;
}[] = [
  {
    provider: 'NAS',
    typeLabel: 'NAS',
    entered: {
      Host: 'nas.example.com',
      Share: 'backups',
      Username: 'backup',
      Password: 'nas-password',
    },
    secretLabels: { password: 'Password' },
    sentOnCreate: {
      type: 'NAS',
      host: 'nas.example.com',
      port: 445,
      share: 'backups',
      username: 'backup',
      password: 'nas-password',
      isSsl: false,
    },
    saved: {
      type: 'NAS',
      host: 'nas.example.com',
      port: 445,
      share: 'backups',
      username: 'backup',
      isSsl: false,
    },
  },
  {
    provider: 'Azure with an account key',
    typeLabel: 'Azure Blob Storage',
    entered: {
      'Account name': 'databasusaccount',
      'Account key': 'azure-account-key',
      'Container name': 'backups',
    },
    secretLabels: { accountKey: 'Account key' },
    sentOnCreate: {
      type: 'AZURE_BLOB',
      authMethod: 'ACCOUNT_KEY',
      accountName: 'databasusaccount',
      accountKey: 'azure-account-key',
      containerName: 'backups',
    },
    saved: {
      type: 'AZURE_BLOB',
      authMethod: 'ACCOUNT_KEY',
      accountName: 'databasusaccount',
      containerName: 'backups',
    },
  },
  {
    provider: 'Azure with a connection string',
    typeLabel: 'Azure Blob Storage',
    authMethodLabel: 'Connection string',
    entered: {
      Connection: 'DefaultEndpointsProtocol=https;AccountName=databasusaccount',
      'Container name': 'backups',
    },
    secretLabels: { connectionString: 'Connection' },
    sentOnCreate: {
      type: 'AZURE_BLOB',
      authMethod: 'CONNECTION_STRING',
      connectionString: 'DefaultEndpointsProtocol=https;AccountName=databasusaccount',
      containerName: 'backups',
    },
    saved: { type: 'AZURE_BLOB', authMethod: 'CONNECTION_STRING', containerName: 'backups' },
  },
  {
    provider: 'FTP',
    typeLabel: 'FTP',
    entered: { Host: 'ftp.example.com', Username: 'backup', Password: 'ftp-password' },
    secretLabels: { password: 'Password' },
    sentOnCreate: {
      type: 'FTP',
      host: 'ftp.example.com',
      port: 21,
      username: 'backup',
      password: 'ftp-password',
      isSsl: false,
      isSkipTlsVerify: false,
    },
    saved: {
      type: 'FTP',
      host: 'ftp.example.com',
      port: 21,
      username: 'backup',
      isSsl: false,
      isSkipTlsVerify: false,
    },
  },
  {
    provider: 'rclone',
    typeLabel: 'Rclone',
    entered: { Config: '[remote]\ntype = sftp', 'Remote path': '/backups' },
    secretLabels: { configContent: 'Config' },
    sentOnCreate: {
      type: 'RCLONE',
      configContent: '[remote]\ntype = sftp',
      remotePath: '/backups',
    },
    saved: { type: 'RCLONE', remotePath: '/backups' },
  },
];

// Their rename and secret rules belong to the provider tests above, so only creation is listed here.
const providerStoragesCreatedOnly: Pick<
  (typeof providerStorages)[number],
  'provider' | 'typeLabel' | 'authMethodLabel' | 'entered' | 'sentOnCreate'
>[] = [
  {
    provider: 'S3',
    typeLabel: 'S3',
    entered: {
      'S3 Bucket': 'databasus-backups',
      Region: 'eu-central-1',
      'Access key': 'entered-access-key',
      'Secret key': 'entered-secret-key',
      Endpoint: 'https://s3.example.com',
    },
    sentOnCreate: {
      type: 'S3',
      s3Bucket: 'databasus-backups',
      s3Region: 'eu-central-1',
      s3AccessKey: 'entered-access-key',
      s3SecretKey: 'entered-secret-key',
      s3Endpoint: 'https://s3.example.com',
      isVirtualHostedStyle: false,
      isSkipTlsVerify: false,
    },
  },
  {
    provider: 'SFTP with a password',
    typeLabel: 'SFTP',
    entered: {
      Host: 'sftp.example.com',
      Username: 'backup',
      Password: 'sftp-password',
      Path: '/backups',
    },
    sentOnCreate: {
      type: 'SFTP',
      host: 'sftp.example.com',
      port: 22,
      username: 'backup',
      authMethods: ['PASSWORD'],
      password: 'sftp-password',
      path: '/backups',
      isSkipHostKeyVerify: false,
    },
  },
];

const getProviderField = (label: string) =>
  screen.getByLabelText(label, { selector: 'input:not([type="radio"]),textarea' });

// Records every storage the form creates and answers it with the storage the request describes.
const mockCreation = () => {
  const sentStorages: ApiComponents['schemas']['StorageCreate'][] = [];
  mockServer.use(
    mockApi.post('/storage-tests', 200, { isConnected: true }),
    http.post(`*${API_BASE_PATH}/storages`, async ({ request }) => {
      const sentStorage = (await request.json()) as ApiComponents['schemas']['StorageCreate'];
      sentStorages.push(sentStorage);

      return HttpResponse.json(
        { ...sentStorage, id: 'created-storage', workspaceId: 'workspace-id' },
        { status: 201 },
      );
    }),
  );

  return sentStorages;
};

const renderNewStorageForm = async (typeLabel: string) => {
  const view = renderScreen(
    <EditStorageComponent
      workspaceId="workspace-id"
      isShowClose={false}
      onClose={() => {}}
      isShowName
      onChanged={() => {}}
    />,
  );

  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
    target: { value: 'New storage' },
  });
  await view.user.click(screen.getByRole('combobox', { name: 'Type' }));
  fireEvent.click(await screen.findByRole('option', { name: typeLabel }));

  return view;
};

describe('EditStorageComponent providers', () => {
  it.each([...providerStorages, ...providerStoragesCreatedOnly])(
    'sends the entered settings when a $provider storage is created',
    async ({ typeLabel, authMethodLabel, entered, sentOnCreate }) => {
      const sentStorages = mockCreation();
      await renderNewStorageForm(typeLabel);

      if (authMethodLabel) fireEvent.click(screen.getByRole('radio', { name: authMethodLabel }));
      for (const [label, value] of Object.entries(entered))
        fireEvent.change(getProviderField(label), { target: { value } });
      fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(sentStorages).toHaveLength(1));
      expect(sentStorages[0]).toMatchObject({ name: 'New storage', settings: sentOnCreate });
      expect(sentStorages[0].settings).toEqual(sentOnCreate);
    },
  );

  it.each(providerStorages)(
    'omits every secret when a saved $provider storage is renamed',
    async ({ saved }) => {
      mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
      const sentUpdates = mockUpdate();
      renderSavedStorageForm({ ...savedS3Storage, settings: saved });

      fireEvent.change(screen.getByDisplayValue('Offsite backups'), {
        target: { value: 'Renamed backups' },
      });
      fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(sentUpdates).toHaveLength(1));
      expect(sentUpdates[0]).toEqual({ name: 'Renamed backups', settings: saved });
    },
  );

  it.each(providerStorages)(
    'sends the replacement when a secret of a saved $provider storage is entered again',
    async ({ saved, secretLabels }) => {
      mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
      const sentUpdates = mockUpdate();
      renderSavedStorageForm({ ...savedS3Storage, settings: saved });

      for (const [secretField, label] of Object.entries(secretLabels))
        fireEvent.change(getProviderField(label), { target: { value: `new-${secretField}` } });
      fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(sentUpdates).toHaveLength(1));
      expect(sentUpdates[0].settings).toEqual({
        ...saved,
        ...Object.fromEntries(
          Object.keys(secretLabels).map((secretField) => [secretField, `new-${secretField}`]),
        ),
      });
    },
  );

  it('sends the password and the private key when an SFTP storage is created with both methods', async () => {
    const sentStorages = mockCreation();
    const { user } = await renderNewStorageForm('SFTP');

    await selectSftpMethods(user, 'Password and private key');
    fireEvent.change(getProviderField('Host'), { target: { value: 'sftp.example.com' } });
    fireEvent.change(getProviderField('Username'), { target: { value: 'backup' } });
    fireEvent.change(getProviderField('Password'), { target: { value: 'sftp-password' } });
    fireEvent.change(getProviderField('Private key'), { target: { value: 'sftp-private-key' } });
    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentStorages).toHaveLength(1));
    expect(sentStorages[0].settings).toEqual({
      type: 'SFTP',
      host: 'sftp.example.com',
      port: 22,
      username: 'backup',
      authMethods: ['PASSWORD', 'PRIVATE_KEY'],
      password: 'sftp-password',
      privateKey: 'sftp-private-key',
      isSkipHostKeyVerify: false,
    });
  });

  it('sends the entered client settings with the token when a Google Drive storage is created', async () => {
    const sentStorages = mockCreation();
    const tokenJson = JSON.stringify({ access_token: 'access', refresh_token: 'refresh' });
    renderScreen(
      <EditStorageComponent
        workspaceId="workspace-id"
        isShowClose={false}
        onClose={() => {}}
        isShowName
        onChanged={() => {}}
        draft={{
          name: 'New storage',
          settings: { type: 'GOOGLE_DRIVE', clientId: 'draft-client', tokenJson },
        }}
      />,
    );

    fireEvent.change(getProviderField('Client ID'), { target: { value: 'entered-client' } });
    fireEvent.change(getProviderField('Client Secret'), { target: { value: 'entered-secret' } });
    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentStorages).toHaveLength(1));
    expect(sentStorages[0]).toMatchObject({ name: 'New storage' });
    expect(sentStorages[0].settings).toEqual({
      type: 'GOOGLE_DRIVE',
      clientId: 'entered-client',
      clientSecret: 'entered-secret',
      tokenJson,
    });
  });

  it('sends the entered keys when S3 keys of a saved storage are entered again', async () => {
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    const sentUpdates = mockUpdate();
    renderSavedStorageForm(savedS3Storage);

    fireEvent.change(getProviderField('Access key'), { target: { value: 'new-access-key' } });
    fireEvent.change(getProviderField('Secret key'), { target: { value: 'new-secret-key' } });
    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0].settings).toEqual({
      ...savedS3Storage.settings,
      s3AccessKey: 'new-access-key',
      s3SecretKey: 'new-secret-key',
    });
  });
});

describe('EditStorageComponent address change', () => {
  const addressChangedMessage = en.common.messages.addressChangedEnterSecretAgain;

  const savedStorageWith = (settings: StorageSettings): Storage => ({
    ...savedS3Storage,
    settings,
  });

  const savedS3AtEndpoint = savedStorageWith({
    ...savedS3Storage.settings,
    s3Endpoint: 'https://s3.example.com',
  });

  const savedFtp = savedStorageWith({
    type: 'FTP',
    host: 'ftp.example.com',
    port: 21,
    username: 'backup',
    path: '/backups',
    isSsl: false,
    isSkipTlsVerify: false,
  });

  const savedNas = savedStorageWith({
    type: 'NAS',
    host: 'nas.example.com',
    port: 445,
    share: 'backups',
    username: 'backup',
    isSsl: false,
  });

  const savedAzureWithAccountKey = savedStorageWith({
    type: 'AZURE_BLOB',
    authMethod: 'ACCOUNT_KEY',
    accountName: 'savedaccount',
    containerName: 'backups',
    endpoint: 'https://blob.example.com',
  });

  const savedAzureWithConnectionString = savedStorageWith({
    type: 'AZURE_BLOB',
    authMethod: 'CONNECTION_STRING',
    containerName: 'backups',
    endpoint: 'https://blob.example.com',
  });

  const getTestButton = () =>
    screen.getByRole('button', { name: 'Test connection' }) as HTMLButtonElement;

  it('shows the message under both keys and offers no test when the S3 endpoint changes', () => {
    renderSavedStorageForm(savedS3AtEndpoint);

    fireEvent.change(getProviderField('Endpoint'), {
      target: { value: 'https://moved.example.com' },
    });

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(2);
    expect(getTestButton().disabled).toBe(true);
  });

  it('takes back the save control when the S3 endpoint changes after a successful test', async () => {
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    renderSavedStorageForm(savedS3AtEndpoint);

    fireEvent.change(screen.getByDisplayValue('Offsite backups'), {
      target: { value: 'Renamed backups' },
    });
    fireEvent.click(getTestButton());
    await screen.findByRole('button', { name: 'Save' });

    fireEvent.change(getProviderField('Endpoint'), {
      target: { value: 'https://moved.example.com' },
    });

    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('removes the message and offers the test when the S3 endpoint returns to the saved value', () => {
    renderSavedStorageForm(savedS3AtEndpoint);

    fireEvent.change(getProviderField('Endpoint'), {
      target: { value: 'https://moved.example.com' },
    });
    fireEvent.change(getProviderField('Endpoint'), {
      target: { value: 'https://s3.example.com' },
    });

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });

  it('keeps the typed password in the test request when the FTP host returns to the saved value', async () => {
    const sentTests: ApiComponents['schemas']['StorageSettingsTest'][] = [];
    mockServer.use(
      http.post(`*${API_BASE_PATH}/storage-tests`, async ({ request }) => {
        sentTests.push((await request.json()) as ApiComponents['schemas']['StorageSettingsTest']);

        return HttpResponse.json({ isConnected: true });
      }),
    );
    renderSavedStorageForm(savedFtp);

    fireEvent.change(getProviderField('Host'), { target: { value: 'moved.example.com' } });
    fireEvent.change(getProviderField('Password'), { target: { value: 'typed-password' } });
    expect(screen.queryByText(addressChangedMessage)).toBeNull();

    fireEvent.change(getProviderField('Host'), { target: { value: 'ftp.example.com' } });
    expect((getProviderField('Password') as HTMLInputElement).value).toBe('typed-password');

    fireEvent.click(getTestButton());

    await waitFor(() => expect(sentTests).toHaveLength(1));
    expect(sentTests[0].settings).toMatchObject({ password: 'typed-password' });
  });

  it('offers the test without a password when only the FTP path changes', async () => {
    const sentTests: ApiComponents['schemas']['StorageSettingsTest'][] = [];
    mockServer.use(
      http.post(`*${API_BASE_PATH}/storage-tests`, async ({ request }) => {
        sentTests.push((await request.json()) as ApiComponents['schemas']['StorageSettingsTest']);

        return HttpResponse.json({ isConnected: true });
      }),
    );
    renderSavedStorageForm(savedFtp);

    fireEvent.change(getProviderField('Path'), { target: { value: '/moved' } });

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    fireEvent.click(getTestButton());

    await waitFor(() => expect(sentTests).toHaveLength(1));
    expect(sentTests[0].settings).not.toHaveProperty('password');
  });

  it.each([
    { provider: 'FTP', saved: savedFtp, label: 'Port', movedValue: '2121' },
    { provider: 'NAS', saved: savedNas, label: 'Host', movedValue: 'moved.example.com' },
    { provider: 'NAS', saved: savedNas, label: 'Port', movedValue: '1445' },
  ])(
    'shows the message under the password and offers no test when the $label of a $provider storage changes',
    ({ saved, label, movedValue }) => {
      renderSavedStorageForm(saved);

      fireEvent.change(getProviderField(label), { target: { value: movedValue } });

      expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
      expect(getTestButton().disabled).toBe(true);
    },
  );

  it('asks only for the private key when an SFTP storage moves with only that method selected', async () => {
    const { user } = renderSavedStorageForm(savedSftpStorage(['PASSWORD', 'PRIVATE_KEY']));

    fireEvent.change(getProviderField('Host'), { target: { value: 'moved.example.com' } });
    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(2);
    expect(screen.queryByText(en.storages.edit.sftp.savedPassword)).toBeNull();

    await selectSftpMethods(user, 'Private key');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(screen.queryByText(en.storages.edit.sftp.savedPrivateKey)).toBeNull();
    expect(screen.queryByLabelText('Password', { exact: true })).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('shows the saved credential text again when the SFTP host returns to the saved value', () => {
    const storage = savedSftpStorage(['PASSWORD']);
    renderSavedStorageForm(storage);
    const savedHost = (getProviderField('Host') as HTMLInputElement).value;

    fireEvent.change(getProviderField('Host'), { target: { value: 'moved.example.com' } });
    fireEvent.change(getProviderField('Host'), { target: { value: savedHost } });

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(screen.getByText(en.storages.edit.sftp.savedPassword)).toBeTruthy();
    expect(getTestButton().disabled).toBe(false);
  });

  it.each([
    { label: 'Endpoint', movedValue: 'https://moved.example.com' },
    { label: 'Account name', movedValue: 'movedaccount' },
  ])(
    'shows the message under the account key and offers no test when the Azure $label changes',
    ({ label, movedValue }) => {
      renderSavedStorageForm(savedAzureWithAccountKey);

      fireEvent.change(getProviderField(label), { target: { value: movedValue } });

      expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
      expect(getTestButton().disabled).toBe(true);
    },
  );

  it('asks for nothing when the Azure account name is left empty', () => {
    renderSavedStorageForm(
      savedStorageWith({
        type: 'AZURE_BLOB',
        authMethod: 'ACCOUNT_KEY',
        containerName: 'backups',
      }),
    );

    fireEvent.change(getProviderField('Container name'), { target: { value: 'moved-container' } });

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });

  it('asks for no secret when the endpoint of an Azure storage with a connection string changes', () => {
    renderSavedStorageForm(savedAzureWithConnectionString);

    fireEvent.click(screen.getByRole('radio', { name: 'Account key' }));
    fireEvent.change(getProviderField('Endpoint'), {
      target: { value: 'https://moved.example.com' },
    });
    fireEvent.click(screen.getByRole('radio', { name: 'Connection string' }));

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });
});
