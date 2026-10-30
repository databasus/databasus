import {
  type ApiComponents,
  accessTokenHelper,
  getApiClient,
  requireData,
  requireSuccess,
} from '../../../shared/api';
import type { SignInOutcome } from '../model/SignInOutcome';
import type { User } from '../model/User';

type Schemas = ApiComponents['schemas'];

const listeners: (() => void)[] = [];

const saveAuthorizedData = (accessToken: string, userId: string) => {
  accessTokenHelper.saveAccessToken(accessToken);
  accessTokenHelper.saveUserId(userId);
};

const notifyAuthListeners = () => {
  for (const listener of listeners) {
    listener();
  }
};

const startSession = (session: Schemas['Session']): Schemas['Session'] => {
  saveAuthorizedData(session.token, session.userId);
  notifyAuthListeners();

  return session;
};

// The current user is the one the stored session belongs to, so its operations take the ID saved
// at sign-in.
const getSignedInUserId = (): string => {
  const userId = accessTokenHelper.getUserId();
  if (!userId) {
    throw new Error('no signed-in user is stored');
  }

  return userId;
};

export const userApi = {
  async signUp(request: Schemas['SignUpRequest']): Promise<Schemas['Session']> {
    return startSession(requireData(await getApiClient().POST('/users', { body: request })));
  },

  async signIn(request: Schemas['SignInRequest']): Promise<SignInOutcome> {
    const outcome = requireData(await getApiClient().POST('/sessions', { body: request }));

    if (outcome.outcome === 'COMPLETED') {
      startSession(outcome);
    }

    return outcome;
  },

  async verifySignInCode(
    pendingSignInId: string,
    request: Schemas['VerifySignInCodeRequest'],
  ): Promise<Schemas['Session']> {
    const session = requireData(
      await getApiClient().POST('/pending-sign-ins/{pendingSignInId}/verify', {
        params: { path: { pendingSignInId } },
        body: request,
      }),
    );

    return startSession(session);
  },

  async resendSignInCode(
    pendingSignInId: string,
    request: Schemas['ResendSignInCodeRequest'],
  ): Promise<Schemas['PendingSignIn']> {
    return requireData(
      await getApiClient().POST('/pending-sign-ins/{pendingSignInId}/resend', {
        params: { path: { pendingSignInId } },
        body: request,
      }),
    );
  },

  async isAnyUserExists(): Promise<boolean> {
    const signInOptions = requireData(await getApiClient().GET('/sign-in-options'));

    return signInOptions.hasAnyUser;
  },

  async changePassword(request: Schemas['PasswordChange']): Promise<void> {
    requireSuccess(
      await getApiClient().PUT('/users/{userId}/password', {
        params: { path: { userId: getSignedInUserId() } },
        body: request,
      }),
    );
  },

  async getCurrentUser(): Promise<User> {
    return requireData(
      await getApiClient().GET('/users/{userId}', {
        params: { path: { userId: getSignedInUserId() } },
      }),
    );
  },

  async updateUserInfo(request: Schemas['UserUpdate']): Promise<User> {
    return requireData(
      await getApiClient().PUT('/users/{userId}', {
        params: { path: { userId: getSignedInUserId() } },
        body: request,
      }),
    );
  },

  async handleGitHubOAuth(request: Schemas['OAuthSignInRequest']): Promise<Schemas['Session']> {
    return startSession(
      requireData(await getApiClient().POST('/github-sessions', { body: request })),
    );
  },

  async handleGoogleOAuth(request: Schemas['OAuthSignInRequest']): Promise<Schemas['Session']> {
    return startSession(
      requireData(await getApiClient().POST('/google-sessions', { body: request })),
    );
  },

  async sendResetPasswordCode(request: Schemas['PasswordResetRequest']): Promise<void> {
    requireSuccess(await getApiClient().POST('/password-resets', { body: request }));
  },

  async resetPassword(request: Schemas['PasswordResetCompletion']): Promise<void> {
    requireSuccess(await getApiClient().POST('/password-reset-completions', { body: request }));
  },

  isAuthorized: (): boolean => !!accessTokenHelper.getAccessToken(),

  logout: () => {
    accessTokenHelper.clearStoredSession();
  },

  addAuthListener: (listener: () => void) => {
    listeners.push(listener);
  },

  removeAuthListener: (listener: () => void) => {
    listeners.splice(listeners.indexOf(listener), 1);
  },

  notifyAuthListeners,

  saveAuthorizedData,
};
