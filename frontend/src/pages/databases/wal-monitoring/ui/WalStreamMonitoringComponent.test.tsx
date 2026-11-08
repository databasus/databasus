// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { type MockInstance, describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { physicalBackupsApi } from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { physicalWalStreamStatisticsApi } from '@/pages/databases/wal-monitoring/api/physicalWalStreamStatisticsApi';
import { WAL_STREAM_LIVE_INTERVAL_MS } from '@/pages/databases/wal-monitoring/lib/walStreamLiveController';
import { WAL_STREAM_REFRESH_INTERVAL_MS } from '@/pages/databases/wal-monitoring/lib/walStreamRefreshController';
import type { WalStreamLiveResponse } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import { WalStreamHealthLevel } from '@/pages/databases/wal-monitoring/model/WalStreamHealthLevel';
import { WalStreamReasonCode } from '@/pages/databases/wal-monitoring/model/WalStreamReasonCode';
import { WalStreamReasonLevel } from '@/pages/databases/wal-monitoring/model/WalStreamReasonLevel';
import { WalStreamMonitoringComponent } from '@/pages/databases/wal-monitoring/ui/WalStreamMonitoringComponent';
import { API_BASE_PATH } from '@/shared/api';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { advanceClockUntil } from '@/shared/lib/test/advanceClockUntil';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { createTestWalStreamStatistics } from '@/shared/lib/test/buildWalStreamStatistics';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { buildProblem, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();

const database: Database = {
  id: 'wal-database',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  notifierIds: [],
};

const brokenChainStatistics = createTestWalStreamStatistics({
  health: {
    level: WalStreamHealthLevel.BROKEN,
    reasons: [{ code: WalStreamReasonCode.WAL_CHAIN_BROKEN, level: WalStreamReasonLevel.BROKEN }],
    notices: [],
    isFullBackupPending: false,
  },
});

const pendingFullBackupStatistics = createTestWalStreamStatistics({
  health: { ...brokenChainStatistics.health, isFullBackupPending: true },
});

const MiB = 1024 * 1024;

const copy = DICTIONARIES.en.backups.physical.monitoring.banner;
const russianCopy = DICTIONARIES.ru.backups.physical.monitoring.banner;

// The tab sits in a page that stays on the screen, so a message shown after leaving would be seen.
const DatabasePage = () => {
  const { setLocale } = useLocale();
  const [isMonitoringOpen, setIsMonitoringOpen] = useState(true);

  return (
    <>
      <button onClick={() => setLocale('ru')}>Switch language</button>
      <button onClick={() => setIsMonitoringOpen((isOpen) => !isOpen)}>Toggle tab</button>
      {isMonitoringOpen && (
        <WalStreamMonitoringComponent database={database} isCanManageDBs onOpenConfig={() => {}} />
      )}
    </>
  );
};

const declareStatistics = (answerStatistics: () => Response | Promise<Response>) => {
  const statisticsRequests = { count: 0 };

  server.use(
    http.get(`*${API_BASE_PATH}/wal-streams/:databaseId/statistics`, () => {
      statisticsRequests.count += 1;

      return answerStatistics();
    }),
    http.get(
      `*${API_BASE_PATH}/wal-streams/:databaseId`,
      () => createPendingApiResponse().response,
    ),
  );

  return statisticsRequests;
};

const declareFullBackupRequest = (answer: () => Response | Promise<Response>) => {
  const fullBackupRequests = { count: 0 };

  server.use(
    http.post(`*${API_BASE_PATH}/physical-backups`, () => {
      fullBackupRequests.count += 1;

      return answer();
    }),
  );

  return fullBackupRequests;
};

const findStartButton = (name: string = copy.startFullBackup) =>
  // Only fixed test fixture names reach this selector.
  // nosemgrep: javascript.lang.security.audit.detect-non-literal-regexp.detect-non-literal-regexp
  waitForCondition(() => screen.getByRole('button', { name: new RegExp(name) }));

const isButtonBusy = (button: HTMLElement) => button.getAttribute('aria-busy') === 'true';

const pressStartButton = async () => {
  fireEvent.click(await findStartButton());

  await waitForCondition(() => expect(physicalBackupsTrigger).toHaveBeenCalledOnce());
};

let physicalBackupsTrigger: MockInstance<typeof physicalBackupsApi.triggerPhysicalBackup>;

const renderDatabasePage = () => {
  vi.useFakeTimers({ now: new Date('2026-10-18T12:00:00Z') });
  physicalBackupsTrigger = vi.spyOn(physicalBackupsApi, 'triggerPhysicalBackup');

  return renderScreen(<DatabasePage />);
};

const answerTriggerAndSettle = (release: () => void) =>
  releaseAndSettle(physicalBackupsTrigger, release);

const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });

