// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Storage } from '../../../../entities/storages';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
import { activateSession } from '../../../../shared/auth';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
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
