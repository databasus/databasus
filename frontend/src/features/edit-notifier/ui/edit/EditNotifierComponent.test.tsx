// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Notifier, NotifierSettings } from '../../../../entities/notifiers';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
import { DICTIONARIES, type Locale } from '../../../../shared/i18n';
import { en } from '../../../../shared/i18n/locales/en';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { watchNoticeTimer } from '../../../../shared/lib/test/noticeTimer';
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

const renderSavedNotifierForm = (notifier: Notifier, options?: { locale: Locale }) =>
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
    options,
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

const headerNameError = DICTIONARIES.en.notifiers.webhook.headerNameInvalid;

const getHeaderNameInput = () => screen.getAllByRole('textbox', { name: 'Header name' })[0];

const expectHeaderNameRefused = () => {
  expect(screen.getByText(headerNameError)).toBeTruthy();
  expect(
    screen.getByRole('button', { name: /Send test notification/ }).hasAttribute('disabled'),
  ).toBe(true);
  expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
};

const expectTestAvailableWithoutHeaderNameError = async () => {
  await waitFor(() => expect(screen.queryByText(headerNameError)).toBeNull());
  expect(
    screen.getByRole('button', { name: /Send test notification/ }).hasAttribute('disabled'),
  ).toBe(false);
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

  it('leaves no notice timer running after the form unmounts', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const noticeTimer = watchNoticeTimer(4_000);
    const view = renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(screen.getByDisplayValue('Team alerts'), { target: { value: 'Changed' } });
    fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
    await waitFor(() => expect(noticeTimer.getStartedCount()).toBeGreaterThan(0));
    view.unmount();

    expect(noticeTimer.getLeftRunningCount()).toBe(0);
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

  it('shows an error and leaves no test or save when a webhook header name has a space', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getHeaderNameInput(), { target: { value: 'X Team' } });

    expectHeaderNameRefused();
  });

  it('shows an error and leaves no test or save when a webhook header name has a character outside the HTTP token', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getHeaderNameInput(), { target: { value: 'X:Team' } });

    expectHeaderNameRefused();
  });

  it('offers the test when a webhook header name holds token punctuation', async () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getHeaderNameInput(), { target: { value: 'X~Team!1' } });

    await expectTestAvailableWithoutHeaderNameError();
  });

  it('shows an error and leaves no test or save right after a webhook header row is added', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.click(screen.getByRole('button', { name: /Add header/ }));

    expectHeaderNameRefused();
  });

  it('offers the test again when the invalid webhook header name is corrected', async () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getHeaderNameInput(), { target: { value: 'X Team' } });

    expectHeaderNameRefused();

    fireEvent.change(getHeaderNameInput(), { target: { value: 'X-Team' } });

    await expectTestAvailableWithoutHeaderNameError();
  });

  it('offers the test again when the invalid webhook header row is removed', async () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(screen.getByDisplayValue('Team alerts'), { target: { value: 'Changed' } });
    fireEvent.click(screen.getByRole('button', { name: /Add header/ }));

    expectHeaderNameRefused();

    fireEvent.click(screen.getAllByRole('button', { name: /Remove/ })[1]);

    await expectTestAvailableWithoutHeaderNameError();
  });

  it('trims spaces around a webhook header name when the field loses focus and saves the trimmed name', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const sentUpdates = mockUpdate();
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getHeaderNameInput(), { target: { value: ' X-Team ' } });
    expect(screen.queryByText(headerNameError)).toBeNull();

    fireEvent.blur(getHeaderNameInput());

    expect((getHeaderNameInput() as HTMLInputElement).value).toBe('X-Team');
    expect(screen.queryByText(headerNameError)).toBeNull();

    await renameAndSave('Renamed webhook');

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0].settings).toMatchObject({ headers: [{ name: 'X-Team' }] });
  });

  it('empties a webhook header name of only spaces when the field loses focus and refuses it', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getHeaderNameInput(), { target: { value: '   ' } });
    fireEvent.blur(getHeaderNameInput());

    expect((getHeaderNameInput() as HTMLInputElement).value).toBe('');
    expectHeaderNameRefused();
  });

  it('keeps Save after a successful test when a webhook header name field is left unchanged', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(screen.getByDisplayValue('Team alerts'), { target: { value: 'Changed' } });
    fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
    await screen.findByRole('button', { name: /Save/ });

    fireEvent.focus(getHeaderNameInput());
    fireEvent.blur(getHeaderNameInput());

    expect(screen.getByRole('button', { name: /Save/ })).toBeTruthy();
  });

  it('shows the error when a saved webhook has a stored invalid header name', () => {
    renderSavedNotifierForm(
      buildSavedNotifier({
        type: 'WEBHOOK',
        webhookUrl: 'https://hooks.example.com/backups',
        webhookMethod: 'POST',
        headers: [{ name: 'X Team' }],
        acceptedCategories: ['ALL'],
      }),
    );

    expect(screen.getByText(headerNameError)).toBeTruthy();

    fireEvent.change(screen.getByDisplayValue('Team alerts'), { target: { value: 'Changed' } });

    expectHeaderNameRefused();
  });

  it('shows the webhook header name error in the language of the interface', () => {
    renderSavedNotifierForm(
      buildSavedNotifier({
        type: 'WEBHOOK',
        webhookUrl: 'https://hooks.example.com/backups',
        webhookMethod: 'POST',
        headers: [{ name: 'X Team' }],
        acceptedCategories: ['ALL'],
      }),
      { locale: 'ru' },
    );

    expect(screen.getByText(DICTIONARIES.ru.notifiers.webhook.headerNameInvalid)).toBeTruthy();
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

const providerNotifiers: {
  provider: string;
  typeLabel: string;
  deliveryModeLabel?: string;
  entered: Record<string, string>;
  secretLabels: Record<string, string>;
  sentOnCreate: NotifierSettings;
  saved: NotifierSettings;
}[] = [
  {
    provider: 'Telegram',
    typeLabel: 'Telegram',
    entered: { 'Bot token': 'telegram-bot-token', 'Target chat ID': '-100123' },
    secretLabels: { botToken: 'Bot token', proxyUrl: 'Proxy URL' },
    sentOnCreate: {
      type: 'TELEGRAM',
      botToken: 'telegram-bot-token',
      targetChatId: '-100123',
      isProxyEnabled: false,
    },
    saved: { type: 'TELEGRAM', targetChatId: '-100123', isProxyEnabled: true },
  },
  {
    provider: 'Discord',
    typeLabel: 'Discord',
    entered: { 'Channel webhook URL': 'https://discord.example.com/hooks/key' },
    secretLabels: { channelWebhookUrl: 'Channel webhook URL' },
    sentOnCreate: { type: 'DISCORD', channelWebhookUrl: 'https://discord.example.com/hooks/key' },
    saved: { type: 'DISCORD' },
  },
  {
    provider: 'Teams',
    typeLabel: 'Teams',
    entered: { 'Power Automate URL': 'https://teams.example.com/flow' },
    secretLabels: { powerAutomateUrl: 'Power Automate URL' },
    sentOnCreate: { type: 'TEAMS', powerAutomateUrl: 'https://teams.example.com/flow' },
    saved: { type: 'TEAMS' },
  },
  {
    provider: 'Mattermost with a webhook',
    typeLabel: 'Mattermost',
    entered: { 'Incoming webhook URL': 'https://mattermost.example.com/hooks/key' },
    secretLabels: { webhookUrl: 'Incoming webhook URL' },
    sentOnCreate: {
      type: 'MATTERMOST',
      deliveryMode: 'WEBHOOK',
      webhookUrl: 'https://mattermost.example.com/hooks/key',
      isInsecureSkipVerify: false,
    },
    saved: { type: 'MATTERMOST', deliveryMode: 'WEBHOOK', isInsecureSkipVerify: false },
  },
  {
    provider: 'Mattermost with a bot',
    typeLabel: 'Mattermost',
    deliveryModeLabel: 'Bot account',
    entered: {
      'Server URL': 'https://mattermost.example.com',
      'Bot token': 'mattermost-bot-token',
      'Channel ID': 'abcdefghijklmnopqrstuvwxyz',
    },
    secretLabels: { botToken: 'Bot token' },
    sentOnCreate: {
      type: 'MATTERMOST',
      deliveryMode: 'BOT',
      serverUrl: 'https://mattermost.example.com',
      botToken: 'mattermost-bot-token',
      targetChannelId: 'abcdefghijklmnopqrstuvwxyz',
      isInsecureSkipVerify: false,
    },
    saved: {
      type: 'MATTERMOST',
      deliveryMode: 'BOT',
      serverUrl: 'https://mattermost.example.com',
      targetChannelId: 'abcdefghijklmnopqrstuvwxyz',
      isInsecureSkipVerify: false,
    },
  },
];

const getProviderField = (label: string) => screen.getByRole('textbox', { name: label });

// Records every notifier the form creates and answers it with the notifier the request describes.
const mockCreation = () => {
  const sentNotifiers: ApiComponents['schemas']['NotifierCreate'][] = [];
  mockServer.use(
    mockApi.post('/notifier-tests', 200, { isDelivered: true }),
    http.post(`*${API_BASE_PATH}/notifiers`, async ({ request }) => {
      const sentNotifier = (await request.json()) as ApiComponents['schemas']['NotifierCreate'];
      sentNotifiers.push(sentNotifier);

      return HttpResponse.json(
        { ...buildSavedNotifier(sentNotifier.settings), name: sentNotifier.name },
        { status: 201 },
      );
    }),
  );

  return sentNotifiers;
};

const renderNewNotifierForm = async (typeLabel: string) => {
  const view = renderScreen(
    <EditNotifierComponent
      workspaceId="workspace-id"
      isShowClose={false}
      onClose={() => {}}
      isShowName
      onPersisted={() => {}}
      onChanged={() => {}}
    />,
  );

  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
    target: { value: 'New alerts' },
  });
  await view.user.click(screen.getByRole('combobox', { name: 'Type' }));
  fireEvent.click((await screen.findAllByTitle(typeLabel)).pop()!);

  return view;
};

