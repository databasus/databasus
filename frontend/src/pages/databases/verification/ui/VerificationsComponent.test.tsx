// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useRef, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { verificationAgentApi } from '@/entities/verification-agents';
import { verificationRunsApi } from '@/pages/databases/verification/api/verificationRunsApi';
import type {
  Verification,
  VerificationDetails,
} from '@/pages/databases/verification/model/verificationContract';
import { VerificationsComponent } from '@/pages/databases/verification/ui/VerificationsComponent';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { advanceClockUntil } from '@/shared/lib/test/advanceClockUntil';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

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
const INTERNAL_ERROR = DICTIONARIES.en.errors.internal;
const REFUSAL = DICTIONARIES.en.errors.permissionDenied;
const runsCopy = DICTIONARIES.en.verification.runs;
const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });
const refusedResponse = () =>
  HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });
const buildDetails = (change: Partial<VerificationDetails> = {}): VerificationDetails => ({
  ...verification('finished', 'MANUAL'),
  tableStats: [],
  ...change,
});

const declareList = (items: Verification[]) =>
  mockServer.use(
    mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
    mockApi.get('/verifications', 200, { items }),
  );

const DatabasePage = ({ isCanManageDBs = true }: { isCanManageDBs?: boolean }) => {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [isListOpen, setIsListOpen] = useState(true);

  return (
    <div ref={scrollContainerRef} data-testid="verification-scroll">
      <button onClick={() => setIsListOpen((isOpen) => !isOpen)}>Toggle tab</button>
      {isListOpen && (
        <VerificationsComponent
          database={database}
          isCanManageDBs={isCanManageDBs}
          scrollContainerRef={scrollContainerRef}
        />
      )}
    </div>
  );
};

const leaveTab = () => fireEvent.click(screen.getByRole('button', { name: 'Toggle tab' }));

