// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Notifier } from '@/entities/notifiers';
import { NotifierComponent } from '@/pages/notifiers/ui/NotifierComponent';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const observeResponse = () =>
  new Promise<void>((resolve) => {
    const onResponse = () => {
      mockServer.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    mockServer.events.on('response:mocked', onResponse);
  });

afterEach(() => {
  vi.restoreAllMocks();
});

const usedNotifier: Notifier = {
  id: 'notifier-id',
  workspaceId: 'workspace-id',
  name: 'Team alerts',
  lastSendError: null,
  settings: { type: 'DISCORD' },
};

describe('NotifierComponent', () => {
  it('retains the acknowledged name when settings editing follows a pending rename', async () => {
    const renamed = { ...usedNotifier, name: 'Renamed alerts' };
    const webhookUrl = 'https://discord.com/api/webhooks/new-token';
    const changedNotifier: Notifier = {
      ...renamed,
      settings: { type: 'DISCORD', channelWebhookUrl: webhookUrl },
    };
    const rename = mockApi.pending('put', '/notifiers/{notifierId}');

    mockServer.use(
      mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
      mockApi.post('/notifier-tests', 200, { isDelivered: true }),
      rename.handler,
    );

    const view = renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={() => {}}
        onNotifierDeleted={() => {}}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );
    await screen.findByText(usedNotifier.name);

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
    expect(screen.queryByRole('region', { name: 'Notifier settings' })).toBeNull();

    mockServer.use(mockApi.put('/notifiers/{notifierId}', 200, changedNotifier));

    rename.respond(HttpResponse.json(renamed));
    await screen.findByText(renamed.name);

    await view.user.click(screen.getAllByRole('button', { name: 'Edit' })[1]);

    expect(screen.getByRole('button', { name: 'Edit' })).toHaveProperty('disabled', true);

    fireEvent.change(screen.getByRole('textbox', { name: 'Channel webhook URL' }), {
      target: { value: webhookUrl },
    });
    await view.user.click(screen.getByRole('button', { name: 'Send test notification' }));
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
      settings: changedNotifier.settings,
    });
    expect(screen.getByText(renamed.name)).toBeTruthy();
  });

  it('offers no retry when the delivery test fails with the notifier loaded', async () => {
    mockServer.use(
      mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
      mockApi.post('/notifiers/{notifierId}/test', 500, buildProblem('internal', 500)),
    );
    const view = renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={() => {}}
        onNotifierDeleted={() => {}}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );

    await view.user.click(await screen.findByRole('button', { name: 'Send test notification' }));
    await screen.findByText('An internal server error occurred. Please try again later.');

    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('shows translated failure when a saved delivery test returns no diagnostic', async () => {
    mockServer.use(
      mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
      mockApi.post('/notifiers/{notifierId}/test', 200, { isDelivered: false }),
    );
    const view = renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={() => {}}
        onNotifierDeleted={() => {}}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );

    await view.user.click(await screen.findByRole('button', { name: 'Send test notification' }));

    expect((await screen.findByRole('alert')).textContent).toContain(
      'The test message was not delivered',
    );
    expect(screen.getByText(usedNotifier.name)).toBeTruthy();
    expect(await screen.findByRole('button', { name: 'Send test notification' })).toHaveProperty(
      'disabled',
      false,
    );
  });

  it.each([
    { operation: 'test', hasFailed: false },
    { operation: 'test', hasFailed: true },
    { operation: 'rename', hasFailed: false },
    { operation: 'rename', hasFailed: true },
    { operation: 'delete', hasFailed: false },
    { operation: 'delete', hasFailed: true },
  ] as const)(
    'suppresses feedback and callbacks after $operation outlives the editor with failure=$hasFailed',
    async ({ operation, hasFailed }) => {
      const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
      const onNotifierChanged = vi.fn();
      const onNotifierDeleted = vi.fn();
      const method = operation === 'test' ? 'post' : operation === 'rename' ? 'put' : 'delete';
      const pendingOperation =
        operation === 'test'
          ? mockApi.pending('post', '/notifiers/{notifierId}/test')
          : operation === 'rename'
            ? mockApi.pending('put', '/notifiers/{notifierId}')
            : mockApi.pending('delete', '/notifiers/{notifierId}');
      mockServer.use(
        mockApi.get('/notifiers/{notifierId}', 200, {
          ...usedNotifier,
          lastSendError: 'Earlier failure',
        }),
        pendingOperation.handler,
      );

      const view = renderScreen(
        <div id="blocks-component-root">
          <NotifierComponent
            notifierId={usedNotifier.id}
            onNotifierChanged={onNotifierChanged}
            onNotifierDeleted={onNotifierDeleted}
            onNotifierTransferred={() => {}}
            isCanManageNotifiers
          />
        </div>,
      );
      await screen.findByText(usedNotifier.name);

      if (operation === 'test')
        fireEvent.click(screen.getByRole('button', { name: 'Send test notification' }));
      else if (operation === 'rename') {
        fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
        fireEvent.change(
          screen.getByPlaceholderText(DICTIONARIES.en.notifiers.config.namePlaceholder),
          { target: { value: 'Renamed notifier' } },
        );
        fireEvent.click(screen.getByRole('button', { name: 'Save' }));
      } else {
        fireEvent.click(screen.getByRole('button', { name: 'Remove' }));
        fireEvent.click(
          within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
        );
      }

      await waitFor(() =>
        expect(
          mockServer.getRequests().some((request) => request.method === method.toUpperCase()),
        ).toBe(true),
      );

      view.rerender(<div id="blocks-component-root">Current interface</div>);
      const processed = observeResponse();
      await act(async () => {
        pendingOperation.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 })
            : operation === 'delete'
              ? new HttpResponse(null, { status: 204 })
              : HttpResponse.json(
                  operation === 'test'
                    ? { isDelivered: true }
                    : { ...usedNotifier, name: 'Renamed notifier' },
                ),
        );
        await processed;
      });

      expect(alert).not.toHaveBeenCalled();
      expect(onNotifierChanged).not.toHaveBeenCalled();
      expect(onNotifierDeleted).not.toHaveBeenCalled();
      expect(screen.queryByText(DICTIONARIES.en.notifiers.testNotification.sent.title)).toBeNull();
    },
  );

  it('shows the in_use text when a notifier that databases use is deleted', async () => {
    const onNotifierDeleted = vi.fn();
    mockServer.use(
      mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
      mockApi.delete('/notifiers/{notifierId}', 409, buildProblem('in_use', 409)),
    );
    renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={() => {}}
        onNotifierDeleted={onNotifierDeleted}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );

    await screen.findByText('Team alerts');

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: 'Remove' }),
    );

    expect(
      await screen.findByText(
        'It is still in use by a database or by a running backup or restore. Try again once nothing uses it.',
      ),
    ).toBeTruthy();
    expect(onNotifierDeleted).not.toHaveBeenCalled();
  });

  it('retains newer name input after a delayed rename is acknowledged', async () => {
    const renamed = { ...usedNotifier, name: 'Submitted' };
    const pendingRename = mockApi.delayed({
      method: 'put',
      path: '/notifiers/{notifierId}',
      status: 200,
      body: renamed,
    });

    mockServer.use(
      mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
      pendingRename.handler,
    );
    const onChanged = vi.fn();
    renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={onChanged}
        onNotifierDeleted={() => {}}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );
    await screen.findByText(usedNotifier.name);

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

  it.each(['delivery', 'request'] as const)(
    'clears a prior %s failure after a saved notifier test succeeds',
    async (failure) => {
      mockServer.use(
        mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
        failure === 'delivery'
          ? mockApi.post('/notifiers/{notifierId}/test', 200, {
              isDelivered: false,
              error: 'Previous delivery failure',
            })
          : mockApi.post('/notifiers/{notifierId}/test', 500, buildProblem('internal', 500)),
      );
      renderScreen(
        <NotifierComponent
          notifierId={usedNotifier.id}
          onNotifierChanged={() => {}}
          onNotifierDeleted={() => {}}
          onNotifierTransferred={() => {}}
          isCanManageNotifiers
        />,
      );

      fireEvent.click(await screen.findByRole('button', { name: 'Send test notification' }));
      const failureText =
        failure === 'delivery' ? /Previous delivery failure/ : /internal|unexpected/i;
      await screen.findAllByText(failureText);

      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Send test notification' })).toHaveProperty(
          'disabled',
          false,
        ),
      );

      mockServer.use(mockApi.post('/notifiers/{notifierId}/test', 200, { isDelivered: true }));

      fireEvent.click(screen.getByRole('button', { name: 'Send test notification' }));
      await screen.findByText('Test notification sent!');

      expect(screen.queryAllByText(failureText)).toHaveLength(0);
      expect(document.querySelector('.ant-alert-error')).toBeNull();
    },
  );

  it.each([true, false])(
    'retains an acknowledged rename when a saved test finishes with delivery=%s',
    async (isDelivered) => {
      const pendingNotifierTest = mockApi.pending('post', '/notifiers/{notifierId}/test');
      const renamedNotifier = {
        ...usedNotifier,
        name: 'Renamed alerts',
        lastSendError: 'Earlier delivery failure',
      };
      mockServer.use(
        mockApi.get('/notifiers/{notifierId}', 200, {
          ...usedNotifier,
          lastSendError: 'Earlier delivery failure',
        }),
        mockApi.put('/notifiers/{notifierId}', 200, renamedNotifier),
        pendingNotifierTest.handler,
      );
      const onNotifierChanged = vi.fn();
      renderScreen(
        <NotifierComponent
          notifierId={usedNotifier.id}
          onNotifierChanged={onNotifierChanged}
          onNotifierDeleted={() => {}}
          onNotifierTransferred={() => {}}
          isCanManageNotifiers
        />,
      );

      fireEvent.click(await screen.findByRole('button', { name: 'Send test notification' }));

      await waitFor(() =>
        expect(mockServer.getRequests().some((request) => request.method === 'POST')).toBe(true),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
      fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
        target: { value: renamedNotifier.name },
      });
      fireEvent.click(screen.getByRole('button', { name: 'Save' }));
      await screen.findByText(renamedNotifier.name);

      await act(async () =>
        pendingNotifierTest.respond(
          HttpResponse.json({
            isDelivered,
            error: isDelivered ? undefined : 'Current delivery failure',
          }),
        ),
      );

      await waitFor(() =>
        expect(onNotifierChanged).toHaveBeenLastCalledWith({
          ...renamedNotifier,
          lastSendError: isDelivered ? null : 'Current delivery failure',
        }),
      );
      expect(screen.getByText(renamedNotifier.name)).toBeTruthy();
      expect(screen.queryByText(usedNotifier.name)).toBeNull();
    },
  );

  it('hides Save when the name returns to the acknowledged value', async () => {
    mockServer.use(mockApi.get('/notifiers/{notifierId}', 200, usedNotifier));
    renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={() => {}}
        onNotifierDeleted={() => {}}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );
    await screen.findByText('Team alerts');

    fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
    const nameInput = screen.getByRole('textbox', { name: 'Name' });

    fireEvent.change(nameInput, { target: { value: 'Changed name' } });

    expect(screen.getByRole('button', { name: 'Save' })).toBeTruthy();

    fireEvent.change(nameInput, { target: { value: 'Team alerts' } });

    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    expect(mockServer.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(0);
  });

  it.each(['success', 'delivery', 'request'] as const)(
    'rejects a saved notifier test after newer settings are saved with old outcome=%s',
    async (outcome) => {
      const oldTest = mockApi.pending('post', '/notifiers/{notifierId}/test');
      const changedNotifier = { ...usedNotifier, lastSendError: null };
      mockServer.use(
        mockApi.get('/notifiers/{notifierId}', 200, {
          ...usedNotifier,
          lastSendError: 'Earlier delivery failure',
        }),
        oldTest.handler,
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        mockApi.put('/notifiers/{notifierId}', 200, changedNotifier),
      );
      const onNotifierChanged = vi.fn();
      const view = renderScreen(
        <NotifierComponent
          notifierId={usedNotifier.id}
          onNotifierChanged={onNotifierChanged}
          onNotifierDeleted={() => {}}
          onNotifierTransferred={() => {}}
          isCanManageNotifiers
        />,
      );

      fireEvent.click(await screen.findByRole('button', { name: 'Send test notification' }));

      await waitFor(() =>
        expect(
          mockServer.getRequests().filter((request) => request.method === 'POST'),
        ).toHaveLength(1),
      );

      fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[1]);
      fireEvent.change(screen.getByRole('textbox', { name: 'Channel webhook URL' }), {
        target: { value: 'https://discord.com/api/webhooks/new-token' },
      });
      fireEvent.click(screen.getByRole('button', { name: 'Send test notification' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

      await waitFor(() => expect(onNotifierChanged).toHaveBeenCalledWith(changedNotifier));

      const notices = screen.queryAllByText('Test notification sent!').length;
      const processed = observeResponse();

      await act(async () => {
        oldTest.respond(
          outcome === 'request'
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json({
                isDelivered: outcome === 'success',
                error: outcome === 'delivery' ? 'Obsolete delivery failure' : undefined,
              }),
        );
        await processed;
      });

      expect(onNotifierChanged).toHaveBeenCalledTimes(1);
      expect(view.container.querySelector('.ant-alert-error')).toBeNull();
      expect(screen.queryAllByText('Test notification sent!')).toHaveLength(notices);
      expect(screen.queryByText('Obsolete delivery failure')).toBeNull();
      expect(screen.getByRole('button', { name: 'Send test notification' })).toHaveProperty(
        'disabled',
        false,
      );
    },
  );
});
