// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Storage } from '@/entities/storages';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

import { EditStorageComponent } from './EditStorageComponent';

const server = setUpMockApi();
const local: Storage = {
  id: 'created-id',
  workspaceId: 'workspace-id',
  name: 'Submitted',
  settings: { type: 'LOCAL' },
};
const props = {
  workspaceId: 'workspace-id',
  isShowName: true,
  isShowClose: true,
  onClose: () => {},
  onChanged: () => {},
};
const changeStorageName = (value: string) =>
  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), { target: { value } });
const testAndSave = async () => {
  fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
  fireEvent.click(await screen.findByRole('button', { name: /Save/ }));
};

describe('EditStorageComponent', () => {
  it('publishes a created ID without completing newer edits and updates that ID on the next save', async () => {
    const pending = mockApi.delayed({
      method: 'post',
      path: '/storages',
      status: 201,
      body: local,
    });
    server.use(
      pending.handler,
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      mockApi.put('/storages/{storageId}', 200, { ...local, name: 'Newer' }),
    );
    const onPersisted = vi.fn();
    const onChanged = vi.fn();
    renderScreen(
      <EditStorageComponent {...props} onPersisted={onPersisted} onChanged={onChanged} />,
      { isStrictMode: true },
    );

    changeStorageName('Submitted');
    await testAndSave();

    await waitFor(() =>
      expect(
        server
          .getRequests()
          .some(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
          ),
      ).toBe(true),
    );

    changeStorageName('Newer');
    pending.release();

    await waitFor(() => expect(onPersisted).toHaveBeenCalledWith(local));
    expect(onChanged).not.toHaveBeenCalled();
    expect(screen.getByDisplayValue('Newer')).toBeTruthy();
    expect(screen.queryByRole('combobox', { name: 'Type' })).toBeNull();

    fireEvent.click(await screen.findByRole('button', { name: /Save/ }));

    await waitFor(() => expect(onChanged).toHaveBeenCalledWith({ ...local, name: 'Newer' }));
    expect(
      server
        .getRequests()
        .filter(
          (request) =>
            request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
        ),
    ).toHaveLength(1);
  });

  it('becomes clean when the name returns to its acknowledged baseline while keeping test approval', async () => {
    server.use(mockApi.post('/storage-tests', 200, { isConnected: true }));
    renderScreen(<EditStorageComponent {...props} editingStorage={local} />);

    changeStorageName('Changed');
    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));
    await screen.findByRole('button', { name: /Save/ });

    changeStorageName('Submitted');

    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();

    changeStorageName('Changed again');

    expect(screen.getByRole('button', { name: /Save/ })).toBeTruthy();
  });

  it.each([true, false])(
    'rejects an obsolete test after settings A-to-B-to-A with success=%s',
    async (isConnected) => {
      const pending = mockApi.pending('post', '/storage-tests');
      server.use(pending.handler);
      renderScreen(
        <EditStorageComponent
          {...props}
          editingStorage={{
            ...local,
            settings: {
              type: 'S3',
              s3Bucket: 'bucket-a',
              isVirtualHostedStyle: false,
              isSkipTlsVerify: false,
            },
          }}
        />,
      );

      fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));

      await waitFor(() => expect(server.getRequests()).toHaveLength(1));

      fireEvent.change(screen.getByDisplayValue('bucket-a'), { target: { value: 'bucket-b' } });
      fireEvent.change(screen.getByDisplayValue('bucket-b'), { target: { value: 'bucket-a' } });
      const processed = new Promise<void>((resolve) =>
        server.events.on('response:mocked', function onResponse() {
          server.events.removeListener('response:mocked', onResponse);
          resolve();
        }),
      );

      await act(async () => {
        pending.respond(
          HttpResponse.json({ isConnected, error: isConnected ? undefined : 'Old failure' }),
        );
        await processed;
      });

      expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
      expect(screen.queryByText(/Old failure/)).toBeNull();
      expect(
        (screen.getByRole('button', { name: /Test connection/ }) as HTMLButtonElement).disabled,
      ).toBe(false);
    },
  );

  it.each([false, true])(
    'acknowledges only submitted S3 credentials when a newer secret exists=%s',
    async (hasNewerSecret) => {
      const settings = {
        type: 'S3' as const,
        s3Bucket: 'archive',
        s3AccessKey: 'submitted-access',
        s3SecretKey: 'submitted-secret',
        isVirtualHostedStyle: false,
        isSkipTlsVerify: false,
      };
      const storedSettings = {
        type: 'S3' as const,
        s3Bucket: 'archive',
        isVirtualHostedStyle: false,
        isSkipTlsVerify: false,
      };
      const createdStorage: Storage = { ...local, settings: storedSettings };
      const creation = mockApi.delayed({
        method: 'post',
        path: '/storages',
        status: 201,
        body: createdStorage,
      });
      server.use(creation.handler, mockApi.post('/storage-tests', 200, { isConnected: true }));
      const onPersisted = vi.fn();
      const onChanged = vi.fn();
      renderScreen(
        <EditStorageComponent
          {...props}
          draft={{ name: 'Submitted', settings }}
          onPersisted={onPersisted}
          onChanged={onChanged}
        />,
      );

      await testAndSave();

      await waitFor(() =>
        expect(
          server
            .getRequests()
            .some(
              (request) =>
                request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
            ),
        ).toBe(true),
      );

      const secret = screen.getByDisplayValue('submitted-secret');

      if (hasNewerSecret) fireEvent.change(secret, { target: { value: 'newer-secret' } });

      creation.release();

      await waitFor(() => expect(onPersisted).toHaveBeenCalledWith(createdStorage));
      expect((secret as HTMLInputElement).value).toBe(hasNewerSecret ? 'newer-secret' : '');
      expect(screen.queryByDisplayValue('submitted-access')).toBeNull();

      if (hasNewerSecret) expect(onChanged).not.toHaveBeenCalled();
      else expect(onChanged).toHaveBeenCalledWith(createdStorage);
    },
  );

  it('uses current callbacks when a pending storage draft receives new callbacks', async () => {
    const creation = mockApi.delayed({
      method: 'post',
      path: '/storages',
      status: 201,
      body: local,
    });
    server.use(creation.handler, mockApi.post('/storage-tests', 200, { isConnected: true }));
    const previousCompletion = vi.fn();
    const currentCompletion = vi.fn();
    const currentPersistence = vi.fn();
    const view = renderScreen(<EditStorageComponent {...props} onChanged={previousCompletion} />);

    changeStorageName('Submitted');
    await testAndSave();

    await waitFor(() =>
      expect(
        server
          .getRequests()
          .some(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
          ),
      ).toBe(true),
    );

    view.rerender(
      <EditStorageComponent
        {...props}
        onChanged={currentCompletion}
        onPersisted={currentPersistence}
      />,
    );

    expect(screen.getByDisplayValue('Submitted')).toBeTruthy();

    creation.release();

    await waitFor(() => expect(currentCompletion).toHaveBeenCalledWith(local));
    expect(currentPersistence).toHaveBeenCalledWith(local);
    expect(previousCompletion).not.toHaveBeenCalled();
  });

  it('keeps pending creates isolated when two storage editors are mounted', async () => {
    const firstRequest = mockApi.pending('post', '/storages');
    const secondRequest = mockApi.pending('post', '/storages');
    server.use(mockApi.post('/storage-tests', 200, { isConnected: true }), firstRequest.handler);
    const firstCompleted = vi.fn();
    const secondCompleted = vi.fn();
    renderScreen(
      <>
        <section aria-label="First editor">
          <EditStorageComponent {...props} onChanged={firstCompleted} />
        </section>
        <section aria-label="Second editor">
          <EditStorageComponent
            {...props}
            workspaceId="second-workspace"
            onChanged={secondCompleted}
          />
        </section>
      </>,
      { isStrictMode: true },
    );
    const first = within(screen.getByRole('region', { name: 'First editor' }));
    const second = within(screen.getByRole('region', { name: 'Second editor' }));

    for (const [editor, editorName] of [
      [first, 'First'],
      [second, 'Second'],
    ] as const) {
      if (editorName === 'Second') server.use(secondRequest.handler);

      fireEvent.change(editor.getByRole('textbox', { name: 'Name' }), {
        target: { value: editorName },
      });
      fireEvent.click(editor.getByRole('button', { name: /Test connection/ }));
      fireEvent.click(await editor.findByRole('button', { name: /Save/ }));

      await waitFor(() =>
        expect(
          server
            .getRequests()
            .filter(
              (request) =>
                request.method === 'POST' && new URL(request.url).pathname.endsWith('/storages'),
            ),
        ).toHaveLength(editorName === 'First' ? 1 : 2),
      );
    }

    await act(async () => {
      secondRequest.respond(
        HttpResponse.json(
          { ...local, id: 'second-id', workspaceId: 'second-workspace', name: 'Second' },
          { status: 201 },
        ),
      );
    });

    await waitFor(() =>
      expect(secondCompleted).toHaveBeenCalledWith(expect.objectContaining({ id: 'second-id' })),
    );
    expect(firstCompleted).not.toHaveBeenCalled();
    expect(first.getByRole('textbox', { name: 'Name' })).toHaveProperty('value', 'First');

    await act(async () => {
      firstRequest.respond(HttpResponse.json({ ...local, name: 'First' }, { status: 201 }));
    });

    await waitFor(() =>
      expect(firstCompleted).toHaveBeenCalledWith(expect.objectContaining({ id: local.id })),
    );
  });
});
