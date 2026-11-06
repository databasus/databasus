// @vitest-environment happy-dom
import { act, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { ApplicationScreen } from '@/app/ui/ApplicationScreen';
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
      HttpResponse.json({ tag_name: 'v99.0.0' }),
    ),
  );

  renderScreen(<ApplicationScreen />, { theme });
  await screen.findByRole('button', { name: 'Profile' });

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
