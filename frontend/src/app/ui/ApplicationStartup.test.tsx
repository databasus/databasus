// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterAll, describe, expect, it, vi } from 'vitest';

import { API_BASE_PATH } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

import { ApplicationScreen } from './ApplicationScreen';
import { renderApp } from './renderApp';

vi.hoisted(() => vi.stubEnv('VITE_APP_VERSION', '1.0.0'));
afterAll(() => vi.unstubAllEnvs());
const server = setUpMockApi();
const user = {
  id: 'owner-id',
  name: 'Owner',
  email: 'owner@example.com',
  role: 'MEMBER' as const,
  status: 'ACTIVE' as const,
  createdAt: '2026-10-01T00:00:00Z',
};
const settings = {
  isAllowExternalRegistrations: false,
  isAllowMemberInvitations: false,
  isMemberAllowedToCreateWorkspaces: false,
  isTwoFactorAuthRequired: false,
  isEmailConfigured: true,
};
const seed = () => {
  activateSession({ token: 'current-session', userId: user.id });
  server.use(
    mockApi.get('/system/version', 200, { version: '1.0.0' }),
    mockApi.get('/users/{userId}', 200, user),
    mockApi.get('/instance-settings', 200, settings),
    mockApi.get('/workspaces', 200, { items: [] }),
    mockApi.get('/system/disk-usages', 200, {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    }),
    mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
      HttpResponse.json({ tag_name: 'dev' }),
    ),
  );
};

describe('Application startup', () => {
  it.each(['/users/{userId}', '/instance-settings', '/workspaces'] as const)(
    'offers recovery without protected actions when essential %s cannot load',
    async (path) => {
      seed();
      server.use(mockApi.get(path, 500, buildProblem('internal', 500)));

      renderScreen(<ApplicationScreen />);
      const retry = await screen.findByRole('button', { name: 'Retry' });
      expect(screen.queryByRole('button', { name: 'Users' })).toBeNull();
      expect(screen.queryByRole('button', { name: 'Create workspace' })).toBeNull();

      server.use(
        mockApi.get('/users/{userId}', 200, user),
        mockApi.get('/instance-settings', 200, settings),
        mockApi.get('/workspaces', 200, { items: [] }),
      );
      fireEvent.click(retry);
      expect(await screen.findByRole('button', { name: 'Profile' })).toBeTruthy();
    },
  );

  it('opens permitted account screens when supplementary disk data fails', async () => {
    seed();
    server.use(mockApi.get('/system/disk-usages', 500, buildProblem('internal', 500)));

    renderScreen(<ApplicationScreen />);
    fireEvent.click(await screen.findByRole('button', { name: 'Profile' }));

    expect(await screen.findByDisplayValue('owner@example.com')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Users' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Databasus settings' })).toBeNull();
  });

  it('rejects previous account startup after logout and activation of another account', async () => {
    seed();
    const oldUser = mockApi.delayed({
      method: 'get',
      path: '/users/{userId}',
      status: 200,
      body: user,
    });
    server.use(oldUser.handler);
    localStorage.setItem('selected_workspace_id', 'old-workspace');
    localStorage.setItem('databasus-theme', 'dark');
    localStorage.setItem('databasus-locale', 'en');
    renderApp();

    await waitFor(() =>
      expect(server.getRequests().some((request) => request.url.endsWith('/users/owner-id'))).toBe(
        true,
      ),
    );

    act(() => {
      invalidateSession();
    });
    await screen.findByRole('button', { name: 'Sign in' });

    server.use(
      mockApi.get('/users/{userId}', 200, {
        ...user,
        id: 'new-owner',
        name: 'Another owner',
        email: 'another@example.com',
      }),
    );
    act(() => {
      activateSession({ token: 'new-session', userId: 'new-owner' });
    });
    fireEvent.click(await screen.findByRole('button', { name: 'Profile' }));
    await screen.findByDisplayValue('another@example.com');

    const responded = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );
    await act(async () => {
      oldUser.release();
      await responded;
    });

    expect(screen.queryByDisplayValue('owner@example.com')).toBeNull();
    expect(localStorage.getItem('selected_workspace_id')).toBeNull();
    expect(localStorage.getItem('databasus-theme')).toBe('dark');
    expect(localStorage.getItem('databasus-locale')).toBe('en');
  });
});

