import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { SignInComponent } from '@/pages/auth/ui/SignInComponent';
// @vitest-environment happy-dom
import { getSessionSnapshot } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const submitPasswordSignIn = () => {
  fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
    target: { value: 'owner@example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText('********'), {
    target: { value: 'correct-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
};

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

      submitPasswordSignIn();

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

    submitPasswordSignIn();

    await waitFor(() =>
      expect(onCodeRequired).toHaveBeenCalledWith({
        pendingSignInId: 'pending-sign-in-id',
        email: 'owner@example.com',
      }),
    );
  });

  it('shows the wrong credentials message when the password is refused', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('invalid_credentials', 401)));
    renderScreen(<SignInComponent />);

    submitPasswordSignIn();

    expect(await screen.findByText('The email or password is incorrect.')).toBeTruthy();
  });

  it('keeps the entered values and sends one request when the password is refused', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('invalid_credentials', 401)));
    renderScreen(<SignInComponent />);

    submitPasswordSignIn();

    await screen.findByText('The email or password is incorrect.');
    expect(screen.getByPlaceholderText<HTMLInputElement>('your@email.com').value).toBe(
      'owner@example.com',
    );
    expect(screen.getByPlaceholderText<HTMLInputElement>('********').value).toBe(
      'correct-password',
    );
    expect(mockServer.getRequests()).toHaveLength(1);
    expect(getSessionSnapshot().token).toBeUndefined();
  });
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
  submitPasswordSignIn();
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
