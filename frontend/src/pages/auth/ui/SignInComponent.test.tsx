import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterAll, describe, expect, it, vi } from 'vitest';

import { userApi } from '@/entities/users';
import { SignInComponent } from '@/pages/auth/ui/SignInComponent';
// @vitest-environment happy-dom
import { getSessionSnapshot } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

vi.hoisted(() => {
  vi.stubEnv('VITE_IS_EMAIL_CONFIGURED', 'true');
  vi.stubEnv('VITE_GOOGLE_CLIENT_ID', 'google-client');
  vi.stubEnv('VITE_GITHUB_CLIENT_ID', 'github-client');
});
afterAll(() => vi.unstubAllEnvs());

const mockServer = setUpMockApi();

const submitPasswordSignIn = async () => {
  const user = userEvent.setup();
  await user.type(screen.getByRole('textbox', { name: 'Your email' }), 'owner@example.com');
  await user.type(screen.getByLabelText('Password', { selector: 'input' }), 'correct-password');
  await user.click(screen.getByRole('button', { name: 'Sign in' }));
};

function LocaleSwitch() {
  const { setLocale } = useLocale();
  return <button onClick={() => setLocale('ru')}>Switch language</button>;
}

describe('SignInComponent', () => {
  it.each([false, true])(
    'stores the session when password sign-in completes with root replay=%s',
    async (isStrictMode) => {
      mockServer.use(
        mockApi.post('/sessions', 200, {
          outcome: 'COMPLETED',
          userId: 'owner-id',
          email: 'owner@example.com',
          token: 'session-token',
        }),
      );
      const onCodeRequired = vi.fn();
      renderScreen(<SignInComponent onCodeRequired={onCodeRequired} />, { isStrictMode });

      await submitPasswordSignIn();

      await waitFor(() => expect(getSessionSnapshot().token).toBe('session-token'));
      expect(getSessionSnapshot().userId).toBe('owner-id');
      expect(onCodeRequired).not.toHaveBeenCalled();
    },
  );

  it('asks for the emailed code when the instance requires a second factor', async () => {
    mockServer.use(
      mockApi.post('/sessions', 200, {
        outcome: 'CODE_REQUIRED',
        pendingSignInId: 'pending-sign-in-id',
        email: 'owner@example.com',
      }),
    );
    const onCodeRequired = vi.fn();
    renderScreen(<SignInComponent onCodeRequired={onCodeRequired} />);

    await submitPasswordSignIn();

    await waitFor(() =>
      expect(onCodeRequired).toHaveBeenCalledWith({
        pendingSignInId: 'pending-sign-in-id',
        email: 'owner@example.com',
      }),
    );
  });

  it('shows the incorrect password beside the password field when sign-in is refused', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('password_incorrect', 401)));
    renderScreen(<SignInComponent />);

    await submitPasswordSignIn();

    expect(await screen.findByText('The password is incorrect.')).toBeTruthy();
  });

  it('keeps the entered values and sends one request when the password is refused', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('password_incorrect', 401)));
    renderScreen(<SignInComponent />);

    await submitPasswordSignIn();

    await screen.findByText('The password is incorrect.');
    expect(screen.getByPlaceholderText<HTMLInputElement>('your@email.com').value).toBe(
      'owner@example.com',
    );
    expect(screen.getByPlaceholderText<HTMLInputElement>('********').value).toBe(
      'correct-password',
    );
    expect(mockServer.getRequests()).toHaveLength(1);
    expect(getSessionSnapshot().token).toBeUndefined();
  });

  it.each([
    ['email_not_found', 'emailNotFound', 'Your email'],
    ['password_incorrect', 'passwordIncorrect', 'Password'],
  ] as const)('marks the affected field when the refusal is %s', async (code, key, label) => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem(code, 401)));
    renderScreen(<SignInComponent />);

    await submitPasswordSignIn();

    const feedback = await screen.findByText(DICTIONARIES.en.errors[key]);
    const input = screen.getByLabelText(label, { selector: 'input' });
    expect(input.getAttribute('aria-invalid')).toBe('true');
    expect(input.getAttribute('aria-describedby')?.split(' ')).toContain(feedback.id);
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it.each([
    ['registration_incomplete', 'registrationIncomplete', 401],
    ['password_not_set', 'passwordNotSet', 401],
    ['password_not_set_google', 'passwordNotSetGoogle', 401],
    ['password_not_set_github', 'passwordNotSetGithub', 401],
    ['password_not_set_google_github', 'passwordNotSetGoogleGithub', 401],
    ['user_deactivated', 'userDeactivated', 403],
  ] as const)('explains the next step when the refusal is %s', async (code, key, status) => {
    const navigate = vi.spyOn(window.location, 'assign').mockImplementation(() => {});
    mockServer.use(mockApi.post('/sessions', status, buildProblem(code, status)));
    renderScreen(<SignInComponent />);

    await submitPasswordSignIn();

    expect(await screen.findByText(DICTIONARIES.en.errors[key])).toBeTruthy();
    expect(screen.getByRole('alert').textContent).toContain(DICTIONARIES.en.errors[key]);
    expect(mockServer.getRequests()).toHaveLength(1);
    expect(navigate).not.toHaveBeenCalled();
    expect(getSessionSnapshot().token).toBeUndefined();
  });

  it('opens registration when an invited user follows the sign-up action', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('registration_incomplete', 401)));
    const onSwitchToSignUp = vi.fn();
    const { user } = renderScreen(<SignInComponent onSwitchToSignUp={onSwitchToSignUp} />);
    await submitPasswordSignIn();
    await screen.findByText(DICTIONARIES.en.errors.registrationIncomplete);

    await user.click(screen.getByRole('button', { name: 'Sign up' }));

    expect(onSwitchToSignUp).toHaveBeenCalledOnce();
  });

  it('offers the existing recovery action when no password or recommended provider is available', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('password_not_set', 401)));
    const onSwitchToResetPassword = vi.fn();
    const { user } = renderScreen(
      <SignInComponent onSwitchToResetPassword={onSwitchToResetPassword} />,
    );
    await submitPasswordSignIn();

    expect(await screen.findByText(DICTIONARIES.en.users.signIn.passwordRecoveryHint)).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Set a password' }));

    expect(onSwitchToResetPassword).toHaveBeenCalledOnce();
    expect(mockServer.getRequests()).toHaveLength(1);
  });

  it.each(['Google', 'GitHub'] as const)(
    'starts the existing %s flow when the user follows a passwordless recommendation',
    async (provider) => {
      const code = provider === 'Google' ? 'password_not_set_google' : 'password_not_set_github';
      mockServer.use(mockApi.post('/sessions', 401, buildProblem(code, 401)));
      const navigate = vi.spyOn(window.location, 'assign').mockImplementation(() => {});
      const { user } = renderScreen(<SignInComponent />);
      await submitPasswordSignIn();
      await screen.findByRole('alert');

      await user.click(screen.getByRole('button', { name: `Continue with ${provider}` }));

      expect(navigate).toHaveBeenCalledOnce();
      expect(new URL(String(navigate.mock.calls[0][0])).hostname).toBe(
        provider === 'Google' ? 'accounts.google.com' : 'github.com',
      );
    },
  );

  it.each(['email_not_found', 'password_incorrect', 'password_not_set'] as const)(
    'retranslates visible feedback and preserves the draft when the language changes after %s',
    async (code) => {
      mockServer.use(mockApi.post('/sessions', 401, buildProblem(code, 401)));
      const { user } = renderScreen(
        <>
          <SignInComponent />
          <LocaleSwitch />
        </>,
      );
      await submitPasswordSignIn();
      const key =
        code === 'email_not_found'
          ? 'emailNotFound'
          : code === 'password_incorrect'
            ? 'passwordIncorrect'
            : 'passwordNotSet';
      await screen.findByText(DICTIONARIES.en.errors[key]);

      await user.click(screen.getByRole('button', { name: 'Switch language' }));

      expect(await screen.findByText(DICTIONARIES.ru.errors[key])).toBeTruthy();
      expect(
        screen.getByRole<HTMLInputElement>('textbox', {
          name: DICTIONARIES.ru.users.fields.yourEmail,
        }).value,
      ).toBe('owner@example.com');
      expect(
        screen.getByLabelText<HTMLInputElement>(DICTIONARIES.ru.common.fields.password, {
          selector: 'input',
        }).value,
      ).toBe('correct-password');
      if (code === 'password_not_set') {
        expect(screen.getByText(DICTIONARIES.ru.users.signIn.passwordRecoveryHint)).toBeTruthy();
      }
    },
  );

  it.each(['Your email', 'Password'])(
    'clears the refusal and guidance when the user edits %s',
    async (label) => {
      mockServer.use(mockApi.post('/sessions', 401, buildProblem('password_not_set', 401)));
      const { user } = renderScreen(<SignInComponent />);
      await submitPasswordSignIn();
      await screen.findByRole('alert');

      await user.type(screen.getByLabelText(label, { selector: 'input' }), 'x');

      expect(screen.queryByRole('alert')).toBeNull();
      expect(screen.queryByRole('button', { name: 'Set a password' })).toBeNull();
      expect(mockServer.getRequests()).toHaveLength(1);
    },
  );

  it.each([
    ['email_not_found', 'emailNotFound'],
    ['password_incorrect', 'passwordIncorrect'],
    ['password_not_set', 'passwordNotSet'],
  ] as const)(
    'discards the delayed %s refusal when OAuth supersedes the mounted password form',
    async (code, key) => {
      const response = mockApi.delayed({
        method: 'post',
        path: '/sessions',
        status: 401,
        body: buildProblem(code, 401),
      });
      mockServer.use(response.handler);
      const signIn = vi.spyOn(userApi, 'signIn');
      const navigate = vi.spyOn(window.location, 'assign').mockImplementation(() => {});
      const { user } = renderScreen(<SignInComponent />);
      const email = screen.getByRole('textbox', { name: 'Your email' });
      const password = screen.getByLabelText('Password', { selector: 'input' });
      await submitPasswordSignIn();
      await waitFor(() => expect(mockServer.getRequests()).toHaveLength(1));

      await user.click(screen.getByRole('button', { name: 'Continue with Google' }));

      expect(navigate).toHaveBeenCalledOnce();
      expect(new URL(String(navigate.mock.calls[0][0])).hostname).toBe('accounts.google.com');

      await releaseAndSettle(signIn, response.release);

      expect(screen.getByRole('textbox', { name: 'Your email' })).toBe(email);
      expect(screen.getByLabelText('Password', { selector: 'input' })).toBe(password);
      expect(email.getAttribute('aria-invalid')).not.toBe('true');
      expect(password.getAttribute('aria-invalid')).not.toBe('true');
      expect(screen.queryByText(DICTIONARIES.en.errors[key])).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
      expect(screen.queryByText(DICTIONARIES.en.users.signIn.passwordRecoveryHint)).toBeNull();
      expect(screen.queryByText(DICTIONARIES.en.users.signIn.contactAdministratorHint)).toBeNull();
      expect(screen.queryByRole('button', { name: 'Set a password' })).toBeNull();
      expect(mockServer.getRequests()).toHaveLength(1);
      expect(getSessionSnapshot().token).toBeUndefined();
    },
  );
});

