// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useRef, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { type BackupConfig, backupConfigApi } from '@/entities/backup-configs';
import { DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { physicalBackupsApi } from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import { PhysicalBackupsComponent } from '@/pages/databases/physical-backups/ui/PhysicalBackupsComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { advanceClockUntil } from '@/shared/lib/test/advanceClockUntil';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

dayjs.extend(utc);

const server = setUpMockApi();
const database = {
  id: 'old-database',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  notifierIds: [],
};
const STORAGE_ID = 'storage-id';
const config: BackupConfig = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: database.id,
  storageId: STORAGE_ID,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
    type: 'PHYSICAL',
    fullBackupSchedule: { type: 'HOURLY' },
    incrementalBackupSchedule: null,
    retention: {
      policy: 'FULL_BACKUPS',
      chainsCount: 0,
      fullBackups: {
        policy: 'LAST_N',
        count: 5,
        gfsHours: 0,
        gfsDays: 0,
        gfsWeeks: 0,
        gfsMonths: 0,
        gfsYears: 0,
      },
    },
    walLagThresholdBytes: 0,
    notificationTriggers: [],
  },
};
const usageText = (size: number) =>
  DICTIONARIES.en.backups.list.totalUsage.replace('{{size}}', `${size} MB`);

const INTERNAL_ERROR = DICTIONARIES.en.errors.internal;
const REFUSAL = DICTIONARIES.en.errors.permissionDenied;
const listCopy = DICTIONARIES.en.backups.list;
const physicalListCopy = DICTIONARIES.en.backups.physical.list;

const buildBackup = (change: Partial<PhysicalBackupListItem> = {}): PhysicalBackupListItem => ({
  id: 'completed-backup',
  type: 'FULL',
  status: 'COMPLETED',
  timelineId: 1,
  startLsn: '0/10000000',
  stopLsn: '1/0',
  hasGapBefore: false,
  sizeMb: 111,
  createdAt: '2026-10-27T10:00:00Z',
  ...change,
});

const failedListResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

const UsageLocaleControl = () => {
  const { setLocale } = useLocale();
  return <button onClick={() => setLocale('ru')}>Switch language</button>;
};

// The database page scrolls the list inside its own container and can leave for another tab, so a
// message shown after leaving would still be seen.
const DatabasePage = ({ isCanManageDBs = true }: { isCanManageDBs?: boolean }) => {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [isListOpen, setIsListOpen] = useState(true);

  return (
    <div ref={scrollContainerRef} className="database-scroll-container">
      <button onClick={() => setIsListOpen((isOpen) => !isOpen)}>Toggle tab</button>
      {isListOpen && (
        <PhysicalBackupsComponent
          database={database}
          isCanManageDBs={isCanManageDBs}
          scrollContainerRef={scrollContainerRef}
        />
      )}
    </div>
  );
};

const renderDatabasePage = (isCanManageDBs = true) => {
  const { container } = renderScreen(<DatabasePage isCanManageDBs={isCanManageDBs} />);

  return container.querySelector('.database-scroll-container') as Element;
};

const queryStartButton = () =>
  screen.queryByRole<HTMLButtonElement>('button', {
    name: new RegExp(physicalListCopy.backUpNow),
  });

const getRowAction = (name: string) => screen.getAllByRole('button', { name })[0];

const leaveTab = () => fireEvent.click(screen.getByRole('button', { name: 'Toggle tab' }));

