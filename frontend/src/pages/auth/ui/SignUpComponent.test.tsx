// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { SignUpComponent } from '@/pages/auth/ui/SignUpComponent';
import { getSessionSnapshot } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { LanguageSelectorComponent } from '@/shared/ui/LanguageSelectorComponent';

const server = setUpMockApi();
const submitRegistration = () => {
  fireEvent.change(screen.getByRole('textbox', { name: 'Your name' }), {
    target: { value: 'New owner' },
  });
  fireEvent.change(screen.getByRole('textbox', { name: 'Your email' }), {
    target: { value: 'owner@example.com' },
  });
  fireEvent.change(screen.getByLabelText('Password', { selector: 'input' }), {
    target: { value: 'registration-password' },
  });
  fireEvent.change(screen.getByLabelText('Confirm password', { selector: 'input' }), {
    target: { value: 'registration-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Sign up' }));
};

describe('SignUpComponent', () => {
  it('activates the returned session without duplicate creation when registration completes', async () => {
    const registration = mockApi.delayed({
      method: 'post',
      path: '/users',
      status: 201,
      body: { token: 'new-session', userId: 'new-owner', email: 'owner@example.com' },
    });
    server.use(registration.handler);

    renderScreen(<SignUpComponent isClaimingInstance />);
    submitRegistration();
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    fireEvent.click(screen.getByRole('button', { name: /Sign up/ }));
    expect(server.getRequests()).toHaveLength(1);

    registration.release();
    await waitFor(() => expect(getSessionSnapshot().token).toBe('new-session'));
    expect(await server.getRequests()[0].json()).toMatchObject({
      name: 'New owner',
      email: 'owner@example.com',
      password: 'registration-password',
    });
  });

  it('preserves input and translates persistent errors when the language changes after refusal', async () => {
    server.use(mockApi.post('/users', 403, buildProblem('permission_denied', 403)));
    const { user } = renderScreen(
      <>
        <SignUpComponent />
        <LanguageSelectorComponent />
      </>,
    );

    submitRegistration();
    await screen.findByText('You do not have permission to do this.');
    await user.click(screen.getByRole('button', { name: /Language/ }));
    await user.click(await screen.findByRole('menuitem', { name: /Español/ }));

    expect(await screen.findByText(DICTIONARIES.es.errors.permissionDenied)).toBeTruthy();
    expect(screen.getByDisplayValue('New owner')).toBeTruthy();
    expect(screen.getByDisplayValue('owner@example.com')).toBeTruthy();
  });

  it('validates beside the associated field without sending a request when the name is empty', async () => {
    renderScreen(<SignUpComponent />);
    fireEvent.click(screen.getByRole('button', { name: 'Sign up' }));
    await screen.findByText('Name is required');
    expect(screen.getByRole('textbox', { name: 'Your name' }).getAttribute('aria-invalid')).toBe(
      'true',
    );
    expect(server.getRequests()).toHaveLength(0);
  });

  it('accepts no late session after registration is abandoned', async () => {
    const registration = mockApi.delayed({
      method: 'post',
      path: '/users',
      status: 201,
      body: { token: 'obsolete-session', userId: 'obsolete-owner', email: 'owner@example.com' },
    });
    server.use(registration.handler);

    const view = renderScreen(<SignUpComponent />);
    submitRegistration();
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    view.rerender(<div>Registration closed</div>);
    const processed = new Promise<void>((resolve) =>
      server.events.on('response:mocked', () => resolve()),
    );

    await act(async () => {
      registration.release();
      await processed;
    });

    expect(getSessionSnapshot().token).toBeUndefined();
  });
});