const workspace = {
  id: 'available-workspace',
  name: 'Available workspace',
  createdAt: '2026-10-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_VIEWER' as const,
};
const seedWorkspaceContent = () =>
  server.use(
    mockApi.get('/databases', 200, { items: [] }),
    mockApi.get('/backup-configs', 200, { items: [] }),
    mockApi.get('/storages', 200, { items: [] }),
  );

it.each(['ADMIN', 'MEMBER'] as const)(
  'derives workspace management from account permissions when a %s has an explicit viewer role',
  async (role) => {
    seed();
    seedWorkspaceContent();
    server.use(
      mockApi.get('/users/{userId}', 200, { ...user, role }),
      mockApi.get('/workspaces', 200, { items: [workspace] }),
    );
    localStorage.setItem('selected_workspace_id', 'unavailable-workspace');

    renderScreen(<ApplicationScreen />);
    await screen.findByRole('button', { name: 'Profile' });

    await waitFor(() =>
      expect(
        server
          .getRequests()
          .some((request) => new URL(request.url).pathname.endsWith('/databases')),
      ).toBe(true),
    );

    expect(localStorage.getItem('selected_workspace_id')).toBe(workspace.id);
    expect(!!screen.queryByRole('button', { name: 'Add database' })).toBe(role === 'ADMIN');
    expect(
      server.getRequests().some((request) => request.url.includes('unavailable-workspace')),
    ).toBe(false);
  },
);

it('selects a newly created workspace when the instance allows member creation', async () => {
  seed();
  seedWorkspaceContent();
  server.use(
    mockApi.get('/instance-settings', 200, {
      ...settings,
      isMemberAllowedToCreateWorkspaces: true,
    }),
    mockApi.post('/workspaces', 201, { ...workspace, currentUserRole: 'WORKSPACE_OWNER' }),
  );

  const view = renderScreen(<ApplicationScreen />);
  await view.user.click((await screen.findAllByRole('button', { name: 'Create workspace' }))[0]);
  await view.user.click(
    within(await screen.findByRole('dialog')).getByRole('button', { name: 'Create workspace' }),
  );

  await waitFor(() => expect(localStorage.getItem('selected_workspace_id')).toBe(workspace.id));
  expect(await screen.findByRole('combobox')).toBeTruthy();
  expect(await screen.findByRole('button', { name: 'Add database' })).toBeTruthy();
});

it('retains retried startup when an older essential request finishes later', async () => {
  seed();
  const oldAccount = mockApi.delayed({
    method: 'get',
    path: '/users/{userId}',
    status: 200,
    body: { ...user, name: 'Obsolete owner' },
  });
  server.use(
    oldAccount.handler,
    mockApi.get('/instance-settings', 500, buildProblem('internal', 500)),
  );

  renderScreen(<ApplicationScreen />);
  await screen.findByRole('button', { name: 'Retry' });

  server.use(
    mockApi.get('/users/{userId}', 200, { ...user, role: 'ADMIN' }),
    mockApi.get('/instance-settings', 200, settings),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
  await screen.findByRole('button', { name: 'Users' });

  const processed = new Promise<void>((resolve) =>
    server.events.on('response:mocked', () => resolve()),
  );
  await act(async () => {
    oldAccount.release();
    await processed;
  });

  expect(screen.getByRole('button', { name: 'Users' })).toBeTruthy();
});

it('updates account capabilities without replacing a newer profile draft after a profile acknowledgement', async () => {
  seed();
  server.use(mockApi.get('/users/{userId}', 200, { ...user, role: 'ADMIN' }));
  const save = mockApi.delayed({
    method: 'put',
    path: '/users/{userId}',
    status: 200,
    body: { ...user, name: 'Accepted member' },
  });
  server.use(save.handler);

  const view = renderScreen(<ApplicationScreen />);
  await view.user.click(await screen.findByRole('button', { name: 'Profile' }));
  await screen.findByDisplayValue('owner@example.com');

  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
    target: { value: 'Submitted profile' },
  });
  await view.user.click(screen.getByRole('button', { name: 'Save changes' }));

  await waitFor(() =>
    expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(true),
  );

  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
    target: { value: 'Newer profile' },
  });
  save.release();

  await screen.findByText('Profile updated successfully');
  expect(screen.queryByRole('button', { name: 'Users' })).toBeNull();
  expect(screen.queryByRole('button', { name: 'Databasus settings' })).toBeNull();
  expect(screen.getByDisplayValue('Newer profile')).toBeTruthy();
});

