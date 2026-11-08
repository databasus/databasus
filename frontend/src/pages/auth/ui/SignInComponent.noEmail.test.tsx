// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { afterAll, describe, expect, it, vi } from 'vitest';

import { SignInComponent } from '@/pages/auth/ui/SignInComponent';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

vi.hoisted(() => {
  vi.stubEnv('VITE_IS_EMAIL_CONFIGURED', 'false');
  vi.stubEnv('VITE_GOOGLE_CLIENT_ID', '');
  vi.stubEnv('VITE_GITHUB_CLIENT_ID', '');
});
afterAll(() => vi.unstubAllEnvs());

const mockServer = setUpMockApi();

describe('SignInComponent without email delivery', () => {
  it('directs the user to an administrator when neither a password nor a provider is available', async () => {
    mockServer.use(mockApi.post('/sessions', 401, buildProblem('password_not_set', 401)));
    const { user } = renderScreen(<SignInComponent />);

    await user.type(screen.getByRole('textbox', { name: 'Your email' }), 'owner@example.com');
    await user.type(screen.getByLabelText('Password', { selector: 'input' }), 'unused-password');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(await screen.findByText(DICTIONARIES.en.errors.passwordNotSet)).toBeTruthy();
    expect(screen.getByText(DICTIONARIES.en.users.signIn.contactAdministratorHint)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Set a password' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Forgot password?' })).toBeNull();
    expect(screen.queryByRole('button', { name: /Continue with/ })).toBeNull();
    expect(mockServer.getRequests()).toHaveLength(1);
  });
});
