import { userApi } from '@/entities/users/api/userApi';
import { useSessionQuery } from '@/shared/api';

// A confirmation step shows the app code field instead of a send action when the signed-in user
// has an app. Undefined while the account loads; a failed load answers false, so the emailed code
// flow stays reachable.
export const useIsAuthenticatorAppConnected = (isNeeded = true): boolean | undefined => {
  const query = useSessionQuery({
    queryKey: ['authenticator-app-state'],
    enabled: isNeeded,
    queryFn: (signal) => userApi.getCurrentUser({ signal }),
  });

  if (query.error) return false;
  if (!query.data) return undefined;

  return query.data.authenticatorAppConnectedAt !== null;
};
