// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { Notifier, NotifierSettings } from '@/entities/notifiers';
import { EditNotifierComponent } from '@/features/edit-notifier/ui/edit/EditNotifierComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { ThemeToggleComponent } from '@/shared/ui';

const server = setUpMockApi();
const emailSettings: NotifierSettings = {
  type: 'EMAIL',
  targetEmail: 'ops@example.com',
  smtpHost: 'smtp.example.com',
  smtpPort: 587,
  security: 'STARTTLS',
  isInsecureSkipVerify: false,
};
const emailNotifier: Notifier = {
  id: 'email-id',
  workspaceId: 'workspace-id',
  name: 'Operations',
  lastSendError: null,
  settings: emailSettings,
};
const props = {
  workspaceId: 'workspace-id',
  isShowClose: true,
  isShowName: true,
  onClose: () => {},
  onPersisted: () => {},
  onChanged: () => {},
};
const LocaleControl = () => {
  const { setLocale } = useLocale();
  return <button onClick={() => setLocale('ru')}>Russian</button>;
};
const rename = (value: string) =>
  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), { target: { value } });
const testAndSave = async () => {
  fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
  fireEvent.click(await screen.findByRole('button', { name: /Save/ }));
};
const beginEmailCreation = async (callbacks: Partial<typeof props> = {}) => {
  const view = renderScreen(<EditNotifierComponent {...props} {...callbacks} />, {
    locale: 'en',
    isStrictMode: true,
  });

  await view.user.click(screen.getByRole('combobox', { name: 'Type' }));
  await view.user.click(screen.getByRole('option', { name: 'Email' }));
  rename('Operations');

  fireEvent.change(screen.getByRole('textbox', { name: 'Target email' }), {
    target: { value: 'ops@example.com' },
  });
  fireEvent.change(screen.getByRole('textbox', { name: 'SMTP host' }), {
    target: { value: 'smtp.example.com' },
  });
  fireEvent.change(screen.getByRole('spinbutton', { name: 'SMTP port' }), {
    target: { value: '587' },
  });
  fireEvent.change(screen.getByLabelText('SMTP password'), {
    target: { value: 'submitted-secret' },
  });

  return view;
};

