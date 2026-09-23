/** The auth page's entry state: whether the instance already holds an account, and whether sign-up is currently allowed. */
export interface AuthEntryState {
  isExist: boolean;
  isSignUpAllowed: boolean;
}
