// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { InstanceAuditLogs } from '@/pages/instance-settings/ui/InstanceAuditLogs';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const buildAuditEvent = (message: string): ApiComponents['schemas']['AuditEvent'] => ({
  id: `${message}-id`,
  message,
  createdAt: '2026-10-01T10:00:00Z',
  author: { userId: 'admin-id', email: 'admin@example.com', name: 'Admin' },
  workspace: { id: 'workspace-id', name: null },
});

describe('InstanceAuditLogs', () => {
  it('opens the log in a sidebar that loads the next page by the cursor when scrolled down', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/audit-events`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        const auditEventPage: ApiComponents['schemas']['AuditEventPage'] =
          cursor === null
            ? {
                items: [buildAuditEvent('Workspace created: first')],
                nextCursor: 'first-page-cursor',
              }
            : { items: [buildAuditEvent('Workspace created: second')] };

        return HttpResponse.json(auditEventPage);
      }),
    );
    const { user } = renderScreen(<InstanceAuditLogs />);

    expect(requestedCursors).toEqual([]);

    await user.click(screen.getByRole('button', { name: 'Open' }));
    const drawer = await screen.findByRole('dialog', { name: 'Audit Logs' });
    await within(drawer).findByText('Workspace created: first');
    const scrollContainer = within(drawer)
      .getByText('Workspace created: first')
      .closest('.overflow-y-auto') as Element;

    fireEvent.scroll(scrollContainer);
    expect(await screen.findByText('Workspace created: second')).toBeTruthy();
    expect(await screen.findByText('All logs loaded')).toBeTruthy();
    fireEvent.scroll(scrollContainer);

    await waitFor(() => expect(requestedCursors).toEqual([null, 'first-page-cursor']));
    expect(screen.getAllByText('workspace-id')).toHaveLength(2);
  });
});
