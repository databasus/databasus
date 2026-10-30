// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Storage } from '../../../../entities/storages';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditStorageComponent } from './EditStorageComponent';

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

const renderSavedStorageForm = (storage: Storage) =>
  renderScreen(
    <EditStorageComponent
      workspaceId="workspace-id"
      isShowClose={false}
      onClose={() => {}}
      isShowName
      editingStorage={storage}
      onChanged={() => {}}
    />,
  );

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
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    mockServer.use(
      mockApi.post('/storage-tests', 200, {
        isConnected: false,
        error: 'the bucket databasus-backups does not exist',
      }),
    );
    renderSavedStorageForm(savedS3Storage);

    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));

    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith(
        'The storage could not be reached: the bucket databasus-backups does not exist',
      ),
    );
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });
});
