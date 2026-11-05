// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { type Storage, storageApi } from '../../../entities/storages';
import { activateSession } from '../../../shared/auth';
import { DICTIONARIES, type Locale, useLocale } from '../../../shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { releaseAndSettle } from '../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { StorageComponent } from './StorageComponent';

const mockServer = setUpMockApi();

afterEach(() => {
  vi.restoreAllMocks();
});

const localStorageOfWorkspace: Storage = {
  id: 'storage-id',
  workspaceId: 'workspace-id',
  name: 'Local backups',
  settings: { type: 'LOCAL' },
};

const renderStorage = (onStorageDeleted: () => void = () => {}) =>
  renderScreen(
    <StorageComponent
      storageId={localStorageOfWorkspace.id}
      onStorageChanged={() => {}}
      onStorageDeleted={onStorageDeleted}
      onStorageTransferred={() => {}}
      isCanManageStorages
    />,
  );

describe('StorageComponent', () => {
  it('retains the acknowledged name when settings editing follows a pending rename', async () => {
    const storage: Storage = {
      ...localStorageOfWorkspace,
      settings: {
        type: 'S3',
        s3Bucket: 'bucket-a',
        isVirtualHostedStyle: false,
        isSkipTlsVerify: false,
      },
    };
    const renamed = { ...storage, name: 'Renamed archive' };
    const changedStorage = { ...renamed, settings: { ...storage.settings, s3Bucket: 'bucket-b' } };
    const rename = mockApi.pending('put', '/storages/{storageId}');

    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, storage),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      rename.handler,
    );

    const view = renderStorage();
    await screen.findByText(storage.name);

    await view.user.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: renamed.name },
    });
    await view.user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() =>
      expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(
        1,
      ),
    );

    const settingsAction = screen.getByRole('button', { name: 'Edit' });

    await view.user.click(settingsAction);

    expect(settingsAction).toHaveProperty('disabled', true);
    expect(screen.queryByRole('region', { name: 'Storage settings' })).toBeNull();

    mockServer.use(mockApi.put('/storages/{storageId}', 200, changedStorage));

    rename.respond(HttpResponse.json(renamed));
    await screen.findByText(renamed.name);

    await view.user.click(screen.getAllByRole('button', { name: 'Edit' })[1]);

    expect(screen.getByRole('button', { name: 'Edit' })).toHaveProperty('disabled', true);

    fireEvent.change(screen.getByDisplayValue('bucket-a'), { target: { value: 'bucket-b' } });
    await view.user.click(screen.getByRole('button', { name: 'Test connection' }));
    await view.user.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(
        2,
      ),
    );
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Save' })).toBeNull());

    const updates = mockServer.getRequests().filter((request) => request.method === 'PUT');

    expect(await updates[1].json()).toMatchObject({
      name: renamed.name,
      settings: { s3Bucket: 'bucket-b' },
    });
    expect(screen.getByText(renamed.name)).toBeTruthy();
  });

  it('offers no retry when the connection test fails with the storage loaded', async () => {
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      mockApi.post('/storages/{storageId}/test', 500, buildProblem('internal', 500)),
    );
    const view = renderStorage();

    await view.user.click(await screen.findByRole('button', { name: 'Test connection' }));
    await screen.findByText('An internal server error occurred. Please try again later.');

    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('shows translated failure when a saved connection test returns no diagnostic', async () => {
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      mockApi.post('/storages/{storageId}/test', 200, { isConnected: false }),
    );
    const view = renderStorage();

    await view.user.click(await screen.findByRole('button', { name: 'Test connection' }));

    expect((await screen.findByRole('alert')).textContent).toContain(
      'The storage could not be reached',
    );
    expect(screen.getByText(localStorageOfWorkspace.name)).toBeTruthy();
    expect(screen.getByRole('button', { name: /Test connection/ })).toHaveProperty(
      'disabled',
      false,
    );
  });

  it.each([
    {
      failure: '403',
      answerTestRequest: () =>
        mockApi.post('/storages/{storageId}/test', 403, buildProblem('permission_denied', 403)),
      reason: 'You do not have permission to do this.',
    },
    {
      failure: '500',
      answerTestRequest: () =>
        mockApi.post('/storages/{storageId}/test', 500, buildProblem('internal', 500)),
      reason: 'An internal server error occurred. Please try again later.',
    },
    {
      failure: 'a network loss',
      answerTestRequest: () => mockApi.network('post', '/storages/{storageId}/test'),
      reason: 'Cannot reach the Databasus server. Check your connection and try again.',
    },
  ])(
    'shows the reason, keeps the storage and allows a retry when the saved connection test answers $failure',
    async ({ answerTestRequest, reason }) => {
      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
        answerTestRequest(),
      );
      const view = renderStorage();

      await view.user.click(await screen.findByRole('button', { name: 'Test connection' }));

      expect(await screen.findByText(reason)).toBeTruthy();
      expect(screen.getByText(localStorageOfWorkspace.name)).toBeTruthy();

      const testAction = screen.getByRole('button', { name: /Test connection/ });

      await waitFor(() => expect(testAction).toHaveProperty('disabled', false));

      mockServer.use(mockApi.post('/storages/{storageId}/test', 200, { isConnected: true }));

      await view.user.click(testAction);

      expect(await screen.findByText('Connection test successful!')).toBeTruthy();
      expect(screen.queryByText(reason)).toBeNull();
      expect(screen.getByText(localStorageOfWorkspace.name)).toBeTruthy();
    },
  );

  it('shows the storage text when a storage that backups use is deleted', async () => {
    const onStorageDeleted = vi.fn();
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      mockApi.delete('/storages/{storageId}', 409, buildProblem('in_use', 409)),
    );
    renderStorage(onStorageDeleted);

    await screen.findByText('Local backups');

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );

    expect(
      await screen.findByText(
        'A backup config uses this storage or it still holds backups. Point those backup configs at another storage and delete its backups first.',
      ),
    ).toBeTruthy();
    expect(onStorageDeleted).not.toHaveBeenCalled();
  });

  it.each(['COLD', 'DEEP_ARCHIVE', '__proto__', 'toString'])(
    'shows the exact class when the storage uses %s',
    async (storageClass) => {
      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, {
          ...localStorageOfWorkspace,
          settings: {
            type: 'S3',
            s3Bucket: 'backups',
            isVirtualHostedStyle: false,
            isSkipTlsVerify: false,
            s3StorageClass: storageClass,
          },
        }),
      );
      renderStorage();

      expect(await screen.findByText(storageClass)).toBeTruthy();
    },
  );

  it('shows no save error box with the storage settings', async () => {
    mockServer.use(mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace));
    renderStorage();

    await screen.findByText('Local backups');

    expect(screen.getByText('Storage settings')).toBeTruthy();
    expect(screen.queryByText('Save error')).toBeNull();
  });

  it.each([
    { methods: ['PASSWORD'] as ('PASSWORD' | 'PRIVATE_KEY')[], labels: ['Password'] },
    { methods: ['PRIVATE_KEY'] as ('PASSWORD' | 'PRIVATE_KEY')[], labels: ['Private key'] },
    {
      methods: ['PASSWORD', 'PRIVATE_KEY'] as ('PASSWORD' | 'PRIVATE_KEY')[],
      labels: ['Password', 'Private key'],
    },
  ])('shows persisted SFTP methods without secrets for $methods', async ({ methods, labels }) => {
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, {
        ...localStorageOfWorkspace,
        settings: {
          type: 'SFTP',
          authMethods: methods,
          host: 'sftp.example.com',
          port: 22,
          username: 'backup',
          isSkipHostKeyVerify: false,
        },
      }),
    );

    renderStorage();
    await screen.findByText('sftp.example.com');

    for (const label of labels) expect(screen.getByText(label, { exact: true })).toBeTruthy();

    if (methods.length === 1)
      expect(
        screen.queryByText(methods[0] === 'PASSWORD' ? 'Private key' : 'Password', { exact: true }),
      ).toBeNull();
  });

  it('retains newer name input after a delayed rename is acknowledged', async () => {
    const renamed = { ...localStorageOfWorkspace, name: 'Submitted' };
    const pendingRename = mockApi.delayed({
      method: 'put',
      path: '/storages/{storageId}',
      status: 200,
      body: renamed,
    });

    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      pendingRename.handler,
    );
    const onChanged = vi.fn();
    renderScreen(
      <StorageComponent
        storageId={localStorageOfWorkspace.id}
        onStorageChanged={onChanged}
        onStorageDeleted={() => {}}
        onStorageTransferred={() => {}}
        isCanManageStorages
      />,
    );
    await screen.findByText(localStorageOfWorkspace.name);

    fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
    const nameInput = screen.getByRole('textbox', { name: 'Name' });

    fireEvent.change(nameInput, { target: { value: 'Submitted' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(mockServer.getRequests().some((request) => request.method === 'PUT')).toBe(true),
    );

    fireEvent.change(nameInput, { target: { value: 'Newer draft' } });
    pendingRename.release();

    await waitFor(() => expect(onChanged).toHaveBeenCalledWith(renamed));
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveProperty('value', 'Newer draft');
    expect(screen.getByRole('button', { name: 'Save' })).toBeTruthy();
  });

  it('closes confirmation and retains the storage when deletion fails', async () => {
    const onStorageDeleted = vi.fn();
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      mockApi.delete('/storages/{storageId}', 500, buildProblem('internal', 500)),
    );
    renderStorage(onStorageDeleted);

    fireEvent.click(await screen.findByRole('button', { name: 'Remove' }));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );

    await screen.findByText('An internal server error occurred. Please try again later.');

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.getByText(localStorageOfWorkspace.name)).toBeTruthy();
    expect(onStorageDeleted).not.toHaveBeenCalled();
  });

  it('hides Save when the name returns to the acknowledged value', async () => {
    mockServer.use(mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace));
    renderStorage();
    await screen.findByText('Local backups');

    fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
    const nameInput = screen.getByRole('textbox', { name: 'Name' });

    fireEvent.change(nameInput, { target: { value: 'Changed name' } });

    expect(screen.getByRole('button', { name: 'Save' })).toBeTruthy();

    fireEvent.change(nameInput, { target: { value: 'Local backups' } });

    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(0);
  });

  it.each(['success', 'connection', 'request'] as const)(
    'rejects a saved storage test after newer settings are saved with old outcome=%s',
    async (outcome) => {
      const storage: Storage = {
        ...localStorageOfWorkspace,
        settings: {
          type: 'S3',
          s3Bucket: 'bucket-a',
          isVirtualHostedStyle: false,
          isSkipTlsVerify: false,
        },
      };
      const changedStorage: Storage = {
        ...storage,
        settings: {
          type: 'S3',
          s3Bucket: 'bucket-b',
          isVirtualHostedStyle: false,
          isSkipTlsVerify: false,
        },
      };

      const testCall = vi.spyOn(storageApi, 'testStorage');
      const oldTest = mockApi.pending('post', '/storages/{storageId}/test');
      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, storage),
        oldTest.handler,
        mockApi.post('/storage-tests', 200, { isConnected: true }),
        mockApi.put('/storages/{storageId}', 200, changedStorage),
      );
      const onStorageChanged = vi.fn();
      const view = renderScreen(
        <StorageComponent
          storageId={storage.id}
          onStorageChanged={onStorageChanged}
          onStorageDeleted={() => {}}
          onStorageTransferred={() => {}}
          isCanManageStorages
        />,
      );

      fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));

      await waitFor(() =>
        expect(
          mockServer.getRequests().filter((request) => request.method === 'POST'),
        ).toHaveLength(1),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[1]);
      fireEvent.change(screen.getByDisplayValue('bucket-a'), { target: { value: 'bucket-b' } });
      fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(onStorageChanged).toHaveBeenCalledWith(changedStorage));

      const notices = screen.queryAllByText('Connection test successful!').length;

      await releaseAndSettle(
        testCall,
        () => {
          oldTest.respond(
            outcome === 'request'
              ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
              : HttpResponse.json({
                  isConnected: outcome === 'success',
                  error: outcome === 'connection' ? 'Obsolete connection failure' : undefined,
                }),
          );
        },
        0,
      );

      expect(onStorageChanged).toHaveBeenCalledTimes(1);
      expect(view.container.querySelector('.ant-alert-error')).toBeNull();
      expect(screen.queryAllByText('Connection test successful!')).toHaveLength(notices);
      expect(screen.getByDisplayValue('bucket-b')).toBeTruthy();
    },
  );

  it.each([true, false])(
    'accepts the saved connection test after only the name changes with success=%s',
    async (isConnected) => {
      const pendingTest = mockApi.pending('post', '/storages/{storageId}/test');
      const renamedStorage = { ...localStorageOfWorkspace, name: 'Renamed backups' };

      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
        mockApi.put('/storages/{storageId}', 200, renamedStorage),
        pendingTest.handler,
      );
      const onStorageChanged = vi.fn();
      renderScreen(
        <StorageComponent
          storageId={localStorageOfWorkspace.id}
          onStorageChanged={onStorageChanged}
          onStorageDeleted={() => {}}
          onStorageTransferred={() => {}}
          isCanManageStorages
        />,
      );

      fireEvent.click(await screen.findByRole('button', { name: 'Test connection' }));

      await waitFor(() =>
        expect(mockServer.getRequests().some((request) => request.method === 'POST')).toBe(true),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
      fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
        target: { value: renamedStorage.name },
      });
      fireEvent.click(screen.getByRole('button', { name: 'Save' }));
      await screen.findByText(renamedStorage.name);

      await act(async () =>
        pendingTest.respond(
          HttpResponse.json({
            isConnected,
            error: isConnected ? undefined : 'Current connection failure',
          }),
        ),
      );

      await screen.findByText(
        isConnected
          ? 'Connection test successful!'
          : 'The storage could not be reached: Current connection failure',
      );

      expect(onStorageChanged).toHaveBeenCalledTimes(1);
      expect(screen.getByText(renamedStorage.name)).toBeTruthy();
      expect(screen.queryByText(localStorageOfWorkspace.name)).toBeNull();
    },
  );
  it('requires a second action after cleanup refusal and retains the warning on explicit failure', async () => {
    const onStorageDeleted = vi.fn();
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      mockApi.delete('/storages/{storageId}', 409, buildProblem('cleanup_pending', 409)),
    );
    const view = renderStorage(onStorageDeleted);

    await view.user.click(await screen.findByRole('button', { name: 'Remove' }));
    await view.user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );
    await screen.findByRole('button', { name: 'Delete anyway' });

    const ordinaryRemoval = mockServer
      .getRequests()
      .filter((request) => request.method === 'DELETE');
    expect(ordinaryRemoval).toHaveLength(1);
    expect(new URL(ordinaryRemoval[0].url).searchParams.has('shouldAbandonCleanup')).toBe(false);

    await view.user.click(screen.getByRole('button', { name: 'Wait' }));

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(onStorageDeleted).not.toHaveBeenCalled();

    await view.user.click(screen.getByRole('button', { name: 'Remove' }));
    await view.user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );
    await screen.findByRole('button', { name: 'Delete anyway' });
    mockServer.use(mockApi.delete('/storages/{storageId}', 500, buildProblem('internal', 500)));
    await view.user.click(screen.getByRole('button', { name: 'Delete anyway' }));
    expect(
      await within(screen.getByRole('dialog')).findByText(
        'An internal server error occurred. Please try again later.',
      ),
    ).toBeTruthy();
    expect(
      new URL(
        mockServer.getRequests().filter((request) => request.method === 'DELETE')[2].url,
      ).searchParams.get('shouldAbandonCleanup'),
    ).toBe('true');

    const explicitRemoval = mockApi.pending('delete', '/storages/{storageId}');
    mockServer.use(explicitRemoval.handler);
    const abandonAction = await screen.findByRole('button', { name: /Delete anyway/ });
    await waitFor(() => expect(abandonAction).toHaveProperty('disabled', false));
    fireEvent.click(abandonAction);
    fireEvent.click(abandonAction);
    await waitFor(() =>
      expect(
        mockServer.getRequests().filter((request) => request.method === 'DELETE'),
      ).toHaveLength(4),
    );
    expect(screen.getByRole('button', { name: 'Wait' })).toHaveProperty('disabled', true);
    explicitRemoval.respond(new HttpResponse(null, { status: 204 }));
    await waitFor(() => expect(onStorageDeleted).toHaveBeenCalledOnce());
  });

  it.each(['storage', 'session'] as const)(
    'closes an open cleanup warning when the %s changes',
    async (contextChange) => {
      const otherStorage = {
        ...localStorageOfWorkspace,
        id: 'other-storage',
        name: 'Other archive',
      };
      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
        mockApi.delete('/storages/{storageId}', 409, buildProblem('cleanup_pending', 409)),
      );
      const renderSelectedStorage = (storageId: string, sessionKey = 'initial') => (
        <StorageComponent
          key={sessionKey}
          storageId={storageId}
          onStorageChanged={() => {}}
          onStorageDeleted={() => {}}
          onStorageTransferred={() => {}}
          isCanManageStorages
        />
      );
      const view = renderScreen(renderSelectedStorage(localStorageOfWorkspace.id));

      await view.user.click(await screen.findByRole('button', { name: 'Remove' }));
      await view.user.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
      );
      await screen.findByRole('button', { name: 'Delete anyway' });

      mockServer.use(mockApi.get('/storages/{storageId}', 200, otherStorage));

      if (contextChange === 'storage') {
        view.rerender(renderSelectedStorage(otherStorage.id));
        await screen.findByText(otherStorage.name);
      } else {
        act(() => activateSession({ token: 'replacement-session', userId: 'replacement-user' }));
      }

      expect(screen.queryByRole('dialog')).toBeNull();

      if (contextChange === 'session') {
        view.rerender(renderSelectedStorage(otherStorage.id, 'replacement'));
        await screen.findByText(otherStorage.name);
      }

      await view.user.click(screen.getByRole('button', { name: 'Remove' }));
      await view.user.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
      );
      await screen.findByRole('button', { name: 'Delete anyway' });

      const removals = mockServer.getRequests().filter((request) => request.method === 'DELETE');
      expect(removals).toHaveLength(2);
      expect(new URL(removals[1].url).pathname).toContain(otherStorage.id);
      expect(new URL(removals[1].url).searchParams.has('shouldAbandonCleanup')).toBe(false);
    },
  );

  it.each(['storage', 'session', 'unmount'] as const)(
    'rejects a cleanup refusal after the %s changes',
    async (contextChange) => {
      const deleteCall = vi.spyOn(storageApi, 'deleteStorage');
      const removal = mockApi.pending('delete', '/storages/{storageId}');
      const onStorageDeleted = vi.fn();
      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
        removal.handler,
      );
      const renderSelectedStorage = (storageId: string) => (
        <StorageComponent
          storageId={storageId}
          onStorageChanged={() => {}}
          onStorageDeleted={onStorageDeleted}
          onStorageTransferred={() => {}}
          isCanManageStorages
        />
      );
      const view = renderScreen(renderSelectedStorage(localStorageOfWorkspace.id));
      await view.user.click(await screen.findByRole('button', { name: 'Remove' }));
      await view.user.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
      );
      await waitFor(() =>
        expect(mockServer.getRequests().some((request) => request.method === 'DELETE')).toBe(true),
      );

      const obsoleteDeleteIndex = deleteCall.mock.calls.length - 1;

      if (contextChange === 'storage') view.rerender(renderSelectedStorage('other-storage'));
      else if (contextChange === 'session')
        act(() => activateSession({ token: 'new-session', userId: 'new-user' }));
      else view.unmount();

      await releaseAndSettle(
        deleteCall,
        () => {
          removal.respond(HttpResponse.json(buildProblem('cleanup_pending', 409), { status: 409 }));
        },
        obsoleteDeleteIndex,
      );

      expect(screen.queryByRole('button', { name: 'Delete anyway' })).toBeNull();
      expect(onStorageDeleted).not.toHaveBeenCalled();
      expect(
        mockServer.getRequests().filter((request) => request.method === 'DELETE'),
      ).toHaveLength(1);
    },
  );

  it.each(['ru', 'fr', 'es', 'pt', 'zh'] as const)(
    'translates an open cleanup warning into %s without acknowledging deletion',
    async (locale: Locale) => {
      const ChangeLanguage = () => {
        const { setLocale } = useLocale();
        return <button onClick={() => setLocale(locale)}>Switch language</button>;
      };
      mockServer.use(
        mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
        mockApi.delete('/storages/{storageId}', 409, buildProblem('cleanup_pending', 409)),
      );
      renderScreen(
        <>
          <ChangeLanguage />
          <StorageComponent
            storageId={localStorageOfWorkspace.id}
            onStorageChanged={() => {}}
            onStorageDeleted={() => {}}
            onStorageTransferred={() => {}}
            isCanManageStorages
          />
        </>,
      );
      fireEvent.click(await screen.findByRole('button', { name: 'Remove' }));
      fireEvent.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
      );
      await screen.findByRole('button', { name: 'Delete anyway' });

      fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

      const dictionary = DICTIONARIES[locale];
      const warning = screen.getByRole('dialog', { name: dictionary.storages.cleanup.title });
      expect(within(warning).getByText(dictionary.storages.cleanup.warning)).toBeTruthy();
      expect(
        within(warning).getByRole('button', { name: dictionary.storages.cleanup.deleteAnyway }),
      ).toBeTruthy();
      fireEvent.click(
        within(warning).getByRole('button', { name: dictionary.storages.cleanup.wait }),
      );
      expect(screen.queryByRole('dialog')).toBeNull();
      expect(
        mockServer.getRequests().filter((request) => request.method === 'DELETE'),
      ).toHaveLength(1);
    },
  );

  it('offers no deletion to a viewer', async () => {
    mockServer.use(mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace));
    renderScreen(
      <StorageComponent
        storageId={localStorageOfWorkspace.id}
        onStorageChanged={() => {}}
        onStorageDeleted={() => {}}
        onStorageTransferred={() => {}}
        isCanManageStorages={false}
      />,
    );
    await screen.findByText(localStorageOfWorkspace.name);

    expect(screen.queryByRole('button', { name: 'Remove' })).toBeNull();
  });
});
