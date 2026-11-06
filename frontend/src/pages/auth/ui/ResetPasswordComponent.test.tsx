// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ResetPasswordComponent } from '@/pages/auth/ui/ResetPasswordComponent';
import { getSessionSnapshot } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const observeResponse = () =>
  new Promise<void>((resolve) => {
    const onResponse = () => {
      server.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    server.events.on('response:mocked', onResponse);
  });

afterEach(() => vi.restoreAllMocks());
const submitReset = () => {
  fireEvent.change(screen.getByRole('textbox', { name: 'Reset Code' }), {
    target: { value: '123456' },
  });
  fireEvent.change(screen.getByLabelText('New Password', { selector: 'input' }), {
    target: { value: 'replacement-password' },
  });
  fireEvent.change(screen.getByLabelText('Confirm Password', { selector: 'input' }), {
    target: { value: 'replacement-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Reset password' }));
};

describe('ResetPasswordComponent', () => {
  it('prevents another completion after success while navigation is pending', async () => {
    const onSignIn = vi.fn();
    const reset = mockApi.delayed({
      method: 'post',
      path: '/password-reset-completions',
      status: 204,
      body: undefined,
    });
    server.use(reset.handler);

    renderScreen(
      <ResetPasswordComponent initialEmail="owner@example.com" onSwitchToSignIn={onSignIn} />,
    );
    submitReset();
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    const processed = observeResponse();
    await act(async () => {
      reset.release();
      await processed;
    });

    expect(screen.getByText('Password reset successfully! Redirecting to sign in...')).toBeTruthy();
    const submit = screen.getByRole('button', { name: 'Reset password' });
    expect(submit.hasAttribute('disabled')).toBe(true);
    fireEvent.click(submit);
    fireEvent.click(submit);
    expect(server.getRequests()).toHaveLength(1);
    expect(getSessionSnapshot().token).toBeUndefined();
    expect(onSignIn).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1999);
    });
    expect(onSignIn).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });

    expect(onSignIn).toHaveBeenCalledOnce();
    expect(server.getRequests()).toHaveLength(1);
  });

  it('returns to ordinary sign in without activating a session after reset completion', async () => {
    const onSignIn = vi.fn();
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    server.use(mockApi.post('/password-reset-completions', 204, undefined));

    renderScreen(
      <ResetPasswordComponent initialEmail="owner@example.com" onSwitchToSignIn={onSignIn} />,
    );

    const processed = observeResponse();
    await act(async () => {
      submitReset();
      await processed;
    });

    expect(screen.getByText('Password reset successfully! Redirecting to sign in...')).toBeTruthy();
    expect(getSessionSnapshot().token).toBeUndefined();
    expect(await server.getRequests()[0].json()).toEqual({
      email: 'owner@example.com',
      code: '123456',
      newPassword: 'replacement-password',
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(onSignIn).toHaveBeenCalledOnce();
  });

  it('retains correction values and the request-code path when the code is refused', async () => {
    const onRequest = vi.fn();
    server.use(
      mockApi.post('/password-reset-completions', 400, buildProblem('invalid_reset_code', 400)),
    );

    const { user } = renderScreen(
      <ResetPasswordComponent initialEmail="owner@example.com" onSwitchToRequestCode={onRequest} />,
    );

    submitReset();
    await screen.findByText('The reset code is incorrect or has expired. Request a new one.');
    expect(screen.getByDisplayValue('123456')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Reset password' }).hasAttribute('disabled')).toBe(
      false,
    );

    await user.click(screen.getByRole('button', { name: 'Request new code' }));
    expect(onRequest).toHaveBeenCalledOnce();
  });

  it('prevents duplicate requests and suppresses completion after the reset page is left', async () => {
    const reset = mockApi.delayed({
      method: 'post',
      path: '/password-reset-completions',
      status: 204,
      body: undefined,
    });
    const onSignIn = vi.fn();
    server.use(reset.handler);

    const view = renderScreen(
      <ResetPasswordComponent initialEmail="owner@example.com" onSwitchToSignIn={onSignIn} />,
    );
    submitReset();
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    fireEvent.click(screen.getByRole('button', { name: /Reset password/ }));
    expect(server.getRequests()).toHaveLength(1);

    view.rerender(<div>Reset closed</div>);
    const processed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );

    await act(async () => {
      reset.release();
      await processed;
    });

    expect(onSignIn).not.toHaveBeenCalled();
    expect(screen.queryByText('Password reset successfully! Redirecting to sign in...')).toBeNull();
  });

  it('shows associated field feedback without submitting when the reset code is incomplete', async () => {
    renderScreen(<ResetPasswordComponent initialEmail="owner@example.com" />);
    fireEvent.change(screen.getByRole('textbox', { name: 'Reset Code' }), {
      target: { value: '12x' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Reset password' }));
    expect(await screen.findByText('Code must be 6 digits')).toBeTruthy();
    expect(screen.getByRole('textbox', { name: 'Reset Code' }).getAttribute('aria-invalid')).toBe(
      'true',
    );
    expect(server.getRequests()).toHaveLength(0);
  });
});
