import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { AuthPage } from '@/pages/auth';
import { getSessionSnapshot } from '@/shared/auth';
// @vitest-environment happy-dom
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { ThemeProvider } from '@/shared/theme';

vi.hoisted(() => {
  vi.stubEnv('VITE_IS_EMAIL_CONFIGURED', 'true');
  vi.stubEnv('VITE_CLOUDFLARE_TURNSTILE_SITE_KEY', 'test-site-key');
});
afterAll(() => vi.unstubAllEnvs());

const mockServer = setUpMockApi();
let completeChallenge: ((token: string) => void) | undefined;

beforeEach(() => {
  const script = document.createElement('script');
  script.type = 'text/plain';
  script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js';
  document.head.appendChild(script);
  window.turnstile = {
    render: (_container, options) => {
      completeChallenge = options.callback;
      return 'widget-id';
    },
    reset: vi.fn(),
    remove: vi.fn(),
    getResponse: () => undefined,
  };
});

afterEach(() => {
  document.querySelector('script[src*="turnstile"]')?.remove();
  delete window.turnstile;
  completeChallenge = undefined;
});

const answerChallenge = async () => {
  await waitFor(() => expect(completeChallenge).toBeTypeOf('function'));
  act(() => completeChallenge?.('challenge-answer'));
};

describe('AuthPage', () => {
  beforeEach(() => {
    mockServer.use(
      http.get('https://api.github.com/repos/databasus/databasus', () =>
        HttpResponse.json({ stargazers_count: 1 }),
      ),
    );
  });
  it('shows sign-in after options load when the instance has users', async () => {
    mockServer.use(mockApi.get('/sign-in-options', 200, { hasAnyUser: true }));
    renderScreen(
      <ThemeProvider>
        <AuthPage />
      </ThemeProvider>,
    );
    expect(await screen.findByRole('button', { name: 'Sign in' })).toBeTruthy();
    expect(screen.queryByPlaceholderText('John Doe')).toBeNull();
    expect(screen.getAllByPlaceholderText('********')).toHaveLength(1);
  });

  it('shows first registration when the instance is unclaimed', async () => {
    mockServer.use(mockApi.get('/sign-in-options', 200, { hasAnyUser: false }));
    renderScreen(
      <ThemeProvider>
        <AuthPage />
      </ThemeProvider>,
    );
    expect(await screen.findByRole('button', { name: 'Sign up' })).toBeTruthy();
    expect(screen.getAllByPlaceholderText('********')).toHaveLength(2);
  });

  it('keeps explicit registration navigation after the initial sign-in selection', async () => {
    mockServer.use(mockApi.get('/sign-in-options', 200, { hasAnyUser: true }));
    renderScreen(
      <ThemeProvider>
        <AuthPage />
      </ThemeProvider>,
    );
    await screen.findByRole('button', { name: 'Sign in' });
    fireEvent.click(screen.getByRole('button', { name: 'Sign up' }));
    expect(screen.getAllByPlaceholderText('********')).toHaveLength(2);
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
    expect(screen.getAllByPlaceholderText('********')).toHaveLength(1);
  });
  it('requires challenges and the emailed code before storing an ordinary sign-in session', async () => {
    mockServer.use(
      mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
      mockApi.post('/sessions', 200, {
        outcome: 'CODE_REQUIRED',
        pendingSignInId: 'pending-id',
        email: 'owner@example.com',
      }),
      mockApi.post('/pending-sign-ins/{pendingSignInId}/verify', 200, {
        token: 'completed-session',
        userId: 'owner-id',
        email: 'owner@example.com',
      }),
    );

    const requests: Request[] = [];
    const recordRequest = ({ request }: { request: Request }) => {
      if (request.method === 'POST') requests.push(request.clone());
    };
    mockServer.events.on('request:start', recordRequest);

    try {
      renderScreen(
        <ThemeProvider>
          <AuthPage />
        </ThemeProvider>,
      );

      const signInButton = await screen.findByRole('button', { name: 'Sign in' });
      fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
        target: { value: 'owner@example.com' },
      });
      fireEvent.change(screen.getByPlaceholderText('********'), {
        target: { value: 'replacement-password' },
      });

      expect(signInButton.hasAttribute('disabled')).toBe(true);
      await answerChallenge();
      fireEvent.click(signInButton);
      await screen.findByPlaceholderText('123456');

      expect(getSessionSnapshot().token).toBeUndefined();
      expect(getSessionSnapshot().userId).toBeUndefined();
      expect(screen.getByRole('button', { name: 'Sign in' }).hasAttribute('disabled')).toBe(true);

      await answerChallenge();
      fireEvent.change(screen.getByPlaceholderText('123456'), { target: { value: '123456' } });
      fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
      await waitFor(() => expect(getSessionSnapshot().token).toBe('completed-session'));

      expect(getSessionSnapshot().userId).toBe('owner-id');
      expect(requests).toHaveLength(2);
      expect(await requests[0].json()).toMatchObject({
        password: 'replacement-password',
        cloudflareTurnstileToken: 'challenge-answer',
      });
      expect(await requests[1].json()).toMatchObject({
        code: '123456',
        cloudflareTurnstileToken: 'challenge-answer',
      });
    } finally {
      mockServer.events.removeListener('request:start', recordRequest);
    }
  });

  it('preserves reset and code-step navigation after options load', async () => {
    mockServer.use(
      mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
      mockApi.post('/password-resets', 204, undefined),
      mockApi.post('/sessions', 200, {
        outcome: 'CODE_REQUIRED',
        pendingSignInId: 'pending-id',
        email: 'owner@example.com',
      }),
    );

    renderScreen(
      <ThemeProvider>
        <AuthPage />
      </ThemeProvider>,
    );

    await screen.findByRole('button', { name: 'Sign in' });
    fireEvent.click(screen.getByRole('button', { name: 'Forgot password?' }));
    fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
      target: { value: 'owner@example.com' },
    });
    await answerChallenge();
    fireEvent.click(screen.getByRole('button', { name: 'Send reset code' }));
    await screen.findByPlaceholderText('123456');

    fireEvent.click(screen.getByRole('button', { name: 'Back to sign in' }));
    fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
      target: { value: 'owner@example.com' },
    });
    fireEvent.change(screen.getByPlaceholderText('********'), {
      target: { value: 'replacement-password' },
    });

    await answerChallenge();
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
    await screen.findByPlaceholderText('123456');

    fireEvent.click(screen.getByRole('button', { name: 'Back to sign in' }));
    expect(screen.getByPlaceholderText('********')).toBeTruthy();
  });
});

