// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useRef } from 'react';
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

// The settings page scrolls the log inside its own container and hands that container in.
function ScrollingSettingsPage() {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  return (
    <div ref={scrollContainerRef} className="settings-scroll-container">
      <InstanceAuditLogs scrollContainerRef={scrollContainerRef} />
    </div>
  );
}

describe('InstanceAuditLogs', () => {
  it('loads the next page by the cursor and stops after the last page when the log is scrolled down', async () => {
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
    const { container } = renderScreen(<ScrollingSettingsPage />);
    await screen.findByText('Workspace created: first');
    const scrollContainer = container.querySelector('.settings-scroll-container') as Element;

    fireEvent.scroll(scrollContainer);
    expect(await screen.findByText('Workspace created: second')).toBeTruthy();
    expect(await screen.findByText('All logs loaded')).toBeTruthy();
    fireEvent.scroll(scrollContainer);

    await waitFor(() => expect(requestedCursors).toEqual([null, 'first-page-cursor']));
    expect(screen.getAllByText('workspace-id')).toHaveLength(2);
  });
});
