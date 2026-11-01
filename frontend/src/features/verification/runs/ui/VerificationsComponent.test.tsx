// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useRef } from 'react';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import type { Verification } from '../../../../entities/verification/runs';
import { API_BASE_PATH } from '../../../../shared/api';
import { advanceClockUntil } from '../../../../shared/lib/test/advanceClockUntil';
import { waitForCondition } from '../../../../shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { VerificationsComponent } from './VerificationsComponent';

dayjs.extend(utc);
const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
};
const verification = (id: string, trigger: Verification['trigger']): Verification => ({
  id,
  databaseId: database.id,
  backupId: 'backup-id',
  trigger,
  status: 'COMPLETED',
  attemptCount: 1,
  createdAt: '2026-10-27T10:00:00Z',
});
function ScrollingVerifications() {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  return (
    <div ref={scrollContainerRef} data-testid="verification-scroll">
      <VerificationsComponent
        database={database}
        isCanManageDBs={true}
        scrollContainerRef={scrollContainerRef}
      />
    </div>
  );
}
describe('VerificationsComponent', () => {
  it('follows nextCursor and displays the after-backup trigger when the list is scrolled', async () => {
    const cursors: (string | null)[] = [];
    mockServer.use(
      mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
      http.get(`*${API_BASE_PATH}/verifications`, ({ request }) => {
        const after = new URL(request.url).searchParams.get('after');
        cursors.push(after);
        return HttpResponse.json(
          after
            ? { items: [verification('oldest', 'MANUAL')] }
            : { items: [verification('newest', 'AFTER_BACKUP')], nextCursor: 'next-verifications' },
        );
      }),
    );

    renderScreen(<ScrollingVerifications />);
    expect(await screen.findAllByText('After backup')).toHaveLength(2);
    fireEvent.scroll(screen.getByTestId('verification-scroll'));
    expect(await screen.findAllByText('Manual')).toHaveLength(2);
    expect(await screen.findAllByText('All restore checks loaded (2 total)')).toHaveLength(2);

    // The list also polls its first page, so polls may fall between these two requests.
    expect(cursors[0]).toBeNull();
    expect(cursors.filter((cursor) => cursor !== null)).toEqual(['next-verifications']);
  });

  describe('with a failing refresh', () => {
    const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

    it('shows no alert, keeps the rows and requests again when polls fail', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let listRequests = 0;
      mockServer.use(
        mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
        http.get(`*${API_BASE_PATH}/verifications`, () =>
          ++listRequests === 1
            ? HttpResponse.json({ items: [verification('listed', 'MANUAL')] })
            : failedResponse(),
        ),
      );
      vi.useFakeTimers();
      renderScreen(<ScrollingVerifications />);
      await waitForCondition(() => screen.getAllByText('Manual'));

      await advanceClockUntil(() => expect(listRequests).toBeGreaterThanOrEqual(3));

      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.getAllByText('Manual')).toHaveLength(2);
    });

    it('shows no alert and refreshes again when the refresh after a successful cancel fails', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const running: Verification = { ...verification('running', 'MANUAL'), status: 'RUNNING' };
      let isCancelled = false;
      let refreshesAfterCancel = 0;
      mockServer.use(
        mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
        http.get(`*${API_BASE_PATH}/verifications`, () => {
          if (!isCancelled) return HttpResponse.json({ items: [running] });

          refreshesAfterCancel += 1;
          return refreshesAfterCancel === 1
            ? failedResponse()
            : HttpResponse.json({ items: [{ ...running, trigger: 'AFTER_BACKUP' }] });
        }),
        http.post(`*${API_BASE_PATH}/verifications/:verificationId/cancel`, () => {
          isCancelled = true;
          return HttpResponse.json(running);
        }),
      );
      vi.useFakeTimers();
      renderScreen(<ScrollingVerifications />);
      await waitForCondition(() => screen.getAllByText('Manual'));

      fireEvent.click(screen.getAllByRole('img', { name: 'close-circle' })[0]);

      await advanceClockUntil(() => expect(screen.getAllByText('After backup')).toHaveLength(2));

      expect(refreshesAfterCancel).toBeGreaterThanOrEqual(2);
      expect(showAlert).not.toHaveBeenCalled();
    });
  });
});