const testAndSaveProvider = async () => {
  fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
  fireEvent.click(await screen.findByRole('button', { name: /Save/ }));
};

describe('EditNotifierComponent providers', () => {
  it.each(providerNotifiers)(
    'sends the entered settings when a $provider notifier is created',
    async ({ typeLabel, deliveryModeLabel, entered, sentOnCreate }) => {
      const sentNotifiers = mockCreation();
      const view = await renderNewNotifierForm(typeLabel);

      if (deliveryModeLabel) {
        await view.user.click(screen.getByRole('combobox', { name: 'Connect via' }));
        fireEvent.click(await screen.findByTitle(deliveryModeLabel));
      }
      for (const [label, value] of Object.entries(entered))
        fireEvent.change(getProviderField(label), { target: { value } });
      await testAndSaveProvider();

      await waitFor(() => expect(sentNotifiers).toHaveLength(1));
      expect(sentNotifiers[0].name).toBe('New alerts');
      expect(sentNotifiers[0].settings).toEqual(sentOnCreate);
    },
  );

  it.each(providerNotifiers)(
    'omits every secret when a saved $provider notifier is renamed',
    async ({ saved }) => {
      mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
      const sentUpdates = mockUpdate();
      renderSavedNotifierForm(buildSavedNotifier(saved));

      await renameAndSave('Renamed alerts');

      await waitFor(() => expect(sentUpdates).toHaveLength(1));
      expect(sentUpdates[0]).toEqual({ name: 'Renamed alerts', settings: saved });
    },
  );

  it.each(providerNotifiers)(
    'sends the replacement when a secret of a saved $provider notifier is entered again',
    async ({ saved, secretLabels }) => {
      mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
      const sentUpdates = mockUpdate();
      renderSavedNotifierForm(buildSavedNotifier(saved));

      const replacedSecrets = Object.fromEntries(
        Object.keys(secretLabels).map((secretField) => [
          secretField,
          `https://example.com/new-${secretField}`,
        ]),
      );
      for (const [secretField, label] of Object.entries(secretLabels))
        fireEvent.change(getProviderField(label), {
          target: { value: replacedSecrets[secretField] },
        });
      fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
      fireEvent.click(await screen.findByRole('button', { name: /Save/ }));

      await waitFor(() => expect(sentUpdates).toHaveLength(1));
      expect(sentUpdates[0].settings).toEqual({ ...saved, ...replacedSecrets });
    },
  );

  it('sends the entered settings when an email notifier is created', async () => {
    const sentNotifiers = mockCreation();
    await renderNewNotifierForm('Email');

    fireEvent.change(getProviderField('Target email'), { target: { value: 'ops@example.com' } });
    fireEvent.change(getProviderField('SMTP host'), { target: { value: 'smtp.example.com' } });
    fireEvent.change(screen.getByRole('spinbutton', { name: 'SMTP port' }), {
      target: { value: '465' },
    });
    fireEvent.change(getProviderField('SMTP user'), { target: { value: 'ops' } });
    fireEvent.change(screen.getByLabelText('SMTP password'), {
      target: { value: 'smtp-password' },
    });
    await testAndSaveProvider();

    await waitFor(() => expect(sentNotifiers).toHaveLength(1));
    expect(sentNotifiers[0].name).toBe('New alerts');
    expect(sentNotifiers[0].settings).toEqual({
      type: 'EMAIL',
      targetEmail: 'ops@example.com',
      smtpHost: 'smtp.example.com',
      smtpPort: 465,
      smtpUser: 'ops',
      smtpPassword: 'smtp-password',
      security: 'TLS',
      isInsecureSkipVerify: false,
    });
  });

  it('sends the entered settings when a Slack notifier is created', async () => {
    const sentNotifiers = mockCreation();
    await renderNewNotifierForm('Slack');

    fireEvent.change(getProviderField('Bot token'), { target: { value: 'xoxb-entered-token' } });
    fireEvent.change(getProviderField('Target chat ID'), { target: { value: 'C0123' } });
    await testAndSaveProvider();

    await waitFor(() => expect(sentNotifiers).toHaveLength(1));
    expect(sentNotifiers[0].name).toBe('New alerts');
    expect(sentNotifiers[0].settings).toEqual({
      type: 'SLACK',
      botToken: 'xoxb-entered-token',
      targetChatId: 'C0123',
    });
  });

  it('sends the entered settings with the header value when a webhook notifier is created', async () => {
    const sentNotifiers = mockCreation();
    const view = await renderNewNotifierForm('Webhook');

    fireEvent.change(getProviderField('Webhook URL'), {
      target: { value: 'https://hooks.example.com/backups' },
    });
    await view.user.click(screen.getByRole('button', { name: /Add header/ }));
    fireEvent.change(getProviderField('Header name'), { target: { value: 'X-Api-Key' } });
    fireEvent.change(getProviderField('Header value'), { target: { value: 'entered-value' } });
    await testAndSaveProvider();

    await waitFor(() => expect(sentNotifiers).toHaveLength(1));
    expect(sentNotifiers[0].name).toBe('New alerts');
    expect(sentNotifiers[0].settings).toEqual({
      type: 'WEBHOOK',
      webhookUrl: 'https://hooks.example.com/backups',
      webhookMethod: 'POST',
      headers: [{ name: 'X-Api-Key', value: 'entered-value' }],
      acceptedCategories: ['ALL'],
    });
  });

  it('sends the entered token when the bot token of a saved Slack notifier is entered again', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const sentUpdates = mockUpdate();
    renderSavedNotifierForm(savedSlackNotifier);

    fireEvent.change(getProviderField('Bot token'), { target: { value: 'xoxb-new-token' } });
    await testAndSaveProvider();

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0].settings).toEqual({
      type: 'SLACK',
      targetChatId: 'C0123',
      botToken: 'xoxb-new-token',
    });
  });

  it('sends the entered value when a header of a saved webhook notifier gets a value', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const sentUpdates = mockUpdate();
    renderSavedNotifierForm(savedWebhookNotifier);

    fireEvent.change(getProviderField('Header value'), { target: { value: 'new-value' } });
    await testAndSaveProvider();

    await waitFor(() => expect(sentUpdates).toHaveLength(1));
    expect(sentUpdates[0].settings).toMatchObject({
      headers: [{ name: 'X-Api-Key', value: 'new-value' }],
    });
  });
});

