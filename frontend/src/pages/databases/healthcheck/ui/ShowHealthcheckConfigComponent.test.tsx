// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { healthcheckConfigApi } from '@/entities/healthcheck';
import { ShowHealthcheckConfigComponent } from '@/pages/databases/healthcheck/ui/ShowHealthcheckConfigComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { useLocale } from '@/shared/i18n';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const FIRST_DATABASE_ID = 'first-database-id';
const SECOND_DATABASE_ID = 'second-database-id';
const REFUSAL_MESSAGE = 'You do not have permission to do this.';
const config: ApiComponents['schemas']['HealthcheckConfig'] = {
  databaseId: FIRST_DATABASE_ID,
  isHealthcheckEnabled: true,
  isNotificationEnabled: true,
  intervalMinutes: 17,
  failedAttemptsBeforeUnavailable: 3,
  attemptRetentionDays: 7,
};
const secondDatabaseConfig = { ...config, databaseId: SECOND_DATABASE_ID, intervalMinutes: 29 };
const ANSWER_OUTCOMES = [
  {
    outcome: 'successful',
    buildAnswer: () => HttpResponse.json(config),
  },
  {
    outcome: 'refused',
    buildAnswer: () => HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
  },
];
const firstDatabasePath = `*${API_BASE_PATH}/healthcheck-configs/${FIRST_DATABASE_ID}`;
const secondDatabasePath = `*${API_BASE_PATH}/healthcheck-configs/${SECOND_DATABASE_ID}`;

describe('ShowHealthcheckConfigComponent', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows the reason with a retry when the load fails, and the configuration after a successful retry', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    renderScreen(<ShowHealthcheckConfigComponent databaseId={FIRST_DATABASE_ID} />);

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();

    mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, config));
    fireEvent.click(retryButton);

    expect(await screen.findByText('17')).toBeTruthy();
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('requests nothing new and keeps the configuration when the language changes', async () => {
    const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
    const LanguageSwitch = () => {
      const { setLocale } = useLocale();

      return <button onClick={() => setLocale('fr')}>Switch language</button>;
    };
    mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, config));

    renderScreen(
      <>
        <LanguageSwitch />
        <ShowHealthcheckConfigComponent databaseId={FIRST_DATABASE_ID} />
      </>,
    );
    await screen.findByText('17');
    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

    await screen.findByText(DICTIONARIES.fr.healthcheck.config.isEnabled);
    expect(screen.getByText('17')).toBeTruthy();
    expect(loadCall).toHaveBeenCalledTimes(1);
  });

  it.each(ANSWER_OUTCOMES)(
    'keeps the second database when a $outcome load of the first one answers late',
    async ({ buildAnswer }) => {
      const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
      const heldLoad = createPendingApiResponse();
      mockServer.use(
        http.get(firstDatabasePath, () => heldLoad.response),
        http.get(secondDatabasePath, () => HttpResponse.json(secondDatabaseConfig)),
      );

      const { rerender } = renderScreen(
        <ShowHealthcheckConfigComponent databaseId={FIRST_DATABASE_ID} />,
      );
      rerender(<ShowHealthcheckConfigComponent databaseId={SECOND_DATABASE_ID} />);
      await screen.findByText('29');

      await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer()), 0);

      expect(screen.getByText('29')).toBeTruthy();
      expect(screen.queryByText('17')).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );

  it.each(ANSWER_OUTCOMES)(
    'keeps the second database when a $outcome retry of the first one answers late',
    async ({ buildAnswer }) => {
      const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
      const heldRetry = createPendingApiResponse();
      mockServer.use(
        mockApi.get(
          '/healthcheck-configs/{databaseId}',
          403,
          buildProblem('permission_denied', 403),
        ),
      );

      const { rerender } = renderScreen(
        <ShowHealthcheckConfigComponent databaseId={FIRST_DATABASE_ID} />,
      );
      const retryButton = await screen.findByRole('button', { name: 'Retry' });

      mockServer.use(
        http.get(firstDatabasePath, () => heldRetry.response),
        http.get(secondDatabasePath, () => HttpResponse.json(secondDatabaseConfig)),
      );
      fireEvent.click(retryButton);
      rerender(<ShowHealthcheckConfigComponent databaseId={SECOND_DATABASE_ID} />);
      await screen.findByText('29');

      await releaseAndSettle(loadCall, () => heldRetry.respond(buildAnswer()), 1);

      expect(screen.getByText('29')).toBeTruthy();
      expect(screen.queryByText('17')).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );

  it.each(ANSWER_OUTCOMES)(
    'shows nothing when a $outcome load answers after the session ended',
    async ({ buildAnswer }) => {
      const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
      const heldLoad = createPendingApiResponse();
      activateSession({ token: 'first-token', userId: 'first-user' });
      mockServer.use(http.get(firstDatabasePath, () => heldLoad.response));

      renderScreen(<ShowHealthcheckConfigComponent databaseId={FIRST_DATABASE_ID} />);
      act(() => {
        invalidateSession();
      });

      await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer()));

      expect(screen.queryByText('17')).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );
});
