// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Storage } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage/ui/edit/EditStorageComponent';
import { installStorageAuthorizationChannels } from '@/features/edit-storage/ui/edit/test/StorageAuthorizationChannels';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { mockApi, setUpMockApi, buildProblem } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { useTheme } from '@/shared/theme';

const mockServer = setUpMockApi();
const draft = {
  name: 'Drive draft',
  settings: {
    type: 'GOOGLE_DRIVE' as const,
    clientId: 'public-client',
    clientSecret: 'synthetic-secret',
  },
};
const token = {
  access_token: 'synthetic-access',
  refresh_token: 'synthetic-refresh',
  token_type: 'Bearer',
  expires_in: 3600,
};
const savedStorage: Storage = {
  id: 'drive-id',
  workspaceId: 'workspace-id',
  name: 'Stored Drive',
  settings: { type: 'GOOGLE_DRIVE', clientId: 'public-client' },
};
const editor = (workspaceId = 'workspace-id', storage?: Storage) => (
  <EditStorageComponent
    workspaceId={workspaceId}
    isShowClose={false}
    onClose={() => {}}
    isShowName
    editingStorage={storage}
    draft={storage ? undefined : draft}
    onChanged={() => {}}
  />
);
const Presentation = () => {
  const { locale, setLocale } = useLocale();
  const { setTheme } = useTheme();

  return (
    <button
      onClick={() => {
        setLocale(locale === 'en' ? 'ru' : 'en');
        setTheme(locale === 'en' ? 'dark' : 'light');
      }}
    >
      presentation
    </button>
  );
};

beforeEach(() => {
  activateSession({ token: 'application-session', userId: 'owner-id' });
});

const setUpAuthorization = () => {
  const communication = installStorageAuthorizationChannels();
  const popup = vi.spyOn(window, 'open').mockReturnValue({} as Window);
  const start = (label = 'Authorize') => {
    fireEvent.click(screen.getByRole('button', { name: label }));
    return new URL(String(popup.mock.calls[popup.mock.calls.length - 1][0])).searchParams.get(
      'state',
    )!;
  };
  const deliver = (state: string) =>
    act(() => communication.send(state, { type: 'code', state, code: 'synthetic-code' }));

  return { ...communication, popup, start, deliver };
};

