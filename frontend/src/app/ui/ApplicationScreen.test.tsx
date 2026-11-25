// @vitest-environment happy-dom
import { act, fireEvent, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { ApplicationScreen } from '@/app/ui/ApplicationScreen';
import type { InstanceSettings, User } from '@/entities/users';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

vi.hoisted(() => vi.stubEnv('VITE_APP_VERSION', 'dev'));

const mockServer = setUpMockApi();

const renderDiskUsage = async ({
  usedPercent,
  theme,
}: {
  usedPercent: number;
  theme: 'light' | 'dark';
}) => {
  activateSession({ token: 'session-token', userId: 'user-id' });
  const usedSpaceBytes = Math.round(usedPercent * 1024 ** 3);
  const totalSpaceBytes = 100 * 1024 ** 3;
  const diskResponse = mockApi.delayed({
    method: 'get',
    path: '/system/disk-usages',
    status: 200,
    body: {
      usedSpaceBytes,
      totalSpaceBytes,
      freeSpaceBytes: totalSpaceBytes - usedSpaceBytes,
    },
  });

  mockServer.use(
    diskResponse.handler,
    mockApi.get('/users/{userId}', 200, {
      id: 'user-id',
      email: 'user@example.com',
      name: 'User',
      role: 'MEMBER',
      status: 'ACTIVE',
      isRootAdmin: false,
      createdAt: '2026-10-01T10:00:00Z',
      authenticatorAppConnectedAt: null,
      unusedRecoveryCodeCount: null,
    }),
    mockApi.get('/workspaces', 200, { items: [] }),
    mockApi.get('/instance-settings', 200, {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isEmailConfigured: false,
      twoFactorRequirement: 'OFF',
    }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'v99.0.0' }),
    ),
  );

  renderScreen(<ApplicationScreen />, { theme });
  await screen.findByRole('link', { name: 'Profile' });

  let acceptDiskResponse = () => {};
  const diskResponseCompleted = new Promise<void>((resolve) => {
    acceptDiskResponse = resolve;
  });
  const observeDiskResponse = ({ request }: { request: Request }) => {
    if (new URL(request.url).pathname.endsWith('/system/disk-usages')) acceptDiskResponse();
  };
  mockServer.events.on('response:mocked', observeDiskResponse);

  try {
    await act(async () => {
      diskResponse.release();
      await diskResponseCompleted;
    });
  } finally {
    mockServer.events.removeListener('response:mocked', observeDiskResponse);
  }

  return within(screen.getByRole('banner'));
};

it('hides the new version link in a development build', async () => {
  await renderDiskUsage({ usedPercent: 50, theme: 'light' });

  expect(screen.queryByRole('link', { name: 'New version available' })).toBeNull();
});

describe.each(['light', 'dark'] as const)('ApplicationScreen in %s theme', (theme) => {
  it.each([50, 85])('hides the disk summary at %s percent', async (usedPercent) => {
    const header = await renderDiskUsage({ usedPercent, theme });

    expect(header.queryByText(/GB used/)).toBeNull();
  });

  it.each([85.01, 86, 90, 95])(
    'shows the disk summary in the muted color at %s percent',
    async (usedPercent) => {
      const header = await renderDiskUsage({ usedPercent, theme });

      const diskSummary = header.getByText(
        `${usedPercent.toFixed(1)} of 100.0 GB used (${usedPercent.toFixed(1)}%)`,
      );

      expect(diskSummary.classList.contains('text-muted')).toBe(true);
      expect(diskSummary.classList.contains('text-danger')).toBe(false);
    },
  );

  it.each([95.01, 96])(
    'shows the disk summary in the danger color at %s percent',
    async (usedPercent) => {
      const header = await renderDiskUsage({ usedPercent, theme });

      const diskSummary = header.getByText(
        `${usedPercent.toFixed(1)} of 100.0 GB used (${usedPercent.toFixed(1)}%)`,
      );

      expect(diskSummary.classList.contains('text-danger')).toBe(true);
      expect(diskSummary.classList.contains('text-muted')).toBe(false);
    },
  );
});

describe('Requiring the authenticator app from the instance settings', () => {
  const admin: User = {
    id: 'admin-id',
    email: 'admin@example.com',
    name: 'Admin',
    role: 'ADMIN',
    status: 'ACTIVE',
    isRootAdmin: false,
    createdAt: '2026-10-01T10:00:00Z',
    authenticatorAppConnectedAt: null,
    unusedRecoveryCodeCount: null,
  };
  const settings: InstanceSettings = {
    isAllowExternalRegistrations: false,
    isAllowMemberInvitations: false,
    isMemberAllowedToCreateWorkspaces: false,
    isEmailConfigured: false,
    twoFactorRequirement: 'OFF',
  };

  const saveAppRequirement = async (account: User) => {
    activateSession({ token: 'session-token', userId: account.id });
    const setupStarts: string[] = [];
    mockServer.use(
      mockApi.get('/users/{userId}', 200, account),
      mockApi.get('/workspaces', 200, { items: [] }),
      mockApi.get('/instance-settings', 200, settings),
      mockApi.put('/instance-settings', 200, {
        ...settings,
        twoFactorRequirement: 'AUTHENTICATOR_APP',
      }),
      mockApi.get('/system/disk-usages', 200, {
        usedSpaceBytes: 1,
        totalSpaceBytes: 10,
        freeSpaceBytes: 9,
      }),
      mockApi.get('/verification-agents', 200, { items: [] }),
      mockApi.get('/audit-events', 200, { items: [] }),
      http.post(`*${API_BASE_PATH}/users/:userId/authenticator-app-setups`, () => {
        setupStarts.push(account.id);
        return HttpResponse.json(
          {
            authenticatorAppSetupId: 'setup-id',
            secret: 'JBSWY3DPEHPK3PXP',
            otpauthUri: 'otpauth://totp/Databasus:admin@example.com?secret=JBSWY3DPEHPK3PXP',
            expiresAt: '2999-01-01T00:00:00Z',
          },
          { status: 201 },
        );
      }),
      http.get('https://api.github.com/repos/databasus/databasus', () =>
        HttpResponse.json({ stargazers_count: 1 }),
      ),
      http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
        HttpResponse.json({ tag_name: 'v99.0.0' }),
      ),
    );
    renderScreen(<ApplicationScreen />, { route: '/databasus-settings' });

    fireEvent.click(await screen.findByRole('switch', { name: 'Require 2FA' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));
    await screen.findByText('Settings updated successfully');

    return setupStarts;
  };

  it('takes an administrator without an app to the profile with the connection open', async () => {
    const setupStarts = await saveAppRequirement(admin);

    expect(
      await screen.findByRole('dialog', { name: 'Connect an authenticator app' }),
    ).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Profile', current: 'page' })).toBeTruthy();
    expect(setupStarts).toEqual([admin.id]);
  });

  it('keeps an administrator with an app on the settings', async () => {
    const setupStarts = await saveAppRequirement({
      ...admin,
      authenticatorAppConnectedAt: '2026-10-02T10:00:00Z',
      unusedRecoveryCodeCount: 10,
    });

    expect(screen.getByRole('switch', { name: 'Require 2FA' })).toBeTruthy();
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(setupStarts).toEqual([]);
  });
});
