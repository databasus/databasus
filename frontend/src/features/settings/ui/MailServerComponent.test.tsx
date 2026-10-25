// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { MailServerComponent } from './MailServerComponent';

const mockServer = setUpMockApi();

describe('MailServerComponent', () => {
  it('shows that the instance has no mail server when the test email is refused for it', async () => {
    mockServer.use(mockApi.post('/test-emails', 409, buildProblem('email_not_configured', 409)));
    renderScreen(<MailServerComponent isEmailConfigured />);

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
    renderScreen(<MailServerComponent isEmailConfigured />);

    fireEvent.click(screen.getByRole('button', { name: 'Send test email' }));

    expect(await screen.findByText('550 mailbox unavailable')).toBeTruthy();
    expect(screen.getByText('admin@example.com')).toBeTruthy();
  });
});
