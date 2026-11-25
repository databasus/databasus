import { getSignedInUserId } from '@/entities/users/api/getSignedInUserId';
import type { AuthenticatorAppSetup } from '@/entities/users/model/AuthenticatorAppSetup';
import type { User } from '@/entities/users/model/User';
import { getApiClient, requireData } from '@/shared/api';

export const authenticatorAppApi = {
  // A user who already has an app proves it with a current app code or a recovery code.
  async startSetup(currentCode?: string): Promise<AuthenticatorAppSetup> {
    return requireData(
      await getApiClient().POST('/users/{userId}/authenticator-app-setups', {
        params: { path: { userId: getSignedInUserId() } },
        body: { code: currentCode },
      }),
    );
  },

  async confirmSetup(authenticatorAppSetupId: string, code: string): Promise<string[]> {
    const issued = requireData(
      await getApiClient().POST(
        '/users/{userId}/authenticator-app-setups/{authenticatorAppSetupId}/confirm',
        {
          params: { path: { userId: getSignedInUserId(), authenticatorAppSetupId } },
          body: { code },
        },
      ),
    );

    return issued.recoveryCodes;
  },

  async disconnect(currentCode: string): Promise<User> {
    return requireData(
      await getApiClient().POST('/users/{userId}/disconnect-authenticator-app', {
        params: { path: { userId: getSignedInUserId() } },
        body: { code: currentCode },
      }),
    );
  },

  async regenerateRecoveryCodes(currentCode: string): Promise<string[]> {
    const issued = requireData(
      await getApiClient().POST('/users/{userId}/recovery-codes', {
        params: { path: { userId: getSignedInUserId() } },
        body: { code: currentCode },
      }),
    );

    return issued.recoveryCodes;
  },
};
