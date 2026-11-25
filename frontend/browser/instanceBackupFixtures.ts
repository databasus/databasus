import type { components as ApiComponents } from '../src/shared/api/schema.gen';
import { accountFixtures } from './accountFixtures';
import { workspace } from './applicationFixtures';
import { apiResponse, type HttpFixture } from './HttpFixture';

type Schemas = ApiComponents['schemas'];

export const offsiteStorage: Schemas['Storage'] = {
  id: 'offsite-storage-id',
  workspaceId: workspace.id,
  name: 'Offsite archive',
  settings: { type: 'LOCAL' },
};

export const preparedRecovery: Schemas['InstanceRecovery'] = {
  id: 'recovery-id',
  status: 'READY',
  origin: 'UPLOAD',
  archiveSha256: 'c'.repeat(64),
  runningRelease: '3.1.0',
  sourceRelease: '3.1.0',
  snapshotAt: '2026-10-18T11:00:00Z',
  createdAt: '2026-10-18T11:55:00Z',
  expiresAt: '2026-10-18T13:00:00Z',
  completedAt: null,
  failure: null,
  discardedCleanupCount: 0,
};

export const buildInstanceBackupStatus = (
  status: Partial<Schemas['InstanceBackupStatus']> = {},
): Schemas['InstanceBackupStatus'] => ({
  health: 'NOT_CONFIGURED',
  destinations: [],
  incident: null,
  isEmailConfigured: true,
  isRootEmailUsable: true,
  isMasterKeyAcknowledged: true,
  isConfirmationCodeRequired: false,
  reminder: { isEligible: false },
  recovery: null,
  ...status,
});

// The account fixtures with the signed-in owner as root admin and every Databasus backup operation
// the settings page and the floating notices read.
export const instanceBackupFixtures = ({
  status = buildInstanceBackupStatus(),
  isRootAdmin = true,
  extra = [],
}: {
  status?: Schemas['InstanceBackupStatus'];
  isRootAdmin?: boolean;
  extra?: HttpFixture[];
} = {}): HttpFixture[] => {
  const replacedPaths = new Set(['GET /users/{userId}', 'GET /instance-settings']);

  return accountFixtures()
    .filter((fixture) => !replacedPaths.has(`${fixture.method} ${fixture.path}`))
    .concat([
      apiResponse({
        method: 'get',
        path: '/users/{userId}',
        status: 200,
        body: {
          id: 'owner-id',
          email: 'owner@example.com',
          name: 'Owner',
          role: 'ADMIN',
          status: 'ACTIVE',
          isRootAdmin,
          createdAt: '2026-10-01T00:00:00Z',
          authenticatorAppConnectedAt: null,
          unusedRecoveryCodeCount: null,
        },
      }),
      apiResponse({
        method: 'get',
        path: '/instance-settings',
        status: 200,
        body: {
          isAllowExternalRegistrations: false,
          isAllowMemberInvitations: false,
          isMemberAllowedToCreateWorkspaces: false,
          isEmailConfigured: status.isEmailConfigured,
          twoFactorRequirement: 'OFF',
        },
      }),
      apiResponse({ method: 'get', path: '/instance-backup-statuses', status: 200, body: status }),
      apiResponse({
        method: 'get',
        path: '/instance-backup-settings',
        status: 200,
        body: {
          schedule: { type: 'HOURLY' },
          retentionDays: 30,
          storageIds: status.destinations.map((destination) => destination.storageId),
        },
      }),
      apiResponse({
        method: 'get',
        path: '/instance-backup-destinations',
        status: 200,
        body: { items: [] },
      }),
      apiResponse({ method: 'get', path: '/instance-backups', status: 200, body: { items: [] } }),
      ...extra,
    ]);
};
