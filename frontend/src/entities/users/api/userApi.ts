import { getSignedInUserId } from '@/entities/users/api/getSignedInUserId';
import type { AuthenticatorAppSetupSession } from '@/entities/users/model/AuthenticatorAppSetupSession';
import type { SignInOutcome } from '@/entities/users/model/SignInOutcome';
import type { User } from '@/entities/users/model/User';
import type { ReadOptions } from '@/shared/api';
import { type ApiComponents, getApiClient, requireData, requireSuccess } from '@/shared/api';

type Schemas = ApiComponents['schemas'];

export const userApi = {
  async signUp(request: Schemas['SignUpRequest']): Promise<Schemas['Session']> {
    return requireData(await getApiClient().POST('/users', { body: request }));
  },

  async signIn(request: Schemas['SignInRequest']): Promise<SignInOutcome> {
    const outcome = requireData(await getApiClient().POST('/sessions', { body: request }));

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

    return session;
  },

  async confirmSignInAuthenticatorAppSetup(
    pendingSignInId: string,
    request: Schemas['VerifySignInCodeRequest'],
  ): Promise<AuthenticatorAppSetupSession> {
    return requireData(
      await getApiClient().POST(
        '/pending-sign-ins/{pendingSignInId}/confirm-authenticator-app-setup',
        {
          params: { path: { pendingSignInId } },
          body: request,
        },
      ),
    );
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

  async isAnyUserExists(options: ReadOptions = {}): Promise<boolean> {
    const signInOptions = requireData(await getApiClient(options).GET('/sign-in-options'));

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

  async getCurrentUser(options: ReadOptions = {}): Promise<User> {
    return requireData(
      await getApiClient(options).GET('/users/{userId}', {
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
    return requireData(await getApiClient().POST('/github-sessions', { body: request }));
  },

  async handleGoogleOAuth(request: Schemas['OAuthSignInRequest']): Promise<Schemas['Session']> {
    return requireData(await getApiClient().POST('/google-sessions', { body: request }));
  },

  async sendResetPasswordCode(request: Schemas['PasswordResetRequest']): Promise<void> {
    requireSuccess(await getApiClient().POST('/password-resets', { body: request }));
  },

  async resetPassword(request: Schemas['PasswordResetCompletion']): Promise<void> {
    requireSuccess(await getApiClient().POST('/password-reset-completions', { body: request }));
  },
};