describe('WalStreamMonitoringComponent', () => {
  it('names the chart period switch', async () => {
    declareStatistics(() => HttpResponse.json(brokenChainStatistics));
    renderDatabasePage();

    await waitForCondition(() =>
      expect(
        screen.getByRole('radiogroup', { name: DICTIONARIES.en.healthcheck.attempts.period }),
      ).toBeTruthy(),
    );
  });

  describe('with a running full backup request', () => {
    it('keeps the button busy through a language switch and a scheduled refresh until the refresh after the answer lands', async () => {
      const fullBackupAnswer = createPendingApiResponse();
      const scheduledRefresh = createPendingApiResponse();
      const refreshAfterAnswer = createPendingApiResponse();
      const heldRefreshes = [scheduledRefresh, refreshAfterAnswer];
      const statisticsRequests = declareStatistics(() =>
        statisticsRequests.count === 1
          ? HttpResponse.json(brokenChainStatistics)
          : heldRefreshes.shift()!.response,
      );
      const fullBackupRequests = declareFullBackupRequest(() => fullBackupAnswer.response);
      const statisticsLoad = vi.spyOn(physicalWalStreamStatisticsApi, 'getWalStreamStatistics');
      renderDatabasePage();

      await pressStartButton();
      fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));
      await findStartButton(russianCopy.startFullBackup);
      await advanceFakeClock(WAL_STREAM_REFRESH_INTERVAL_MS);
      await waitForCondition(() => expect(statisticsRequests.count).toBe(2));
      await releaseAndSettle(statisticsLoad, () =>
        scheduledRefresh.respond(HttpResponse.json(brokenChainStatistics)),
      );

      const busyButton = await findStartButton(russianCopy.startFullBackup);
      expect(isButtonBusy(busyButton)).toBe(true);

      fireEvent.click(busyButton);
      await answerTriggerAndSettle(() =>
        fullBackupAnswer.respond(new HttpResponse(null, { status: 202 })),
      );
      await waitForCondition(() => expect(statisticsRequests.count).toBe(3));

      expect(isButtonBusy(await findStartButton(russianCopy.startFullBackup))).toBe(true);
      expect(fullBackupRequests.count).toBe(1);

      await releaseAndSettle(statisticsLoad, () =>
        refreshAfterAnswer.respond(HttpResponse.json(pendingFullBackupStatistics)),
      );

      await waitForCondition(() =>
        expect(
          screen.queryByRole('button', { name: new RegExp(russianCopy.startFullBackup) }),
        ).toBeNull(),
      );
      expect(
        screen.getByText(
          DICTIONARIES.ru.backups.physical.monitoring.reasons.walChainBroken.whatToDoPending,
        ),
      ).toBeTruthy();
    });

    it('shows a failure in the language chosen during the request and offers the button again', async () => {
      const fullBackupAnswer = createPendingApiResponse();
      declareStatistics(() => HttpResponse.json(brokenChainStatistics));
      declareFullBackupRequest(() => fullBackupAnswer.response);
      renderDatabasePage();

      await pressStartButton();
      fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));
      await findStartButton(russianCopy.startFullBackup);
      await answerTriggerAndSettle(() => fullBackupAnswer.respond(failedResponse()));

      await waitForCondition(() => screen.getByText(DICTIONARIES.ru.errors.internal));
      expect(screen.queryByText(DICTIONARIES.en.errors.internal)).toBeNull();
      expect(isButtonBusy(await findStartButton(russianCopy.startFullBackup))).toBe(false);
    });

    it('shows a success in the language chosen during the request', async () => {
      const fullBackupAnswer = createPendingApiResponse();
      declareStatistics(() => HttpResponse.json(brokenChainStatistics));
      declareFullBackupRequest(() => fullBackupAnswer.response);
      renderDatabasePage();

      await pressStartButton();
      fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));
      await findStartButton(russianCopy.startFullBackup);
      await answerTriggerAndSettle(() =>
        fullBackupAnswer.respond(new HttpResponse(null, { status: 202 })),
      );

      await waitForCondition(() => screen.getByText(russianCopy.fullBackupRequested));
      expect(screen.queryByText(copy.fullBackupRequested)).toBeNull();
    });

    it('shows no message, sends no statistics request and leaves no refresh running after the user leaves', async () => {
      const fullBackupAnswer = createPendingApiResponse();
      const statisticsRequests = declareStatistics(() => HttpResponse.json(brokenChainStatistics));
      declareFullBackupRequest(() => fullBackupAnswer.response);
      renderDatabasePage();

      await pressStartButton();
      fireEvent.click(screen.getByRole('button', { name: 'Toggle tab' }));
      const statisticsRequestsWhenLeaving = statisticsRequests.count;
      const timersWhenLeaving = vi.getTimerCount();

      await answerTriggerAndSettle(() =>
        fullBackupAnswer.respond(new HttpResponse(null, { status: 202 })),
      );
      expect(vi.getTimerCount()).toBeLessThanOrEqual(timersWhenLeaving);

      await advanceFakeClock(WAL_STREAM_REFRESH_INTERVAL_MS * 2);

      expect(screen.queryByText(copy.fullBackupRequested)).toBeNull();
      expect(statisticsRequests.count).toBe(statisticsRequestsWhenLeaving);
    });

    it('shows no error after the user leaves and offers a working button on return', async () => {
      const fullBackupAnswer = createPendingApiResponse();
      declareStatistics(() => HttpResponse.json(brokenChainStatistics));
      const fullBackupRequests = declareFullBackupRequest(() => fullBackupAnswer.response);
      renderDatabasePage();

      await pressStartButton();
      fireEvent.click(screen.getByRole('button', { name: 'Toggle tab' }));
      await answerTriggerAndSettle(() => fullBackupAnswer.respond(failedResponse()));

      expect(screen.queryByText(DICTIONARIES.en.errors.internal)).toBeNull();

      fireEvent.click(screen.getByRole('button', { name: 'Toggle tab' }));
      const offeredButton = await findStartButton();
      expect(isButtonBusy(offeredButton)).toBe(false);

      fireEvent.click(offeredButton);

      await waitForCondition(() => expect(fullBackupRequests.count).toBe(2));
    });

    it('keeps the message and the busy button when the browser tab is hidden during the request', async () => {
      const fullBackupAnswer = createPendingApiResponse();
      const refreshAfterAnswer = createPendingApiResponse();
      let isAnswered = false;
      declareStatistics(() =>
        isAnswered ? refreshAfterAnswer.response : HttpResponse.json(brokenChainStatistics),
      );
      declareFullBackupRequest(() => fullBackupAnswer.response);
      renderDatabasePage();

      await pressStartButton();
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
      document.dispatchEvent(new Event('visibilitychange'));

      isAnswered = true;
      await answerTriggerAndSettle(() =>
        fullBackupAnswer.respond(new HttpResponse(null, { status: 202 })),
      );

      await waitForCondition(() => screen.getByText(copy.fullBackupRequested));
      expect(isButtonBusy(await findStartButton())).toBe(true);

      await act(async () => {
        refreshAfterAnswer.respond(HttpResponse.json(pendingFullBackupStatistics));
      });

      await waitForCondition(() =>
        expect(screen.queryByRole('button', { name: new RegExp(copy.startFullBackup) })).toBeNull(),
      );
    });
  });
});

