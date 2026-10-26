// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Storage } from '../../../entity/storages';
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

  it('shows no save error box with the storage settings', async () => {
    mockServer.use(mockApi.get('/storages/{storageId}', 200, localStorageOfWorkspace));
    renderStorage();

    await screen.findByText('Local backups');

    expect(screen.getByText('Storage settings')).toBeTruthy();
    expect(screen.queryByText('Save error')).toBeNull();
  });
});
