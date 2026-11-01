import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { ApplicationScreen } from '@/app/ui/ApplicationScreen';
// @vitest-environment happy-dom
import { activateSession } from '@/shared/auth';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { ThemeProvider } from '@/shared/theme';

const mockServer = setUpMockApi();

describe('ApplicationScreen', () => {
  it('shows the returned disk capacity and percentage when the system disk endpoint responds', async () => {
    activateSession({ token: 'session-token', userId: 'user-id' });

    mockServer.use(
      mockApi.get('/system/disk-usages', 200, {
        usedSpaceBytes: 90 * 1024 ** 3,
        totalSpaceBytes: 100 * 1024 ** 3,
        freeSpaceBytes: 10 * 1024 ** 3,
      }),
      mockApi.get('/users/{userId}', 200, {
        id: 'user-id',
        email: 'user@example.com',
        name: 'User',
        role: 'MEMBER',
        status: 'ACTIVE',
        createdAt: '2026-10-01T10:00:00Z',
      }),
      mockApi.get('/workspaces', 200, { items: [] }),
      mockApi.get('/instance-settings', 200, {
        isAllowExternalRegistrations: false,
        isAllowMemberInvitations: false,
        isMemberAllowedToCreateWorkspaces: false,
        isEmailConfigured: false,
        isTwoFactorAuthRequired: false,
      }),
      http.get('https://api.github.com/repos/databasus/databasus', () =>
        HttpResponse.json({ stargazers_count: 1 }),
      ),
      http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
        HttpResponse.json({ tag_name: 'v0.0.0' }),
      ),
    );

    renderScreen(
      <ThemeProvider>
        <ApplicationScreen />
      </ThemeProvider>,
    );

    expect(
      await screen.findByText(
        (_, element) => element?.textContent === '90.0 of 100.0 GB used (90.0%)',
      ),
    ).toBeTruthy();
  });
});
