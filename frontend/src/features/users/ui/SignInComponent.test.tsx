// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { accessTokenHelper } from '../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { SignInComponent } from './SignInComponent';

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
  it('stores the session when the sign-in completes', async () => {
    mockServer.use(
      mockApi.post('/sessions', 200, {
        outcome: 'COMPLETED',
        userId: 'owner-id',
        email: 'owner@example.com',
        token: 'session-token',
      }),
    );
    const onCodeRequired = vi.fn();
    renderScreen(<SignInComponent onCodeRequired={onCodeRequired} />);

    submitPasswordSignIn();

    await waitFor(() => expect(accessTokenHelper.getAccessToken()).toBe('session-token'));
    expect(accessTokenHelper.getUserId()).toBe('owner-id');
    expect(onCodeRequired).not.toHaveBeenCalled();
  });

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
});