it('retains an acknowledged workspace-creation policy when profile acceptance changes the account role', async () => {
  seed();
  server.use(
    mockApi.get('/users/{userId}', 200, { ...user, role: 'ADMIN' }),
    mockApi.get('/verification-agents', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
    mockApi.put('/instance-settings', 200, {
      ...settings,
      isMemberAllowedToCreateWorkspaces: true,
    }),
    mockApi.put('/users/{userId}', 200, { ...user, name: 'Member profile' }),
  );

  const view = renderScreen(<ApplicationScreen />);
  await view.user.click(await screen.findByRole('button', { name: 'Databasus settings' }));
  await view.user.click(
    await screen.findByRole('switch', { name: 'Members can create workspaces' }),
  );
  await view.user.click(screen.getByRole('button', { name: 'Save Changes' }));
  await screen.findByText('Settings updated successfully');

  await view.user.click(screen.getByRole('button', { name: 'Profile' }));
  await screen.findByDisplayValue('owner@example.com');
  fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
    target: { value: 'Member profile' },
  });
  await view.user.click(screen.getByRole('button', { name: 'Save changes' }));
  await screen.findByText('Profile updated successfully');

  expect(screen.queryByRole('button', { name: 'Users' })).toBeNull();
  expect(screen.getByRole('button', { name: 'Create workspace' })).toBeTruthy();
});

it.each([false, true])(
  'clears account confirmations and suppresses deletion when session refusal arrives with confirmation submitted=%s',
  async (hasConfirmed) => {
    seed();
    seedWorkspaceContent();
    const ownedWorkspace = { ...workspace, currentUserRole: 'WORKSPACE_OWNER' as const };
    const refusal = mockApi.delayed({
      method: 'get',
      path: '/system/disk-usages',
      status: 401,
      body: buildProblem('unauthenticated', 401),
    });
    server.use(
      refusal.handler,
      mockApi.get('/workspaces', 200, { items: [ownedWorkspace] }),
      mockApi.get('/workspaces/{workspaceId}', 200, ownedWorkspace),
      mockApi.get('/workspace-members', 200, { items: [] }),
      mockApi.get('/audit-events', 200, { items: [] }),
      mockApi.delete('/workspaces/{workspaceId}', 204, undefined),
    );

    renderApp();
    fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Delete workspace' }));
    const confirmation = await screen.findByRole('dialog');

    const deletionCheck = mockApi.delayed({
      method: 'get',
      path: '/databases',
      status: 200,
      body: { items: [] },
    });

    if (hasConfirmed) {
      server.use(deletionCheck.handler);
      fireEvent.click(within(confirmation).getByRole('button', { name: 'Delete Workspace' }));
      await waitFor(() =>
        expect(
          server
            .getRequests()
            .filter((request) => new URL(request.url).pathname.endsWith('/databases')),
        ).toHaveLength(2),
      );
    }

    await act(async () => refusal.release());
    await screen.findByRole('button', { name: 'Sign in' });
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

    if (hasConfirmed) {
      const processed = new Promise<void>((resolve) => {
        const onResponse = () => {
          server.events.removeListener('response:mocked', onResponse);
          resolve();
        };
        server.events.on('response:mocked', onResponse);
      });
      await act(async () => {
        deletionCheck.release();
        await processed;
      });
    }

    expect(server.getRequests().filter((request) => request.method === 'DELETE')).toHaveLength(0);
    expect(screen.queryByText('Workspace deleted successfully')).toBeNull();
  },
);