const getRowAction = (name: string) => screen.getAllByRole('button', { name })[0];

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
  it('keeps the next availability tick when an agent-count response finishes between ticks', async () => {
    declareList([]);
    const availabilityRead = vi.spyOn(verificationAgentApi, 'getAgentCount');
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });

    renderScreen(<DatabasePage />);
    await act(async () => {
      await availabilityRead.mock.results[0].value;
    });

    const slowRead = mockApi.pending('get', '/verification-agent-counts');
    mockServer.use(slowRead.handler);

    await advanceFakeClock(5000);

    expect(availabilityRead).toHaveBeenCalledTimes(2);

    await advanceFakeClock(2000);
    await releaseAndSettle(availabilityRead, () =>
      slowRead.respond(HttpResponse.json({ agentCount: 1 })),
    );

    mockServer.use(mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }));

    await advanceFakeClock(3000);

    expect(availabilityRead).toHaveBeenCalledTimes(3);
  });

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

  describe('with a failing load', () => {
    it('shows the reason with a retry instead of the empty text, and the retry shows the same database', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const requestedDatabaseIds: (string | null)[] = [];
      const requestedCursors: (string | null)[] = [];
      let isListFailing = true;
      mockServer.use(
        mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
        http.get(`*${API_BASE_PATH}/verifications`, ({ request }) => {
          const { searchParams } = new URL(request.url);
          requestedDatabaseIds.push(searchParams.get('databaseId'));
          requestedCursors.push(searchParams.get('after'));

          return isListFailing
            ? failedResponse()
            : HttpResponse.json({ items: [verification('listed', 'MANUAL')] });
        }),
      );
      renderScreen(<DatabasePage />);

      const retryButton = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(screen.queryByText(runsCopy.empty)).toBeNull();
      expect(showAlert).not.toHaveBeenCalled();

      isListFailing = false;
      fireEvent.click(retryButton);

      expect((await screen.findAllByText('Manual')).length).toBeGreaterThan(0);
      expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();
      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
      // The list polls its first page once the retry succeeds, so more requests may follow.
      expect(requestedDatabaseIds.length).toBeGreaterThanOrEqual(2);
      expect(requestedDatabaseIds.every((databaseId) => databaseId === database.id)).toBe(true);
      expect(requestedCursors.every((cursor) => cursor === null)).toBe(true);
    });

    it('keeps the rows, reports once and requests again on a later scroll when the next page fails', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const requestedCursors: string[] = [];
      let isNextPageFailing = true;
      mockServer.use(
        mockApi.get('/verification-agent-counts', 200, { agentCount: 1 }),
        http.get(`*${API_BASE_PATH}/verifications`, ({ request }) => {
          const cursor = new URL(request.url).searchParams.get('after');

          if (!cursor) {
            return HttpResponse.json({
              items: [verification('newest', 'AFTER_BACKUP')],
              nextCursor: 'first-page-cursor',
            });
          }

          requestedCursors.push(cursor);

          return isNextPageFailing
            ? failedResponse()
            : HttpResponse.json({ items: [verification('oldest', 'MANUAL')] });
        }),
      );
      renderScreen(<DatabasePage />);
      await screen.findAllByText('After backup');

      fireEvent.scroll(screen.getByTestId('verification-scroll'));

      expect(await screen.findAllByText(INTERNAL_ERROR)).toHaveLength(1);
      expect(screen.getAllByText('After backup').length).toBeGreaterThan(0);
      expect(showAlert).not.toHaveBeenCalled();

      isNextPageFailing = false;
      fireEvent.scroll(screen.getByTestId('verification-scroll'));

      expect((await screen.findAllByText('Manual')).length).toBeGreaterThan(0);
      expect(screen.getAllByText(INTERNAL_ERROR)).toHaveLength(1);
      expect(requestedCursors).toEqual(['first-page-cursor', 'first-page-cursor']);
    });
  });

  describe('its cancel action', () => {
    const running: Verification = { ...verification('running', 'MANUAL'), status: 'RUNNING' };
    const cancelName = runsCopy.actions.cancel;

    const renderRunningVerification = async (...otherVerifications: Verification[]) => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      declareList([running, ...otherVerifications]);
      const view = renderScreen(<DatabasePage />);
      await screen.findAllByText('Manual');

      return { showAlert, view };
    };

    // Only the intervals are faked and nothing here moves them, because vi.waitFor would move the
    // clock. No poll runs, so the only list request the answer can cause comes from the cancel.
    const renderRunningVerificationWithStoppedPolls = async () => {
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      declareList([running]);
      const listCall = vi.spyOn(verificationRunsApi, 'listByDatabase');
      renderScreen(<DatabasePage />);
      await act(async () => {
        await Promise.allSettled([listCall.mock.results[0].value]);
      });

      return { showAlert, listCall };
    };

    const declareHeldCancel = () => {
      const heldCancel = createPendingApiResponse();
      mockServer.use(
        http.post(
          `*${API_BASE_PATH}/verifications/:verificationId/cancel`,
          () => heldCancel.response,
        ),
      );

      return heldCancel;
    };

    it('reports a refused cancel once, keeps the rows and leaves the action available', async () => {
      const { showAlert } = await renderRunningVerification();
      mockServer.use(
        mockApi.post(
          '/verifications/{verificationId}/cancel',
          403,
          buildProblem('permission_denied', 403),
        ),
      );

      fireEvent.click(getRowAction(cancelName));

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getAllByText('Manual').length).toBeGreaterThan(0);
      expect(showAlert).not.toHaveBeenCalled();
      await waitFor(() =>
        expect((getRowAction(cancelName) as HTMLButtonElement).disabled).toBe(false),
      );
      expect(screen.getAllByText(REFUSAL)).toHaveLength(1);
    });

    it('starts the cancel from the keyboard and blocks the cancel of another row while it runs', async () => {
      const { view } = await renderRunningVerification({
        ...running,
        id: 'other-running',
      });
      const cancelCall = vi.spyOn(verificationRunsApi, 'cancel');
      const heldCancel = declareHeldCancel();

      getRowAction(cancelName).focus();
      await view.user.keyboard('{Enter}');
      await waitFor(() => expect(cancelCall).toHaveBeenCalledOnce());

      const otherRow = within(screen.getByRole('table')).getAllByRole('row')[2];
      fireEvent.click(within(otherRow).getByRole('button', { name: cancelName }));

      expect(cancelCall).toHaveBeenCalledOnce();
      expect(cancelCall).toHaveBeenCalledWith('running');
      await releaseAndSettle(cancelCall, () => heldCancel.respond(HttpResponse.json(running)));
    });

    it.each([
      { answer: 'a refusal', buildAnswer: refusedResponse },
      { answer: 'an acceptance', buildAnswer: () => HttpResponse.json(running) },
    ])(
      'shows nothing and requests no list when $answer arrives after the user left',
      async ({ buildAnswer }) => {
        const { showAlert, listCall } = await renderRunningVerificationWithStoppedPolls();
        const cancelCall = vi.spyOn(verificationRunsApi, 'cancel');
        const heldCancel = declareHeldCancel();

        await act(async () => {
          fireEvent.click(getRowAction(cancelName));
        });
        expect(cancelCall).toHaveBeenCalledOnce();
        leaveTab();
        await releaseAndSettle(cancelCall, () => heldCancel.respond(buildAnswer()));

        expect(listCall).toHaveBeenCalledOnce();
        expect(document.querySelector('[role="alert"], [role="status"]')).toBeNull();
        expect(showAlert).not.toHaveBeenCalled();
        expect(screen.getByRole('button', { name: 'Toggle tab' })).toBeTruthy();
      },
    );

    it.each([
      { answer: 'a refusal', buildAnswer: refusedResponse },
      { answer: 'an acceptance', buildAnswer: () => HttpResponse.json(running) },
    ])(
      'shows nothing and requests no list when $answer arrives after the session changed',
      async ({ buildAnswer }) => {
        activateSession({ token: 'first-token', userId: 'first-user' });
        const { showAlert, listCall } = await renderRunningVerificationWithStoppedPolls();
        const cancelCall = vi.spyOn(verificationRunsApi, 'cancel');
        const heldCancel = declareHeldCancel();

        await act(async () => {
          fireEvent.click(getRowAction(cancelName));
        });
        expect(cancelCall).toHaveBeenCalledOnce();
        act(() => {
          invalidateSession();
        });
        await releaseAndSettle(cancelCall, () => heldCancel.respond(buildAnswer()));

        expect(listCall).toHaveBeenCalledOnce();
        expect(document.querySelector('[role="alert"], [role="status"]')).toBeNull();
        expect(showAlert).not.toHaveBeenCalled();
      },
    );
  });

  describe('its no-agents warning', () => {
    const DOCS_LINK_NAME = 'Learn more';

    const queryDocsLink = () => screen.queryByRole('link', { name: DOCS_LINK_NAME });

    const renderAfterAgentCountSettles = async () => {
      const agentCountCall = vi.spyOn(verificationAgentApi, 'getAgentCount');
      renderScreen(<DatabasePage />);
      await screen.findAllByText('Manual');
      await act(async () => {
        await Promise.allSettled([agentCountCall.mock.results[0].value]);
      });
    };

    it('shows the warning with its docs link when no agent is registered', async () => {
      mockServer.use(
        mockApi.get('/verification-agent-counts', 200, { agentCount: 0 }),
        mockApi.get('/verifications', 200, { items: [] }),
      );
      renderScreen(<DatabasePage />);

      const docsLink = await screen.findByRole('link', { name: DOCS_LINK_NAME });

      expect(docsLink.getAttribute('href')).toBe('https://databasus.com/restore-verification/');
      expect(screen.getByRole('alert').textContent).toContain('No verification agents registered');
    });

    it('shows no warning when an agent is registered', async () => {
      declareList([verification('listed', 'MANUAL')]);

      await renderAfterAgentCountSettles();

      expect(queryDocsLink()).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    });

    it('shows no warning when the agent count fails', async () => {
      mockServer.use(
        http.get(`*${API_BASE_PATH}/verification-agent-counts`, () => failedResponse()),
        mockApi.get('/verifications', 200, { items: [verification('listed', 'MANUAL')] }),
      );

      await renderAfterAgentCountSettles();

      expect(queryDocsLink()).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    });

    it('removes the warning when a later agent count fails', async () => {
      let isAgentCountFailing = false;
      mockServer.use(
        http.get(`*${API_BASE_PATH}/verification-agent-counts`, () =>
          isAgentCountFailing ? failedResponse() : HttpResponse.json({ agentCount: 0 }),
        ),
        mockApi.get('/verifications', 200, { items: [] }),
      );
      vi.useFakeTimers();
      renderScreen(<DatabasePage />);
      await waitForCondition(() => screen.getByRole('link', { name: DOCS_LINK_NAME }));

      isAgentCountFailing = true;

      await advanceClockUntil(() => expect(queryDocsLink()).toBeNull());
    });
  });

  describe('its status labels', () => {
    const statusCopy = DICTIONARIES.en.status.verification;

    it.each([
      { status: 'PENDING', label: statusCopy.pending },
      { status: 'RUNNING', label: statusCopy.running },
      { status: 'COMPLETED', label: statusCopy.completed },
      { status: 'FAILED', label: statusCopy.failed },
      { status: 'CANCELED', label: statusCopy.canceled },
    ] as const)(
      'shows $label in the table and the cards with status $status',
      async ({ status, label }) => {
        declareList([{ ...verification('listed', 'MANUAL'), status }]);
        renderScreen(<DatabasePage />);

        const shownLabels = await screen.findAllByText(label);

        expect(shownLabels).toHaveLength(2);
        expect(shownLabels.filter((shownLabel) => shownLabel.closest('tr'))).toHaveLength(1);
      },
    );
  });

  describe('its icon actions', () => {
    it('names the cancel action of a pending and a running verification in the table and the cards', async () => {
      declareList([
        { ...verification('running', 'MANUAL'), status: 'RUNNING' },
        { ...verification('pending', 'MANUAL'), status: 'PENDING' },
      ]);
      renderScreen(<DatabasePage />);
      await screen.findAllByText('Manual');

      expect(screen.getAllByRole('button', { name: runsCopy.actions.cancel })).toHaveLength(4);
      expect(screen.queryByRole('button', { name: runsCopy.actions.viewDetails })).toBeNull();
    });

    it.each(['COMPLETED', 'FAILED', 'CANCELED'] as const)(
      'names the details action of a verification with status %s in the table and the cards',
      async (status) => {
        declareList([{ ...verification('finished', 'MANUAL'), status }]);
        renderScreen(<DatabasePage />);
        await screen.findAllByText('Manual');

        expect(screen.getAllByRole('button', { name: runsCopy.actions.viewDetails })).toHaveLength(
          2,
        );
        expect(screen.queryByRole('button', { name: runsCopy.actions.cancel })).toBeNull();
      },
    );

    it('offers a viewer the details and no cancel', async () => {
      declareList([
        { ...verification('running', 'MANUAL'), status: 'RUNNING' },
        verification('finished', 'MANUAL'),
      ]);
      renderScreen(<DatabasePage isCanManageDBs={false} />);
      await screen.findAllByText('Manual');

      expect(screen.queryByRole('button', { name: runsCopy.actions.cancel })).toBeNull();
      expect(screen.getAllByRole('button', { name: runsCopy.actions.viewDetails })).toHaveLength(2);
    });
  });

  describe('its details', () => {
    const viewDetailsName = runsCopy.actions.viewDetails;

    const declareListedFinished = (...items: Verification[]) => {
      declareList(items.length > 0 ? items : [verification('finished', 'MANUAL')]);
    };

    it('shows the reason with a retry and no alert when the details fail, and the retry shows them', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isDetailsFailing = true;
      declareListedFinished();
      mockServer.use(
        http.get(`*${API_BASE_PATH}/verifications/:verificationId`, () =>
          isDetailsFailing
            ? failedResponse()
            : HttpResponse.json(buildDetails({ schemaCount: 37 })),
        ),
      );
      renderScreen(<DatabasePage />);
      await screen.findAllByText('Manual');

      fireEvent.click(getRowAction(viewDetailsName));

      const drawer = await screen.findByRole('dialog');
      const retryButton = await within(drawer).findByRole('button', { name: 'Retry' });

      expect(within(drawer).getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();

      isDetailsFailing = false;
      fireEvent.click(retryButton);

      expect(await within(drawer).findByText('37')).toBeTruthy();
      expect(within(drawer).queryByText(INTERNAL_ERROR)).toBeNull();
      expect(within(drawer).queryByRole('button', { name: 'Retry' })).toBeNull();
    });

    it('sends no request on close and shows the other verification when it opens next', async () => {
      const requestedIds: unknown[] = [];
      declareListedFinished(
        verification('first', 'MANUAL'),
        verification('second', 'AFTER_BACKUP'),
      );
      mockServer.use(
        http.get(`*${API_BASE_PATH}/verifications/:verificationId`, ({ params }) => {
          requestedIds.push(params.verificationId);

          return HttpResponse.json(
            buildDetails(
              params.verificationId === 'first' ? { schemaCount: 11 } : { schemaCount: 22 },
            ),
          );
        }),
      );
      const { user } = renderScreen(<DatabasePage />);
      await screen.findAllByText('Manual');

      await user.click(screen.getAllByRole('button', { name: viewDetailsName })[0]);
      expect(await screen.findByText('11')).toBeTruthy();

      await user.click(screen.getByRole('button', { name: 'Close' }));
      await waitFor(() => expect(screen.queryByText('11')).toBeNull());

      expect(requestedIds).toEqual(['first']);

      await user.click(screen.getAllByRole('button', { name: viewDetailsName })[1]);

      expect(await screen.findByText('22')).toBeTruthy();
      expect(screen.queryByText('11')).toBeNull();
      expect(requestedIds).toEqual(['first', 'second']);
    });

    it.each([
      { duration: '1h 2m 0s', restoreDurationMs: 3_720_000 },
      { duration: '0m 5s', restoreDurationMs: 5_000 },
      { duration: '0m 0s', restoreDurationMs: 0 },
    ])(
      'shows the same restore duration $duration in the list and the details',
      async ({ duration, restoreDurationMs }) => {
        const finished = { ...verification('finished', 'MANUAL'), restoreDurationMs };
        declareList([finished]);
        mockServer.use(
          mockApi.get('/verifications/{verificationId}', 200, { ...finished, tableStats: [] }),
        );
        renderScreen(<DatabasePage />);
        await screen.findAllByText('Manual');

        expect(screen.getByText(`restore ${duration}`)).toBeTruthy();

        fireEvent.click(getRowAction(viewDetailsName));

        const restoreRowLabel = await within(await screen.findByRole('dialog')).findByText(
          runsCopy.details.fields.restoreDuration,
        );

        expect(restoreRowLabel.parentElement?.textContent).toContain(duration);
      },
    );
  });

  describe('with a failing refresh', () => {
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
      expect(document.querySelector('[role="alert"], [role="status"]')).toBeNull();
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

      fireEvent.click(screen.getAllByRole('button', { name: runsCopy.actions.cancel })[0]);

      await advanceClockUntil(() => expect(screen.getAllByText('After backup')).toHaveLength(2));

      expect(refreshesAfterCancel).toBeGreaterThanOrEqual(2);
      expect(showAlert).not.toHaveBeenCalled();
    });
  });
});
