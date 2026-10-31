// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { SignInCodeComponent } from '@/pages/auth/ui/SignInCodeComponent';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const pendingSignIn = { pendingSignInId: 'pending-sign-in-id', email: 'owner@example.com' };

const renderCodeScreen = (onPendingSignInLost = vi.fn()) =>
  renderScreen(
    <SignInCodeComponent
      pendingSignIn={pendingSignIn}
      onPendingSignInReplaced={vi.fn()}
      onPendingSignInLost={onPendingSignInLost}
      onSwitchToSignIn={vi.fn()}
    />,
  );

const submitCode = () => {
  fireEvent.change(screen.getByPlaceholderText('123456'), { target: { value: '123456' } });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
};

describe('SignInCodeComponent', () => {
  it('shows the incorrect code message when the code is wrong', async () => {
    mockServer.use(
      mockApi.post(
        '/pending-sign-ins/{pendingSignInId}/verify',
        400,
        buildProblem('sign_in_code_incorrect', 400),
      ),
    );
    renderCodeScreen();

    submitCode();

    expect(await screen.findByText('The code is incorrect.')).toBeTruthy();
  });

  it('asks to wait when another code is requested too soon', async () => {
    mockServer.use(
      mockApi.post(
        '/pending-sign-ins/{pendingSignInId}/resend',
        429,
        buildProblem('sign_in_code_resent_too_soon', 429),
      ),
    );
    renderCodeScreen();

    fireEvent.click(screen.getByRole('button', { name: 'Send another code' }));

    expect(
      await screen.findByText(
        'A code was sent less than a minute ago. Wait a minute before requesting another.',
      ),
    ).toBeTruthy();
  });

  it('returns to the sign-in when the pending sign-in can no longer be completed', async () => {
    mockServer.use(
      mockApi.post(
        '/pending-sign-ins/{pendingSignInId}/verify',
        410,
        buildProblem('pending_sign_in_not_usable', 410),
      ),
    );
    const onPendingSignInLost = vi.fn();
    renderCodeScreen(onPendingSignInLost);

    submitCode();

    await waitFor(() => expect(onPendingSignInLost).toHaveBeenCalledOnce());
  });
});
