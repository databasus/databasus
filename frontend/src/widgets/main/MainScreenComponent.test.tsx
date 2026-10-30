// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { accessTokenHelper } from '../../shared/api';
import { mockApi, setUpMockApi } from '../../shared/lib/test/mockApi';
import { renderScreen } from '../../shared/lib/test/renderScreen';
import { ThemeProvider } from '../../shared/theme';
import { MainScreenComponent } from './MainScreenComponent';

const mockServer = setUpMockApi();

describe('MainScreenComponent', () => {
  it('shows the returned disk capacity and percentage when the system disk endpoint responds', async () => {
    accessTokenHelper.saveUserId('user-id');
    accessTokenHelper.saveAccessToken('session-token');
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
        <MainScreenComponent />
      </ThemeProvider>,
    );

    expect(
      await screen.findByText(
        (_, element) => element?.textContent === '90.0 of 100.0 GBROM used (90.0%)',
      ),
    ).toBeTruthy();
  });
});
