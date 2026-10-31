// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import type { Notifier } from '../../../entities/notifiers';
import { API_BASE_PATH } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { NotifiersComponent } from './NotifiersComponent';

const server = setUpMockApi();
const notifier: Notifier = {
  id: 'created-id',
  name: 'Created SMTP',
  workspaceId: 'workspace-id',
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

describe('Notifier creation persistence', () => {
  it('preserves the newer draft and created notifier when the list refreshes and editing is cancelled', async () => {
    const creation = mockApi.delayed({
      method: 'post',
      path: '/notifiers',
      status: 201,
      body: notifier,
    });
    let listedNotifiers: Notifier[] = [];
    server.use(
      creation.handler,
      mockApi.post('/notifier-tests', 200, { isDelivered: true }),
      http.get(`*${API_BASE_PATH}/notifiers`, () => HttpResponse.json({ items: listedNotifiers })),
    );
    const view = renderScreen(
      <NotifiersComponent
        workspace={{
          id: 'workspace-id',
          name: 'Workspace',
          currentUserRole: 'WORKSPACE_OWNER',
          createdAt: '2026-10-01T00:00:00Z',
        }}
        isCanManageNotifiers
      />,
      { locale: 'en' },
    );

    fireEvent.click(await screen.findByRole('button', { name: 'Add notifier' }));
    const dialog = within(screen.getByRole('dialog'));

    await view.user.click(dialog.getByRole('combobox', { name: 'Type' }));
    await view.user.click(screen.getByRole('option', { name: 'Email' }));
    fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), {
      target: { value: notifier.name },
    });
    fireEvent.change(dialog.getByLabelText('Target email'), {
      target: { value: 'ops@example.com' },
    });
    fireEvent.change(dialog.getByLabelText('SMTP host'), { target: { value: 'smtp.example.com' } });
    fireEvent.change(dialog.getByLabelText('SMTP port'), { target: { value: '587' } });
    fireEvent.click(dialog.getByRole('button', { name: /Send test notification/ }));
    fireEvent.click(await dialog.findByRole('button', { name: /Save/ }));

    await waitFor(() =>
      expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(2),
    );

    const nameInput = dialog.getByRole('textbox', { name: 'Name' });

    fireEvent.change(nameInput, { target: { value: 'Newer local name' } });
    listedNotifiers = [notifier];

    creation.release();
    await screen.findByRole('button', { name: notifier.name });

    expect(dialog.getByDisplayValue('Newer local name')).toBe(nameInput);
    expect(screen.getByRole('dialog')).toBeTruthy();

    fireEvent.click(dialog.getByRole('button', { name: 'Close' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(screen.getByRole('button', { name: notifier.name })).toBeTruthy();
    expect(
      server
        .getRequests()
        .filter(
          (request) =>
            request.method === 'POST' && new URL(request.url).pathname.endsWith('/notifiers'),
        ),
    ).toHaveLength(1);
  });
});
