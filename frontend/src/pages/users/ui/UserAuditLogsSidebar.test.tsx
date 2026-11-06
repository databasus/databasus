// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { type User } from '@/entities/users';
import { UserAuditLogsSidebar } from '@/pages/users/ui/UserAuditLogsSidebar';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const firstUser: User = {
  id: 'first-user',
  name: 'First',
  email: 'first@example.com',
  role: 'MEMBER',
  status: 'ACTIVE',
  createdAt: '2026-10-01T00:00:00Z',
};
const secondUser: User = { ...firstUser, id: 'second-user', name: 'Second' };
const auditEvent = {
  id: 'event-id',
  message: 'Second account event',
  createdAt: '2026-10-01T10:00:00Z',
  author: null,
  workspace: null,
};

describe('UserAuditLogsSidebar', () => {
  it('retains the current author and cursor after a previous selected user responds', async () => {
    const previousUserEvents = mockApi.delayed({
      method: 'get',
      path: '/audit-events',
      status: 200,
      body: { items: [{ ...auditEvent, message: 'Obsolete account event' }] },
    });
    server.use(previousUserEvents.handler);

    const view = renderScreen(<UserAuditLogsSidebar user={firstUser} />);
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    server.use(
      mockApi.get('/audit-events', 200, { items: [auditEvent], nextCursor: 'second-cursor' }),
    );
    view.rerender(<UserAuditLogsSidebar user={secondUser} />);
    await screen.findByText(auditEvent.message);

    const processed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );
    await act(async () => {
      previousUserEvents.release();
      await processed;
    });

    expect(screen.queryByText('Obsolete account event')).toBeNull();
    fireEvent.scroll(view.container.querySelector('.overflow-y-auto')!);
    await waitFor(() => expect(server.getRequests()).toHaveLength(3));

    expect(
      server.getRequests().map((request) => new URL(request.url).searchParams.get('authorUserId')),
    ).toEqual(['first-user', 'second-user', 'second-user']);
    const nextPageUrl = new URL(server.getRequests()[2].url);
    expect(nextPageUrl.searchParams.get('after')).toBe('second-cursor');
    expect(nextPageUrl.searchParams.get('limit')).toBe('50');
  });

  it('loads the events again when a failed load is retried', async () => {
    server.use(mockApi.get('/audit-events', 500, buildProblem('internal', 500)));
    const view = renderScreen(<UserAuditLogsSidebar user={firstUser} />);
    const retry = await screen.findByRole('button', { name: 'Retry' });

    server.use(mockApi.get('/audit-events', 200, { items: [auditEvent] }));
    await view.user.click(retry);

    expect(await screen.findByText(auditEvent.message)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });
});
