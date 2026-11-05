import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  BackupEncryption,
  BackupSettingsType,
  type LogicalBackupConfigInput,
  type PhysicalBackupConfigInput,
} from '../../../entities/backup-configs';
import { API_BASE_PATH } from '../../../shared/api';
import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { createDefaultLogicalBackupConfig } from './createDefaultLogicalBackupConfig';
import { startFirstBackup } from './startFirstBackup';

const mockServer = setUpMockApi();

const logicalConfig: LogicalBackupConfigInput = createDefaultLogicalBackupConfig();

const physicalConfig = {
  isBackupsEnabled: true,
  storageId: 'archive-storage-id',
  encryption: BackupEncryption.NONE,
  settings: { type: BackupSettingsType.PHYSICAL },
} as PhysicalBackupConfigInput;

const declareBackupStarts = () =>
  mockServer.use(
    http.post(`*${API_BASE_PATH}/logical-backups`, () =>
      HttpResponse.json({ id: 'backup-id' }, { status: 202 }),
    ),
    http.post(`*${API_BASE_PATH}/physical-backups`, () => new HttpResponse(null, { status: 202 })),
  );

const getStartedBackupPaths = () =>
  mockServer.getRequests().map((request) => new URL(request.url).pathname);

describe('startFirstBackup', () => {
  it.each([
    { kind: 'logical', backupConfig: logicalConfig, startPath: '/logical-backups' },
    { kind: 'physical', backupConfig: physicalConfig, startPath: '/physical-backups' },
  ])('starts a $kind backup when backups are enabled', async ({ backupConfig, startPath }) => {
    declareBackupStarts();

    await startFirstBackup('created-id', backupConfig);

    expect(getStartedBackupPaths()).toEqual([`${API_BASE_PATH}${startPath}`]);
  });

  it.each([
    { kind: 'logical', backupConfig: logicalConfig },
    { kind: 'physical', backupConfig: physicalConfig },
  ])('starts no $kind backup when backups are disabled', async ({ backupConfig }) => {
    declareBackupStarts();

    await startFirstBackup('created-id', { ...backupConfig, isBackupsEnabled: false });

    expect(getStartedBackupPaths()).toEqual([]);
  });
});
