import type { SessionSnapshot } from '@/shared/auth';

const DEFERRAL_KEY = 'databasus_master_key_onboarding_deferred';

// Deferral lasts for one sign-in, reloads included, so it is bound to the session's token rather
// than to the page; a later sign-in brings a new token and the dialog back. Only the token's tail is
// kept, which identifies the sign-in without storing a usable credential a second time.
const getSessionMark = (session: SessionSnapshot) =>
  session.token ? `${session.userId ?? ''}:${session.token.slice(-12)}` : undefined;

export const isMasterKeyOnboardingDeferred = (session: SessionSnapshot): boolean => {
  const mark = getSessionMark(session);
  if (!mark) return false;

  try {
    return globalThis.sessionStorage?.getItem(DEFERRAL_KEY) === mark;
  } catch {
    return false;
  }
};

export const deferMasterKeyOnboarding = (session: SessionSnapshot) => {
  const mark = getSessionMark(session);
  if (!mark) return;

  try {
    globalThis.sessionStorage?.setItem(DEFERRAL_KEY, mark);
  } catch {}
};
