// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { afterAll, afterEach, describe, expect, it, vi } from 'vitest';

import { OAuthSignInButtons } from '@/features/oauth-sign-in/ui/OAuthSignInButtons';
import { beginAuthIntent, isCurrentAuthIntent } from '@/shared/auth';
import { renderScreen } from '@/shared/lib/test/renderScreen';

vi.hoisted(() => {
  vi.stubEnv('VITE_GITHUB_CLIENT_ID', 'github-client');
  vi.stubEnv('VITE_GOOGLE_CLIENT_ID', 'google-client');
});
afterAll(() => vi.unstubAllEnvs());
afterEach(() => vi.restoreAllMocks());

describe('OAuthSignInButtons', () => {
  it.each(['GitHub', 'Google'])(
    'binds an unpredictable tab record when %s sign-in is started',
    async (provider) => {
      const navigate = vi.spyOn(window.location, 'assign').mockImplementation(() => {});
      const returnPath = '/workspaces/team-id/databases/shop-id/config';
      window.history.replaceState(null, '', returnPath);
      const { user } = renderScreen(<OAuthSignInButtons />, { route: returnPath });
      const button = screen.getByRole('button', { name: `Continue with ${provider}` });

      await user.click(button);
      const firstAttempt = JSON.parse(sessionStorage.getItem('databasus_oauth_sign_in_attempt')!);
      await user.click(button);
      const secondAttempt = JSON.parse(sessionStorage.getItem('databasus_oauth_sign_in_attempt')!);

      expect(secondAttempt.state).toMatch(/^[a-f0-9]{64}$/);
      expect(secondAttempt.state).not.toBe(firstAttempt.state);
      expect(Object.keys(secondAttempt).sort()).toEqual([
        'createdAt',
        'provider',
        'redirectUri',
        'returnPath',
        'state',
      ]);
      expect(firstAttempt.returnPath).toBe(returnPath);
      expect(secondAttempt.returnPath).toBe(returnPath);
      expect(secondAttempt.provider).toBe(provider.toLowerCase());
      expect(new URL(String(navigate.mock.calls[1][0])).searchParams.get('state')).toBe(
        secondAttempt.state,
      );
      expect(localStorage.getItem('databasus_oauth_sign_in_attempt')).toBeNull();
    },
  );
  it.each(['randomness', 'storage'])(
    'stays on the page when secure %s is unavailable',
    async (failure) => {
      const navigate = vi.spyOn(window.location, 'assign').mockImplementation(() => {});
      const { user } = renderScreen(<OAuthSignInButtons />);

      if (failure === 'randomness')
        vi.spyOn(crypto, 'getRandomValues').mockImplementation(() => {
          throw new Error('unavailable');
        });
      else
        vi.spyOn(sessionStorage, 'setItem').mockImplementation(() => {
          throw new Error('unavailable');
        });

      await user.click(screen.getByRole('button', { name: 'Continue with GitHub' }));

      expect(await screen.findByRole('alert')).toBeTruthy();
      expect(navigate).not.toHaveBeenCalled();
    },
  );
});

it('abandons a pending password operation when a new OAuth attempt starts', async () => {
  vi.spyOn(window.location, 'assign').mockImplementation(() => {});
  const passwordIntent = beginAuthIntent();
  const { user } = renderScreen(<OAuthSignInButtons />);
  await user.click(screen.getByRole('button', { name: 'Continue with GitHub' }));
  expect(isCurrentAuthIntent(passwordIntent)).toBe(false);
});
