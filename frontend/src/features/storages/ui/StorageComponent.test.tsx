// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Storage } from '../../../entities/storages';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
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
  it('shows the storage text when a storage that backups use is deleted', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const onStorageDeleted = vi.fn();
    mockServer.use(
      mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace),
      mockApi.delete('/storages/{storageId}', 409, buildProblem('in_use', 409)),
    );
    renderStorage(onStorageDeleted);

    await screen.findByText('Local backups');
    fireEvent.click(screen.getByRole('button', { name: 'delete' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Remove' }));

    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith(
        'A backup config uses this storage or it still holds backups. Point those backup configs at another storage and delete its backups first.',
      ),
    );
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
