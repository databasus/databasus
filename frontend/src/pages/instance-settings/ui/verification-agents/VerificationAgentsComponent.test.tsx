// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { activateSession, getSessionSnapshot } from '@/shared/auth';

import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { VerificationAgentsComponent } from './VerificationAgentsComponent';

const mockServer = setUpMockApi();
describe('VerificationAgentsComponent', () => {
  it('reveals the token once after an agent is created and clears it after confirmation', async () => {
    mockServer.use(
      mockApi.get('/verification-agents', 200, { items: [] }),
      mockApi.post('/verification-agents', 201, {
        agent: {
          id: 'agent-id',
          name: 'My agent',
          maxCpu: 0,
          maxRamMb: 0,
          maxDiskGb: 0,
          maxConcurrentJobs: 0,
          createdAt: '2026-10-27T10:00:00Z',
        },
        token: 'issued-once-token',
      }),
    );

    renderScreen(<VerificationAgentsComponent />);
    fireEvent.click(await screen.findByRole('button', { name: /Create/ }));
    const dialog = await screen.findByRole('dialog');

    fireEvent.change(within(dialog).getByPlaceholderText('Agent name'), {
      target: { value: 'My agent' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }));

    expect(await screen.findByText('issued-once-token')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: "I've saved the token" }));
    await waitFor(() => expect(screen.queryByText('issued-once-token')).toBeNull());
    expect(screen.getByText('My agent')).toBeTruthy();
  });
  it.each([
    { memory: 512, label: '512 MB RAM' },
    { memory: 4096, label: '4 GB RAM' },
  ])('shows $label when the agent reports $memory megabytes', async ({ memory, label }) => {
    mockServer.use(
      mockApi.get('/verification-agents', 200, {
        items: [
          {
            id: 'agent-id',
            name: 'My agent',
            maxCpu: 0,
            maxRamMb: memory,
            maxDiskGb: 0,
            maxConcurrentJobs: 0,
            createdAt: '2026-10-27T10:00:00Z',
          },
        ],
      }),
    );

    renderScreen(<VerificationAgentsComponent />);
    expect(await screen.findByText(label)).toBeTruthy();
  });
});

it.each([
  { operation: 'delete', isRefused: false },
  { operation: 'delete', isRefused: true },
  { operation: 'poll', isRefused: false },
  { operation: 'poll', isRefused: true },
] as const)(
  'suppresses old agent completion after session replacement with $operation refusal=$isRefused',
  async ({ operation, isRefused }) => {
    activateSession({ token: 'old-session', userId: 'old-owner' });
    const agent = {
      id: 'agent-id',
      name: 'Original agent',
      maxCpu: 0,
      maxRamMb: 0,
      maxDiskGb: 0,
      maxConcurrentJobs: 0,
      createdAt: '2026-10-27T10:00:00Z',
    };

    mockServer.use(mockApi.get('/verification-agents', 200, { items: [agent] }));
    vi.spyOn(window, 'setInterval');
    const view = renderScreen(<VerificationAgentsComponent />);
    await screen.findByText('Original agent');
    const pending =
      operation === 'delete'
        ? mockApi.pending('delete', '/verification-agents/{agentId}')
        : mockApi.pending('get', '/verification-agents');
    mockServer.use(pending.handler);

    if (operation === 'delete') {
      await view.user.click(screen.getByRole('button', { name: 'delete' }));
      await view.user.click(await screen.findByRole('button', { name: 'Delete' }));
    } else {
      const tick = vi.mocked(window.setInterval).mock.calls.find((call) => call[1] === 15_000)?.[0];
      if (typeof tick !== 'function') throw new Error('Agent polling is not installed');
      act(() => tick());
    }
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(2));

    act(() => activateSession({ token: 'new-session', userId: 'new-owner' }));
    const provider = document.querySelector('.ant-app');
    if (operation === 'delete') view.rerender(<div>New account</div>);

    const processed = new Promise<void>((resolve) => {
      const onResponse = () => {
        mockServer.events.removeListener('response:mocked', onResponse);
        resolve();
      };
      mockServer.events.on('response:mocked', onResponse);
    });

    await act(async () => {
      pending.respond(
        isRefused
          ? HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 })
          : operation === 'delete'
            ? new HttpResponse(null, { status: 204 })
            : HttpResponse.json({ items: [{ ...agent, name: 'Obsolete agent' }] }),
      );
      await processed;
    });

    expect(document.querySelector('.ant-app')).toBe(provider);
    expect(document.querySelector('.ant-message-notice')).toBeNull();
    expect(screen.queryByText('Obsolete agent')).toBeNull();
    expect(getSessionSnapshot().token).toBe('new-session');
  },
);