it('recovers the current authentication page when loading sign-in options fails', async () => {
  mockServer.use(
    mockApi.get('/sign-in-options', 500, buildProblem('internal', 500)),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
  );

  const view = renderScreen(<AuthPage />);
  const retry = await screen.findByRole('button', { name: 'Retry' });
  mockServer.use(mockApi.get('/sign-in-options', 200, { hasAnyUser: true }));
  await view.user.click(retry);
  expect(await screen.findByRole('button', { name: 'Sign in' })).toBeTruthy();
});

it('verifies the replacement attempt without activating a session before successful code resend and verification', async () => {
  mockServer.use(
    mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    mockApi.post('/sessions', 200, {
      outcome: 'CODE_REQUIRED',
      pendingSignInId: 'original-pending',
      email: 'owner@example.com',
    }),
    mockApi.post('/pending-sign-ins/{pendingSignInId}/resend', 200, {
      pendingSignInId: 'replacement-pending',
      email: 'owner@example.com',
    }),
  );

  const verification = mockApi.delayed({
    method: 'post',
    path: '/pending-sign-ins/{pendingSignInId}/verify',
    status: 200,
    body: { userId: 'owner-id', token: 'verified-replacement', email: 'owner@example.com' },
  });
  mockServer.use(verification.handler);

  const { user } = renderScreen(<AuthPage />, { isStrictMode: true });
  await screen.findByRole('button', { name: 'Sign in' });
  fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
    target: { value: 'owner@example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText('********'), {
    target: { value: 'correct-password' },
  });

  await answerChallenge();
  await user.click(screen.getByRole('button', { name: 'Sign in' }));
  await screen.findByPlaceholderText('123456');
  await answerChallenge();
  await user.click(screen.getByRole('button', { name: 'Send another code' }));
  await screen.findByText(DICTIONARIES.en.users.signInCode.codeResent);
  expect(getSessionSnapshot().token).toBeUndefined();

  await answerChallenge();
  fireEvent.change(screen.getByPlaceholderText('123456'), { target: { value: '123456' } });
  await user.click(screen.getByRole('button', { name: 'Sign in' }));

  await waitFor(() =>
    expect(
      mockServer
        .getRequests()
        .some((request) => request.url.endsWith('/replacement-pending/verify')),
    ).toBe(true),
  );

  expect(
    mockServer.getRequests().some((request) => request.url.endsWith('/original-pending/verify')),
  ).toBe(false);
  expect(getSessionSnapshot().token).toBeUndefined();

  verification.release();
  await waitFor(() => expect(getSessionSnapshot().token).toBe('verified-replacement'));
});
