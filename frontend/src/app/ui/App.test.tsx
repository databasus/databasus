import { fireEvent, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { renderApp } from '@/app/ui/renderApp';
// @vitest-environment happy-dom
import { activateSession, getSessionSnapshot, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';

const mockServer = setUpMockApi();

const openProfile = async (isTwoFactorAuthRequired: boolean) => {
  activateSession({ token: 'old-session', userId: 'owner-id' });

  mockServer.use(
    mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
    mockApi.get('/system/disk-usages', 200, {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    }),
    mockApi.get('/users/{userId}', 200, {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Owner',
      role: 'ADMIN',
      status: 'ACTIVE',
      createdAt: '2026-10-01T00:00:00Z',
    }),
    mockApi.get('/workspaces', 200, { items: [] }),
    mockApi.get('/instance-settings', 200, {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isEmailConfigured: true,
      isTwoFactorAuthRequired,
    }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'v0.0.0' }),
    ),
  );

  renderApp();
  fireEvent.click(await screen.findByRole('link', { name: 'Profile' }));
  await screen.findByDisplayValue('owner@example.com');
  fireEvent.change(screen.getByPlaceholderText('Enter new password'), {
    target: { value: 'replacement-password' },
  });
  fireEvent.change(screen.getByPlaceholderText('Confirm new password'), {
    target: { value: 'replacement-password' },
  });
};

describe('App password change', () => {
  for (const isTwoFactorAuthRequired of [true, false]) {
    it(`shows ordinary sign-in and retains the notice after password change with second factor ${isTwoFactorAuthRequired}`, async () => {
      let sessionRequestCount = 0;
      const onRequest = ({ request }: { request: Request }) => {
        if (request.method === 'POST' && new URL(request.url).pathname.endsWith('/sessions'))
          sessionRequestCount++;
      };
      mockServer.events.on('request:start', onRequest);

      mockServer.use(
        mockApi.put('/users/{userId}/password', 204, undefined),
        mockApi.post(
          '/sessions',
          200,
          isTwoFactorAuthRequired
            ? {
                outcome: 'CODE_REQUIRED',
                pendingSignInId: 'pending-id',
                email: 'owner@example.com',
              }
            : {
                outcome: 'COMPLETED',
                userId: 'owner-id',
                email: 'owner@example.com',
                token: 'new-session',
              },
        ),
      );

      try {
        await openProfile(isTwoFactorAuthRequired);
        fireEvent.click(screen.getByRole('button', { name: 'Change password' }));
        expect(await screen.findByRole('button', { name: 'Sign in' })).toBeTruthy();
        expect(screen.getAllByPlaceholderText('********')).toHaveLength(1);
        expect(window.location.pathname).toBe('/');
        expect(
          await screen.findByText('Password changed. Sign in again with your new password.'),
        ).toBeTruthy();
        expect(getSessionSnapshot().token).toBeUndefined();
        expect(getSessionSnapshot().userId).toBeUndefined();
        expect(sessionRequestCount).toBe(0);
      } finally {
        mockServer.events.removeListener('request:start', onRequest);
      }
    });
  }

  it('keeps the authenticated profile when the password change fails', async () => {
    mockServer.use(
      mockApi.put('/users/{userId}/password', 403, buildProblem('permission_denied', 403)),
    );

    await openProfile(true);
    fireEvent.click(screen.getByRole('button', { name: 'Change password' }));
    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(getSessionSnapshot().token).toBe('old-session');
    expect(screen.getByDisplayValue('owner@example.com')).toBeTruthy();
    expect(
      screen.queryByText('Password changed. Sign in again with your new password.'),
    ).toBeNull();
  });
});

describe('App routes', () => {
  it.each([
    '/unrecognized/path',
    '/auth/callback?error=access_denied',
    '/storages/google-oauth?code=synthetic-code&state=obsolete',
  ])('opens a fresh sign-in page after recovery from %s', async (path) => {
    const previousAddress = window.location.href;
    invalidateSession();
    mockServer.use(
      mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
      http.get('https://api.github.com/repos/databasus/databasus', () =>
        HttpResponse.json({ stargazers_count: 1 }),
      ),
    );
    window.history.replaceState(null, '', path);

    try {
      renderApp();
      if (path.startsWith('/auth/callback')) {
        fireEvent.click(await screen.findByRole('button', { name: 'Return to sign in' }));
      }

      expect(await screen.findByRole('button', { name: 'Sign in' })).toBeTruthy();
      expect(window.location.pathname).toBe('/');
      expect(window.location.search).toBe('');
      expect(getSessionSnapshot().token).toBeUndefined();
      expect(mockServer.getRequests().filter((request) => request.method === 'POST')).toHaveLength(
        0,
      );
    } finally {
      window.history.replaceState(null, '', previousAddress);
    }
  });
});
