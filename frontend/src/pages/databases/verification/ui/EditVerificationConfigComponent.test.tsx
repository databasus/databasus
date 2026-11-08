// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { verificationConfigApi } from '@/pages/databases/verification/api/verificationConfigApi';
import type { VerificationConfig } from '@/pages/databases/verification/model/verificationContract';
import { EditVerificationConfigComponent } from '@/pages/databases/verification/ui/EditVerificationConfigComponent';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

type VerificationConfigInput = Omit<VerificationConfig, 'databaseId'>;

const mockServer = setUpMockApi();

const weeklySchedule: VerificationConfig['schedule'] = {
  type: 'WEEKLY',
  timeOfDay: '12:00',
  weekday: 1,
};
const INVALID_CRON_EXPRESSION = '0 0 31 2 *';
const REFUSAL_MESSAGE = 'You do not have permission to do this.';

const buildConfig = (overrides: Partial<VerificationConfig> = {}): VerificationConfig => ({
  databaseId: 'database-id',
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: weeklySchedule,
  notificationTriggers: [],
  ...overrides,
});

const serveConfig = (config: VerificationConfig) => {
  const submitted: VerificationConfigInput[] = [];

  mockServer.use(
    mockApi.get('/verification-configs/{databaseId}', 200, config),
    http.put(`*${API_BASE_PATH}/verification-configs/:databaseId`, async ({ request }) => {
      const saved = (await request.json()) as VerificationConfigInput;
      submitted.push(saved);

      return HttpResponse.json({ ...saved, databaseId: config.databaseId });
    }),
  );

  return submitted;
};

const renderEditor = (config: VerificationConfig) => {
  const submitted = serveConfig(config);
  const onClose = vi.fn();

  renderScreen(
    <EditVerificationConfigComponent databaseId={config.databaseId} onClose={onClose} />,
  );

  return { submitted, onClose };
};

const renderConfig = (schedule: VerificationConfig['schedule']) =>
  renderEditor(buildConfig({ schedule })).submitted;

const chooseOption = async ({ label, optionTitle }: { label: string; optionTitle: string }) => {
  const fieldLabel = await screen.findByText(label);

  fireEvent.click(fieldLabel.parentElement!.querySelector('[role="combobox"]')!);

  const option = await waitFor(() => {
    const visibleOptions = screen.queryAllByRole('option', { name: optionTitle });
    expect(visibleOptions.length).toBeGreaterThan(0);
    return visibleOptions[0];
  });

  fireEvent.click(option);
};

const getSaveButton = () => screen.getByRole('button', { name: /Save$/ });
const findSaveButton = () => screen.findByRole('button', { name: /Save$/ });
const getCancelButton = () => screen.getByRole('button', { name: 'Cancel' });

const isDisabled = (button: HTMLElement) => (button as HTMLButtonElement).disabled;

const typeCronExpression = async (cronExpression: string) => {
  fireEvent.change(await screen.findByLabelText('Cron expression (UTC)'), {
    target: { value: cronExpression },
  });
};

const turnVerificationOff = () => {
  fireEvent.click(screen.getByRole('switch', { name: 'Scheduled verification' }));
};

const saveAndGetSubmittedConfig = async (submitted: VerificationConfigInput[]) => {
  fireEvent.click(getSaveButton());
  await waitFor(() => expect(submitted).toHaveLength(1));

  return submitted[0];
};

const saveAndGetSubmittedSchedule = async (submitted: VerificationConfigInput[]) =>
  (await saveAndGetSubmittedConfig(submitted)).schedule;

const EditorWithLanguageSwitch = ({ databaseId }: { databaseId: string }) => {
  const { setLocale } = useLocale();

  return (
    <>
      <button onClick={() => setLocale('fr')}>Change language</button>
      <EditVerificationConfigComponent databaseId={databaseId} onClose={() => {}} />
    </>
  );
};