describe('EditNotifierComponent', () => {
  describe('Notifier editing sessions', () => {
    it('retains newer edits and the saved notifier ID when creation completes', async () => {
      const creation = mockApi.delayed({
        method: 'post',
        path: '/notifiers',
        status: 201,
        body: emailNotifier,
      });
      server.use(
        creation.handler,
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        mockApi.put('/notifiers/{notifierId}', 200, { ...emailNotifier, name: 'Newer name' }),
      );
      await beginEmailCreation();

      await testAndSave();

      await waitFor(() =>
        expect(
          server
            .getRequests()
            .filter((request) => request.method === 'POST' && request.url.endsWith('/notifiers')),
        ).toHaveLength(1),
      );
      expect((screen.getByRole('combobox', { name: 'Type' }) as HTMLInputElement).disabled).toBe(
        true,
      );

      fireEvent.keyDown(screen.getByRole('combobox', { name: 'Type' }), { key: 't' });
      rename('Newer name');

      fireEvent.change(screen.getByRole('textbox', { name: 'SMTP host' }), {
        target: { value: 'newer.example.com' },
      });
      creation.release();

      await waitFor(() => expect(screen.queryByRole('combobox', { name: 'Type' })).toBeNull());
      expect(screen.getByDisplayValue('Newer name')).toBeTruthy();
      expect(screen.getByDisplayValue('newer.example.com')).toBeTruthy();
      expect((screen.getByLabelText('SMTP password') as HTMLInputElement).value).toBe('');

      await testAndSave();

      await waitFor(() =>
        expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(true),
      );

      const requests = server.getRequests();

      expect(
        requests.filter(
          (request) => request.method === 'POST' && request.url.endsWith('/notifiers'),
        ),
      ).toHaveLength(1);

      const tests = await Promise.all(
        requests
          .filter((request) => request.url.endsWith('/notifier-tests'))
          .map((request) => request.json()),
      );

      expect(tests).toHaveLength(2);
      expect(tests[1]).toMatchObject({
        notifierId: 'email-id',
        settings: { type: 'EMAIL', smtpHost: 'newer.example.com' },
      });
      expect(tests[1].settings).not.toHaveProperty('smtpPassword');

      const update = requests.find((request) => request.method === 'PUT')!;

      expect(new URL(update.url).pathname).toBe(`${API_BASE_PATH}/notifiers/email-id`);
      expect(await update.json()).toMatchObject({
        name: 'Newer name',
        settings: { type: 'EMAIL', smtpHost: 'newer.example.com' },
      });
    });

    it('unlocks provider selection after a failed creation without an ID', async () => {
      server.use(
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        mockApi.post('/notifiers', 500, buildProblem('internal', 500)),
      );
      const view = await beginEmailCreation();

      await testAndSave();
      await screen.findByRole('alert');

      await waitFor(() =>
        expect((screen.getByRole('combobox', { name: 'Type' }) as HTMLInputElement).disabled).toBe(
          false,
        ),
      );
      expect(screen.getByDisplayValue('Operations')).toBeTruthy();

      await view.user.click(screen.getByRole('combobox', { name: 'Type' }));
      await view.user.click(screen.getByRole('option', { name: 'Telegram' }));

      expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
      expect(screen.getByDisplayValue('Operations')).toBeTruthy();
    });

    it.each([
      {
        failure: '403',
        answerTestRequest: () =>
          mockApi.post('/notifier-tests', 403, buildProblem('permission_denied', 403)),
        reason: DICTIONARIES.en.errors.permissionDenied,
      },
      {
        failure: '500',
        answerTestRequest: () =>
          mockApi.post('/notifier-tests', 500, buildProblem('internal', 500)),
        reason: DICTIONARIES.en.errors.internal,
      },
      {
        failure: 'a network loss',
        answerTestRequest: () => mockApi.network('post', '/notifier-tests'),
        reason: DICTIONARIES.en.errors.networkUnreachable,
      },
    ])(
      'shows the reason, keeps the draft and allows a retry when the test notification answers $failure',
      async ({ answerTestRequest, reason }) => {
        server.use(answerTestRequest());
        renderScreen(<EditNotifierComponent {...props} editingNotifier={emailNotifier} />, {
          locale: 'en',
        });
        rename('Renamed draft');
        fireEvent.change(screen.getByRole('textbox', { name: 'SMTP host' }), {
          target: { value: 'draft.example.com' },
        });

        fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

        expect((await screen.findByRole('alert')).textContent).toContain(reason);
        expect(screen.getByDisplayValue('Renamed draft')).toBeTruthy();
        expect(screen.getByDisplayValue('draft.example.com')).toBeTruthy();
        expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();

        const testAction = screen.getByRole('button', { name: /Send test notification/ });

        await waitFor(() => expect(testAction).toHaveProperty('disabled', false));

        server.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));

        fireEvent.click(testAction);

        expect(await screen.findByRole('button', { name: /Save/ })).toBeTruthy();
        expect(screen.queryByText(reason)).toBeNull();
        expect(screen.getByDisplayValue('Renamed draft')).toBeTruthy();
        expect(screen.getByDisplayValue('draft.example.com')).toBeTruthy();
      },
    );

    it('retains test validity and sends one request per activation when name, locale or theme changes', async () => {
      server.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));
      const view = renderScreen(
        <>
          <LocaleControl />
          <ThemeToggleComponent />
          <EditNotifierComponent {...props} editingNotifier={emailNotifier} />
        </>,
        { locale: 'en' },
      );
      rename('Renamed');

      fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));
      await screen.findByRole('button', { name: /Save/ });
      rename('Another name');

      await view.user.click(screen.getByRole('button', { name: /Theme/ }));
      await view.user.click(screen.getByRole('menuitemradio', { name: /Dark/ }));

      expect(document.documentElement.classList.contains('dark')).toBe(true);

      await view.user.click(screen.getByRole('button', { name: 'Russian' }));

      expect(
        screen.getByRole('button', { name: DICTIONARIES.ru.common.actions.save }),
      ).toBeTruthy();
      expect(server.getRequests()).toHaveLength(1);
    });

    it('rejects an old test after settings change from A to B and back to A', async () => {
      const response = mockApi.delayed({
        method: 'post',
        path: '/notifier-tests',
        status: 200,
        body: { isDelivered: true },
      });
      server.use(response.handler);
      renderScreen(<EditNotifierComponent {...props} editingNotifier={emailNotifier} />, {
        locale: 'en',
      });
      rename('Renamed');

      fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

      await waitFor(() => expect(server.getRequests()).toHaveLength(1));

      const host = screen.getByRole('textbox', { name: 'SMTP host' });

      fireEvent.change(host, { target: { value: 'b.example.com' } });
      fireEvent.change(host, { target: { value: 'smtp.example.com' } });
      const responded = new Promise<void>((resolve) =>
        server.events.on('response:mocked', () => resolve()),
      );
      await act(async () => {
        response.release();
        await responded;
      });

      expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
      expect(
        screen.queryByText(DICTIONARIES.en.notifiers.testNotification.sent.title, { exact: true }),
      ).toBeNull();
    });

    it('retains the draft and literal diagnostics when a delayed failure arrives after a language change', async () => {
      const diagnostic = '<img src=x onerror=alert(1)> {{name}}';
      const response = mockApi.delayed({
        method: 'post',
        path: '/notifier-tests',
        status: 200,
        body: { isDelivered: false, error: diagnostic },
      });
      server.use(response.handler);
      renderScreen(
        <>
          <LocaleControl />
          <EditNotifierComponent {...props} editingNotifier={emailNotifier} />
        </>,
        { locale: 'en' },
      );
      rename('Renamed');

      fireEvent.click(screen.getByRole('button', { name: /Advanced settings/ }));
      fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

      await waitFor(() => expect(server.getRequests()).toHaveLength(1));

      fireEvent.click(screen.getByRole('button', { name: 'Russian' }));
      response.release();

      expect((await screen.findByRole('alert')).textContent).toContain(
        DICTIONARIES.ru.notifiers.testNotification.notDelivered.replace('{{error}}', diagnostic),
      );
      expect(screen.getByRole('alert').querySelector('img')).toBeNull();
      expect(screen.getByDisplayValue('Renamed')).toBeTruthy();
      expect(
        screen
          .getByRole('button', { name: DICTIONARIES.ru.notifiers.advancedSettings })
          .getAttribute('aria-expanded'),
      ).toBe('true');

      const host = screen.getByRole('textbox', { name: DICTIONARIES.ru.notifiers.email.smtpHost });

      fireEvent.change(host, { target: { value: '' } });
      fireEvent.blur(host);

      expect(host.getAttribute('aria-invalid')).toBe('true');
      expect(document.getElementById(host.getAttribute('aria-describedby')!)?.textContent).toBe(
        DICTIONARIES.ru.notifiers.email.validation.smtpHost,
      );
    });

    it('translates a visible field error and retains the input when the language changes', () => {
      renderScreen(
        <>
          <LocaleControl />
          <EditNotifierComponent {...props} editingNotifier={emailNotifier} />
        </>,
        { locale: 'en' },
      );
      rename('Local draft');
      const host = screen.getByRole('textbox', { name: 'SMTP host' });

      fireEvent.change(host, { target: { value: '' } });
      fireEvent.blur(host);

      expect(document.getElementById(host.getAttribute('aria-describedby')!)?.textContent).toBe(
        DICTIONARIES.en.notifiers.email.validation.smtpHost,
      );

      fireEvent.click(screen.getByRole('button', { name: 'Russian' }));

      expect(screen.getByRole('textbox', { name: DICTIONARIES.ru.notifiers.email.smtpHost })).toBe(
        host,
      );
      expect(document.getElementById(host.getAttribute('aria-describedby')!)?.textContent).toBe(
        DICTIONARIES.ru.notifiers.email.validation.smtpHost,
      );
      expect(screen.getByDisplayValue('Local draft')).toBeTruthy();
      expect((host as HTMLInputElement).value).toBe('');
    });

    it.each(['settings', 'entity', 'workspace'] as const)(
      'suppresses an obsolete HTTP failure after the %s context changes',
      async (changedContext) => {
        const response = mockApi.delayed({
          method: 'post',
          path: '/notifier-tests',
          status: 500,
          body: buildProblem('internal', 500),
        });
        server.use(response.handler);
        const view = renderScreen(
          <EditNotifierComponent {...props} editingNotifier={emailNotifier} />,
          { locale: 'en', isStrictMode: true },
        );
        rename('Pending draft');
        const testControl = screen.getByRole('button', { name: /Send test notification/ });

        fireEvent.click(testControl);
        fireEvent.click(testControl);

        await waitFor(() => expect(server.getRequests()).toHaveLength(1));

        if (changedContext === 'settings')
          fireEvent.change(screen.getByRole('textbox', { name: 'SMTP host' }), {
            target: { value: 'new.example.com' },
          });
        else
          view.rerender(
            <EditNotifierComponent
              {...props}
              workspaceId={changedContext === 'workspace' ? 'other-workspace' : props.workspaceId}
              editingNotifier={{
                ...emailNotifier,
                id: changedContext === 'entity' ? 'other-id' : emailNotifier.id,
              }}
            />,
          );

        const responded = new Promise<void>((resolve) =>
          server.events.on('response:mocked', () => resolve()),
        );
        await act(async () => {
          response.release();
          await responded;
        });

        expect(screen.queryByRole('alert')).toBeNull();
        expect(
          screen.queryByText(DICTIONARIES.en.notifiers.testNotification.sent.title, {
            exact: true,
          }),
        ).toBeNull();
        expect(server.getRequests()).toHaveLength(1);
      },
    );

    it('accepts asynchronous values when the same notifier draft is clean', () => {
      const view = renderScreen(
        <EditNotifierComponent {...props} editingNotifier={emailNotifier} />,
        { locale: 'en' },
      );

      view.rerender(
        <EditNotifierComponent
          {...props}
          editingNotifier={{ ...emailNotifier, name: 'Refreshed' }}
        />,
      );

      expect(screen.getByDisplayValue('Refreshed')).toBeTruthy();

      rename('Local draft');

      view.rerender(
        <EditNotifierComponent
          {...props}
          editingNotifier={{ ...emailNotifier, name: 'Remote refresh' }}
        />,
      );

      expect(screen.getByDisplayValue('Local draft')).toBeTruthy();

      view.rerender(
        <EditNotifierComponent
          {...props}
          editingNotifier={{ ...emailNotifier, id: 'another-id', name: 'Another notifier' }}
        />,
      );

      expect(screen.getByDisplayValue('Another notifier')).toBeTruthy();
    });

    it.each([undefined, 'replacement-secret'])(
      'preserves stored SMTP secrets when the replacement is %s',
      async (replacement) => {
        const updates: ApiComponents['schemas']['NotifierUpdate'][] = [];
        server.use(
          mockApi.post('/notifier-tests', 200, { isDelivered: true }),
          http.put(`*${API_BASE_PATH}/notifiers/:notifierId`, async ({ request }) => {
            const update = (await request.json()) as ApiComponents['schemas']['NotifierUpdate'];
            updates.push(update);
            return HttpResponse.json({ ...emailNotifier, name: update.name });
          }),
        );

        renderScreen(<EditNotifierComponent {...props} editingNotifier={emailNotifier} />, {
          locale: 'en',
        });
        rename('Renamed');

        if (replacement)
          fireEvent.change(screen.getByLabelText('SMTP password'), {
            target: { value: replacement },
          });

        await testAndSave();

        await waitFor(() => expect(updates).toHaveLength(1));

        if (replacement) expect(updates[0].settings).toHaveProperty('smtpPassword', replacement);
        else expect(updates[0].settings).not.toHaveProperty('smtpPassword');
        const testRequest = server
          .getRequests()
          .find((request) => request.url.endsWith('/notifier-tests'))!;
        const testBody = await testRequest.json();
        if (replacement) expect(testBody.settings).toHaveProperty('smtpPassword', replacement);
        else expect(testBody.settings).not.toHaveProperty('smtpPassword');
      },
    );

    it('ignores save completion after the editing context changes', async () => {
      const persisted = vi.fn();
      const response = mockApi.delayed({
        method: 'put',
        path: '/notifiers/{notifierId}',
        status: 200,
        body: emailNotifier,
      });

      server.use(response.handler, mockApi.post('/notifier-tests', 200, { isDelivered: true }));
      const view = renderScreen(
        <EditNotifierComponent
          {...props}
          onPersisted={persisted}
          editingNotifier={emailNotifier}
        />,
        { locale: 'en' },
      );
      rename('Submitted');

      await testAndSave();

      await waitFor(() => expect(server.getRequests()).toHaveLength(2));

      view.rerender(
        <EditNotifierComponent
          {...props}
          onPersisted={persisted}
          editingNotifier={{ ...emailNotifier, id: 'new-id', name: 'New session' }}
        />,
      );

      const responded = new Promise<void>((resolve) =>
        server.events.on('response:mocked', () => resolve()),
      );
      await act(async () => {
        response.release();
        await responded;
      });

      expect(screen.getByDisplayValue('New session')).toBeTruthy();
      expect(persisted).not.toHaveBeenCalled();
    });
  });

  describe('Notifier session cleanup', () => {
    it('assigns separate label and help IDs when SMTP editors coexist', () => {
      renderScreen(
        <>
          <EditNotifierComponent {...props} editingNotifier={emailNotifier} />
          <EditNotifierComponent {...props} editingNotifier={emailNotifier} />
        </>,
        { locale: 'en' },
      );
      const targets = screen.getAllByRole('textbox', { name: 'Target email' });

      expect(new Set(targets.map((target) => target.id)).size).toBe(2);

      for (const target of targets)
        expect(
          document.getElementById(target.getAttribute('aria-describedby')!)?.textContent,
        ).toContain('email');
    });

    it('starts a clean reopened session after unmount suppresses a delayed test failure', async () => {
      const response = mockApi.delayed({
        method: 'post',
        path: '/notifier-tests',
        status: 200,
        body: { isDelivered: false, error: 'obsolete failure' },
      });
      server.use(response.handler);
      const view = renderScreen(
        <EditNotifierComponent {...props} editingNotifier={emailNotifier} />,
        { locale: 'en', isStrictMode: true },
      );
      rename('Old draft');

      fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

      await waitFor(() => expect(server.getRequests()).toHaveLength(1));

      view.unmount();
      const responded = new Promise<void>((resolve) =>
        server.events.on('response:mocked', () => resolve()),
      );
      await act(async () => {
        response.release();
        await responded;
      });

      renderScreen(<EditNotifierComponent {...props} editingNotifier={emailNotifier} />, {
        locale: 'en',
        isStrictMode: true,
      });

      expect(screen.queryByRole('alert')).toBeNull();
      expect(screen.getByDisplayValue('Operations')).toBeTruthy();
      expect(screen.queryByDisplayValue('Old draft')).toBeNull();
    });

    it.each(['test', 'save'] as const)(
      'suppresses a delayed successful %s after editor unmount while providers remain',
      async (operation) => {
        const persisted = vi.fn();
        const completed = vi.fn();
        const response =
          operation === 'test'
            ? mockApi.delayed({
                method: 'post',
                path: '/notifier-tests',
                status: 200,
                body: { isDelivered: true },
              })
            : mockApi.delayed({
                method: 'put',
                path: '/notifiers/{notifierId}',
                status: 200,
                body: { ...emailNotifier, name: 'Submitted' },
              });
        server.use(response.handler);

        if (operation === 'save')
          server.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }));

        const view = renderScreen(
          <EditNotifierComponent
            {...props}
            editingNotifier={emailNotifier}
            onPersisted={persisted}
            onChanged={completed}
          />,
          { locale: 'en', isStrictMode: true },
        );
        rename('Submitted');

        if (operation === 'save') await testAndSave();
        else fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

        await waitFor(() =>
          expect(server.getRequests()).toHaveLength(operation === 'save' ? 2 : 1),
        );

        const provider = document.querySelector('style[data-mantine-styles="classes"]');
        expect(provider).not.toBeNull();

        view.rerender(<div>Editor closed</div>);

        expect(document.querySelector('style[data-mantine-styles="classes"]')).toBe(provider);
        await waitFor(() =>
          expect(
            screen.queryByText(DICTIONARIES.en.notifiers.testNotification.sent.title, {
              exact: true,
            }),
          ).toBeNull(),
        );

        const responded = new Promise<void>((resolve) =>
          server.events.on('response:mocked', () => resolve()),
        );
        await act(async () => {
          response.release();
          await responded;
        });

        expect(
          screen.queryByText(DICTIONARIES.en.notifiers.testNotification.sent.title, {
            exact: true,
          }),
        ).toBeNull();
        expect(persisted).not.toHaveBeenCalled();
        expect(completed).not.toHaveBeenCalled();
      },
    );

    it('uses the latest persistence callback when save is activated repeatedly', async () => {
      const oldPersisted = vi.fn();
      const latestPersisted = vi.fn();
      const completed = vi.fn();
      const response = mockApi.delayed({
        method: 'put',
        path: '/notifiers/{notifierId}',
        status: 200,
        body: { ...emailNotifier, name: 'Submitted' },
      });

      server.use(response.handler, mockApi.post('/notifier-tests', 200, { isDelivered: true }));
      const view = renderScreen(
        <EditNotifierComponent
          {...props}
          editingNotifier={emailNotifier}
          onPersisted={oldPersisted}
          onChanged={completed}
        />,
        { locale: 'en' },
      );
      rename('Submitted');

      await testAndSave();
      fireEvent.click(screen.getByRole('button', { name: /Save/ }));

      await waitFor(() => expect(server.getRequests()).toHaveLength(2));

      rename('Newer draft');

      view.rerender(
        <EditNotifierComponent
          {...props}
          editingNotifier={emailNotifier}
          onPersisted={latestPersisted}
          onChanged={completed}
        />,
      );
      response.release();

      await waitFor(() => expect(latestPersisted).toHaveBeenCalledTimes(1));
      expect(oldPersisted).not.toHaveBeenCalled();
      expect(completed).not.toHaveBeenCalled();
      expect(server.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(1);
      expect(screen.getByDisplayValue('Newer draft')).toBeTruthy();
    });
  });

  it('completes a created notifier when newer name input returns to the submitted value', async () => {
    const creation = mockApi.pending('post', '/notifiers');
    server.use(creation.handler, mockApi.post('/notifier-tests', 200, { isDelivered: true }));
    const onChanged = vi.fn();
    await beginEmailCreation({ onChanged });

    await testAndSave();

    await waitFor(() =>
      expect(
        server
          .getRequests()
          .some(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/notifiers'),
          ),
      ).toBe(true),
    );

    rename('Newer name');
    rename(emailNotifier.name);
    await act(async () => creation.respond(HttpResponse.json(emailNotifier, { status: 201 })));

    await waitFor(() => expect(onChanged).toHaveBeenCalledWith(emailNotifier));
    expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
  });

  it.each(['test success', 'test failure', 'save success', 'save failure'] as const)(
    'rejects a pending notifier %s after authentication replacement without unmounting',
    async (operation) => {
      activateSession({ token: 'originating-session', userId: 'original-owner' });
      const fetchObservation = vi.spyOn(globalThis, 'fetch');
      const isTest = operation.startsWith('test');
      const isSuccess = operation.endsWith('success');
      const pending = isTest
        ? mockApi.pending('post', '/notifier-tests')
        : mockApi.pending('put', '/notifiers/{notifierId}');
      server.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }), pending.handler);
      const onPersisted = vi.fn();
      const onChanged = vi.fn();

      renderScreen(
        <EditNotifierComponent
          {...props}
          editingNotifier={emailNotifier}
          onPersisted={onPersisted}
          onChanged={onChanged}
        />,
      );
      rename('Pending name');

      fireEvent.click(screen.getByRole('button', { name: /Send test notification/ }));

      if (!isTest) fireEvent.click(await screen.findByRole('button', { name: /Save/ }));

      await waitFor(() =>
        expect(
          server.getRequests().some((request) => request.method === (isTest ? 'POST' : 'PUT')),
        ).toBe(true),
      );

      act(() => activateSession({ token: 'replacement-session', userId: 'new-owner' }));

      const feedbackBefore = screen.queryAllByRole('alert').map((element) => element.textContent);
      const pendingFetch =
        fetchObservation.mock.results[fetchObservation.mock.results.length - 1].value;

      await act(async () => {
        pending.respond(
          isSuccess
            ? HttpResponse.json(
                isTest ? { isDelivered: true } : { ...emailNotifier, name: 'Pending name' },
              )
            : HttpResponse.json(buildProblem('internal', 500), { status: 500 }),
        );
        await pendingFetch.catch(() => undefined);
      });

      expect(onPersisted).not.toHaveBeenCalled();
      expect(onChanged).not.toHaveBeenCalled();
      expect(screen.queryAllByRole('alert').map((element) => element.textContent)).toEqual(
        feedbackBefore,
      );
      expect(screen.queryByRole('button', { name: /Save/ })).toBeNull();
      expect(screen.getByRole('textbox', { name: 'Name' })).toHaveProperty('value', 'Pending name');
    },
  );

  it('merges normalized settings when the notifier name changes during saving', async () => {
    const submittedNotifier = {
      ...emailNotifier,
      name: 'Submitted',
      settings: { ...emailSettings, smtpHost: 'normalized.example.com' },
    };

    const save = mockApi.pending('put', '/notifiers/{notifierId}');
    server.use(mockApi.post('/notifier-tests', 200, { isDelivered: true }), save.handler);
    const onChanged = vi.fn();
    const onPersisted = vi.fn();
    renderScreen(
      <EditNotifierComponent
        {...props}
        editingNotifier={emailNotifier}
        onChanged={onChanged}
        onPersisted={onPersisted}
      />,
    );
    rename('Submitted');

    await testAndSave();

    await waitFor(() =>
      expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(true),
    );

    rename('Newer name');
    await act(async () => save.respond(HttpResponse.json(submittedNotifier)));

    await waitFor(() => expect(onPersisted).toHaveBeenCalledWith(submittedNotifier));
    expect(screen.getByRole('textbox', { name: 'SMTP host' })).toHaveProperty(
      'value',
      'normalized.example.com',
    );
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveProperty('value', 'Newer name');
    expect(onChanged).not.toHaveBeenCalled();
  });
});
