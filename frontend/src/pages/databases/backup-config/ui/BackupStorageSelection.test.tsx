// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Storage } from '@/entities/storages';
import { BackupStorageSelection } from '@/pages/databases/backup-config/ui/BackupStorageSelection';
import { finishCancelledApiResponse } from '@/shared/lib/test/finishCancelledApiResponse';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const storage: Storage = {
  id: 'created-storage',
  workspaceId: 'first-workspace',
  name: 'Submitted',
  settings: { type: 'LOCAL' },
};

describe('Composed backup storage creation', () => {
  it('offers every supported provider when creating storage from backup settings', async () => {
    server.use(mockApi.get('/storages', 200, { items: [] }));
    const { user } = renderScreen(
      <BackupStorageSelection workspaceId={storage.workspaceId} canClear onSelect={() => {}} />,
    );

    await user.click(await screen.findByRole('combobox', { name: 'Storage' }));
    await user.click(await screen.findByRole('button', { name: 'Create new storage' }));
    const dialog = within(await screen.findByRole('dialog'));
    await user.click(dialog.getByRole('combobox', { name: 'Type' }));

    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'Local storage',
      'S3',
      'NAS',
      'Azure Blob Storage',
      'FTP',
      'SFTP',
    ]);
  });

  it('keeps newer input open after creation and selects the assigned ID after updating it', async () => {
    const pending = mockApi.delayed({
      method: 'post',
      path: '/storages',
      status: 201,
      body: storage,
    });
    server.use(
      mockApi.get('/storages', 200, { items: [] }),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      pending.handler,
      mockApi.put('/storages/{storageId}', 200, { ...storage, name: 'Newer' }),
    );
    const onSelect = vi.fn();
    const view = renderScreen(
      <BackupStorageSelection
        workspaceId={storage.workspaceId}
        selectedStorageId="previous-id"
        canClear
        onSelect={onSelect}
      />,
    );

    expect(screen.queryByRole('button', { name: 'Create new storage' })).toBeNull();

    await view.user.click(await screen.findByRole('combobox', { name: 'Storage' }));
    await view.user.tab();

    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Create new storage' }));

    await view.user.keyboard('{Enter}');
    const dialog = within(await screen.findByRole('dialog'));

    fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), {
      target: { value: storage.name },
    });
    fireEvent.click(dialog.getByRole('button', { name: /Test connection/ }));
    fireEvent.click(await dialog.findByRole('button', { name: /Save/ }));

    await waitFor(() =>
      expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(2),
    );

    fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), { target: { value: 'Newer' } });
    pending.release();

    await waitFor(() => expect(dialog.queryByRole('combobox', { name: 'Type' })).toBeNull());
    expect(onSelect).not.toHaveBeenCalled();
    expect(dialog.getByDisplayValue('Newer')).toBeTruthy();

    const saveButton = await dialog.findByRole('button', { name: /Save/ });
    await waitFor(() => {
      expect(saveButton.hasAttribute('disabled')).toBe(false);
      expect(saveButton.getAttribute('aria-busy') === 'true').toBe(false);
    });
    await view.user.click(saveButton);

    await waitFor(() => expect(onSelect).toHaveBeenCalledWith(storage.id));
    expect(
      server
        .getRequests()
        .filter(
          (request) =>
            request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
        ),
    ).toHaveLength(1);
  });

  it.each(['cancel', 'workspace', 'unmount'] as const)(
    'rejects a late nested create after %s',
    async (change) => {
      const pending = mockApi.delayed({
        method: 'post',
        path: '/storages',
        status: 201,
        body: storage,
      });
      server.use(
        mockApi.get('/storages', 200, { items: [] }),
        mockApi.post('/storage-tests', 200, { isConnected: true }),
        pending.handler,
      );
      const onSelect = vi.fn();
      const view = renderScreen(
        <BackupStorageSelection
          workspaceId={storage.workspaceId}
          selectedStorageId="previous-id"
          canClear
          onSelect={onSelect}
        />,
        { isStrictMode: true },
      );

      fireEvent.click(await screen.findByRole('combobox', { name: 'Storage' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Create new storage' }));
      const dialog = within(await screen.findByRole('dialog'));

      fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), {
        target: { value: storage.name },
      });
      fireEvent.click(dialog.getByRole('button', { name: /Test connection/ }));
      fireEvent.click(await dialog.findByRole('button', { name: /Save/ }));

      await waitFor(() =>
        expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(2),
      );

      if (change === 'cancel') fireEvent.click(dialog.getByRole('button', { name: 'Close' }));
      else if (change === 'workspace')
        view.rerender(
          <BackupStorageSelection
            workspaceId="second-workspace"
            selectedStorageId="second-storage"
            canClear
            onSelect={onSelect}
          />,
        );
      else view.unmount();
      const response = new Promise<void>((resolve) =>
        server.events.on('response:mocked', function onResponse() {
          server.events.removeListener('response:mocked', onResponse);
          resolve();
        }),
      );

      await act(async () => {
        pending.release();
        await response;
      });

      expect(onSelect).not.toHaveBeenCalled();
      expect(screen.queryByDisplayValue(storage.name)).toBeNull();
    },
  );

  it.each([false, true])(
    'retains the created storage after the initial list completes with failure=%s',
    async (hasFailed) => {
      const initialList = mockApi.pending('get', '/storages');
      server.use(
        initialList.handler,
        mockApi.post('/storage-tests', 200, { isConnected: true }),
        mockApi.post('/storages', 201, storage),
      );

      const existingStorage = { ...storage, id: 'existing-storage', name: 'Existing archive' };
      const onSelect = vi.fn();
      const props = { workspaceId: storage.workspaceId, canClear: true, onSelect };
      const view = renderScreen(<BackupStorageSelection {...props} />, { isStrictMode: true });

      await waitFor(() =>
        expect(server.getRequests().some((request) => request.method === 'GET')).toBe(true),
      );

      server.use(mockApi.get('/storages', 200, { items: [existingStorage] }));

      fireEvent.click(await screen.findByRole('combobox', { name: 'Storage' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Create new storage' }));
      const dialog = within(await screen.findByRole('dialog'));

      fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), {
        target: { value: storage.name },
      });
      fireEvent.click(dialog.getByRole('button', { name: 'Test connection' }));
      fireEvent.click(await dialog.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(onSelect).toHaveBeenCalledWith(storage.id));

      view.rerender(<BackupStorageSelection {...props} selectedStorageId={storage.id} />);
      await within(screen.getByRole('combobox', { name: 'Storage' })).findByText(storage.name);
      await finishCancelledApiResponse(server, '/api/v1/storages', () => {
        initialList.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [] }),
        );
      });

      expect(
        within(screen.getByRole('combobox', { name: 'Storage' })).getByText(storage.name),
      ).toBeTruthy();
      expect(screen.getByRole('combobox', { name: 'Storage' })).toBeTruthy();
      expect(view.container.querySelector('[role="alert"]')).toBeNull();
      expect(onSelect).toHaveBeenCalledTimes(1);

      await view.user.click(screen.getByRole('combobox', { name: 'Storage' }));
      await view.user.click(await screen.findByRole('option', { name: existingStorage.name }));

      expect(onSelect).toHaveBeenLastCalledWith(existingStorage.id);

      view.rerender(<BackupStorageSelection {...props} selectedStorageId={existingStorage.id} />);
      await view.user.click(screen.getByRole('combobox', { name: 'Storage' }));
      await view.user.click(await screen.findByRole('option', { name: storage.name }));

      expect(onSelect).toHaveBeenLastCalledWith(storage.id);
    },
  );

  it('retains the acknowledged option and retries when the post-creation refresh fails', async () => {
    server.use(
      mockApi.get('/storages', 200, { items: [] }),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      mockApi.post('/storages', 201, storage),
    );
    const onSelect = vi.fn();
    const props = { workspaceId: storage.workspaceId, canClear: true, onSelect };
    const view = renderScreen(<BackupStorageSelection {...props} />);

    fireEvent.click(await screen.findByRole('combobox', { name: 'Storage' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Create new storage' }));
    const editor = within(await screen.findByRole('dialog'));

    fireEvent.change(editor.getByRole('textbox', { name: 'Name' }), {
      target: { value: storage.name },
    });
    fireEvent.click(editor.getByRole('button', { name: 'Test connection' }));
    server.use(mockApi.get('/storages', 500, buildProblem('internal', 500)));

    fireEvent.click(await editor.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(onSelect).toHaveBeenCalledWith(storage.id));

    view.rerender(<BackupStorageSelection {...props} selectedStorageId={storage.id} />);
    await screen.findByText('An internal server error occurred. Please try again later.');

    expect(
      within(screen.getByRole('combobox', { name: 'Storage' })).getByText(storage.name),
    ).toBeTruthy();

    const existingStorage = { ...storage, id: 'existing-storage', name: 'Existing archive' };
    server.use(mockApi.get('/storages', 200, { items: [existingStorage] }));

    await view.user.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() =>
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull(),
    );

    await view.user.click(screen.getByRole('combobox', { name: 'Storage' }));
    await screen.findByRole('option', { name: existingStorage.name });

    expect(screen.getByRole('option', { name: storage.name })).toBeTruthy();
    expect(onSelect).toHaveBeenCalledTimes(1);
  });
});
