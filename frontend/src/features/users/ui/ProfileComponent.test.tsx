// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { userApi } from '../../../entities/users';
import { accessTokenHelper } from '../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { ProfileComponent } from './ProfileComponent';

const mockServer = setUpMockApi();

const openProfile = async () => {
  userApi.saveAuthorizedData('old-session', 'owner-id');
  mockServer.use(
    mockApi.get('/users/{userId}', 200, {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Owner',
      role: 'ADMIN',
      status: 'ACTIVE',
      createdAt: '2026-10-01T00:00:00Z',
    }),
  );
  renderScreen(<ProfileComponent contentHeight={800} />);
  await screen.findByDisplayValue('owner@example.com');
};

const submitPasswordChange = () => {
  fireEvent.change(screen.getByPlaceholderText('Enter new password'), {
    target: { value: 'replacement-password' },
  });
  fireEvent.change(screen.getByPlaceholderText('Confirm new password'), {
    target: { value: 'replacement-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Change password' }));
};

describe('ProfileComponent', () => {
  it('clears authentication and notifies listeners after a successful password change', async () => {
    mockServer.use(
      mockApi.put('/users/{userId}/password', 204, undefined),
      mockApi.post('/sessions', 200, {
        outcome: 'CODE_REQUIRED',
        pendingSignInId: 'pending-id',
        email: 'owner@example.com',
      }),
    );
    await openProfile();
    const onAuthChanged = vi.fn();
    userApi.addAuthListener(onAuthChanged);
    try {
      submitPasswordChange();
      await waitFor(() => expect(onAuthChanged).toHaveBeenCalledOnce());
      expect(accessTokenHelper.getAccessToken()).toBeUndefined();
      expect(accessTokenHelper.getUserId()).toBeUndefined();
      expect(
        await screen.findByText('Password changed. Sign in again with your new password.'),
      ).toBeTruthy();
    } finally {
      userApi.removeAuthListener(onAuthChanged);
    }
  });

  it('keeps the profile and session when the password change is refused', async () => {
    mockServer.use(
      mockApi.put('/users/{userId}/password', 403, buildProblem('permission_denied', 403)),
    );
    await openProfile();
    submitPasswordChange();
    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(accessTokenHelper.getAccessToken()).toBe('old-session');
    expect(accessTokenHelper.getUserId()).toBe('owner-id');
    expect(screen.getByDisplayValue('owner@example.com')).toBeTruthy();
    expect(
      screen.queryByText('Password changed. Sign in again with your new password.'),
    ).toBeNull();
  });
});
