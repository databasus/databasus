// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { type MockInstance, describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { API_BASE_PATH } from '../../../../shared/api';
import { DICTIONARIES, useLocale } from '../../../../shared/i18n';
import { advanceFakeClock } from '../../../../shared/lib/test/advanceFakeClock';
import { createTestWalStreamStatistics } from '../../../../shared/lib/test/buildWalStreamStatistics';
import { waitForCondition } from '../../../../shared/lib/test/conditionWait';
import { buildProblem, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '../../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { physicalBackupsApi } from '../../api/physicalBackupsApi';
import { physicalWalStreamStatisticsApi } from '../../api/physicalWalStreamStatisticsApi';
import { WAL_STREAM_REFRESH_INTERVAL_MS } from '../../lib/wal-stream-monitoring/walStreamRefreshController';
import { WalStreamHealthLevel } from '../../model/wal-stream-monitoring/WalStreamHealthLevel';
import { WalStreamReasonCode } from '../../model/wal-stream-monitoring/WalStreamReasonCode';
import { WalStreamReasonLevel } from '../../model/wal-stream-monitoring/WalStreamReasonLevel';
import { WalStreamMonitoringComponent } from './WalStreamMonitoringComponent';

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
        <WalStreamMonitoringComponent
          database={database}
          isCanManageDBs
          isDirectlyUnderTab
          onOpenConfig={() => {}}
        />
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

// A busy AntD button adds its loading icon to the accessible name.
const findStartButton = (name: string = copy.startFullBackup) =>
  waitForCondition(() => screen.getByRole('button', { name: new RegExp(name) }));

const isButtonBusy = (button: HTMLElement) => button.classList.contains('ant-btn-loading');

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
