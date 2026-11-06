// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { type MockInstance, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { verificationConfigApi } from '@/pages/databases/api/verificationConfigApi';
import type { VerificationConfig } from '@/pages/databases/model/verification/VerificationConfig';
import { ShowVerificationConfigComponent } from '@/pages/databases/ui/verification/ShowVerificationConfigComponent';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const config: VerificationConfig = {
  databaseId: 'database-id',
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: { type: 'WEEKLY', timeOfDay: '12:00', weekday: 0 },
  notificationTriggers: [],
};
const REFUSAL_MESSAGE = 'You do not have permission to do this.';

describe('ShowVerificationConfigComponent', () => {
  beforeEach(() => {
    vi.stubEnv('TZ', 'UTC');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  it('shows a saved weekday 0 as Sunday', async () => {
    mockServer.use(mockApi.get('/verification-configs/{databaseId}', 200, config));

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    expect(await screen.findByText('Sun')).toBeTruthy();
  });

  it('names the fields Weekday and Time of day without the word Verification', async () => {
    mockServer.use(mockApi.get('/verification-configs/{databaseId}', 200, config));

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    expect(await screen.findByText('Weekday')).toBeTruthy();
    expect(screen.getByText('Time of day')).toBeTruthy();
    expect(screen.queryByText('Verification weekday')).toBeNull();
    expect(screen.queryByText('Verification time of day')).toBeNull();
  });

  it('shows the next run of a stored valid expression', async () => {
    mockServer.use(
      mockApi.get('/verification-configs/{databaseId}', 200, {
        ...config,
        schedule: { type: 'CRON', cronExpression: '0 2 * * *' },
      }),
    );

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    expect(await screen.findByText('0 2 * * *')).toBeTruthy();
    expect(screen.getByText(/^Next run/)).toBeTruthy();
  });

  it('shows no next run for a stored expression with no date to run on', async () => {
    mockServer.use(
      mockApi.get('/verification-configs/{databaseId}', 200, {
        ...config,
        schedule: { type: 'CRON', cronExpression: '0 0 31 2 *' },
      }),
    );

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    expect(await screen.findByText('0 0 31 2 *')).toBeTruthy();
    expect(screen.queryByText(/^Next run/)).toBeNull();
  });

  it('offers a retry without an alert after a failed load, and a successful retry shows the config', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});

    mockServer.use(
      mockApi.get(
        '/verification-configs/{databaseId}',
        403,
        buildProblem('permission_denied', 403),
      ),
    );

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();

    mockServer.use(mockApi.get('/verification-configs/{databaseId}', 200, config));
    fireEvent.click(retryButton);

    expect(await screen.findByText('Sun')).toBeTruthy();
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('converts a stored weekday and time to the local zone', async () => {
    vi.stubEnv('TZ', 'Europe/Moscow');
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('en-GB');
    mockServer.use(
      mockApi.get('/verification-configs/{databaseId}', 200, {
        ...config,
        schedule: { type: 'WEEKLY', timeOfDay: '22:00', weekday: 1 },
      }),
    );

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    expect(await screen.findByText('Tue')).toBeTruthy();
    expect(screen.getByText('01:00')).toBeTruthy();
  });

  describe('late answers', () => {
    const SECOND_DATABASE_ID = 'second-database-id';
    const ANSWER_OUTCOMES = [
      { outcome: 'successful', status: 200 },
      { outcome: 'refused', status: 403 },
    ];
    const firstDatabaseConfig: VerificationConfig = {
      ...config,
      schedule: { type: 'WEEKLY', timeOfDay: '12:00', weekday: 3 },
    };

    const buildAnswer = (status: number) =>
      status === 200
        ? HttpResponse.json(firstDatabaseConfig)
        : HttpResponse.json(buildProblem('permission_denied', 403), { status });

    const expectNoFeedback = (showAlert: MockInstance) => {
      expect(showAlert).not.toHaveBeenCalled();
      expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    };

    it.each(ANSWER_OUTCOMES)(
      'shows nothing for a $outcome load answer after unmount',
      async ({ status }) => {
        const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
        const loadCall = vi.spyOn(verificationConfigApi, 'getByDatabaseId');
        const heldLoad = createPendingApiResponse();

        mockServer.use(
          http.get(`*${API_BASE_PATH}/verification-configs/:databaseId`, () => heldLoad.response),
        );

        const { unmount } = renderScreen(
          <ShowVerificationConfigComponent databaseId={config.databaseId} />,
        );

        unmount();
        await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer(status)));

        expectNoFeedback(showAlert);
        expect(screen.queryByText('Wed')).toBeNull();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'shows nothing for a $outcome load answer after a session change',
      async ({ status }) => {
        const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
        const loadCall = vi.spyOn(verificationConfigApi, 'getByDatabaseId');
        const heldLoad = createPendingApiResponse();

        activateSession({ token: 'first-token', userId: 'first-user' });
        mockServer.use(
          http.get(`*${API_BASE_PATH}/verification-configs/:databaseId`, () => heldLoad.response),
        );
        renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

        act(() => {
          invalidateSession();
        });
        await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer(status)));

        expectNoFeedback(showAlert);
        expect(screen.queryByText('Wed')).toBeNull();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'keeps the second database for a $outcome load answer of the first one',
      async ({ status }) => {
        const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
        const loadCall = vi.spyOn(verificationConfigApi, 'getByDatabaseId');
        const heldLoad = createPendingApiResponse();

        mockServer.use(
          http.get(
            `*${API_BASE_PATH}/verification-configs/${config.databaseId}`,
            () => heldLoad.response,
          ),
          http.get(`*${API_BASE_PATH}/verification-configs/${SECOND_DATABASE_ID}`, () =>
            HttpResponse.json({ ...config, databaseId: SECOND_DATABASE_ID }),
          ),
        );

        const { rerender } = renderScreen(
          <ShowVerificationConfigComponent databaseId={config.databaseId} />,
        );

        rerender(<ShowVerificationConfigComponent databaseId={SECOND_DATABASE_ID} />);
        await screen.findByText('Sun');

        await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer(status)), 0);

        expectNoFeedback(showAlert);
        expect(screen.getByText('Sun')).toBeTruthy();
        expect(screen.queryByText('Wed')).toBeNull();
      },
    );
  });
});
