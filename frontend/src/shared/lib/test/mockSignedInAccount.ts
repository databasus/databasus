import { mockApi } from '@/shared/lib/test/mockApi';

export const mockSignedInAccount = ({
  userId = 'root',
  isAuthenticatorAppConnected = false,
}: { userId?: string; isAuthenticatorAppConnected?: boolean } = {}) =>
  mockApi.get('/users/{userId}', 200, {
    id: userId,
    email: 'root@example.com',
    name: 'Root',
    role: 'ADMIN',
    status: 'ACTIVE',
    isRootAdmin: true,
    createdAt: '2026-10-01T00:00:00Z',
    authenticatorAppConnectedAt: isAuthenticatorAppConnected ? '2026-11-01T00:00:00Z' : null,
    unusedRecoveryCodeCount: isAuthenticatorAppConnected ? 10 : null,
  });
