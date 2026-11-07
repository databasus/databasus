import type { components as ApiComponents } from '../src/shared/api/schema.gen';
import { apiResponse, type HttpFixture } from './HttpFixture';

export const workspace: ApiComponents['schemas']['Workspace'] = {
  id: 'workspace-id',
  name: 'Test workspace',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T00:00:00Z',
};

export const smtpNotifier: ApiComponents['schemas']['Notifier'] = {
  id: 'smtp-id',
  workspaceId: 'workspace-id',
  name: 'Operations mail',
  lastSendError: null,
  settings: {
    type: 'EMAIL',
    targetEmail: 'ops@example.com',
    smtpHost: 'smtp.example.com',
    smtpPort: 587,
    security: 'STARTTLS',
    isInsecureSkipVerify: false,
  },
};
export const applicationFixtures = (): HttpFixture[] => [
  apiResponse({ method: 'get', path: '/sign-in-options', status: 200, body: { hasAnyUser: true } }),
  apiResponse({
    method: 'get',
    path: '/system/disk-usages',
    status: 200,
    body: {
      usedSpaceBytes: 1,
      totalSpaceBytes: 10,
      freeSpaceBytes: 9,
    },
  }),
  apiResponse({
    method: 'get',
    path: '/users/{userId}',
    status: 200,
    body: {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Owner',
      role: 'ADMIN',
      status: 'ACTIVE',
      createdAt: '2026-10-01T00:00:00Z',
    },
  }),
  apiResponse({
    method: 'get',
    path: '/workspaces',
    status: 200,
    body: {
      items: [workspace],
    },
  }),
  apiResponse({
    method: 'get',
    path: '/instance-settings',
    status: 200,
    body: {
      isAllowExternalRegistrations: false,
      isAllowMemberInvitations: false,
      isMemberAllowedToCreateWorkspaces: false,
      isEmailConfigured: true,
      isTwoFactorAuthRequired: false,
    },
  }),
  apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [] } }),
  apiResponse({ method: 'get', path: '/backup-configs', status: 200, body: { items: [] } }),
  apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [] } }),
  apiResponse({
    method: 'get',
    path: '/storages/{storageId}/backup-usage',
    status: 200,
    body: { totalUsageMb: 0 },
  }),
  apiResponse({ method: 'get', path: '/notifiers', status: 200, body: { items: [smtpNotifier] } }),
  apiResponse({ method: 'get', path: '/notifiers/{notifierId}', status: 200, body: smtpNotifier }),
  apiResponse({
    method: 'post',
    path: '/notifier-tests',
    status: 200,
    body: { isDelivered: true },
  }),
  apiResponse({ method: 'put', path: '/notifiers/{notifierId}', status: 200, body: smtpNotifier }),
  apiResponse({ method: 'delete', path: '/notifiers/{notifierId}', status: 204, body: undefined }),
  {
    method: 'GET',
    path: 'https://api.github.com/repos/databasus/databasus',
    status: 200,
    body: { stargazers_count: 1 },
  },
  {
    method: 'GET',
    path: 'https://api.github.com/repos/databasus/databasus/releases/latest',
    status: 200,
    body: { tag_name: 'v0.0.0' },
  },
];
