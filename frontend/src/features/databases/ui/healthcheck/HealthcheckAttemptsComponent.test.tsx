// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entity/databases';
import { API_BASE_PATH, type ApiComponents } from '../../../../shared/api';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { HealthcheckAttemptsComponent } from './HealthcheckAttemptsComponent';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
};
const config: ApiComponents['schemas']['HealthcheckConfig'] = {
  databaseId: database.id,
  isHealthcheckEnabled: true,
  isNotificationEnabled: true,
  intervalMinutes: 1,
  failedAttemptsBeforeUnavailable: 3,
  attemptRetentionDays: 7,
};
const attempt = (
  id: string,
  status: 'AVAILABLE' | 'UNAVAILABLE',
): ApiComponents['schemas']['HealthcheckAttempt'] => ({
  id,
  status,
  createdAt: new Date().toISOString(),
});
const attemptsPath = `*${API_BASE_PATH}/healthcheck-attempts`;

describe('HealthcheckAttemptsComponent', () => {
  it('follows every cursor and displays all attempts after the last page', async () => {
    const cursors: (string | null)[] = [];
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, ({ request }) => {
        const after = new URL(request.url).searchParams.get('after');
        cursors.push(after);
        const page: ApiComponents['schemas']['HealthcheckAttemptPage'] =
          after === 'second'
            ? { items: [attempt('last', 'AVAILABLE')] }
            : {
                items: [attempt('first', 'AVAILABLE'), attempt('middle', 'UNAVAILABLE')],
                nextCursor: 'second',
              };
        return HttpResponse.json(page);
      }),
    );
    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);

    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(3));
    expect(cursors).toEqual([null, 'second']);
    expect(container.querySelectorAll('[data-status="UNAVAILABLE"]')).toHaveLength(1);
  });

  it('discards every old page when the period changes during a multi-page load', async () => {
    let releaseOldPage!: () => void;
    const oldPage = new Promise<void>((resolve) => {
      releaseOldPage = resolve;
    });
    let hasRequestedOldPage = false;
    let hasChangedPeriod = false;
    const sinceValues: string[] = [];
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, async ({ request }) => {
        const query = new URL(request.url).searchParams;
        sinceValues.push(query.get('since')!);
        if (query.get('after')) {
          hasRequestedOldPage = true;
          await oldPage;
          return HttpResponse.json({ items: [attempt('old-last', 'UNAVAILABLE')] });
        }
        if (hasChangedPeriod) return HttpResponse.json({ items: [attempt('new', 'AVAILABLE')] });
        return HttpResponse.json({
          items: [attempt('old-first', 'UNAVAILABLE')],
          nextCursor: 'old-second',
        });
      }),
    );
    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    await waitFor(() => expect(hasRequestedOldPage).toBe(true));
    hasChangedPeriod = true;
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('7 days'));
    await waitFor(() =>
      expect(container.querySelectorAll('[data-status="AVAILABLE"]')).toHaveLength(1),
    );
    await act(async () => {
      releaseOldPage();
      await oldPage;
    });

    expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
    expect(container.querySelectorAll('[data-status="UNAVAILABLE"]')).toHaveLength(0);
    expect(sinceValues[0]).toEqual(sinceValues[1]);
    expect(new Date(sinceValues[2]).getTime()).toBeLessThan(new Date(sinceValues[0]).getTime());
  });
});
