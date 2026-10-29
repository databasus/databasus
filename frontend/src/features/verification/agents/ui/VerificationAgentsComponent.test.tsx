// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
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
          maxRamGb: 0,
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
});
