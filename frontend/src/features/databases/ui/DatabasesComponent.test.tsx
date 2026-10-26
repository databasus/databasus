// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { http } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  type Database,
  DatabaseType,
  type HealthStatus,
  createDefaultDatabaseSettings,
} from '../../../entity/databases';
import type { Workspace } from '../../../entity/workspaces';
import { API_BASE_PATH } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { DatabasesComponent } from './DatabasesComponent';

const mockServer = setUpMockApi();

const workspace: Workspace = {
  id: 'workspace-id',
  name: 'Production',
  createdAt: '2026-01-01T00:00:00Z',
  currentUserRole: 'WORKSPACE_VIEWER',
};

const buildDatabase = (name: string, healthStatus: HealthStatus): Database => ({
  id: `${name}-id`,
  workspaceId: workspace.id,
  name,
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
  healthStatus,
});

describe('DatabasesComponent', () => {
  it('lists unavailable databases first when the server lists them after available ones', async () => {
    mockServer.use(
      mockApi.get('/databases', 200, {
        items: [
          buildDatabase('Alpha', 'AVAILABLE'),
          buildDatabase('Bravo', 'UNAVAILABLE'),
          buildDatabase('Charlie', 'AVAILABLE'),
          buildDatabase('Delta', 'UNAVAILABLE'),
        ],
      }),
      // The selected database's details load through the old API, which this test does not cover.
      http.all(`*${API_BASE_PATH}/*`, () => new Promise<never>(() => {})),
    );

    renderScreen(<DatabasesComponent contentHeight={800} workspace={workspace} isCanManageDBs />);

    await screen.findByText('Alpha');
    const listedNames = screen
      .getAllByText(/^(Alpha|Bravo|Charlie|Delta)$/)
      .map((element) => element.textContent);
    expect(listedNames).toEqual(['Bravo', 'Delta', 'Alpha', 'Charlie']);
  });
});
