// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Notifier } from '../../../entities/notifiers';
import { DICTIONARIES } from '../../../shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { NotifierComponent } from './NotifierComponent';

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
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
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

    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith(
        'It is still in use by a database or by a running backup or restore. Try again once nothing uses it.',
      ),
    );
    expect(onNotifierDeleted).not.toHaveBeenCalled();
  });
});
