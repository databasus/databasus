// @vitest-environment happy-dom
import { act, screen, waitFor } from '@testing-library/react';
import { useNavigate } from 'react-router';
import { describe, expect, it } from 'vitest';

import { AuthCallbackPage } from '@/pages/auth-callback/ui/AuthCallbackPage';
import { activateSession, getSessionSnapshot } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const state = 'a'.repeat(64);
const attemptKey = 'databasus_oauth_sign_in_attempt';
const session = { token: 'oauth-session', userId: 'oauth-owner', email: 'oauth@example.com' };
const storeAttempt = (changes: Record<string, unknown> = {}) =>
  sessionStorage.setItem(
    attemptKey,
    JSON.stringify({
      state,
      provider: 'github',
      createdAt: Date.now(),
      redirectUri: `${window.location.origin}/auth/callback`,
      ...changes,
    }),
  );
const expectUnfinishedSignInMessage = async () => {
  await screen.findByRole('button', { name: 'Return to sign in' });

  expect(screen.getByText(DICTIONARIES.en.app.oauthCallback.notCompleted)).toBeTruthy();
  expect(screen.queryByText('Invalid OAuth provider')).toBeNull();
};

describe('AuthCallbackPage', () => {
  it.each(['github', 'google'] as const)(
    'activates one session when a live %s callback is replayed by StrictMode',
    async (provider) => {
      storeAttempt({ provider });
      server.use(
        mockApi.post(provider === 'github' ? '/github-sessions' : '/google-sessions', 200, session),
      );
      window.history.replaceState(
        null,
        '',
        `/auth/callback?code=private-code&state=${state}&error_description=private-description`,
      );

      renderScreen(<AuthCallbackPage />, {
        route: `/auth/callback?code=private-code&state=${state}`,
        isStrictMode: true,
      });

      await waitFor(() => expect(getSessionSnapshot().token).toBe(session.token));
      expect(server.getRequests()).toHaveLength(1);
      expect(await server.getRequests()[0].json()).toEqual({
        code: 'private-code',
        redirectUri: `${window.location.origin}/auth/callback`,
      });
      expect(sessionStorage.getItem(attemptKey)).toBeNull();
      expect(window.location.search).toBe('');
    },
  );

  it('sends no second exchange when the same callback address is opened after a completed one', async () => {
    const callbackRoute = `/auth/callback?code=private-code&state=${state}`;
    storeAttempt();
    server.use(mockApi.post('/github-sessions', 200, session));
    const completedView = renderScreen(<AuthCallbackPage />, { route: callbackRoute });
    await waitFor(() => expect(getSessionSnapshot().token).toBe(session.token));
    completedView.unmount();

    renderScreen(<AuthCallbackPage />, { route: callbackRoute });

    await expectUnfinishedSignInMessage();
    expect(server.getRequests()).toHaveLength(1);
    expect(getSessionSnapshot().token).toBe(session.token);
  });

  it.each([
    { name: 'missing code', query: `state=${state}`, record: {} },
    { name: 'missing state', query: 'code=private-code', record: {} },
    { name: 'wrong state', query: 'code=private-code&state=unrelated', record: {} },
    { name: 'legacy provider state', query: 'code=private-code&state=github', record: {} },
    {
      name: 'expired attempt',
      query: `code=private-code&state=${state}`,
      record: { createdAt: Date.now() - 600_001 },
    },
    {
      name: 'future attempt',
      query: `code=private-code&state=${state}`,
      record: { createdAt: Date.now() + 600_000 },
    },
    {
      name: 'redirect mismatch',
      query: `code=private-code&state=${state}`,
      record: { redirectUri: 'https://unrelated.example/auth/callback' },
    },
    {
      name: 'invalid provider',
      query: `code=private-code&state=${state}`,
      record: { provider: 'other' },
    },
  ])('rejects exchange when the callback has $name', async ({ query, record }) => {
    storeAttempt(record);
    renderScreen(<AuthCallbackPage />, { route: `/auth/callback?${query}`, isStrictMode: true });
    await expectUnfinishedSignInMessage();
    expect(server.getRequests()).toHaveLength(0);
    expect(getSessionSnapshot().token).toBeUndefined();
  });

  it.each(['missing', 'malformed', 'consumed'])(
    'offers recovery when attempt metadata is %s',
    async (mode) => {
      if (mode === 'malformed') sessionStorage.setItem(attemptKey, '{');
      if (mode === 'consumed') {
        storeAttempt();
        sessionStorage.removeItem(attemptKey);
      }
      renderScreen(<AuthCallbackPage />, {
        route: `/auth/callback?code=private-code&state=${state}`,
      });
      await expectUnfinishedSignInMessage();
      expect(server.getRequests()).toHaveLength(0);
    },
  );

  it('explains that sign-in did not finish when the provider refuses consent', async () => {
    storeAttempt();

    renderScreen(<AuthCallbackPage />, {
      route: `/auth/callback?error=access_denied&state=${state}`,
      isStrictMode: true,
    });

    await expectUnfinishedSignInMessage();
    expect(server.getRequests()).toHaveLength(0);
    expect(sessionStorage.getItem(attemptKey)).toBeNull();
    expect(getSessionSnapshot().token).toBeUndefined();
  });

  it('requires a new attempt when the provider exchange is refused', async () => {
    storeAttempt();
    server.use(mockApi.post('/github-sessions', 403, buildProblem('permission_denied', 403)));
    const view = renderScreen(<AuthCallbackPage />, {
      route: `/auth/callback?code=private-code&state=${state}`,
      isStrictMode: true,
    });

    await screen.findByText('You do not have permission to do this.');
    view.rerender(<AuthCallbackPage />);
    expect(server.getRequests()).toHaveLength(1);
    expect(sessionStorage.getItem(attemptKey)).toBeNull();
  });

  it.each([
    { mode: 'unmount', isRefused: false },
    { mode: 'replacement', isRefused: false },
    { mode: 'unmount', isRefused: true },
    { mode: 'replacement', isRefused: true },
  ] as const)(
    'suppresses completion after callback $mode while providers remain with refusal=$isRefused',
    async ({ mode, isRefused }) => {
      storeAttempt();
      const exchange = isRefused
        ? mockApi.delayed({
            method: 'post',
            path: '/github-sessions',
            status: 403,
            body: buildProblem('permission_denied', 403),
          })
        : mockApi.delayed({ method: 'post', path: '/github-sessions', status: 200, body: session });
      server.use(exchange.handler);

      const view = renderScreen(<AuthCallbackPage />, {
        route: `/auth/callback?code=private-code&state=${state}`,
        isStrictMode: true,
      });
      await waitFor(() => expect(server.getRequests()).toHaveLength(1));

      const provider = document.querySelector('style[data-mantine-styles="classes"]');
      expect(provider).not.toBeNull();
      if (mode === 'unmount') view.rerender(<div>Callback closed</div>);
      else activateSession({ token: 'new-session', userId: 'new-owner' });

      const responded = new Promise<void>((resolve) =>
        server.events.on('response:mocked', () => resolve()),
      );
      await act(async () => {
        exchange.release();
        await responded;
      });

      expect(getSessionSnapshot().token).toBe(mode === 'unmount' ? undefined : 'new-session');
      expect(document.querySelector('style[data-mantine-styles="classes"]')).toBe(provider);
      expect(screen.queryByText(DICTIONARIES.en.app.oauthCallback.failedTitle)).toBeNull();
      expect(screen.queryByText(DICTIONARIES.en.errors.permissionDenied)).toBeNull();
    },
  );
});

