// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Storage } from '@/entities/storages';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

import { BackupStorageSelection } from './BackupStorageSelection';

const server = setUpMockApi();
const storage: Storage = {
  id: 'created-storage',
  workspaceId: 'first-workspace',
  name: 'Submitted',
  settings: { type: 'LOCAL' },
};

describe('Composed backup storage creation', () => {
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

    fireEvent.click(await dialog.findByRole('button', { name: /Save/ }));

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

      fireEvent.mouseDown(await screen.findByRole('combobox', { name: 'Storage' }));
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

      fireEvent.mouseDown(await screen.findByRole('combobox', { name: 'Storage' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Create new storage' }));
      const dialog = within(await screen.findByRole('dialog'));

      fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), {
        target: { value: storage.name },
      });
      fireEvent.click(dialog.getByRole('button', { name: 'Test connection' }));
      fireEvent.click(await dialog.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(onSelect).toHaveBeenCalledWith(storage.id));

      view.rerender(<BackupStorageSelection {...props} selectedStorageId={storage.id} />);
      await screen.findByText(storage.name, { selector: '.ant-select-selection-item' });
      const processed = new Promise<void>((resolve) => {
        server.events.on('response:mocked', function onResponse() {
          server.events.removeListener('response:mocked', onResponse);
          resolve();
        });
      });

      await act(async () => {
        initialList.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({ items: [] }),
        );
        await processed;
      });

      expect(
        screen.getByText(storage.name, { selector: '.ant-select-selection-item' }),
      ).toBeTruthy();
      expect(screen.getByRole('combobox', { name: 'Storage' })).toBeTruthy();
      expect(view.container.querySelector('.ant-alert-error')).toBeNull();
      expect(onSelect).toHaveBeenCalledTimes(1);

      await view.user.click(screen.getByRole('combobox', { name: 'Storage' }));
      await view.user.click(
        await screen.findByText(existingStorage.name, {
          selector: '.ant-select-item-option-content',
        }),
      );

      expect(onSelect).toHaveBeenLastCalledWith(existingStorage.id);

      view.rerender(<BackupStorageSelection {...props} selectedStorageId={existingStorage.id} />);
      await view.user.click(screen.getByRole('combobox', { name: 'Storage' }));
      await view.user.click(
        await screen.findByText(storage.name, { selector: '.ant-select-item-option-content' }),
      );

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

    fireEvent.mouseDown(await screen.findByRole('combobox', { name: 'Storage' }));
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

    expect(screen.getByText(storage.name, { selector: '.ant-select-selection-item' })).toBeTruthy();

    const existingStorage = { ...storage, id: 'existing-storage', name: 'Existing archive' };
    server.use(mockApi.get('/storages', 200, { items: [existingStorage] }));

    await view.user.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() =>
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull(),
    );

    await view.user.click(screen.getByRole('combobox', { name: 'Storage' }));
    await screen.findByText(existingStorage.name, { selector: '.ant-select-item-option-content' });

    expect(
      screen.getByText(storage.name, { selector: '.ant-select-item-option-content' }),
    ).toBeTruthy();
    expect(onSelect).toHaveBeenCalledTimes(1);
  });
});
