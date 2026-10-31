// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Notifier, NotifierSettings } from '../../../../entities/notifiers';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
import { DICTIONARIES } from '../../../../shared/i18n';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditNotifierComponent } from './EditNotifierComponent';

const mockServer = setUpMockApi();

afterEach(() => {
  vi.restoreAllMocks();
});

const buildSavedNotifier = (settings: NotifierSettings): Notifier => ({
  id: 'notifier-id',
  workspaceId: 'workspace-id',
  name: 'Team alerts',
  lastSendError: null,
  settings,
});

const savedSlackNotifier = buildSavedNotifier({
  type: 'SLACK',
  targetChatId: 'C0123',
});

const savedWebhookNotifier = buildSavedNotifier({
  type: 'WEBHOOK',
  webhookUrl: 'https://hooks.example.com/backups',
  webhookMethod: 'POST',
  headers: [{ name: 'X-Api-Key' }],
  acceptedCategories: ['ALL'],
});

const renderSavedNotifierForm = (notifier: Notifier) =>
  renderScreen(
    <EditNotifierComponent
      workspaceId="workspace-id"
      isShowClose={false}
      onClose={() => {}}
      isShowName
      editingNotifier={notifier}
      onPersisted={() => {}}
      onChanged={() => {}}
    />,
  );

const mockUpdate = () => {
  const sentUpdates: ApiComponents['schemas']['NotifierUpdate'][] = [];
  mockServer.use(
    http.put(`*${API_BASE_PATH}/notifiers/:notifierId`, async ({ request }) => {
      const sentUpdate = (await request.json()) as ApiComponents['schemas']['NotifierUpdate'];
      sentUpdates.push(sentUpdate);

      return HttpResponse.json({
        ...buildSavedNotifier(sentUpdate.settings),
        name: sentUpdate.name,
      });
    }),
  );

  return sentUpdates;
};

const renameAndSave = async (name: string) => {
  fireEvent.change(screen.getByDisplayValue('Team alerts'), {
    target: { value: name },
  });
  fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
  fireEvent.click(await screen.findByRole('button', { name: /Save/ }));
};

describe('EditNotifierComponent', () => {
  it('hides actions when an edit returns to the saved baseline', () => {
    renderSavedNotifierForm(savedWebhookNotifier);
    const name = screen.getByDisplayValue('Team alerts');

    fireEvent.change(name, { target: { value: 'Changed alerts' } });
    fireEvent.change(name, { target: { value: 'Team alerts' } });

    expect(screen.queryByRole('button', { name: /Send test notification/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
  });

  it('rejects a successful test when settings change before its response', async () => {
    const testResponse = mockApi.delayed({
      method: 'post',
      path: '/notifier-tests',
      status: 200,
      body: { isDelivered: true },
    });
    mockServer.use(testResponse.handler);
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(screen.getByDisplayValue('Team alerts'), { target: { value: 'Changed' } });
    fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
    fireEvent.change(screen.getByDisplayValue('https://hooks.example.com/backups'), {
      target: { value: 'https://hooks.example.com/changed' },
    });
    const responded = new Promise<void>((resolve) =>
      mockServer.events.on('response:mocked', () => resolve()),
    );
    await act(async () => {
      testResponse.release();
      await responded;
    });

    expect(
      screen.queryByText(DICTIONARIES.en.notifiers.testNotification.sent.title, { exact: true }),
    ).toBeNull();
    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
    expect(screen.getByRole('button', { name: /Send test notification/ })).toBeTruthy();
  });

  it('keeps newer edits when a submitted save completes', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const saveResponse = mockApi.delayed({
      method: 'put',
      path: '/notifiers/{notifierId}',
      status: 200,
      body: { ...savedWebhookNotifier, name: 'Submitted' },
    });
    mockServer.use(saveResponse.handler);
    renderSavedNotifierForm(savedWebhookNotifier);
    await renameAndSave('Submitted');

    fireEvent.change(screen.getByDisplayValue('Submitted'), { target: { value: 'Newer edit' } });
    saveResponse.release();

    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: /Save/ })?.classList.contains('ant-btn-loading') ??
          false,
      ).toBe(false),
    );
    expect(screen.getByDisplayValue('Newer edit')).toBeTruthy();
    expect(screen.getByRole('button', { name: /Save/ })).toBeTruthy();
  });
  it('shows the type of a saved notifier as text instead of a type selector', () => {
    renderSavedNotifierForm(savedSlackNotifier);
    expect(screen.getByText('Slack')).toBeTruthy();
    expect(screen.queryByRole('combobox')).toBeNull();
  });

  it('sends no bot token when a renamed Slack notifier is saved', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const sentUpdates = mockUpdate();
    renderSavedNotifierForm(savedSlackNotifier);

    await renameAndSave('Renamed alerts');

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0]).toEqual({
      name: 'Renamed alerts',
      settings: { type: 'SLACK', targetChatId: 'C0123' },
    });
  });

  it('keeps the name of a saved webhook header and sends no value for it', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const sentUpdates = mockUpdate();
    renderSavedNotifierForm(savedWebhookNotifier);

    expect(screen.getByDisplayValue('X-Api-Key')).toBeTruthy();

    await renameAndSave('Renamed webhook');

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0].settings).toMatchObject({
      headers: [{ name: 'X-Api-Key' }],
    });
    expect(sentUpdates[0].settings).not.toHaveProperty('headers.0.value');
  });

  it("shows the channel's error when the test message is not delivered", async () => {
    mockServer.use(
      mockApi.post('/notifier-tests', 200, {
        isDelivered: false,
        error: 'the webhook answered 500 Internal Server Error',
      }),
    );
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(screen.getByDisplayValue('Team alerts'), {
      target: { value: 'Renamed' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

    expect((await screen.findByRole('alert')).textContent).toContain(
      'The test message was not delivered: the webhook answered 500 Internal Server Error',
    );
    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
  });
});
