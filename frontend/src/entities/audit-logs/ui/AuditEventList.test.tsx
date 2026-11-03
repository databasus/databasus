// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import dayjs from 'dayjs';
import { describe, expect, it, vi } from 'vitest';

import { renderScreen } from '@/shared/lib/test/renderScreen';
import { getUserShortTimeFormat, getUserTimeFormat } from '@/shared/time';

import type { AuditEvent } from '../model/AuditEvent';
import { AuditEventList } from './AuditEventList';

const event: AuditEvent = {
  id: 'event-id',
  message: '<script>alert(1)</script> {{message}}',
  createdAt: '2026-10-01T10:12:00Z',
  author: { userId: 'author', name: '<Owner>', email: 'owner@example.com' },
  workspace: { id: 'deleted-workspace', name: '{{workspace}} <primary>' },
};

describe('AuditEventList', () => {
  it.each(['table', 'cards'] as const)(
    'displays literal event and captured labels when the %s view includes author and workspace',
    (mode) => {
      const { container } = renderScreen(
        <AuditEventList
          events={[event]}
          isLoading={false}
          isLoadingMore={false}
          hasMore={false}
          view={{
            mode,
            shouldShowAuthor: true,
            shouldShowWorkspace: true,
            timeFormat: 'full',
            emptyKey: 'auditLogs.empty',
          }}
        />,
      );

      expect(screen.getByText(event.message)).toBeTruthy();
      expect(screen.getByText('<Owner> (owner@example.com)')).toBeTruthy();
      expect(screen.getByText('{{workspace}} <primary>')).toBeTruthy();
      expect(
        screen.getByText(
          new RegExp(
            dayjs(event.createdAt)
              .format(getUserTimeFormat().format)
              .replace(/[.*+?^${}()|[\]\\]/g, '\\$&'),
          ),
        ),
      ).toBeTruthy();
      expect(container.querySelector('script')).toBeNull();
      if (mode === 'table')
        expect(screen.getAllByRole('columnheader').map((column) => column.textContent)).toEqual([
          'User',
          'Message',
          'Workspace',
          'Created',
        ]);
    },
  );

  it('hides authors and uses the selected short timestamp when the user view omits that column', () => {
    renderScreen(
      <AuditEventList
        events={[{ ...event, workspace: { id: 'deleted-workspace', name: null } }]}
        isLoading={false}
        isLoadingMore={false}
        hasMore
        view={{
          mode: 'table',
          shouldShowAuthor: false,
          shouldShowWorkspace: true,
          timeFormat: 'short',
          emptyKey: 'users.auditLogs.empty',
        }}
      />,
    );

    expect(screen.getAllByRole('columnheader').map((column) => column.textContent)).toEqual([
      'Message',
      'Workspace',
      'Created',
    ]);
    expect(screen.queryByText('<Owner> (owner@example.com)')).toBeNull();
    expect(screen.getByText('deleted-workspace')).toBeTruthy();
    expect(
      screen.getByText((text) =>
        text.startsWith(dayjs(event.createdAt).format(getUserShortTimeFormat().format)),
      ),
    ).toBeTruthy();
  });

  it.each([
    { emptyKey: 'auditLogs.empty' as const, text: 'No audit logs found.' },
    { emptyKey: 'users.auditLogs.empty' as const, text: 'No audit logs found for this user.' },
  ])(
    'preserves the view-specific empty message when there are no events for $emptyKey',
    ({ emptyKey, text }) => {
      renderScreen(
        <AuditEventList
          events={[]}
          isLoading={false}
          isLoadingMore={false}
          hasMore={false}
          view={{
            mode: 'table',
            shouldShowAuthor: false,
            shouldShowWorkspace: true,
            timeFormat: 'full',
            emptyKey,
          }}
        />,
      );
      expect(screen.getByText(text)).toBeTruthy();
    },
  );

  it.each([
    { state: 'an empty list', events: [] },
    { state: 'loaded events', events: [event] },
  ])('shows the failure and retries on click with $state', async ({ events }) => {
    const onRetry = vi.fn();
    const { user } = renderScreen(
      <AuditEventList
        events={events}
        failure={{ error: new Error('offline'), onRetry }}
        isLoading={false}
        isLoadingMore={false}
        hasMore={false}
        view={{
          mode: 'table',
          shouldShowAuthor: false,
          shouldShowWorkspace: true,
          timeFormat: 'full',
          emptyKey: 'auditLogs.empty',
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Retry' }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
