import { applicationFixtures } from './applicationFixtures';
import { apiResponse } from './HttpFixture';

const account = {
  id: 'member-id',
  name: 'Member <literal>',
  email: 'member@example.com',
  role: 'MEMBER' as const,
  status: 'ACTIVE' as const,
  createdAt: '2026-10-01T00:00:00Z',
};
export const accountFixtures = () =>
  applicationFixtures().concat([
    apiResponse({
      method: 'post',
      path: '/users/{userId}/deactivate',
      status: 403,
      body: {
        type: 'urn:databasus:problem:human_verification_failed',
        title: 'Refused',
        detail: 'Synthetic refusal',
        status: 403,
        code: 'human_verification_failed',
      },
    }),
    apiResponse({ method: 'get', path: '/users', status: 200, body: { items: [account] } }),
    apiResponse({ method: 'get', path: '/verification-agents', status: 200, body: { items: [] } }),
    apiResponse({
      method: 'get',
      path: '/audit-events',
      status: 200,
      body: {
        items: [
          {
            id: 'audit-id',
            message: '<literal> {{audit}}',
            createdAt: '2026-10-01T00:00:00Z',
            author: null,
            workspace: null,
          },
        ],
      },
    }),
    apiResponse({
      method: 'put',
      path: '/users/{userId}',
      status: 403,
      body: {
        type: 'urn:databasus:problem:human_verification_failed',
        title: 'Refused',
        detail: 'Synthetic refusal',
        status: 403,
        code: 'human_verification_failed',
      },
    }),
    apiResponse({
      method: 'put',
      path: '/instance-settings',
      status: 403,
      body: {
        type: 'urn:databasus:problem:human_verification_failed',
        title: 'Refused',
        detail: 'Synthetic refusal',
        status: 403,
        code: 'human_verification_failed',
      },
    }),
    apiResponse({
      method: 'post',
      path: '/sessions',
      status: 403,
      body: {
        type: 'urn:databasus:problem:human_verification_failed',
        title: 'Refused',
        detail: 'Synthetic refusal',
        status: 403,
        code: 'human_verification_failed',
      },
    }),
  ]);
