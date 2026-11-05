import { beginAuthIntent } from '@/shared/auth';

import { getOAuthRedirectUri } from './getOAuthRedirectUri';

export type OAuthProvider = 'github' | 'google';
export interface OAuthSignInAttempt {
  state: string;
  provider: OAuthProvider;
  createdAt: number;
  redirectUri: string;
}
const ATTEMPT_KEY = 'databasus_oauth_sign_in_attempt';
const ATTEMPT_TTL_MS = 10 * 60 * 1000;

export const createOAuthSignInAttempt = (provider: OAuthProvider): OAuthSignInAttempt => {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  const attempt = {
    provider,
    state: Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join(''),
    createdAt: Date.now(),
    redirectUri: getOAuthRedirectUri(),
  };

  sessionStorage.setItem(ATTEMPT_KEY, JSON.stringify(attempt));
  beginAuthIntent();
  return attempt;
};

export const consumeOAuthSignInAttempt = (state: string | null): OAuthSignInAttempt => {
  const raw = sessionStorage.getItem(ATTEMPT_KEY);
  if (!raw || !state) throw new Error('OAuth sign-in attempt is missing');
  const attempt: OAuthSignInAttempt = JSON.parse(raw);

  if (
    !attempt ||
    typeof attempt.state !== 'string' ||
    !/^[a-f0-9]{64}$/.test(attempt.state) ||
    state !== attempt.state ||
    !['github', 'google'].includes(attempt.provider) ||
    !Number.isFinite(attempt.createdAt) ||
    attempt.createdAt > Date.now() ||
    Date.now() - attempt.createdAt >= ATTEMPT_TTL_MS ||
    attempt.redirectUri !== getOAuthRedirectUri()
  )
    throw new Error('OAuth sign-in attempt is invalid');

  sessionStorage.removeItem(ATTEMPT_KEY);
  return attempt;
};