it('publishes neither a session nor a code transition after the sign-in page is left', async () => {
  const response = mockApi.delayed({
    method: 'post',
    path: '/sessions',
    status: 200,
    body: { outcome: 'CODE_REQUIRED', pendingSignInId: 'obsolete', email: 'owner@example.com' },
  });
  mockServer.use(response.handler);

  const onCodeRequired = vi.fn();
  const view = renderScreen(<SignInComponent onCodeRequired={onCodeRequired} />);
  await submitPasswordSignIn();
  await waitFor(() => expect(mockServer.getRequests()).toHaveLength(1));

  view.rerender(<div>Sign in closed</div>);
  const processed = new Promise<void>((resolve) =>
    mockServer.events.on('response:mocked', () => resolve()),
  );

  await act(async () => {
    response.release();
    await processed;
  });

  expect(onCodeRequired).not.toHaveBeenCalled();
  expect(getSessionSnapshot().token).toBeUndefined();
});

it('preserves the bootstrap login when the existing administrator has no email address', async () => {
  mockServer.use(
    mockApi.post('/sessions', 200, {
      outcome: 'COMPLETED',
      token: 'bootstrap-session',
      userId: 'bootstrap-admin',
      email: 'admin',
    }),
  );

  renderScreen(<SignInComponent />);
  fireEvent.change(screen.getByRole('textbox', { name: 'Your email' }), {
    target: { value: 'admin' },
  });
  fireEvent.change(screen.getByLabelText('Password', { selector: 'input' }), {
    target: { value: 'bootstrap-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));

  await waitFor(() => expect(getSessionSnapshot().token).toBe('bootstrap-session'));
  expect(await mockServer.getRequests()[0].json()).toMatchObject({ email: 'admin' });
});