describe('WalStreamMonitoringComponent readings', () => {
  const healthyStatistics = createTestWalStreamStatistics();

  // Newer than the sample of the statistics fixture, so the screen lays it over the statistics.
  const buildLiveReading = (unarchivedWalBytes: number): WalStreamLiveResponse => ({
    isStreamerRunning: true,
    current: {
      measuredAt: '2026-10-18T12:00:01Z',
      sourceProbedAt: '2026-10-18T12:00:01Z',
      sourceState: 'REACHABLE',
      sourceCurrentLsn: '0/5000000',
      slotWalStatus: 'reserved',
      slotLagBytes: 2 * MiB,
      slotSafeWalSizeBytes: null,
      archiveProgressLsn: '0/4F00000',
      lastArchivedWalAt: '2026-10-18T11:59:20Z',
      unarchivedWalBytes,
      localQueueBytes: 0,
      localQueuePauseBytes: 512 * MiB,
      isReceiverRunning: true,
    },
  });

  const buildStatisticsWaiting = (unarchivedWalBytes: number) =>
    createTestWalStreamStatistics({
      keepingUp: { ...healthyStatistics.keepingUp, unarchivedWalBytes },
    });

  const declareLiveReadings = (answerReading: () => Response) => {
    const liveReadings = { count: 0 };

    server.use(
      http.get(`*${API_BASE_PATH}/wal-streams/:databaseId`, () => {
        liveReadings.count += 1;

        return answerReading();
      }),
    );

    return liveReadings;
  };

  const getWaitingToBeSaved = () =>
    screen.getByText(DICTIONARIES.en.backups.physical.monitoring.figures.waitingToBeSaved)
      .nextElementSibling?.textContent;

  const setBrowserTabVisibility = (visibility: DocumentVisibilityState) => {
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue(visibility);
    document.dispatchEvent(new Event('visibilitychange'));
  };

  const settleAllCalls = (apiCall: MockInstance) =>
    act(async () => {
      await Promise.allSettled(apiCall.mock.results.map((call) => call.value));
    });

  it('sends no request when the browser tab is hidden and reads at once after it is shown', async () => {
    const statisticsRequests = declareStatistics(() => HttpResponse.json(healthyStatistics));
    const liveReadings = declareLiveReadings(() => HttpResponse.json(buildLiveReading(30 * MiB)));
    const statisticsLoad = vi.spyOn(physicalWalStreamStatisticsApi, 'getWalStreamStatistics');
    const liveLoad = vi.spyOn(physicalWalStreamStatisticsApi, 'getWalStreamLive');
    renderDatabasePage();
    await waitForCondition(() => expect(liveReadings.count).toBeGreaterThan(0));
    await settleAllCalls(statisticsLoad);
    await settleAllCalls(liveLoad);

    setBrowserTabVisibility('hidden');
    const statisticsCallsWhenHidden = statisticsLoad.mock.calls.length;
    const liveCallsWhenHidden = liveLoad.mock.calls.length;
    await advanceFakeClock(WAL_STREAM_REFRESH_INTERVAL_MS * 2);

    expect(statisticsLoad).toHaveBeenCalledTimes(statisticsCallsWhenHidden);
    expect(liveLoad).toHaveBeenCalledTimes(liveCallsWhenHidden);
    expect(statisticsRequests.count).toBe(statisticsCallsWhenHidden);
    expect(liveReadings.count).toBe(liveCallsWhenHidden);

    setBrowserTabVisibility('visible');

    expect(statisticsLoad).toHaveBeenCalledTimes(statisticsCallsWhenHidden + 1);
    expect(liveLoad).toHaveBeenCalledTimes(liveCallsWhenHidden + 1);

    await settleAllCalls(liveLoad);
    await advanceFakeClock(WAL_STREAM_LIVE_INTERVAL_MS);

    expect(liveLoad.mock.calls.length).toBeGreaterThan(liveCallsWhenHidden + 1);
    await waitForCondition(() => {
      expect(statisticsRequests.count).toBe(statisticsCallsWhenHidden + 1);
      expect(liveReadings.count).toBeGreaterThan(liveCallsWhenHidden + 1);
    });
  });

  it('sends no live reading after the user leaves the monitoring tab', async () => {
    declareStatistics(() => HttpResponse.json(healthyStatistics));
    const liveReadings = declareLiveReadings(() => HttpResponse.json(buildLiveReading(30 * MiB)));
    const liveLoad = vi.spyOn(physicalWalStreamStatisticsApi, 'getWalStreamLive');
    renderDatabasePage();
    await waitForCondition(() => expect(liveReadings.count).toBeGreaterThan(0));
    await settleAllCalls(liveLoad);
    const callsBeforeNextReading = liveLoad.mock.calls.length;
    await advanceFakeClock(WAL_STREAM_LIVE_INTERVAL_MS);
    expect(liveLoad.mock.calls.length).toBeGreaterThan(callsBeforeNextReading);
    await waitForCondition(() => expect(liveReadings.count).toBe(liveLoad.mock.calls.length));
    await settleAllCalls(liveLoad);

    fireEvent.click(screen.getByRole('button', { name: 'Toggle tab' }));
    const callsWhenLeaving = liveLoad.mock.calls.length;
    await advanceFakeClock(WAL_STREAM_LIVE_INTERVAL_MS * 3);
    document.dispatchEvent(new Event('visibilitychange'));

    expect(liveLoad).toHaveBeenCalledTimes(callsWhenLeaving);
    expect(liveReadings.count).toBe(callsWhenLeaving);
  });

  it('keeps the last live values without a warning when a reading is rejected, and updates them after the next success', async () => {
    declareStatistics(() => HttpResponse.json(healthyStatistics));
    let answerReading = () => HttpResponse.json(buildLiveReading(30 * MiB)) as Response;
    declareLiveReadings(() => answerReading());
    const liveLoad = vi.spyOn(physicalWalStreamStatisticsApi, 'getWalStreamLive');
    renderDatabasePage();
    await advanceClockUntil(() => expect(getWaitingToBeSaved()).toBe('30 MB'));

    answerReading = () => HttpResponse.error();
    await settleAllCalls(liveLoad);
    const callsBeforeRejection = liveLoad.mock.calls.length;
    await advanceClockUntil(() =>
      expect(liveLoad.mock.calls.length).toBeGreaterThan(callsBeforeRejection),
    );
    await settleAllCalls(liveLoad);
    await advanceFakeClock(WAL_STREAM_LIVE_INTERVAL_MS);

    expect(getWaitingToBeSaved()).toBe('30 MB');
    expect(screen.queryByText(copy.refreshFailed)).toBeNull();

    answerReading = () => HttpResponse.json(buildLiveReading(40 * MiB));

    await advanceClockUntil(() => expect(getWaitingToBeSaved()).toBe('40 MB'));
  });

  it('keeps the last statistics with a warning when a refresh is rejected, and updates them after the next success', async () => {
    let answerStatistics = () => HttpResponse.json(buildStatisticsWaiting(20 * MiB)) as Response;
    declareStatistics(() => answerStatistics());
    const statisticsLoad = vi.spyOn(physicalWalStreamStatisticsApi, 'getWalStreamStatistics');
    renderDatabasePage();
    await waitForCondition(() => expect(getWaitingToBeSaved()).toBe('20 MB'));

    answerStatistics = () => HttpResponse.error();
    await advanceFakeClock(WAL_STREAM_REFRESH_INTERVAL_MS);

    await waitForCondition(() => screen.getByText(copy.refreshFailed));
    expect(getWaitingToBeSaved()).toBe('20 MB');
    await settleAllCalls(statisticsLoad);

    answerStatistics = () => HttpResponse.json(buildStatisticsWaiting(25 * MiB));
    await advanceFakeClock(WAL_STREAM_REFRESH_INTERVAL_MS);

    await advanceClockUntil(() => expect(getWaitingToBeSaved()).toBe('25 MB'));
    expect(screen.queryByText(copy.refreshFailed)).toBeNull();
  });
});