describe('EditVerificationConfigComponent', () => {
  beforeEach(() => {
    vi.stubEnv('TZ', 'Europe/Moscow');
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-27T22:00:00Z'));
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('en-GB');
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  it('keeps a weekday chosen before a time and converts it with the time', async () => {
    const submitted = renderConfig({ type: 'HOURLY' });

    await chooseOption({ label: 'Verification interval', optionTitle: 'Weekly' });
    await chooseOption({ label: 'Weekday', optionTitle: 'Mon' });

    const timeLabel = await screen.findByText('Time of day');
    const timeInput = timeLabel.parentElement!.querySelector('input')!;

    fireEvent.click(timeInput);
    fireEvent.paste(timeInput, { clipboardData: { getData: () => '12:00' } });
    fireEvent.keyDown(timeInput, { key: 'Enter' });

    const submittedSchedule = await saveAndGetSubmittedSchedule(submitted);

    expect(submittedSchedule).toEqual({ type: 'WEEKLY', weekday: 1, timeOfDay: '09:00' });
  });

  it('shows a saved weekday 0 as Sunday', async () => {
    renderConfig({ type: 'WEEKLY', timeOfDay: '12:00', weekday: 0 });

    expect(
      await within(await screen.findByRole('combobox', { name: 'Weekday' })).findByText('Sun'),
    ).toBeTruthy();
  });

  it('names the fields Weekday and Time of day without the word Verification', async () => {
    renderConfig(weeklySchedule);

    expect(await screen.findByLabelText('Weekday')).toBeTruthy();
    expect(screen.getByLabelText('Time of day')).toBeTruthy();
    expect(screen.queryByText('Verification weekday')).toBeNull();
    expect(screen.queryByText('Verification time of day')).toBeNull();
  });

  it('keeps the draft when the language changes', async () => {
    const config = buildConfig({ schedule: { type: 'CRON', cronExpression: '0 2 * * *' } });
    const submitted = serveConfig(config);
    renderScreen(<EditorWithLanguageSwitch databaseId={config.databaseId} />);

    await typeCronExpression('0 3 * * *');
    fireEvent.click(screen.getByRole('button', { name: 'Change language' }));

    const translatedField = await screen.findByLabelText(DICTIONARIES.fr.schedules.cronExpression);

    expect((translatedField as HTMLInputElement).value).toBe('0 3 * * *');
    expect(
      isDisabled(
        screen.getByRole('button', {
          name: new RegExp(`${DICTIONARIES.fr.common.actions.save}$`),
        }),
      ),
    ).toBe(false);
    expect(submitted).toHaveLength(0);
  });

  it('shows an expression with no date to run on as invalid and keeps Save disabled', async () => {
    renderConfig({ type: 'CRON', cronExpression: '0 2 * * *' });

    await typeCronExpression(INVALID_CRON_EXPRESSION);

    expect(await screen.findByText('Invalid cron expression')).toBeTruthy();
    expect(screen.queryByText(/^Next run/)).toBeNull();
    expect(isDisabled(getSaveButton())).toBe(true);
  });

  it('sends the acknowledged schedule when an invalid expression is hidden by choosing after-backup', async () => {
    const submitted = renderConfig(weeklySchedule);

    await chooseOption({ label: 'Verification interval', optionTitle: 'Cron' });
    await typeCronExpression(INVALID_CRON_EXPRESSION);
    await chooseOption({ label: 'Verification interval', optionTitle: 'After backup' });

    expect(screen.queryByLabelText('Cron expression (UTC)')).toBeNull();
    expect(isDisabled(getSaveButton())).toBe(false);

    const submittedConfig = await saveAndGetSubmittedConfig(submitted);

    expect(submittedConfig.scheduleType).toBe('AFTER_BACKUP');
    expect(submittedConfig.schedule).toEqual(weeklySchedule);
  });

  it('sends the acknowledged schedule when an invalid expression is hidden by turning verification off', async () => {
    const submitted = renderConfig(weeklySchedule);

    await chooseOption({ label: 'Verification interval', optionTitle: 'Cron' });
    await typeCronExpression(INVALID_CRON_EXPRESSION);
    turnVerificationOff();

    expect(screen.queryByLabelText('Cron expression (UTC)')).toBeNull();
    expect(isDisabled(getSaveButton())).toBe(false);

    const submittedConfig = await saveAndGetSubmittedConfig(submitted);

    expect(submittedConfig.isAutomaticVerificationEnabled).toBe(false);
    expect(submittedConfig.schedule).toEqual(weeklySchedule);
  });

  it('offers a retry and Cancel after a failed load, and a successful retry shows the form', async () => {
    const onClose = vi.fn();

    mockServer.use(
      mockApi.get(
        '/verification-configs/{databaseId}',
        403,
        buildProblem('permission_denied', 403),
      ),
    );

    renderScreen(<EditVerificationConfigComponent databaseId="database-id" onClose={onClose} />);

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();

    fireEvent.click(getCancelButton());
    expect(onClose).toHaveBeenCalledTimes(1);

    serveConfig(buildConfig());
    fireEvent.click(retryButton);

    expect(await screen.findByLabelText('Weekday')).toBeTruthy();
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });

  it('keeps the input and shows the reason when the server refuses a save', async () => {
    const { onClose } = renderEditor(buildConfig({ schedule: { type: 'HOURLY' } }));

    await chooseOption({ label: 'Verification interval', optionTitle: 'After backup' });
    mockServer.use(
      mockApi.put(
        '/verification-configs/{databaseId}',
        403,
        buildProblem('permission_denied', 403),
      ),
    );
    fireEvent.click(getSaveButton());

    expect(await screen.findByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(screen.getByText(/^Runs automatically after each successful backup/)).toBeTruthy();
    expect(onClose).not.toHaveBeenCalled();
    expect(isDisabled(getCancelButton())).toBe(false);
  });

  it('disables Cancel while a save is pending', async () => {
    const config = buildConfig({ schedule: { type: 'HOURLY' } });
    const save = mockApi.delayed({
      method: 'put',
      path: '/verification-configs/{databaseId}',
      status: 200,
      body: config,
    });
    const { onClose } = renderEditor(config);

    await chooseOption({ label: 'Verification interval', optionTitle: 'After backup' });
    mockServer.use(save.handler);
    fireEvent.click(getSaveButton());

    await waitFor(() => expect(isDisabled(getCancelButton())).toBe(true));

    save.release();

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
  });

  it('disables Save again after an edit is reverted', async () => {
    renderConfig({ type: 'HOURLY' });

    const saveButton = await findSaveButton();

    expect(isDisabled(saveButton)).toBe(true);

    await chooseOption({ label: 'Verification interval', optionTitle: 'Daily' });
    await chooseOption({ label: 'Verification interval', optionTitle: 'Hourly' });

    expect(isDisabled(getSaveButton())).toBe(true);
  });

  it('disables Save after a schedule edit hidden by returning to after-backup', async () => {
    renderEditor(buildConfig({ scheduleType: 'AFTER_BACKUP' }));

    await chooseOption({ label: 'Verification interval', optionTitle: 'Weekly' });

    const timeInput = await screen.findByLabelText('Time of day');

    fireEvent.click(timeInput);
    fireEvent.paste(timeInput, { clipboardData: { getData: () => '18:00' } });
    fireEvent.keyDown(timeInput, { key: 'Enter' });

    expect(isDisabled(getSaveButton())).toBe(false);

    await chooseOption({ label: 'Verification interval', optionTitle: 'After backup' });

    expect(isDisabled(getSaveButton())).toBe(true);
  });

  it('disables Save after a notification edit hidden by turning verification off', async () => {
    renderEditor(buildConfig({ isAutomaticVerificationEnabled: false }));

    await findSaveButton();
    fireEvent.click(screen.getByRole('switch', { name: 'Scheduled verification' }));
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Verification failed' }));
    turnVerificationOff();

    expect(isDisabled(getSaveButton())).toBe(true);
  });

  it('sends the acknowledged values after a notification trigger and the schedule choice are hidden by turning verification off', async () => {
    const { submitted } = renderEditor(
      buildConfig({ notificationTriggers: ['VERIFICATION_SUCCESS'] }),
    );

    fireEvent.click(await screen.findByRole('checkbox', { name: 'Verification failed' }));
    await chooseOption({ label: 'Verification interval', optionTitle: 'After backup' });
    turnVerificationOff();

    expect(await saveAndGetSubmittedConfig(submitted)).toEqual({
      isAutomaticVerificationEnabled: false,
      scheduleType: 'INTERVAL',
      schedule: weeklySchedule,
      notificationTriggers: ['VERIFICATION_SUCCESS'],
    });
  });

  describe('late answers', () => {
    const SECOND_DATABASE_ID = 'second-database-id';
    const ANSWER_OUTCOMES = [
      { outcome: 'successful', status: 200 },
      { outcome: 'refused', status: 403 },
    ];

    const buildAnswer = (config: VerificationConfig, status: number) =>
      status === 200
        ? HttpResponse.json(config)
        : HttpResponse.json(buildProblem('permission_denied', 403), { status });

    const expectNoFeedback = () => {
      expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    };

    const serveSecondDatabaseConfig = () => {
      mockServer.use(
        http.get(`*${API_BASE_PATH}/verification-configs/${SECOND_DATABASE_ID}`, () =>
          HttpResponse.json(buildConfig({ databaseId: SECOND_DATABASE_ID })),
        ),
      );
    };

    const startHeldSave = async (onClose: () => void) => {
      const config = buildConfig({ schedule: { type: 'HOURLY' } });
      const heldSave = createPendingApiResponse();
      const saveCall = vi.spyOn(verificationConfigApi, 'save');

      serveConfig(config);

      const view = renderScreen(
        <EditVerificationConfigComponent databaseId={config.databaseId} onClose={onClose} />,
      );

      await chooseOption({ label: 'Verification interval', optionTitle: 'After backup' });
      mockServer.use(
        http.put(`*${API_BASE_PATH}/verification-configs/:databaseId`, () => heldSave.response),
      );
      fireEvent.click(getSaveButton());
      await waitFor(() => expect(saveCall).toHaveBeenCalledTimes(1));

      return { config, heldSave, saveCall, view };
    };

    it.each(ANSWER_OUTCOMES)(
      'shows nothing for a $outcome load answer after unmount',
      async ({ status }) => {
        const loadCall = vi.spyOn(verificationConfigApi, 'getByDatabaseId');
        const heldLoad = createPendingApiResponse();

        mockServer.use(
          http.get(`*${API_BASE_PATH}/verification-configs/:databaseId`, () => heldLoad.response),
        );

        const { unmount } = renderScreen(
          <EditVerificationConfigComponent databaseId="database-id" onClose={() => {}} />,
        );

        unmount();
        await releaseAndSettle(loadCall, () =>
          heldLoad.respond(buildAnswer(buildConfig(), status)),
        );

        expectNoFeedback();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'shows nothing for a $outcome load answer after a session change',
      async ({ status }) => {
        const loadCall = vi.spyOn(verificationConfigApi, 'getByDatabaseId');
        const heldLoad = createPendingApiResponse();

        activateSession({ token: 'first-token', userId: 'first-user' });
        mockServer.use(
          http.get(`*${API_BASE_PATH}/verification-configs/:databaseId`, () => heldLoad.response),
        );
        renderScreen(
          <EditVerificationConfigComponent databaseId="database-id" onClose={() => {}} />,
        );

        act(() => {
          invalidateSession();
        });
        await releaseAndSettle(loadCall, () =>
          heldLoad.respond(buildAnswer(buildConfig(), status)),
        );

        expectNoFeedback();
        expect(screen.queryByRole('button', { name: /Save$/ })).toBeNull();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'keeps the second database for a $outcome load answer of the first one',
      async ({ status }) => {
        const loadCall = vi.spyOn(verificationConfigApi, 'getByDatabaseId');
        const heldLoad = createPendingApiResponse();

        mockServer.use(
          http.get(`*${API_BASE_PATH}/verification-configs/database-id`, () => heldLoad.response),
        );
        serveSecondDatabaseConfig();

        const { rerender } = renderScreen(
          <EditVerificationConfigComponent databaseId="database-id" onClose={() => {}} />,
        );

        rerender(
          <EditVerificationConfigComponent databaseId={SECOND_DATABASE_ID} onClose={() => {}} />,
        );
        await screen.findByLabelText('Weekday');

        await releaseAndSettle(
          loadCall,
          () =>
            heldLoad.respond(buildAnswer(buildConfig({ schedule: { type: 'HOURLY' } }), status)),
          0,
        );

        expectNoFeedback();
        expect(screen.getByLabelText('Weekday')).toBeTruthy();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'shows no feedback for a $outcome save answer after unmount',
      async ({ status }) => {
        const onClose = vi.fn();
        const { config, heldSave, saveCall, view } = await startHeldSave(onClose);

        view.unmount();
        await releaseAndSettle(saveCall, () => heldSave.respond(buildAnswer(config, status)));

        expectNoFeedback();
        expect(onClose).not.toHaveBeenCalled();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'shows no feedback for a $outcome save answer after a session change',
      async ({ status }) => {
        const onClose = vi.fn();

        activateSession({ token: 'first-token', userId: 'first-user' });

        const { config, heldSave, saveCall } = await startHeldSave(onClose);

        act(() => {
          invalidateSession();
        });
        await releaseAndSettle(saveCall, () => heldSave.respond(buildAnswer(config, status)));

        expectNoFeedback();
        expect(onClose).not.toHaveBeenCalled();
      },
    );

    it.each(ANSWER_OUTCOMES)(
      'leaves the section untouched for a $outcome save answer after the database changed',
      async ({ status }) => {
        const onClose = vi.fn();
        const { config, heldSave, saveCall, view } = await startHeldSave(onClose);

        serveSecondDatabaseConfig();
        view.rerender(
          <EditVerificationConfigComponent databaseId={SECOND_DATABASE_ID} onClose={onClose} />,
        );
        await screen.findByLabelText('Weekday');
        expect(isDisabled(getCancelButton())).toBe(false);

        await releaseAndSettle(saveCall, () => heldSave.respond(buildAnswer(config, status)));

        expectNoFeedback();
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByLabelText('Weekday')).toBeTruthy();
        expect(isDisabled(getSaveButton())).toBe(true);
      },
    );
  });
});