describe('PhysicalBackupsComponent', () => {
  it.each([
    ['Full backup', 'FULL'],
    ['Incremental backup', 'INCREMENTAL'],
  ])('requests %s with the public backup kind when chosen in the menu', async (label, kind) => {
    const requestedBackups: unknown[] = [];
    const user = userEvent.setup();
    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
      http.post(`*${API_BASE_PATH}/physical-backups`, async ({ request }) => {
        requestedBackups.push(await request.json());

        return new HttpResponse(null, { status: 202 });
      }),
    );
    renderScreen(
      <PhysicalBackupsComponent
        database={{
          ...database,
          settings: { ...database.settings, backupType: 'FULL_INCREMENTAL_WAL_STREAM' },
        }}
        isCanManageDBs
      />,
    );
    await screen.findByRole('button', { name: physicalListCopy.backUpNow });

    await user.click(screen.getByRole('button', { name: 'Type' }));
    await user.click(await screen.findByRole('menuitem', { name: label }));

    await waitFor(() => expect(requestedBackups).toHaveLength(1));
    expect(requestedBackups).toEqual([{ databaseId: database.id, kind }]);
  });

  describe('with a failing load', () => {
    it('shows the reason with a retry, ends the spinner and does not say backups are disabled when the config cannot load', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isConfigFailing = true;
      server.use(
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () =>
          isConfigFailing
            ? failedListResponse()
            : HttpResponse.json({ ...config, isBackupsEnabled: false }),
        ),
        mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
      );
      renderDatabasePage();

      const retryButton = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(screen.queryByText(listCopy.scheduledBackupsDisabled)).toBeNull();
      expect(queryStartButton()).toBeTruthy();
      expect(showAlert).not.toHaveBeenCalled();

      isConfigFailing = false;
      fireEvent.click(retryButton);

      expect(await screen.findByText(listCopy.scheduledBackupsDisabled)).toBeTruthy();
    });

    it.each([
      { answer: 'a failure', buildAnswer: failedListResponse },
      {
        answer: 'a loaded config',
        buildAnswer: () => HttpResponse.json({ ...config, isBackupsEnabled: false }),
      },
    ])(
      'changes nothing when the config answers with $answer after the user left',
      async ({ buildAnswer }) => {
        const configLoad = vi.spyOn(backupConfigApi, 'getBackupConfig');
        const heldConfig = createPendingApiResponse();
        server.use(
          http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => heldConfig.response),
          mockApi.get('/physical-backups', 200, { items: [], totalUsageMb: 0 }),
        );
        renderDatabasePage();
        await waitFor(() => expect(configLoad).toHaveBeenCalledOnce());

        leaveTab();
        await releaseAndSettle(configLoad, () => heldConfig.respond(buildAnswer()));

        expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();
        expect(screen.queryByText(listCopy.scheduledBackupsDisabled)).toBeNull();
        expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
        expect(screen.getByRole('button', { name: 'Toggle tab' })).toBeTruthy();
      },
    );

    it('shows a retry, no empty text and no usage and keeps the start disabled when the first page fails', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isListFailing = true;
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () =>
          isListFailing
            ? failedListResponse()
            : HttpResponse.json({ items: [buildBackup()], totalUsageMb: 10 }),
        ),
      );
      renderDatabasePage();

      const retryButton = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(screen.queryByText(listCopy.empty)).toBeNull();
      expect(screen.queryByText(/Total/)).toBeNull();
      expect(queryStartButton()?.disabled).toBe(true);
      expect(showAlert).not.toHaveBeenCalled();

      isListFailing = false;
      fireEvent.click(retryButton);

      expect(await screen.findByText(usageText(10))).toBeTruthy();
      expect(queryStartButton()?.disabled).toBe(false);
    });

    it('shows no usage before the first page has loaded', async () => {
      const heldList = createPendingApiResponse();
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () => heldList.response),
      );
      renderDatabasePage();

      await waitFor(() => expect(queryStartButton()).toBeTruthy());

      expect(screen.queryByText(usageText(0))).toBeNull();
    });

    it('keeps the rows, reports once and requests again on a later scroll when the next page fails', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      const requestedCursors: (string | null)[] = [];
      let isNextPageFailing = true;
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', ({ request }) => {
          const cursor = new URL(request.url).searchParams.get('after');
          requestedCursors.push(cursor);

          if (!cursor) {
            return HttpResponse.json({
              items: [buildBackup()],
              totalUsageMb: 10,
              nextCursor: 'first-page-cursor',
            });
          }

          return isNextPageFailing
            ? failedListResponse()
            : HttpResponse.json({
                items: [buildBackup({ id: 'oldest-backup', sizeMb: 222 })],
                totalUsageMb: 10,
              });
        }),
      );
      const scrollContainer = renderDatabasePage();
      await screen.findAllByText('111 MB');

      fireEvent.scroll(scrollContainer);

      expect(await screen.findAllByText(INTERNAL_ERROR)).toHaveLength(1);
      expect(screen.getAllByText('111 MB').length).toBeGreaterThan(0);
      expect(showAlert).not.toHaveBeenCalled();

      isNextPageFailing = false;
      fireEvent.scroll(scrollContainer);

      expect((await screen.findAllByText('222 MB')).length).toBeGreaterThan(0);
      expect(screen.getAllByText(INTERNAL_ERROR)).toHaveLength(1);
      expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual([
        'first-page-cursor',
        'first-page-cursor',
      ]);
    });
  });

  describe('for a user role', () => {
    const declareListedBackups = () =>
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () =>
          HttpResponse.json({
            items: [
              buildBackup(),
              buildBackup({ id: 'running-backup', status: 'IN_PROGRESS', sizeMb: 222 }),
            ],
            totalUsageMb: 10,
          }),
        ),
      );

    it('offers a viewer restore and no start, cancel or delete', async () => {
      declareListedBackups();
      renderDatabasePage(false);
      await screen.findAllByText('111 MB');

      expect(queryStartButton()).toBeNull();
      expect(screen.queryByRole('button', { name: listCopy.actions.cancelBackup })).toBeNull();
      expect(screen.queryByRole('button', { name: listCopy.actions.deleteBackup })).toBeNull();
      expect(screen.getByRole('button', { name: physicalListCopy.restore })).toBeTruthy();
      expect(getRowAction(physicalListCopy.restoreFromThisBackup)).toBeTruthy();
    });

    it('offers a member the backup start', async () => {
      declareListedBackups();
      renderDatabasePage();
      await screen.findAllByText('111 MB');

      expect(queryStartButton()).toBeTruthy();
    });

    it.each([
      {
        condition: 'backups are turned off',
        change: { isBackupsEnabled: false, storageId: STORAGE_ID },
      },
      {
        condition: 'backups are on and no storage is named',
        change: { isBackupsEnabled: true, storageId: null },
      },
    ])('disables the start and says what it needs when $condition', async ({ change }) => {
      declareListedBackups();
      server.use(mockApi.get('/backup-configs/{databaseId}', 200, { ...config, ...change }));
      renderDatabasePage();
      await screen.findAllByText('111 MB');

      const startMenuTrigger = screen.getByRole<HTMLButtonElement>('button', { name: 'Type' });

      fireEvent.click(queryStartButton() as HTMLButtonElement);
      fireEvent.click(startMenuTrigger);

      expect(queryStartButton()?.disabled).toBe(true);
      expect(startMenuTrigger.disabled).toBe(true);
      expect(screen.queryByText(physicalListCopy.fullBackup)).toBeNull();
      expect(server.getRequests().filter(({ method }) => method === 'POST')).toHaveLength(0);
      expect(screen.getByText(physicalListCopy.startRequirement)).toBeTruthy();
      expect(screen.queryAllByText(listCopy.scheduledBackupsDisabled)).toHaveLength(
        change.isBackupsEnabled === false ? 1 : 0,
      );
    });

    it('keeps the start enabled without saying what it needs when the config fails to load', async () => {
      declareListedBackups();
      server.use(
        http.get(`*${API_BASE_PATH}/backup-configs/:databaseId`, () => failedListResponse()),
      );
      renderDatabasePage();
      await screen.findAllByText('111 MB');

      expect(queryStartButton()?.disabled).toBe(false);
      expect(screen.queryByText(physicalListCopy.startRequirement)).toBeNull();
    });
  });

  it.each([
    { status: 'CHAIN_BROKEN', label: DICTIONARIES.en.status.physicalBackup.chainBroken },
    { status: 'CANCELED', label: DICTIONARIES.en.status.physicalBackup.canceled },
  ] as const)(
    'shows $label and offers no row action with a $status backup',
    async ({ status, label }) => {
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        mockApi.get('/physical-backups', 200, {
          items: [buildBackup({ status })],
          totalUsageMb: 10,
        }),
      );
      renderDatabasePage();

      expect(await screen.findAllByText(label)).toHaveLength(2);

      for (const actionName of [
        listCopy.actions.cancelBackup,
        listCopy.actions.deleteBackup,
        physicalListCopy.restoreFromThisBackup,
      ]) {
        expect(screen.queryByRole('button', { name: actionName })).toBeNull();
      }
    },
  );

  it('keeps the tail and shows no loading state after a manual backup, and shows the new backup through a later poll', async () => {
    const requestedCursors: (string | null)[] = [];
    let isNewBackupListed = false;
    let startRequests = 0;
    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.post('*/api/v1/physical-backups', () => {
        startRequests += 1;

        return new HttpResponse(null, { status: 202 });
      }),
      http.get('*/api/v1/physical-backups', ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        if (cursor) {
          return HttpResponse.json({
            items: [buildBackup({ id: 'oldest-backup', sizeMb: 222 })],
            totalUsageMb: 10,
          });
        }

        return HttpResponse.json({
          items: isNewBackupListed
            ? [buildBackup({ id: 'new-backup', sizeMb: 333 }), buildBackup()]
            : [buildBackup()],
          totalUsageMb: 10,
          nextCursor: 'first-page-cursor',
        });
      }),
    );
    vi.useFakeTimers();
    const scrollContainer = renderDatabasePage();
    await waitForCondition(() => expect(screen.getAllByText('111 MB').length).toBeGreaterThan(0));
    fireEvent.scroll(scrollContainer);
    await waitForCondition(() => expect(screen.getAllByText('222 MB').length).toBeGreaterThan(0));

    fireEvent.click(queryStartButton() as HTMLButtonElement);
    await waitForCondition(() => expect(startRequests).toBe(1));
    await waitForCondition(() => expect(queryStartButton()?.disabled).toBe(false));
    await advanceFakeClock(2_000);

    expect(screen.getAllByText('222 MB').length).toBeGreaterThan(0);
    expect(document.querySelector('[role="status"]')).toBeNull();
    expect(requestedCursors.filter((cursor) => cursor !== null)).toEqual(['first-page-cursor']);

    isNewBackupListed = true;

    await advanceClockUntil(() => expect(screen.getAllByText('333 MB').length).toBeGreaterThan(0));

    const shownRows = screen.getAllByRole('row').slice(1);
    expect(shownRows.map((row) => within(row).getByText(/^(333|111|222) MB$/).textContent)).toEqual(
      ['333 MB', '111 MB', '222 MB'],
    );
    expect(startRequests).toBe(1);
  });

  describe('its one-shot actions', () => {
    const renderListedBackup = async (status: PhysicalBackupListItem['status']) => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () =>
          HttpResponse.json({ items: [buildBackup({ status })], totalUsageMb: 10 }),
        ),
      );
      renderDatabasePage();
      await screen.findAllByText('111 MB');

      return showAlert;
    };

    const clickDeleteAndConfirm = async () => {
      fireEvent.click(getRowAction(listCopy.actions.deleteBackup));
      fireEvent.click(await waitForCondition(() => screen.getByRole('button', { name: 'Delete' })));
    };

    it('reports a refused start once, keeps the rows and leaves the start usable', async () => {
      const showAlert = await renderListedBackup('COMPLETED');
      server.use(mockApi.post('/physical-backups', 403, buildProblem('permission_denied', 403)));

      fireEvent.click(queryStartButton() as HTMLButtonElement);

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getAllByText('111 MB').length).toBeGreaterThan(0);
      expect(showAlert).not.toHaveBeenCalled();
      await waitFor(() => expect(queryStartButton()?.disabled).toBe(false));
      expect(screen.getAllByText(REFUSAL)).toHaveLength(1);
    });

    it('reports a refused cancel once, keeps the rows and leaves the action usable', async () => {
      const showAlert = await renderListedBackup('IN_PROGRESS');
      server.use(
        mockApi.post(
          '/physical-backups/{backupId}/cancel',
          403,
          buildProblem('permission_denied', 403),
        ),
      );

      fireEvent.click(getRowAction(listCopy.actions.cancelBackup));

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getAllByText('111 MB').length).toBeGreaterThan(0);
      expect(showAlert).not.toHaveBeenCalled();
      await waitFor(() =>
        expect((getRowAction(listCopy.actions.cancelBackup) as HTMLButtonElement).disabled).toBe(
          false,
        ),
      );
      expect(screen.getAllByText(REFUSAL)).toHaveLength(1);
    });

    it('reports a refused delete once, keeps the rows and leaves the action usable', async () => {
      const showAlert = await renderListedBackup('COMPLETED');
      server.use(
        mockApi.delete('/physical-backups/{backupId}', 403, buildProblem('permission_denied', 403)),
      );

      await clickDeleteAndConfirm();

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getAllByText('111 MB').length).toBeGreaterThan(0);
      expect(showAlert).not.toHaveBeenCalled();
      await waitFor(() =>
        expect((getRowAction(listCopy.actions.deleteBackup) as HTMLButtonElement).disabled).toBe(
          false,
        ),
      );
      expect(screen.getAllByText(REFUSAL)).toHaveLength(1);
    });

    const declareHeldAnswer = () => {
      const heldAnswer = createPendingApiResponse();

      server.use(
        http.post('*/api/v1/physical-backups', () => heldAnswer.response),
        http.post('*/api/v1/physical-backups/:backupId/cancel', () => heldAnswer.response),
        http.delete('*/api/v1/physical-backups/:backupId', () => heldAnswer.response),
      );

      return heldAnswer;
    };

    interface HeldAction {
      action: string;
      status: PhysicalBackupListItem['status'];
      apiMethod: 'triggerPhysicalBackup' | 'cancelPhysicalBackup' | 'deletePhysicalBackup';
      buildAcceptedAnswer: () => Response;
      press: () => unknown;
    }

    const HELD_ACTIONS: HeldAction[] = [
      {
        action: 'start',
        status: 'COMPLETED',
        apiMethod: 'triggerPhysicalBackup',
        buildAcceptedAnswer: () => new HttpResponse(null, { status: 202 }),
        press: () => fireEvent.click(queryStartButton() as HTMLButtonElement),
      },
      {
        action: 'cancel',
        status: 'IN_PROGRESS',
        apiMethod: 'cancelPhysicalBackup',
        buildAcceptedAnswer: () => HttpResponse.json(buildBackup({ status: 'CANCELED' })),
        press: () => fireEvent.click(getRowAction(listCopy.actions.cancelBackup)),
      },
      {
        action: 'delete',
        status: 'COMPLETED',
        apiMethod: 'deletePhysicalBackup',
        buildAcceptedAnswer: () => new HttpResponse(null, { status: 204 }),
        press: clickDeleteAndConfirm,
      },
    ];

    it.each(HELD_ACTIONS)(
      'shows nothing when a refused $action answers after the user left',
      async ({ status, apiMethod, press }) => {
        const showAlert = await renderListedBackup(status);
        const heldAnswer = declareHeldAnswer();
        const actionCall = vi.spyOn(physicalBackupsApi, apiMethod);

        await press();
        await waitFor(() => expect(actionCall).toHaveBeenCalledOnce());
        leaveTab();
        await releaseAndSettle(actionCall, () =>
          heldAnswer.respond(
            HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
          ),
        );

        expect(screen.queryByText(REFUSAL)).toBeNull();
        expect(showAlert).not.toHaveBeenCalled();
      },
    );

    it.each(HELD_ACTIONS)(
      'shows nothing when a refused $action answers after the session changed',
      async ({ status, apiMethod, press }) => {
        activateSession({ token: 'first-token', userId: 'first-user' });
        const showAlert = await renderListedBackup(status);
        const heldAnswer = declareHeldAnswer();
        const actionCall = vi.spyOn(physicalBackupsApi, apiMethod);

        await press();
        await waitFor(() => expect(actionCall).toHaveBeenCalledOnce());
        act(() => {
          invalidateSession();
        });
        await releaseAndSettle(actionCall, () =>
          heldAnswer.respond(
            HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
          ),
        );

        expect(screen.queryByText(REFUSAL)).toBeNull();
        expect(showAlert).not.toHaveBeenCalled();
      },
    );

    // The frozen clock holds the polls, so any later list request comes from the action itself.
    const renderAcceptedAction = async ({ status, apiMethod, press }: HeldAction) => {
      vi.useFakeTimers();
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () =>
          HttpResponse.json({ items: [buildBackup({ status })], totalUsageMb: 10 }),
        ),
      );
      const listCall = vi.spyOn(physicalBackupsApi, 'getPhysicalBackups');
      const actionCall = vi.spyOn(physicalBackupsApi, apiMethod);
      renderDatabasePage();
      await waitForCondition(() => expect(screen.getAllByText('111 MB').length).toBeGreaterThan(0));
      const heldAnswer = declareHeldAnswer();

      await press();
      await waitForCondition(() => expect(actionCall).toHaveBeenCalledOnce());

      return { heldAnswer, listCall, actionCall };
    };

    it.each(HELD_ACTIONS)(
      'requests no list and shows nothing when an accepted $action answers after the user left',
      async (heldAction) => {
        const { heldAnswer, listCall, actionCall } = await renderAcceptedAction(heldAction);

        leaveTab();
        const listCallsWhenLeaving = listCall.mock.calls.length;
        await releaseAndSettle(actionCall, () =>
          heldAnswer.respond(heldAction.buildAcceptedAnswer()),
        );

        expect(listCall.mock.calls).toHaveLength(listCallsWhenLeaving);
        expect(document.querySelector('[role="alert"], [role="status"]')).toBeNull();
        expect(screen.getByRole('button', { name: 'Toggle tab' })).toBeTruthy();
      },
    );

    it.each(HELD_ACTIONS.filter(({ action }) => action !== 'start'))(
      'requests no list and clears protected rows when an accepted $action answers after the session changed',
      async (heldAction) => {
        activateSession({ token: 'first-token', userId: 'first-user' });
        const { heldAnswer, listCall, actionCall } = await renderAcceptedAction(heldAction);

        act(() => {
          invalidateSession();
        });
        const listCallsWhenChanged = listCall.mock.calls.length;
        await releaseAndSettle(actionCall, () =>
          heldAnswer.respond(heldAction.buildAcceptedAnswer()),
        );

        expect(listCall.mock.calls).toHaveLength(listCallsWhenChanged);
        expect(screen.queryAllByText('111 MB')).toHaveLength(0);
      },
    );
  });

  it('keeps the current usage when an old refresh answers after the replacement list loads', async () => {
    const oldRefresh = createPendingApiResponse();
    let oldRequestCount = 0;

    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.get('*/api/v1/physical-backups', ({ request }) => {
        if (new URL(request.url).searchParams.get('databaseId') === 'new-database') {
          return HttpResponse.json({ items: [], totalUsageMb: 30 });
        }
        oldRequestCount++;
        return oldRequestCount === 1
          ? HttpResponse.json({ items: [], totalUsageMb: 10 })
          : oldRefresh.response;
      }),
    );

    vi.useFakeTimers();
    const view = renderScreen(
      <PhysicalBackupsComponent database={database} isCanManageDBs={false} />,
    );

    await waitForCondition(() => expect(screen.getByText(usageText(10))).toBeTruthy());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });
    await waitForCondition(() => expect(oldRequestCount).toBe(2));

    view.rerender(
      <PhysicalBackupsComponent
        database={{ ...database, id: 'new-database' }}
        isCanManageDBs={false}
      />,
    );
    await waitForCondition(() => expect(screen.getByText(usageText(30))).toBeTruthy());

    const oldResponseProcessed = new Promise<void>((resolve) => {
      server.events.on('response:mocked', ({ request }) => {
        if (new URL(request.url).searchParams.get('databaseId') === 'old-database') resolve();
      });
    });

    await act(async () => {
      oldRefresh.respond(HttpResponse.json({ items: [], totalUsageMb: 20 }));
      await oldResponseProcessed;
    });

    expect(screen.getByText(usageText(30))).toBeTruthy();
    expect(screen.queryByText(usageText(20))).toBeNull();
  });

  describe('with a failing refresh', () => {
    const physicalBackup = buildBackup({
      id: 'running-backup',
      status: 'IN_PROGRESS',
      sizeMb: 12,
      createdAt: '2026-10-27T10:00:00Z',
    });

    const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

    const countListRequests = () =>
      server.getRequests().filter((request) => request.url.includes('/physical-backups?')).length;

    it('shows no alert, keeps the rows and requests again when polls fail', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let listRequests = 0;
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () =>
          ++listRequests === 1
            ? HttpResponse.json({ items: [physicalBackup], totalUsageMb: 10 })
            : failedResponse(),
        ),
      );
      vi.useFakeTimers();
      renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs={false} />);
      await waitForCondition(() => screen.getByText(usageText(10)));
      const shownRunningBackups = screen.getAllByText('In progress').length;

      await advanceClockUntil(() => expect(countListRequests()).toBeGreaterThanOrEqual(3));

      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.getByText(usageText(10))).toBeTruthy();
      expect(shownRunningBackups).toBeGreaterThan(0);
      expect(screen.getAllByText('In progress')).toHaveLength(shownRunningBackups);
    });

    it('shows no alert and refreshes again when the refresh after a successful cancel fails', async () => {
      const showAlert = vi.fn();
      vi.stubGlobal('alert', showAlert);
      let isCancelled = false;
      let refreshesAfterCancel = 0;
      server.use(
        mockApi.get('/backup-configs/{databaseId}', 200, config),
        http.get('*/api/v1/physical-backups', () => {
          if (!isCancelled) {
            return HttpResponse.json({ items: [physicalBackup], totalUsageMb: 10 });
          }

          refreshesAfterCancel += 1;
          return refreshesAfterCancel === 1
            ? failedResponse()
            : HttpResponse.json({
                items: [{ ...physicalBackup, status: 'CANCELED' }],
                totalUsageMb: 20,
              });
        }),
        http.post('*/api/v1/physical-backups/:backupId/cancel', () => {
          isCancelled = true;
          return HttpResponse.json({ ...physicalBackup, status: 'CANCELED' });
        }),
      );
      vi.useFakeTimers();
      renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs />);
      await waitForCondition(() => screen.getByText(usageText(10)));

      fireEvent.click(getRowAction(listCopy.actions.cancelBackup));

      await advanceClockUntil(() => screen.getByText(usageText(20)));

      expect(refreshesAfterCancel).toBeGreaterThanOrEqual(2);
      expect(showAlert).not.toHaveBeenCalled();
    });
  });
});

