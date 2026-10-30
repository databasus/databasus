// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Notifier, NotifierSettings } from '../../../../entities/notifiers';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
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
      onChanged={() => {}}
    />,
  );

// Records every update the form sends and answers it with the notifier the update describes.
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
  fireEvent.click(screen.getByRole('button', { name: 'Send test notification' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Save' }));
};

describe('EditNotifierComponent', () => {
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
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
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
    fireEvent.click(screen.getByRole('button', { name: 'Send test notification' }));

    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith(
        'The test message was not delivered: the webhook answered 500 Internal Server Error',
      ),
    );
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });
});
