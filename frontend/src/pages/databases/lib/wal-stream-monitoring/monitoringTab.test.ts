import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { isMonitoringTabShown } from './monitoringTab';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const buildDatabase = (settings: Database['settings']): Database => ({
  id: 'database-id',
  name: 'database',
  workspaceId: 'workspace-id',
  settings,
  notifierIds: [],
});

const buildPhysicalDatabase = (backupType: PhysicalDatabaseBackupType): Database =>
  buildDatabase({
    ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
    backupType,
  });

describe('isMonitoringTabShown', () => {
  it('shows the tab for a physical database in WAL streaming mode', () => {
    expect(
      isMonitoringTabShown(
        buildPhysicalDatabase(PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM),
      ),
    ).toBe(true);
  });

  it('hides the tab for a physical database without WAL streaming', () => {
    expect(
      isMonitoringTabShown(buildPhysicalDatabase(PhysicalDatabaseBackupType.FULL_INCREMENTAL)),
    ).toBe(false);
  });

  it('hides the tab for a logical database', () => {
    expect(
      isMonitoringTabShown(
        buildDatabase(createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL)),
      ),
    ).toBe(false);
  });
});
