// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { healthcheckAttemptApi, healthcheckConfigApi } from '@/entities/healthcheck';
import { HealthcheckAttemptsComponent } from '@/pages/databases/healthcheck/ui/HealthcheckAttemptsComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { useLocale } from '@/shared/i18n';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

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
const secondDatabase: Database = { ...database, id: 'second-database-id' };
const REFUSAL_MESSAGE = 'You do not have permission to do this.';
const REFRESH_INTERVAL_MS = 60_000;
const refusal = () => HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 });
const singlePage = (...items: ApiComponents['schemas']['HealthcheckAttempt'][]) =>
  HttpResponse.json({ items });
const selectSevenDays = async () => {
  fireEvent.click(screen.getByRole('combobox'));
  fireEvent.click(await screen.findByText('7 days'));
};

describe('HealthcheckAttemptsComponent', () => {
  it.each([20_000, 70_000])(
    'keeps scheduled ticks and skips pending reads when a refresh takes %s milliseconds',
    async (duration) => {
      const attemptsRead = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
      mockServer.use(
        mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
        http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))),
      );
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });

      const view = renderScreen(<HealthcheckAttemptsComponent database={database} />);
      await waitFor(() => expect(view.container.querySelectorAll('[data-status]')).toHaveLength(1));

      const slowRead = mockApi.pending('get', '/healthcheck-attempts');
      mockServer.use(slowRead.handler);

      await advanceFakeClock(60_000);
      await advanceFakeClock(duration);

      expect(attemptsRead).toHaveBeenCalledTimes(2);

      await releaseAndSettle(attemptsRead, () =>
        slowRead.respond(singlePage(attempt('second', 'AVAILABLE'))),
      );

      mockServer.use(http.get(attemptsPath, () => singlePage(attempt('second', 'AVAILABLE'))));

      await advanceFakeClock(duration === 20_000 ? 40_000 : 50_000);

      expect(attemptsRead).toHaveBeenCalledTimes(3);
    },
  );

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

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
    const oldPage = createPendingApiResponse();
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
          return oldPage.response;
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
    fireEvent.click(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('7 days'));
    await waitFor(() =>
      expect(container.querySelectorAll('[data-status="AVAILABLE"]')).toHaveLength(1),
    );

    await act(async () => {
      oldPage.respond(HttpResponse.json({ items: [attempt('old-last', 'UNAVAILABLE')] }));
      await oldPage.response;
    });

    expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
    expect(container.querySelectorAll('[data-status="UNAVAILABLE"]')).toHaveLength(0);
    expect(sinceValues[0]).toEqual(sinceValues[1]);
    expect(new Date(sinceValues[2]).getTime()).toBeLessThan(new Date(sinceValues[0]).getTime());
  });

  it('names the period field by its label', async () => {
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage()),
    );

    renderScreen(<HealthcheckAttemptsComponent database={database} />);

    expect(await screen.findByRole('combobox', { name: 'Period' })).toBeTruthy();
  });

  it('shows a retry when the config load fails, and the attempts after a retry finds healthcheck enabled', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();

    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))),
    );
    fireEvent.click(retryButton);

    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('shows nothing after a retry of the config load finds healthcheck disabled', async () => {
    const configCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, {
        ...config,
        isHealthcheckEnabled: false,
      }),
    );
    fireEvent.click(retryButton);

    await waitFor(() => expect(configCall).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(container.innerText).toBe(''));
  });

  it('shows a retry when the first load of attempts fails, and the attempts after a successful retry', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, refusal),
    );

    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(screen.queryByText('No attempts')).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();

    mockServer.use(http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))));
    fireEvent.click(retryButton);

    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('keeps the shown attempts with no message when a refresh fails', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))),
    );

    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));

    mockServer.use(http.get(attemptsPath, refusal));
    const callsBeforeRefresh = attemptsCall.mock.calls.length;
    await advanceFakeClock(REFRESH_INTERVAL_MS);
    await waitFor(() => expect(attemptsCall.mock.calls.length).toBe(callsBeforeRefresh + 1));
    await releaseAndSettle(attemptsCall, () => {});

    expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('keeps refreshing after a failed refresh', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
    const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))),
    );

    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));

    mockServer.use(http.get(attemptsPath, refusal));
    const callsBeforeRefresh = attemptsCall.mock.calls.length;
    await advanceFakeClock(REFRESH_INTERVAL_MS);
    await waitFor(() => expect(attemptsCall.mock.calls.length).toBe(callsBeforeRefresh + 1));
    await releaseAndSettle(attemptsCall, () => {});

    mockServer.use(
      http.get(attemptsPath, () =>
        singlePage(attempt('first', 'AVAILABLE'), attempt('second', 'UNAVAILABLE')),
      ),
    );
    await advanceFakeClock(REFRESH_INTERVAL_MS);

    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(2));
  });

  it('shows the reason without the old attempts when the load after a period change fails, and a later tick does not replace it', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
    const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage(attempt('old', 'AVAILABLE'))),
    );

    const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));

    mockServer.use(http.get(attemptsPath, refusal));
    await selectSevenDays();

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(container.querySelectorAll('[data-status]')).toHaveLength(0);

    fireEvent.click(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('Today'));
    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();

    const callsAfterFailure = attemptsCall.mock.calls.length;
    mockServer.use(http.get(attemptsPath, () => singlePage(attempt('new', 'AVAILABLE'))));
    await advanceFakeClock(REFRESH_INTERVAL_MS * 2);

    expect(attemptsCall.mock.calls.length).toBe(callsAfterFailure);
    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it.each([
    { outcome: 'successful', buildAnswer: () => singlePage(attempt('late', 'UNAVAILABLE')) },
    { outcome: 'refused', buildAnswer: refusal },
  ])(
    'keeps the second database when a $outcome load of the first one answers late',
    async ({ buildAnswer }) => {
      const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
      const heldLoad = createPendingApiResponse();
      mockServer.use(
        mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
        http.get(attemptsPath, ({ request }) =>
          new URL(request.url).searchParams.get('databaseId') === database.id
            ? heldLoad.response
            : singlePage(attempt('second', 'AVAILABLE')),
        ),
      );

      const { container, rerender } = renderScreen(
        <HealthcheckAttemptsComponent database={database} />,
      );
      await waitFor(() => expect(attemptsCall).toHaveBeenCalledTimes(1));

      rerender(<HealthcheckAttemptsComponent database={secondDatabase} />);
      await waitFor(() =>
        expect(container.querySelectorAll('[data-status="AVAILABLE"]')).toHaveLength(1),
      );

      await releaseAndSettle(attemptsCall, () => heldLoad.respond(buildAnswer()), 0);

      expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
      expect(container.querySelectorAll('[data-status="UNAVAILABLE"]')).toHaveLength(0);
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );

  it('keeps the second database when a successful config load of the first one answers late', async () => {
    const configCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
    const heldConfig = createPendingApiResponse();
    mockServer.use(
      http.get(`*${API_BASE_PATH}/healthcheck-configs/${database.id}`, () => heldConfig.response),
      http.get(`*${API_BASE_PATH}/healthcheck-configs/${secondDatabase.id}`, () =>
        HttpResponse.json({ ...config, databaseId: secondDatabase.id }),
      ),
      http.get(attemptsPath, () => singlePage(attempt('second', 'AVAILABLE'))),
    );

    const { container, rerender } = renderScreen(
      <HealthcheckAttemptsComponent database={database} />,
    );
    await waitFor(() => expect(configCall).toHaveBeenCalledTimes(1));
    rerender(<HealthcheckAttemptsComponent database={secondDatabase} />);
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));

    await releaseAndSettle(
      configCall,
      () => heldConfig.respond(HttpResponse.json({ ...config, isHealthcheckEnabled: false })),
      0,
    );

    expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
  });

  it.each([
    { outcome: 'successful', buildAnswer: () => HttpResponse.json(config) },
    { outcome: 'refused', buildAnswer: refusal },
  ])(
    'keeps the spinner when a $outcome config load answers after the session ended',
    async ({ buildAnswer }) => {
      const configCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
      const heldConfig = createPendingApiResponse();
      activateSession({ token: 'first-token', userId: 'first-user' });
      mockServer.use(
        http.get(`*${API_BASE_PATH}/healthcheck-configs/${database.id}`, () => heldConfig.response),
      );

      renderScreen(<HealthcheckAttemptsComponent database={database} />);
      await waitFor(() => expect(configCall).toHaveBeenCalledTimes(1));
      act(() => {
        invalidateSession();
      });

      await releaseAndSettle(configCall, () => heldConfig.respond(buildAnswer()));

      expect(document.querySelector('[role="status"]')).toBeTruthy();
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );

  it.each([
    { outcome: 'successful', buildAnswer: () => singlePage(attempt('late', 'AVAILABLE')) },
    { outcome: 'refused', buildAnswer: refusal },
  ])(
    'shows nothing when a $outcome attempts load answers after the session ended',
    async ({ buildAnswer }) => {
      const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
      const heldLoad = createPendingApiResponse();
      activateSession({ token: 'first-token', userId: 'first-user' });
      mockServer.use(
        mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
        http.get(attemptsPath, () => heldLoad.response),
      );

      const { container } = renderScreen(<HealthcheckAttemptsComponent database={database} />);
      await waitFor(() => expect(attemptsCall).toHaveBeenCalledTimes(1));
      act(() => {
        invalidateSession();
      });

      await releaseAndSettle(attemptsCall, () => heldLoad.respond(buildAnswer()));

      expect(container.querySelectorAll('[data-status]')).toHaveLength(0);
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );

  it('requests no further attempts after it unmounts with polling running', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
    const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))),
    );

    const { container, unmount } = renderScreen(
      <HealthcheckAttemptsComponent database={database} />,
    );
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));
    const callsBeforeUnmount = attemptsCall.mock.calls.length;

    unmount();
    await advanceFakeClock(REFRESH_INTERVAL_MS * 2);

    expect(attemptsCall.mock.calls.length).toBe(callsBeforeUnmount);
  });

  it('repeats no request when the language changes', async () => {
    const configCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
    const attemptsCall = vi.spyOn(healthcheckAttemptApi, 'getAttemptsByDatabase');
    const LanguageSwitch = () => {
      const { setLocale } = useLocale();

      return <button onClick={() => setLocale('fr')}>Switch language</button>;
    };
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      http.get(attemptsPath, () => singlePage(attempt('first', 'AVAILABLE'))),
    );

    const { container } = renderScreen(
      <>
        <LanguageSwitch />
        <HealthcheckAttemptsComponent database={database} />
      </>,
    );
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));

    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));
    await screen.findByText(DICTIONARIES.fr.healthcheck.attempts.title);

    expect(configCall).toHaveBeenCalledTimes(1);
    expect(attemptsCall).toHaveBeenCalledTimes(1);
    expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
  });

  it('keeps the second database when a retry of the first one answers late', async () => {
    const configCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
    const heldRetry = createPendingApiResponse();
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    const { container, rerender } = renderScreen(
      <HealthcheckAttemptsComponent database={database} />,
    );
    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    mockServer.use(
      http.get(`*${API_BASE_PATH}/healthcheck-configs/${database.id}`, () => heldRetry.response),
      http.get(`*${API_BASE_PATH}/healthcheck-configs/${secondDatabase.id}`, () =>
        HttpResponse.json({ ...config, databaseId: secondDatabase.id }),
      ),
      http.get(attemptsPath, () => singlePage(attempt('second', 'AVAILABLE'))),
    );
    fireEvent.click(retryButton);
    await waitFor(() => expect(configCall).toHaveBeenCalledTimes(2));

    rerender(<HealthcheckAttemptsComponent database={secondDatabase} />);
    await waitFor(() => expect(container.querySelectorAll('[data-status]')).toHaveLength(1));

    await releaseAndSettle(configCall, () => heldRetry.respond(refusal()), 1);

    expect(container.querySelectorAll('[data-status]')).toHaveLength(1);
    expect(screen.queryByRole('alert')).toBeNull();
  });
});
