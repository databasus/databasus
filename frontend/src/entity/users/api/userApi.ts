import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { accessTokenHelper } from '../../../shared/api/accessTokenHelper';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { AuthEntryState } from '../model/AuthEntryState';
import type { ChangePasswordRequest } from '../model/ChangePasswordRequest';
import type { InviteUserRequest } from '../model/InviteUserRequest';
import type { InviteUserResponse } from '../model/InviteUserResponse';
import type { OAuthCallbackRequest } from '../model/OAuthCallbackRequest';
import type { OAuthCallbackResponse } from '../model/OAuthCallbackResponse';
import type { PendingSignIn } from '../model/PendingSignIn';
import type { ResendSignInCodeRequest } from '../model/ResendSignInCodeRequest';
import type { ResetPasswordRequest } from '../model/ResetPasswordRequest';
import type { SendResetPasswordCodeRequest } from '../model/SendResetPasswordCodeRequest';
import type { SignInOutcome } from '../model/SignInOutcome';
import type { SignInRequest } from '../model/SignInRequest';
import type { SignInResponse } from '../model/SignInResponse';
import type { SignUpRequest } from '../model/SignUpRequest';
import type { UpdateUserInfoRequest } from '../model/UpdateUserInfoRequest';
import type { UserProfile } from '../model/UserProfile';
import type { VerifySignInCodeRequest } from '../model/VerifySignInCodeRequest';
import { readSignInOutcome } from './readSignInOutcome';

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

export const userApi = {
  async signUp(signUpRequest: SignUpRequest): Promise<SignInResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(signUpRequest));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/users/signup`, requestOptions)
      .then((response: unknown): SignInResponse => {
        const typedResponse = response as SignInResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  async signIn(signInRequest: SignInRequest): Promise<SignInOutcome> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(signInRequest));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/users/signin`, requestOptions)
      .then((response: unknown): SignInOutcome => {
        const outcome = readSignInOutcome(response);

        if (outcome.isCompleted) {
          saveAuthorizedData(outcome.signIn.token, outcome.signIn.userId);
          notifyAuthListeners();
        }

        return outcome;
      });
  },

  async verifySignInCode(request: VerifySignInCodeRequest): Promise<SignInResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/users/verify-signin-code`, requestOptions)
      .then((response: unknown): SignInResponse => {
        const typedResponse = response as SignInResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  async resendSignInCode(request: ResendSignInCodeRequest): Promise<PendingSignIn> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/resend-signin-code`,
      requestOptions,
    );
  },

  /** Fetches whether the instance already holds an account and whether sign-up is currently allowed, for the auth page to decide what to show before anyone is signed in. */
  async getAuthEntryState(): Promise<AuthEntryState> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/users/is-any-user-exist`,
      requestOptions,
      true,
    );
  },

  async changePassword(request: ChangePasswordRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/users/change-password`,
      requestOptions,
    );
  },

  async inviteUser(request: InviteUserRequest): Promise<InviteUserResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(`${getApplicationServer()}/api/v1/users/invite`, requestOptions);
  },

  async getCurrentUser(): Promise<UserProfile> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(`${getApplicationServer()}/api/v1/users/me`, requestOptions);
  },

  async updateUserInfo(request: UpdateUserInfoRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPutJson(`${getApplicationServer()}/api/v1/users/me`, requestOptions);
  },

  async handleGitHubOAuth(request: OAuthCallbackRequest): Promise<OAuthCallbackResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/auth/github/callback`, requestOptions)
      .then((response: unknown): OAuthCallbackResponse => {
        const typedResponse = response as OAuthCallbackResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  async handleGoogleOAuth(request: OAuthCallbackRequest): Promise<OAuthCallbackResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/auth/google/callback`, requestOptions)
      .then((response: unknown): OAuthCallbackResponse => {
        const typedResponse = response as OAuthCallbackResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  async sendResetPasswordCode(request: SendResetPasswordCodeRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/send-reset-password-code`,
      requestOptions,
    );
  },

  async resetPassword(request: ResetPasswordRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/reset-password`,
      requestOptions,
    );
  },

  isAuthorized: (): boolean => !!accessTokenHelper.getAccessToken(),

  logout: () => {
    accessTokenHelper.cleanAccessToken();
  },

  // listeners

  addAuthListener: (listener: () => void) => {
    listeners.push(listener);
  },

  removeAuthListener: (listener: () => void) => {
    listeners.splice(listeners.indexOf(listener), 1);
  },

  notifyAuthListeners: (): void => {
    for (const listener of listeners) {
      listener();
    }
  },

  saveAuthorizedData,
};
