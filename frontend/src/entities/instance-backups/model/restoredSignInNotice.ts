// A tab-scoped marker without secrets: after an accepted recovery replaces the accounts, the sign-in
// screen explains that the accounts from the Databasus backup now apply.
const RESTORED_SIGN_IN_KEY = 'databasus_restored_sign_in_expected';

export const expectRestoredSignIn = () => {
  try {
    globalThis.sessionStorage?.setItem(RESTORED_SIGN_IN_KEY, 'true');
  } catch {}
};

export const isRestoredSignInExpected = (): boolean => {
  try {
    return globalThis.sessionStorage?.getItem(RESTORED_SIGN_IN_KEY) === 'true';
  } catch {
    return false;
  }
};

export const clearRestoredSignInExpectation = () => {
  try {
    globalThis.sessionStorage?.removeItem(RESTORED_SIGN_IN_KEY);
  } catch {}
};