it('keeps account navigation usable when production release information fails', async () => {
  seed();
  localStorage.removeItem('githubLatestVersion');
  server.use(
    http.get(
      'https://api.github.com/repos/databasus/databasus/releases/latest',
      () => new HttpResponse(null, { status: 503 }),
    ),
  );

  const failedRelease = new Promise<void>((resolve) => {
    const onResponse = ({ response, request }: { response: Response; request: Request }) => {
      if (!request.url.endsWith('/releases/latest')) return;
      expect(response.status).toBe(503);
      server.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    server.events.on('response:mocked', onResponse);
  });

  const view = renderScreen(<ApplicationScreen />);
  await failedRelease;
  await view.user.click(await screen.findByRole('button', { name: 'Profile' }));
  expect(await screen.findByDisplayValue('owner@example.com')).toBeTruthy();
  expect(document.querySelector('.ant-spin-spinning')).toBeNull();
});

describe('New-version link', () => {
  const NEW_VERSION_EXPLANATION = 'New version available';

  const serveLatestRelease = (tagName: string) => {
    seed();
    localStorage.removeItem('githubLatestVersion');
    server.use(
      http.get('https://api.github.com/repos/databasus/databasus/releases/latest', () =>
        HttpResponse.json({ tag_name: tagName }),
      ),
    );
  };

  const waitForReleaseAnswer = () =>
    new Promise<void>((resolve) => {
      const onResponse = ({ request }: { request: Request }) => {
        if (!request.url.endsWith('/releases/latest')) return;
        server.events.removeListener('response:mocked', onResponse);
        resolve();
      };
      server.events.on('response:mocked', onResponse);
    });

  it('explains the new version link when a newer release exists', async () => {
    serveLatestRelease('v2.0.0');

    const { user } = renderScreen(<ApplicationScreen />);
    const link = await screen.findByRole('link', { name: NEW_VERSION_EXPLANATION });

    expect(link.getAttribute('href')).toBe(
      'https://github.com/databasus/databasus/releases/latest',
    );
    expect(link.textContent).toBe('');

    await user.hover(link);
    expect((await screen.findByRole('tooltip')).textContent).toBe(NEW_VERSION_EXPLANATION);
  });

  it('explains the new version link on keyboard focus', async () => {
    serveLatestRelease('v2.0.0');

    renderScreen(<ApplicationScreen />);
    const link = await screen.findByRole('link', { name: NEW_VERSION_EXPLANATION });

    fireEvent.focus(link);

    expect((await screen.findByRole('tooltip')).textContent).toBe(NEW_VERSION_EXPLANATION);
  });

  it('hides the new version link when the running version is the latest', async () => {
    serveLatestRelease('v1.0.0');
    const releaseAnswered = waitForReleaseAnswer();

    renderScreen(<ApplicationScreen />);
    await releaseAnswered;
    await screen.findByRole('button', { name: 'Profile' });

    expect(screen.queryByRole('link', { name: NEW_VERSION_EXPLANATION })).toBeNull();
  });

  it('hides the new version link when the release lookup fails', async () => {
    seed();
    localStorage.removeItem('githubLatestVersion');
    server.use(
      http.get(
        'https://api.github.com/repos/databasus/databasus/releases/latest',
        () => new HttpResponse(null, { status: 503 }),
      ),
    );
    const releaseAnswered = waitForReleaseAnswer();

    renderScreen(<ApplicationScreen />);
    await releaseAnswered;
    await screen.findByRole('button', { name: 'Profile' });

    expect(screen.queryByRole('link', { name: NEW_VERSION_EXPLANATION })).toBeNull();
  });
});

it('accepts only the active startup after root StrictMode repeats setup', async () => {
  seed();
  const firstSetup = mockApi.pending('get', '/users/{userId}');
  let profileReads = 0;
  server.use(
    http.get(`*${API_BASE_PATH}/users/:userId`, () =>
      ++profileReads === 1
        ? firstSetup.response
        : HttpResponse.json({ ...user, name: 'Replay owner' }),
    ),
  );

  const view = renderScreen(<ApplicationScreen />, { isStrictMode: true });
  await view.user.click(await screen.findByRole('button', { name: 'Profile' }));
  await screen.findByDisplayValue('Replay owner');

  const processed = new Promise<void>((resolve) => {
    const onResponse = () => {
      server.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    server.events.on('response:mocked', onResponse);
  });

  await act(async () => {
    firstSetup.respond(HttpResponse.json({ ...user, name: 'Obsolete owner', role: 'ADMIN' }));
    await processed;
  });

  expect(screen.queryByDisplayValue('Obsolete owner')).toBeNull();
  expect(screen.getByDisplayValue('Replay owner')).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Users' })).toBeNull();
});