describe('GoogleDriveAuthorization', () => {
  it('accepts a code once and preserves newer names before explicit test and save', async () => {
    const authorization = setUpAuthorization();
    let exchanges = 0;
    mockServer.use(
      http.post('https://oauth2.googleapis.com/token', async ({ request }) => {
        exchanges++;

        expect(request.headers.get('Authorization')).toBeNull();
        expect(Object.fromEntries(new URLSearchParams(await request.text()))).toEqual({
          client_id: 'public-client',
          client_secret: 'synthetic-secret',
          code: 'synthetic-code',
          redirect_uri: `${window.location.origin}/storages/google-oauth`,
          grant_type: 'authorization_code',
        });

        return HttpResponse.json(token);
      }),
    );

    renderScreen(editor(), { isStrictMode: true });
    const state = authorization.start();

    fireEvent.change(screen.getByDisplayValue('Drive draft'), { target: { value: 'Newer name' } });
    authorization.deliver(state);
    authorization.deliver(state);

    expect(await screen.findByText('Authorized')).toBeTruthy();
    expect(exchanges).toBe(1);
    expect(screen.getByDisplayValue('Newer name')).toBeTruthy();
    expect(screen.queryByDisplayValue(JSON.stringify(token))).toBeNull();
    expect(
      mockServer.getRequests().filter((request) => new URL(request.url).pathname.includes('/api/')),
    ).toHaveLength(0);
    expect(JSON.stringify(authorization.messages)).not.toContain('synthetic-secret');
    expect(JSON.stringify(authorization.messages)).not.toContain('synthetic-refresh');
    expect(JSON.stringify({ ...localStorage, ...sessionStorage })).not.toContain(
      'synthetic-secret',
    );
  });

  it.each(['refusal', 'missing refresh', 'invalid access', 'network failure', 'provider failure'])(
    'preserves stored credentials after %s and allows a fresh attempt',
    async (failure) => {
      const authorization = setUpAuthorization();
      mockServer.use(
        http.post('https://oauth2.googleapis.com/token', () =>
          failure === 'network failure'
            ? HttpResponse.error()
            : HttpResponse.json(
                failure === 'missing refresh'
                  ? { access_token: 'access' }
                  : failure === 'invalid access'
                    ? { access_token: '', refresh_token: 'refresh' }
                    : { error_description: 'raw-private-provider-error' },
                { status: failure === 'provider failure' ? 400 : 200 },
              ),
        ),
      );

      renderScreen(editor('workspace-id', savedStorage));

      expect(
        (screen.getByRole('button', { name: 'Reconnect' }) as HTMLButtonElement).disabled,
      ).toBe(true);

      fireEvent.change(screen.getByLabelText('Client Secret'), {
        target: { value: 'reentered-secret' },
      });
      const state = authorization.start('Reconnect');

      if (failure === 'refusal') act(() => authorization.send(state, { type: 'refusal', state }));
      else authorization.deliver(state);

      expect(await screen.findByRole('alert')).toBeTruthy();
      expect(screen.getByText('Authorized')).toBeTruthy();
      expect(screen.queryByText('raw-private-provider-error')).toBeNull();
      expect(
        mockServer
          .getRequests()
          .some((request) => new URL(request.url).pathname.startsWith('/api/')),
      ).toBe(false);
      expect(savedStorage.settings).toEqual({ type: 'GOOGLE_DRIVE', clientId: 'public-client' });
      expect(authorization.start('Restart authorization')).not.toBe(state);
    },
  );

  it('retains an earlier draft token when reconnection is refused', async () => {
    const authorization = setUpAuthorization();
    mockServer.use(mockApi.post('/storage-tests', 200, { isConnected: true }));

    renderScreen(
      <EditStorageComponent
        workspaceId="workspace-id"
        isShowClose={false}
        onClose={() => {}}
        isShowName
        onChanged={() => {}}
        draft={{ ...draft, settings: { ...draft.settings, tokenJson: JSON.stringify(token) } }}
      />,
    );

    const state = authorization.start('Reconnect');
    act(() => authorization.send(state, { type: 'refusal', state }));

    expect(await screen.findByRole('alert')).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    await screen.findByRole('button', { name: 'Save' });
    const connectionTest = mockServer.getRequests()[0];

    expect((await connectionTest.clone().json()).settings.tokenJson).toBe(JSON.stringify(token));
    expect(screen.queryByDisplayValue(JSON.stringify(token))).toBeNull();
  });

  it('tests and saves a renamed saved storage without secrets or authorization', async () => {
    const authorization = setUpAuthorization();
    mockServer.use(
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      mockApi.put('/storages/{storageId}', 200, savedStorage),
    );
    renderScreen(editor('workspace-id', savedStorage));

    fireEvent.change(screen.getByDisplayValue('Stored Drive'), {
      target: { value: 'Renamed Drive' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(2));
    expect(await mockServer.getRequests()[1].clone().json()).toEqual({
      name: 'Renamed Drive',
      settings: savedStorage.settings,
    });
    expect(authorization.popup).not.toHaveBeenCalled();
  });

  it('retains authorized draft tokens when explicit save fails', async () => {
    const authorization = setUpAuthorization();
    mockServer.use(
      http.post('https://oauth2.googleapis.com/token', () => HttpResponse.json(token)),
      mockApi.post('/storage-tests', 200, { isConnected: true }),
      mockApi.put('/storages/{storageId}', 403, buildProblem('permission_denied', 403)),
    );
    renderScreen(editor('workspace-id', savedStorage));

    fireEvent.change(screen.getByLabelText('Client Secret'), {
      target: { value: 'reentered-secret' },
    });
    authorization.deliver(authorization.start('Reconnect'));
    await screen.findByText('Authorized');

    await waitFor(() => expect(screen.getByRole('button', { name: 'Reconnect' })).toBeTruthy());

    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText(DICTIONARIES.en.errors.permissionDenied)).toBeTruthy();
    expect(screen.getByText('Authorized')).toBeTruthy();
    expect(savedStorage.settings).not.toHaveProperty('tokenJson');

    const update = mockServer.getRequests().find((request) => request.method === 'PUT')!;

    expect((await update.clone().json()).settings.tokenJson).toBe(JSON.stringify(token));

    mockServer.use(mockApi.put('/storages/{storageId}', 200, savedStorage));

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByRole('button', { name: 'Save' })).toBeNull());

    const updates = mockServer.getRequests().filter((request) => request.method === 'PUT');

    expect(updates).toHaveLength(2);
    expect(await updates[1].clone().json()).toEqual({
      name: savedStorage.name,
      settings: {
        ...savedStorage.settings,
        clientSecret: 'reentered-secret',
        tokenJson: JSON.stringify(token),
      },
    });
    expect(authorization.popup).toHaveBeenCalledTimes(1);
    expect(
      mockServer
        .getRequests()
        .filter((request) => new URL(request.url).hostname === 'oauth2.googleapis.com'),
    ).toHaveLength(1);
    expect(screen.queryByText(DICTIONARIES.en.errors.permissionDenied)).toBeNull();
  });

  it.each(
    ['cancel', 'replace', 'settings', 'workspace', 'session', 'unmount'].flatMap((change) =>
      ['success', 'network rejection'].map((outcome) => ({ change, outcome })),
    ),
  )('ignores a late $outcome after $change', async ({ change, outcome }) => {
    const authorization = setUpAuthorization();
    const pending = createPendingApiResponse();
    const fetchObservation = vi.spyOn(globalThis, 'fetch');
    let exchanges = 0;

    mockServer.use(
      http.post('https://oauth2.googleapis.com/token', () => {
        exchanges++;
        return exchanges === 1 ? pending.response : HttpResponse.json(token);
      }),
    );

    const screenView = renderScreen(editor());
    authorization.deliver(authorization.start());

    await waitFor(() => expect(exchanges).toBe(1));

    let currentState: string | undefined;

    if (change === 'cancel')
      fireEvent.click(screen.getByRole('button', { name: 'Cancel authorization' }));

    if (change === 'replace') currentState = authorization.start('Restart authorization');

    if (change === 'settings')
      fireEvent.change(screen.getByLabelText('Client ID'), {
        target: { value: 'replacement-client' },
      });

    if (change === 'workspace') screenView.rerender(editor('replacement-workspace'));

    if (change === 'session')
      act(() => activateSession({ token: 'replacement-session', userId: 'owner-id' }));

    if (change === 'unmount') screenView.unmount();
    else {
      if (!currentState) currentState = authorization.start();

      fireEvent.change(screen.getByDisplayValue('Drive draft'), {
        target: { value: 'Current Drive draft' },
      });
    }
    await act(async () => {
      pending.respond(outcome === 'success' ? HttpResponse.json(token) : HttpResponse.error());
      await fetchObservation.mock.results[0].value.catch(() => undefined);
    });

    expect(screen.queryByText('Authorized')).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();

    if (currentState) {
      expect(screen.getByText('Waiting for Google authorization...')).toBeTruthy();
      expect(screen.getByDisplayValue('Current Drive draft')).toBeTruthy();

      authorization.deliver(currentState);

      expect(await screen.findByText('Authorized')).toBeTruthy();
      expect(exchanges).toBe(2);
      expect(screen.getByDisplayValue('Current Drive draft')).toBeTruthy();
    }
    screenView.unmount();

    expect(authorization.channels.every((channel) => channel.isClosed)).toBe(true);
  });

  it('requires a fresh connection test when reauthorization replaces credentials during an earlier test', async () => {
    const authorization = setUpAuthorization();
    const firstConnectionTest = createPendingApiResponse();
    const fetchObservation = vi.spyOn(globalThis, 'fetch');
    let connectionTests = 0;
    mockServer.use(
      http.post('https://oauth2.googleapis.com/token', () => HttpResponse.json(token)),
      http.post('*/api/v1/storage-tests', () => {
        connectionTests++;
        return connectionTests === 1
          ? firstConnectionTest.response
          : HttpResponse.json({ isConnected: true });
      }),
    );

    renderScreen(editor('workspace-id', savedStorage));

    fireEvent.change(screen.getByLabelText('Client Secret'), {
      target: { value: 'reentered-secret' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Test connection/ }));

    await waitFor(() => expect(connectionTests).toBe(1));

    authorization.deliver(authorization.start('Reconnect'));

    await waitFor(() => expect(screen.getByRole('button', { name: 'Reconnect' })).toBeTruthy());

    await act(async () => {
      firstConnectionTest.respond(HttpResponse.json({ isConnected: true }));
      await fetchObservation.mock.results[0].value;
    });

    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();

    const connectionTestControl = await screen.findByRole('button', { name: /Test connection/ });

    expect((connectionTestControl as HTMLButtonElement).disabled).toBe(false);

    fireEvent.click(connectionTestControl);

    expect(await screen.findByRole('button', { name: 'Save' })).toBeTruthy();

    const currentTest = mockServer
      .getRequests()
      .filter((request) => request.url.endsWith('/storage-tests'))[1];

    expect((await currentTest.clone().json()).settings.tokenJson).toBe(JSON.stringify(token));
  });

  it('rejects invalid state, foreign origins and malformed messages before accepting the matching code', async () => {
    const authorization = setUpAuthorization();
    mockServer.use(
      http.post('https://oauth2.googleapis.com/token', () => HttpResponse.json(token)),
    );
    renderScreen(editor());
    const state = authorization.start();

    act(() => {
      authorization.send(state, { type: 'code', state: 'a'.repeat(64), code: 'wrong' });
      authorization.send(state, { type: 'code', state, code: 'wrong' }, 'https://other.example');
      authorization.send(state, { type: 'code', state, code: '' });
      authorization.send(state, { type: 'code', state, code: 'wrong', clientSecret: 'unexpected' });
    });

    expect(mockServer.getRequests()).toHaveLength(0);

    authorization.deliver(state);

    expect(await screen.findByText('Authorized')).toBeTruthy();
    expect(mockServer.getRequests()).toHaveLength(1);
  });

  it('expires exactly at ten minutes and rejects a future-dated attempt', () => {
    vi.useFakeTimers({ toFake: ['Date', 'setTimeout', 'clearTimeout'] });
    const authorization = setUpAuthorization();
    renderScreen(editor());
    const state = authorization.start();
    act(() => vi.advanceTimersByTime(599999));

    expect(screen.getByText('Waiting for Google authorization...')).toBeTruthy();

    act(() => vi.advanceTimersByTime(1));

    expect(screen.getByRole('alert').textContent).toContain('expired');

    authorization.deliver(state);
    const newState = authorization.start('Restart authorization');
    vi.setSystemTime(Date.now() - 1);
    authorization.deliver(newState);

    expect(mockServer.getRequests()).toHaveLength(0);
    expect(screen.getByRole('alert').textContent).toContain('expired');
  });

  it('isolates two editors and releases their channels on unmount', async () => {
    const authorization = setUpAuthorization();
    mockServer.use(
      http.post('https://oauth2.googleapis.com/token', () => HttpResponse.json(token)),
    );
    const screenView = renderScreen(
      <>
        <section aria-label="First">{editor()}</section>
        <section aria-label="Second">{editor()}</section>
      </>,
    );

    fireEvent.click(
      within(screen.getByRole('region', { name: 'First' })).getByRole('button', {
        name: 'Authorize',
      }),
    );
    const firstState = new URL(String(authorization.popup.mock.calls[0][0])).searchParams.get(
      'state',
    )!;

    fireEvent.click(
      within(screen.getByRole('region', { name: 'Second' })).getByRole('button', {
        name: 'Authorize',
      }),
    );
    const secondState = new URL(String(authorization.popup.mock.calls[1][0])).searchParams.get(
      'state',
    )!;

    expect(secondState).not.toBe(firstState);

    authorization.deliver(firstState);

    expect(
      await within(screen.getByRole('region', { name: 'First' })).findByText('Authorized'),
    ).toBeTruthy();
    expect(
      within(screen.getByRole('region', { name: 'Second' })).queryByText('Authorized'),
    ).toBeNull();

    screenView.unmount();

    expect(authorization.channels.every((channel) => channel.isClosed)).toBe(true);
  });

  it('preserves the attempt, input and translated error across language and theme changes', async () => {
    const authorization = setUpAuthorization();
    const screenView = renderScreen(
      <>
        {editor()}
        <Presentation />
      </>,
    );
    const state = authorization.start();

    fireEvent.click(screen.getByRole('button', { name: 'presentation' }));

    expect(screen.getByDisplayValue('synthetic-secret')).toBeTruthy();

    act(() => authorization.send(state, { type: 'refusal', state }));

    expect(await screen.findByText(DICTIONARIES.ru.storages.edit.googleDrive.refused)).toBeTruthy();

    screenView.rerender(
      <>
        {editor()}
        <Presentation />
      </>,
    );

    expect(screen.getByRole('alert').textContent).toContain(
      DICTIONARIES.ru.storages.edit.googleDrive.refused,
    );

    fireEvent.click(screen.getByRole('button', { name: 'presentation' }));

    expect(screen.getByRole('alert').textContent).toContain(
      DICTIONARIES.en.storages.edit.googleDrive.refused,
    );
    expect(screen.getByDisplayValue('synthetic-secret')).toBeTruthy();
  });

  it.each(['popup', 'channel'])('releases pending state when the %s is unavailable', (failure) => {
    const authorization = setUpAuthorization();

    if (failure === 'popup') authorization.popup.mockReturnValue(null);
    else
      vi.stubGlobal(
        'BroadcastChannel',
        class {
          constructor() {
            throw new Error('unavailable');
          }
        },
      );
    renderScreen(editor());

    fireEvent.click(screen.getByRole('button', { name: 'Authorize' }));

    expect(screen.getByRole('alert')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Cancel authorization' })).toBeNull();
    expect(screen.getByDisplayValue('Drive draft')).toBeTruthy();
    expect(authorization.channels.every((channel) => channel.isClosed)).toBe(true);
  });
});
