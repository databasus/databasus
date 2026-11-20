import type {
  InstanceBackup,
  InstanceBackupDestinationChoice,
  InstanceBackupSettings,
  InstanceBackupStatus,
  InstanceRecovery,
} from '@/entities/instance-backups';

export const s3Choice: InstanceBackupDestinationChoice = {
  storageId: '11111111-1111-4111-8111-111111111111',
  storageName: 'Offsite S3',
  storageType: 'S3',
  workspaceId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  workspaceName: 'Production',
  isSelected: false,
};

export const localChoice: InstanceBackupDestinationChoice = {
  storageId: '22222222-2222-4222-8222-222222222222',
  storageName: 'Local disk',
  storageType: 'LOCAL',
  workspaceId: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
  workspaceName: 'Staging',
  isSelected: false,
};

export const buildSettings = (
  settings: Partial<InstanceBackupSettings> = {},
): InstanceBackupSettings => ({
  schedule: { type: 'HOURLY' },
  retentionDays: 30,
  storageIds: [],
  ...settings,
});

export const buildStatus = (status: Partial<InstanceBackupStatus> = {}): InstanceBackupStatus => ({
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

export const buildBackup = (backup: Partial<InstanceBackup> = {}): InstanceBackup => ({
  id: '33333333-3333-4333-8333-333333333333',
  status: 'COMPLETED',
  trigger: 'SCHEDULE',
  sourceRelease: '3.1.0',
  startedAt: '2026-11-14T10:00:00Z',
  snapshotAt: '2026-11-14T10:00:05Z',
  completedAt: '2026-11-14T10:00:30Z',
  sizeBytes: 2048,
  failureReason: null,
  copies: [
    {
      id: '44444444-4444-4444-8444-444444444444',
      storageId: s3Choice.storageId,
      storageName: s3Choice.storageName,
      workspaceName: s3Choice.workspaceName,
      status: 'STORED',
      attemptCount: 1,
      storedAt: '2026-11-14T10:00:30Z',
      lastError: null,
      isDownloadable: true,
    },
  ],
  ...backup,
});

export const buildRecovery = (recovery: Partial<InstanceRecovery> = {}): InstanceRecovery => ({
  id: '99999999-9999-4999-8999-999999999999',
  status: 'VALIDATING',
  origin: 'UPLOAD',
  archiveSha256: 'b'.repeat(64),
  runningRelease: '3.1.0',
  sourceRelease: null,
  snapshotAt: null,
  createdAt: '2026-11-14T10:00:00Z',
  expiresAt: '2026-11-14T11:00:00Z',
  completedAt: null,
  failure: null,
  discardedCleanupCount: 0,
  ...recovery,
});
