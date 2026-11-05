// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { MailServer } from '@/pages/instance-settings/ui/MailServer';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

describe('MailServer', () => {
  it('disables the test email and names the missing server when the instance has no mail server', () => {
    renderScreen(<MailServer isEmailConfigured={false} />);

    const sendTestEmail = screen.getByRole<HTMLButtonElement>('button', {
      name: 'Send test email',
    });

    expect(sendTestEmail.disabled).toBe(true);
    expect(screen.getByText('not configured')).toBeTruthy();
  });

  it('enables the test email and shows the server as configured when the instance has one', () => {
    renderScreen(<MailServer isEmailConfigured />);

    const sendTestEmail = screen.getByRole<HTMLButtonElement>('button', {
      name: 'Send test email',
    });

    expect(sendTestEmail.disabled).toBe(false);
    expect(screen.getByText('configured')).toBeTruthy();
  });

  it('shows that the instance has no mail server when the test email is refused for it', async () => {
    mockServer.use(mockApi.post('/test-emails', 409, buildProblem('email_not_configured', 409)));
    renderScreen(<MailServer isEmailConfigured />);

    fireEvent.click(screen.getByRole('button', { name: 'Send test email' }));

    expect(await screen.findByText('The instance has no mail server configured.')).toBeTruthy();
  });

  it('shows the mail server reply when it does not accept the test email', async () => {
    mockServer.use(
      mockApi.post('/test-emails', 201, {
        recipientEmail: 'admin@example.com',
        isDelivered: false,
        failureDetail: '550 mailbox unavailable',
      }),
    );
    renderScreen(<MailServer isEmailConfigured />);

    fireEvent.click(screen.getByRole('button', { name: 'Send test email' }));

    expect(await screen.findByText('550 mailbox unavailable')).toBeTruthy();
    expect(screen.getByText('admin@example.com')).toBeTruthy();
  });
});

it.each([false, true])(
  'publishes no mail feedback after unmount with refusal=%s',
  async (isRefused) => {
    const pending = isRefused
      ? mockApi.delayed({
          method: 'post',
          path: '/test-emails',
          status: 403,
          body: buildProblem('permission_denied', 403),
        })
      : mockApi.delayed({
          method: 'post',
          path: '/test-emails',
          status: 201,
          body: { recipientEmail: 'admin@example.com', isDelivered: true },
        });
    mockServer.use(pending.handler);
    const view = renderScreen(<MailServer isEmailConfigured />);

    await view.user.click(screen.getByRole('button', { name: 'Send test email' }));
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(1));

    view.rerender(<div>Mail settings closed</div>);
    const processed = new Promise<void>((resolve) => {
      const onResponse = () => {
        mockServer.events.removeListener('response:mocked', onResponse);
        resolve();
      };
      mockServer.events.on('response:mocked', onResponse);
    });
    await act(async () => {
      pending.release();
      await processed;
    });

    expect(document.querySelector('.ant-message-notice')).toBeNull();
    expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
  },
);

it('reports delivery when the configured mail server accepts the test email', async () => {
  mockServer.use(
    mockApi.post('/test-emails', 201, { recipientEmail: 'admin@example.com', isDelivered: true }),
  );
  const { user } = renderScreen(<MailServer isEmailConfigured />);
  await user.click(screen.getByRole('button', { name: 'Send test email' }));
  expect(await screen.findByText('Test email sent to admin@example.com')).toBeTruthy();
});
