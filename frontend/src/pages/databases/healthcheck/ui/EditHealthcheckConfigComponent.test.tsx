// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, onTestFinished, vi } from 'vitest';

import { healthcheckConfigApi } from '@/entities/healthcheck';
import { EditHealthcheckConfigComponent } from '@/pages/databases/healthcheck/ui/EditHealthcheckConfigComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const databaseId = 'database-id';
const SECOND_DATABASE_ID = 'second-database-id';
const REFUSAL_MESSAGE = 'You do not have permission to do this.';
const config: ApiComponents['schemas']['HealthcheckConfig'] = {
  databaseId,
  isHealthcheckEnabled: true,
  isNotificationEnabled: true,
  intervalMinutes: 1,
  failedAttemptsBeforeUnavailable: 3,
  attemptRetentionDays: 7,
  shouldConfirmEnabling: false,
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
const neonConfig = { ...config, isHealthcheckEnabled: false, shouldConfirmEnabling: true };
const configPath = (id: string) => `*${API_BASE_PATH}/healthcheck-configs/${id}`;
const getIntervalField = async () =>
  (await screen.findAllByRole('spinbutton'))[0] as HTMLInputElement;

describe('EditHealthcheckConfigComponent', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('connects every field to its label', async () => {
    mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, config));

    renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />);

    expect(await screen.findByRole('switch', { name: 'Enable healthcheck' })).toBeTruthy();
    expect(screen.getByRole('switch', { name: 'Notify when unavailable' })).toBeTruthy();
    expect(screen.getByRole('spinbutton', { name: 'Check interval (minutes)' })).toBeTruthy();
    expect(screen.getByRole('spinbutton', { name: 'Attempts before down' })).toBeTruthy();
    expect(screen.getByRole('spinbutton', { name: 'Store attempts (days)' })).toBeTruthy();
  });

  it('keeps the entered values and shows the reason when the server refuses the save', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const onClose = vi.fn();
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      mockApi.put('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />);
    fireEvent.change(await getIntervalField(), { target: { value: '12' } });
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect((await getIntervalField()).value).toBe('12');
    expect(showAlert).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('removes the shown reason when the next save starts', async () => {
    const heldSave = createPendingApiResponse();
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      mockApi.put('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />);
    fireEvent.change(await getIntervalField(), { target: { value: '12' } });
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));
    await screen.findByText(REFUSAL_MESSAGE);

    mockServer.use(http.put(configPath(databaseId), () => heldSave.response));
    fireEvent.click(await screen.findByRole('button', { name: /Save/ }));

    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
    heldSave.cancel();
  });

  it('offers retry and close when the load fails, and the form after a successful retry', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const onClose = vi.fn();
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );

    renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />);

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onClose).toHaveBeenCalledTimes(1);

    mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, config));
    fireEvent.click(retryButton);

    expect((await getIntervalField()).value).toBe('1');
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  describe('late answers', () => {
    it.each(ANSWER_OUTCOMES)(
      'keeps the second database when a $outcome load of the first one answers late',
      async ({ buildAnswer }) => {
        const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
        const heldLoad = createPendingApiResponse();
        const onClose = vi.fn();
        mockServer.use(
          http.get(configPath(databaseId), () => heldLoad.response),
          http.get(configPath(SECOND_DATABASE_ID), () => HttpResponse.json(secondDatabaseConfig)),
        );

        const { rerender } = renderScreen(
          <EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />,
        );
        rerender(
          <EditHealthcheckConfigComponent databaseId={SECOND_DATABASE_ID} onClose={onClose} />,
        );
        expect((await getIntervalField()).value).toBe('29');

        await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer()), 0);

        expect((await getIntervalField()).value).toBe('29');
        expect(screen.queryByRole('alert')).toBeNull();
        expect(onClose).not.toHaveBeenCalled();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'keeps the second database when a $outcome retry of the first one answers late',
      async ({ buildAnswer }) => {
        const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
        const heldRetry = createPendingApiResponse();
        const onClose = vi.fn();
        mockServer.use(
          mockApi.get(
            '/healthcheck-configs/{databaseId}',
            403,
            buildProblem('permission_denied', 403),
          ),
        );

        const { rerender } = renderScreen(
          <EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />,
        );
        const retryButton = await screen.findByRole('button', { name: 'Retry' });

        mockServer.use(
          http.get(configPath(databaseId), () => heldRetry.response),
          http.get(configPath(SECOND_DATABASE_ID), () => HttpResponse.json(secondDatabaseConfig)),
        );
        fireEvent.click(retryButton);
        rerender(
          <EditHealthcheckConfigComponent databaseId={SECOND_DATABASE_ID} onClose={onClose} />,
        );
        expect((await getIntervalField()).value).toBe('29');

        await releaseAndSettle(loadCall, () => heldRetry.respond(buildAnswer()), 1);

        expect((await getIntervalField()).value).toBe('29');
        expect(screen.queryByRole('alert')).toBeNull();
        expect(onClose).not.toHaveBeenCalled();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'neither closes the section nor shows a reason when a $outcome save answers after the database changed',
      async ({ buildAnswer }) => {
        const saveCall = vi.spyOn(healthcheckConfigApi, 'saveHealthcheckConfig');
        const heldSave = createPendingApiResponse();
        const onClose = vi.fn();
        mockServer.use(
          http.get(configPath(databaseId), () => HttpResponse.json(config)),
          http.put(configPath(databaseId), () => heldSave.response),
          http.get(configPath(SECOND_DATABASE_ID), () => HttpResponse.json(secondDatabaseConfig)),
        );

        const { rerender } = renderScreen(
          <EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />,
        );
        fireEvent.change(await getIntervalField(), { target: { value: '12' } });
        fireEvent.click(await screen.findByRole('button', { name: 'Save' }));
        rerender(
          <EditHealthcheckConfigComponent databaseId={SECOND_DATABASE_ID} onClose={onClose} />,
        );
        expect((await getIntervalField()).value).toBe('29');

        await releaseAndSettle(saveCall, () => heldSave.respond(buildAnswer()));

        expect((await getIntervalField()).value).toBe('29');
        expect(screen.queryByRole('alert')).toBeNull();
        expect(onClose).not.toHaveBeenCalled();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'keeps the spinner when a $outcome load answers after the session ended',
      async ({ buildAnswer }) => {
        const loadCall = vi.spyOn(healthcheckConfigApi, 'getHealthcheckConfig');
        const heldLoad = createPendingApiResponse();
        activateSession({ token: 'first-token', userId: 'first-user' });
        mockServer.use(http.get(configPath(databaseId), () => heldLoad.response));

        renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />);
        act(() => {
          invalidateSession();
        });

        await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer()));

        expect(document.querySelector('[role="status"]')).toBeTruthy();
        expect(screen.queryByRole('spinbutton')).toBeNull();
        expect(screen.queryByRole('alert')).toBeNull();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'neither closes the section nor shows a reason when a $outcome save answers after the session ended',
      async ({ buildAnswer }) => {
        const saveCall = vi.spyOn(healthcheckConfigApi, 'saveHealthcheckConfig');
        const heldSave = createPendingApiResponse();
        const onClose = vi.fn();
        activateSession({ token: 'first-token', userId: 'first-user' });
        mockServer.use(
          http.get(configPath(databaseId), () => HttpResponse.json(config)),
          http.put(configPath(databaseId), () => heldSave.response),
        );

        renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />);
        fireEvent.change(await getIntervalField(), { target: { value: '12' } });
        fireEvent.click(await screen.findByRole('button', { name: 'Save' }));
        await waitFor(() => expect(saveCall).toHaveBeenCalledOnce());

        act(() => {
          invalidateSession();
        });

        await releaseAndSettle(saveCall, () => heldSave.respond(buildAnswer()));

        expect(screen.queryByRole('alert')).toBeNull();
        expect(onClose).not.toHaveBeenCalled();
      },
    );
  });

  describe('Neon confirmation', () => {
    const useFakeIntervals = () => {
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
      onTestFinished(() => {
        vi.useRealTimers();
      });
    };
    const getEnableSwitch = () =>
      screen.findByRole('switch', { name: 'Enable healthcheck' }) as Promise<HTMLInputElement>;
    const getConfirmButton = (name: string) =>
      within(screen.getByRole('dialog')).getByRole('button', { name }) as HTMLButtonElement;

    it('keeps the switch off and the confirm button locked with a countdown for 10 seconds', async () => {
      useFakeIntervals();
      mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, neonConfig));

      const { user } = renderScreen(
        <EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />,
      );
      await user.click(await getEnableSwitch());

      expect(screen.getByRole('dialog').textContent).toContain('Neon');
      expect((await getEnableSwitch()).checked).toBe(false);
      expect(getConfirmButton('Enable healthcheck (10)').disabled).toBe(true);

      act(() => vi.advanceTimersByTime(9_000));

      expect(getConfirmButton('Enable healthcheck (1)').disabled).toBe(true);
    });

    it('turns the checks on and saves them without the confirmation flag after the wait and a confirmation', async () => {
      useFakeIntervals();
      const savedBodies: unknown[] = [];
      mockServer.use(
        mockApi.get('/healthcheck-configs/{databaseId}', 200, neonConfig),
        http.put(configPath(databaseId), async ({ request }) => {
          savedBodies.push(await request.json());
          return HttpResponse.json({ ...neonConfig, isHealthcheckEnabled: true });
        }),
      );

      const { user } = renderScreen(
        <EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />,
      );
      await user.click(await getEnableSwitch());
      act(() => vi.advanceTimersByTime(10_000));
      await user.click(getConfirmButton('Enable healthcheck'));

      expect(screen.queryByRole('dialog')).toBeNull();
      expect((await getEnableSwitch()).checked).toBe(true);
      expect((await getIntervalField()).value).toBe('1');

      await user.click(screen.getByRole('button', { name: 'Save' }));

      await waitFor(() => expect(savedBodies).toHaveLength(1));
      expect(savedBodies[0]).toEqual({
        isHealthcheckEnabled: true,
        isNotificationEnabled: true,
        intervalMinutes: 1,
        failedAttemptsBeforeUnavailable: 3,
        attemptRetentionDays: 7,
      });
    });

    it('keeps the switch off and offers no save when the dialog is cancelled', async () => {
      mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, neonConfig));

      const { user } = renderScreen(
        <EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />,
      );
      await user.click(await getEnableSwitch());
      await user.click(getConfirmButton('Cancel'));

      expect(screen.queryByRole('dialog')).toBeNull();
      expect((await getEnableSwitch()).checked).toBe(false);
      expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    });

    it.each([
      { control: 'its close control', close: () => fireEvent.click(getConfirmButton('Close')) },
      {
        control: 'Escape',
        close: () => fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' }),
      },
    ])(
      'keeps the switch off and offers no save when the dialog is closed with $control',
      async ({ close }) => {
        mockServer.use(mockApi.get('/healthcheck-configs/{databaseId}', 200, neonConfig));

        const { user } = renderScreen(
          <EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />,
        );
        await user.click(await getEnableSwitch());
        close();

        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
        expect((await getEnableSwitch()).checked).toBe(false);
        expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
      },
    );

    it('keeps an earlier unsaved change offered when the dialog is cancelled', async () => {
      mockServer.use(
        mockApi.get('/healthcheck-configs/{databaseId}', 200, {
          ...neonConfig,
          isHealthcheckEnabled: true,
        }),
      );

      const { user } = renderScreen(
        <EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />,
      );
      await user.click(await getEnableSwitch());

      expect(screen.queryByRole('dialog')).toBeNull();
      expect((await getEnableSwitch()).checked).toBe(false);

      await user.click(await getEnableSwitch());
      await user.click(getConfirmButton('Cancel'));

      expect((await getEnableSwitch()).checked).toBe(false);
      expect(screen.getByRole('button', { name: 'Save' })).toBeTruthy();
    });

    it('turns the checks on at once without a dialog when the config needs no confirmation', async () => {
      mockServer.use(
        mockApi.get('/healthcheck-configs/{databaseId}', 200, {
          ...config,
          isHealthcheckEnabled: false,
        }),
      );

      const { user } = renderScreen(
        <EditHealthcheckConfigComponent databaseId={databaseId} onClose={vi.fn()} />,
      );
      await user.click(await getEnableSwitch());

      expect(screen.queryByRole('dialog')).toBeNull();
      expect((await getEnableSwitch()).checked).toBe(true);
    });
  });
});