describe('EditNotifierComponent address change', () => {
  const addressChangedMessage = en.common.messages.addressChangedEnterSecretAgain;
  const keptSecretPlaceholder = en.notifiers.secretKeptPlaceholder;
  const savedHeadersNotice = en.notifiers.webhook.savedHeadersHidden;

  const savedEmailNotifier = buildSavedNotifier({
    type: 'EMAIL',
    targetEmail: 'alerts@example.com',
    smtpHost: 'smtp.example.com',
    smtpPort: 587,
    smtpUser: 'mailer',
    isInsecureSkipVerify: false,
    security: 'STARTTLS',
  });

  const savedTelegramNotifier = buildSavedNotifier({
    type: 'TELEGRAM',
    targetChatId: '-100123',
    botApiUrl: 'https://telegram.example.com',
    isProxyEnabled: false,
  });

  const savedMattermostBot = buildSavedNotifier({
    type: 'MATTERMOST',
    deliveryMode: 'BOT',
    serverUrl: 'https://mattermost.example.com',
    targetChannelId: 'abcdefghijklmnopqrstuvwxyz',
    isInsecureSkipVerify: false,
  });

  const savedMattermostWebhook = buildSavedNotifier({
    type: 'MATTERMOST',
    deliveryMode: 'WEBHOOK',
    serverUrl: 'https://mattermost.example.com',
    isInsecureSkipVerify: false,
  });

  const getTestButton = () =>
    screen.getByRole('button', { name: /Send test notification/ }) as HTMLButtonElement;

  const renameNotifier = () =>
    fireEvent.change(screen.getByDisplayValue('Team alerts'), { target: { value: 'Renamed' } });

  const changeField = (label: string, value: string) =>
    fireEvent.change(screen.getByRole('textbox', { name: label }), { target: { value } });

  it('takes back the save control when the SMTP host changes after a successful test', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    renderSavedNotifierForm(savedEmailNotifier);

    renameNotifier();
    fireEvent.click(getTestButton());
    await screen.findByRole('button', { name: /Save/ });

    changeField('SMTP host', 'moved.example.com');

    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('takes back the save control when the webhook URL changes after a successful test', async () => {
    mockServer.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    renderSavedNotifierForm(savedWebhookNotifier);

    renameNotifier();
    fireEvent.click(getTestButton());
    await screen.findByRole('button', { name: /Save/ });

    changeField('Webhook URL', 'https://moved.example.com');

    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for the SMTP password and hides the kept placeholder when the SMTP host changes', () => {
    renderSavedNotifierForm(savedEmailNotifier);

    changeField('SMTP host', 'moved.example.com');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(screen.queryByPlaceholderText(keptSecretPlaceholder)).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for the SMTP password when the SMTP port changes', () => {
    renderSavedNotifierForm(savedEmailNotifier);

    fireEvent.change(screen.getByLabelText('SMTP port'), { target: { value: '2525' } });

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(getTestButton().disabled).toBe(true);
  });

  it('brings the kept placeholder back when the SMTP host returns to the saved value', () => {
    renderSavedNotifierForm(savedEmailNotifier);

    renameNotifier();
    changeField('SMTP host', 'moved.example.com');
    changeField('SMTP host', 'smtp.example.com');

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(screen.getByPlaceholderText(keptSecretPlaceholder)).toBeTruthy();
    expect(getTestButton().disabled).toBe(false);
  });

  it('asks for no password when the SMTP host changes and the SMTP user is cleared', () => {
    renderSavedNotifierForm(savedEmailNotifier);

    changeField('SMTP host', 'moved.example.com');
    changeField('SMTP user', '');

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });

  it('keeps a typed SMTP password when the SMTP host returns to the saved value', () => {
    renderSavedNotifierForm(savedEmailNotifier);

    changeField('SMTP host', 'moved.example.com');
    fireEvent.change(screen.getByLabelText('SMTP password'), { target: { value: 'typed' } });
    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);

    changeField('SMTP host', 'smtp.example.com');

    expect((screen.getByLabelText('SMTP password') as HTMLInputElement).value).toBe('typed');
  });

  it('asks for the header value and hides the notice when the webhook URL changes', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    changeField('Webhook URL', 'https://hooks.example.com/backups/');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(screen.queryByText(savedHeadersNotice)).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for the saved header under another letter case when the webhook URL changes', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    changeField('Webhook URL', 'https://moved.example.com');
    changeField('Header name', 'x-api-key');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for no value of a new header when the webhook URL changes', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    changeField('Webhook URL', 'https://moved.example.com');
    changeField('Header name', 'X-New-Header');

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });

  it('brings the notice back when the webhook URL returns to the saved value', () => {
    renderSavedNotifierForm(savedWebhookNotifier);

    renameNotifier();
    changeField('Webhook URL', 'https://moved.example.com');
    changeField('Webhook URL', 'https://hooks.example.com/backups');

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(screen.getByText(savedHeadersNotice)).toBeTruthy();
    expect(getTestButton().disabled).toBe(false);
  });

  it('asks for the bot token when the Telegram Bot API URL changes', () => {
    renderSavedNotifierForm(savedTelegramNotifier);

    changeField('Bot API URL', 'https://moved.example.com');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(screen.queryByPlaceholderText(keptSecretPlaceholder)).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for the bot token when the Telegram proxy is turned on', () => {
    renderSavedNotifierForm(savedTelegramNotifier);

    fireEvent.click(screen.getByRole('switch', { name: 'Use proxy' }));

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for the bot token when the Telegram proxy is turned off', () => {
    renderSavedNotifierForm(
      buildSavedNotifier({ type: 'TELEGRAM', targetChatId: '-100123', isProxyEnabled: true }),
    );

    fireEvent.click(screen.getByRole('switch', { name: 'Use proxy' }));

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for the bot token and not the proxy URL when a Telegram proxy URL is typed', () => {
    renderSavedNotifierForm(
      buildSavedNotifier({ type: 'TELEGRAM', targetChatId: '-100123', isProxyEnabled: true }),
    );

    changeField('Proxy URL', 'socks5://proxy.example.com:1080');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(getTestButton().disabled).toBe(true);

    changeField('Bot token', 'typed-token');

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });

  it('keeps the placeholder and the test when only the Telegram chat changes', () => {
    renderSavedNotifierForm(savedTelegramNotifier);

    changeField('Target chat ID', '-100456');

    expect(screen.getByPlaceholderText(keptSecretPlaceholder)).toBeTruthy();
    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });

  it('asks for the bot token when the Mattermost server URL changes in the bot mode', () => {
    renderSavedNotifierForm(savedMattermostBot);

    changeField('Server URL', 'https://moved.example.com');

    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);
    expect(screen.queryByPlaceholderText(keptSecretPlaceholder)).toBeNull();
    expect(getTestButton().disabled).toBe(true);
  });

  it('asks for no bot token when the Mattermost server URL differs in the webhook mode', async () => {
    const { user } = renderSavedNotifierForm(savedMattermostWebhook);
    const selectDeliveryMode = async (optionTitle: string) => {
      await user.click(screen.getByRole('combobox', { name: 'Connect via' }));
      fireEvent.click((await screen.findAllByTitle(optionTitle)).pop()!);
    };

    await selectDeliveryMode('Bot account');
    changeField('Server URL', 'https://moved.example.com');
    expect(screen.getAllByText(addressChangedMessage)).toHaveLength(1);

    await selectDeliveryMode('Incoming webhook');

    expect(screen.queryByText(addressChangedMessage)).toBeNull();
    expect(getTestButton().disabled).toBe(false);
  });
});
