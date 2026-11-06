// @vitest-environment happy-dom
import { act, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useRef } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { WorkspaceAuditLogsComponent } from '@/pages/workspace-settings/ui/WorkspaceAuditLogsComponent';
import { API_BASE_PATH } from '@/shared/api';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();

afterEach(() => {
  vi.restoreAllMocks();
});
const event = (message: string) => ({
  id: message,
  message,
  createdAt: '2026-10-01T10:00:00Z',
  author: { userId: 'owner', name: '<owner>{{name}}', email: 'owner@example.com' },
  workspace: { id: 'workspace-a', name: 'Team' },
});
function AuditScreen({
  workspaceId,
  onUnavailable,
}: {
  workspaceId: string;
  onUnavailable?: (error: unknown) => void;
}) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  return (
    <div ref={scrollContainerRef}>
      <WorkspaceAuditLogsComponent
        workspaceId={workspaceId}
        onUnavailable={onUnavailable}
        scrollContainerRef={scrollContainerRef}
      />
    </div>
  );
}

describe('Workspace audit', () => {
  it('offers retry with the same workspace filter and ten-event page size after failure', async () => {
    server.use(mockApi.get('/audit-events', 500, buildProblem('internal', 500)));
    const view = renderScreen(<AuditScreen workspaceId="workspace-a" />);
    const retry = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.queryByText('No audit logs yet')).toBeNull();

    server.use(mockApi.get('/audit-events', 200, { items: [event('<literal>{{message}}')] }));

    await view.user.click(retry);

    expect(await screen.findByText('<literal>{{message}}')).toBeTruthy();
    expect(screen.getByText('<owner>{{name}} (owner@example.com)')).toBeTruthy();

    for (const request of server.getRequests()) {
      expect(new URL(request.url).searchParams.get('workspaceId')).toBe('workspace-a');
      expect(new URL(request.url).searchParams.get('limit')).toBe('10');
    }

    expect(screen.queryByRole('columnheader', { name: 'Workspace' })).toBeNull();
  });

  it.each([200, 404, 500])(
    'ignores an old workspace response with status %s after scope changes',
    async (status) => {
      const fetchObservation = vi.spyOn(globalThis, 'fetch');
      const onUnavailable = vi.fn();
      const pending = mockApi.pending('get', '/audit-events');
      server.use(pending.handler);
      const view = renderScreen(
        <AuditScreen workspaceId="workspace-a" onUnavailable={onUnavailable} />,
        { isStrictMode: true },
      );

      await waitFor(() => expect(server.getRequests().length).toBeGreaterThan(0));

      const pendingFetches = fetchObservation.mock.results.map((request) => request.value);
      server.use(
        http.get(`*${API_BASE_PATH}/audit-events`, () =>
          HttpResponse.json({ items: [event('Current workspace')] }),
        ),
      );

      view.rerender(<AuditScreen workspaceId="workspace-b" onUnavailable={onUnavailable} />);
      await screen.findByText('Current workspace');

      await act(async () => {
        pending.respond(
          HttpResponse.json(
            status === 200
              ? { items: [event('Old workspace')] }
              : buildProblem(status === 404 ? 'not_found' : 'internal', status),
            { status },
          ),
        );
        await Promise.all(pendingFetches);
      });

      expect(onUnavailable).not.toHaveBeenCalled();
      expect(screen.queryByText('Old workspace')).toBeNull();
      expect(screen.getByText('Current workspace')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    },
  );

  it.each([320, 1200])(
    'renders audit columns and cards when viewport width is %s',
    async (width) => {
      const originalWidth = window.innerWidth;
      Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
      vi.spyOn(navigator, 'language', 'get').mockReturnValue('en-US');
      server.use(mockApi.get('/audit-events', 200, { items: [event('Workspace event')] }));

      try {
        renderScreen(<AuditScreen workspaceId="workspace-a" />);
        await screen.findByText('Workspace event');

        expect(screen.getByText('<owner>{{name}} (owner@example.com)')).toBeTruthy();
        expect(screen.getByText((text) => /^Oct 01, 2026 /.test(text))).toBeTruthy();

        if (width === 320) {
          expect(screen.getAllByRole('article')).toHaveLength(1);
          expect(screen.queryByRole('table')).toBeNull();
        } else {
          expect(screen.getByRole('table')).toBeTruthy();
          expect(screen.queryByRole('article')).toBeNull();
          expect(screen.queryByRole('columnheader', { name: 'Workspace' })).toBeNull();
        }
      } finally {
        Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalWidth });
      }
    },
  );

  it.each([320, 1200])(
    'shows workspace-specific empty copy when viewport width is %s',
    async (width) => {
      const originalWidth = window.innerWidth;
      Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
      server.use(mockApi.get('/audit-events', 200, { items: [] }));

      try {
        renderScreen(<AuditScreen workspaceId="workspace-a" />);

        expect(await screen.findByText('No audit logs found for this workspace.')).toBeTruthy();
        expect(screen.queryByRole('table')).toBeNull();
        expect(screen.queryByRole('article')).toBeNull();
      } finally {
        Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalWidth });
      }
    },
  );
});