describe('PhysicalBackupsComponent usage', () => {
  it('updates the Size label and decimal separator when the interface language changes', async () => {
    let requestCount = 0;
    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.get('*/api/v1/physical-backups', () => {
        requestCount++;
        return HttpResponse.json<ApiComponents['schemas']['PhysicalBackupPage']>({
          items: [],
          totalUsageMb: 10752,
        });
      }),
    );
    vi.useFakeTimers();
    renderScreen(
      <>
        <UsageLocaleControl />
        <PhysicalBackupsComponent database={database} isCanManageDBs={false} />
      </>,
    );
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*10.5 GB/ })).toBeTruthy(),
    );
    const requestsBeforeLanguageChange = requestCount;

    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

    await waitForCondition(() =>
      expect(
        screen.getByRole('columnheader', {
          name: `${DICTIONARIES.ru.backups.list.columns.size} 10,5 GB`,
        }),
      ).toBeTruthy(),
    );
    expect(requestCount).toBe(requestsBeforeLanguageChange);
    expect(screen.queryByRole('columnheader', { name: /^Size/ })).toBeNull();
  });

  it('updates the inline total when only aggregate metadata changes', async () => {
    let totalUsageMb = 104;
    const backupRows = [buildBackup()];
    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.get('*/api/v1/physical-backups', () =>
        HttpResponse.json({ items: backupRows, totalUsageMb }),
      ),
    );
    vi.useFakeTimers();
    renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs={false} />);
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*104 MB/ })).toBeTruthy(),
    );

    totalUsageMb = 10752;
    await advanceClockUntil(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*10.5 GB/ })).toBeTruthy(),
    );

    totalUsageMb = 0;
    await advanceClockUntil(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*0 MB/ })).toBeTruthy(),
    );
  });

  it('hides the previous total and waits for explicit retry when reloading after deletion fails', async () => {
    let hasBackupBeenDeleted = false;
    let hasReadRecovered = false;
    let requestCount = 0;
    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.get('*/api/v1/physical-backups', () => {
        requestCount++;

        if (hasReadRecovered)
          return HttpResponse.json<ApiComponents['schemas']['PhysicalBackupPage']>({
            items: [],
            totalUsageMb: 0,
          });

        if (hasBackupBeenDeleted)
          return HttpResponse.json(buildProblem('internal', 500), { status: 500 });

        return HttpResponse.json<ApiComponents['schemas']['PhysicalBackupPage']>({
          items: [buildBackup({ sizeMb: 104 })],
          totalUsageMb: 104,
        });
      }),
      http.delete('*/api/v1/physical-backups/:backupId', () => {
        hasBackupBeenDeleted = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    vi.useFakeTimers();
    renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs />);
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*104 MB/ })).toBeTruthy(),
    );

    fireEvent.click(screen.getAllByRole('button', { name: 'Delete backup' })[0]);
    await waitForCondition(() =>
      expect(screen.getByRole('button', { name: 'Delete' })).toBeTruthy(),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    await waitForCondition(() =>
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
    );
    expect(screen.queryByRole('columnheader', { name: /^Size.*MB/ })).toBeNull();
    expect(screen.queryByText('Total usage: 104 MB')).toBeNull();
    const requestsBeforeRetry = requestCount;

    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    fireEvent(document, new Event('visibilitychange'));
    await advanceFakeClock(3000);
    expect(requestCount).toBe(requestsBeforeRetry);

    visibility.mockReturnValue('visible');
    fireEvent(document, new Event('visibilitychange'));
    await advanceFakeClock(3000);
    expect(requestCount).toBe(requestsBeforeRetry);
    expect(screen.queryByRole('columnheader', { name: /^Size.*MB/ })).toBeNull();
    expect(screen.queryByText('Total usage: 104 MB')).toBeNull();

    hasReadRecovered = true;
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await waitForCondition(() =>
      expect(screen.getByRole('columnheader', { name: /^Size.*0 MB/ })).toBeTruthy(),
    );
    await advanceClockUntil(() => expect(requestCount).toBeGreaterThan(requestsBeforeRetry + 1));
    expect(screen.queryByRole('columnheader', { name: /^Size.*104 MB/ })).toBeNull();
  });

  it('requires explicit retry after a failed first page despite polling and visibility changes', async () => {
    let requestCount = 0;
    server.use(
      mockApi.get('/backup-configs/{databaseId}', 200, config),
      http.get('*/api/v1/physical-backups', () => {
        requestCount++;
        return HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });
      }),
    );
    vi.useFakeTimers();
    renderScreen(<PhysicalBackupsComponent database={database} isCanManageDBs={false} />);
    await waitForCondition(() =>
      expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy(),
    );

    fireEvent(document, new Event('visibilitychange'));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });

    expect(requestCount).toBe(1);
    expect(screen.queryByRole('columnheader', { name: /^Size.*MB/ })).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await waitForCondition(() => expect(requestCount).toBe(2));
  });
});