it('preserves a replacement attempt when an older callback state arrives', async () => {
  storeAttempt({ state: 'b'.repeat(64), provider: 'google' });
  renderScreen(<AuthCallbackPage />, { route: `/auth/callback?code=old-code&state=${state}` });
  await expectUnfinishedSignInMessage();
  expect(JSON.parse(sessionStorage.getItem(attemptKey)!).state).toBe('b'.repeat(64));
  expect(server.getRequests()).toHaveLength(0);
});

function ReplaceableCallback() {
  const navigate = useNavigate();
  return (
    <>
      <button onClick={() => navigate(`/auth/callback?code=new-code&state=${'b'.repeat(64)}`)}>
        Open replacement callback
      </button>
      <AuthCallbackPage />
    </>
  );
}

it.each([false, true])(
  'accepts only the replacement parameters after an earlier exchange settles with refusal=%s',
  async (isRefused) => {
    storeAttempt();
    const oldExchange = isRefused
      ? mockApi.delayed({
          method: 'post',
          path: '/github-sessions',
          status: 403,
          body: buildProblem('permission_denied', 403),
        })
      : mockApi.delayed({ method: 'post', path: '/github-sessions', status: 200, body: session });
    const replacementExchange = mockApi.delayed({
      method: 'post',
      path: '/google-sessions',
      status: 200,
      body: { ...session, token: 'replacement-session' },
    });
    server.use(oldExchange.handler, replacementExchange.handler);

    const view = renderScreen(<ReplaceableCallback />, {
      route: `/auth/callback?code=old-code&state=${state}`,
      isStrictMode: true,
    });
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    storeAttempt({ state: 'b'.repeat(64), provider: 'google' });
    await view.user.click(screen.getByRole('button', { name: 'Open replacement callback' }));
    await waitFor(() => expect(server.getRequests()).toHaveLength(2));

    const processed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );
    await act(async () => {
      oldExchange.release();
      await processed;
    });

    expect(getSessionSnapshot().token).toBeUndefined();
    expect(screen.queryByText(DICTIONARIES.en.app.oauthCallback.failedTitle)).toBeNull();
    expect(screen.queryByText(DICTIONARIES.en.errors.permissionDenied)).toBeNull();
    expect(screen.getByText(DICTIONARIES.en.app.oauthCallback.completing)).toBeTruthy();

    replacementExchange.release();
    await waitFor(() => expect(getSessionSnapshot().token).toBe('replacement-session'));
    expect(server.getRequests()).toHaveLength(2);
  },
);
