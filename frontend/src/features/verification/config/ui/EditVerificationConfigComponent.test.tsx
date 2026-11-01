// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { VerificationConfig } from '../../../../entities/verification/config';
import { API_BASE_PATH } from '../../../../shared/api';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditVerificationConfigComponent } from './EditVerificationConfigComponent';

const mockServer = setUpMockApi();

const renderConfig = (schedule: VerificationConfig['schedule']) => {
  const submitted: VerificationConfig[] = [];
  const config: VerificationConfig = {
    databaseId: 'database-id',
    isAutomaticVerificationEnabled: true,
    scheduleType: 'INTERVAL',
    schedule,
    notificationTriggers: [],
  };

  mockServer.use(
    mockApi.get('/verification-configs/{databaseId}', 200, config),
    http.put(`*${API_BASE_PATH}/verification-configs/:databaseId`, async ({ request }) => {
      const saved = (await request.json()) as VerificationConfig;
      submitted.push(saved);

      return HttpResponse.json({ ...saved, databaseId: config.databaseId });
    }),
  );

  renderScreen(
    <EditVerificationConfigComponent databaseId={config.databaseId} onClose={() => {}} />,
  );

  return submitted;
};

const chooseOption = async ({ label, optionTitle }: { label: string; optionTitle: string }) => {
  const fieldLabel = await screen.findByText(label);

  fireEvent.mouseDown(fieldLabel.parentElement!.querySelector('.ant-select-selector')!);

  const option = await waitFor(() => {
    const visibleOption = document.querySelector(`.ant-select-item-option[title="${optionTitle}"]`);
    expect(visibleOption).toBeTruthy();
    return visibleOption!;
  });

  fireEvent.click(option);
};

const saveAndGetSubmittedSchedule = async (submitted: VerificationConfig[]) => {
  fireEvent.click(screen.getByRole('button', { name: /Save$/ }));
  await waitFor(() => expect(submitted).toHaveLength(1));

  return submitted[0].schedule;
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
    await chooseOption({ label: 'Verification weekday', optionTitle: 'Mon' });

    const timeLabel = await screen.findByText('Verification time of day');
    const timeInput = timeLabel.parentElement!.querySelector('input')!;

    fireEvent.mouseDown(timeInput);
    fireEvent.change(timeInput, { target: { value: '12:00' } });
    fireEvent.keyDown(timeInput, { key: 'Enter' });

    const submittedSchedule = await saveAndGetSubmittedSchedule(submitted);

    expect(submittedSchedule).toEqual({ type: 'WEEKLY', weekday: 1, timeOfDay: '09:00' });
  });

  it('shows a saved weekday 0 as Sunday', async () => {
    renderConfig({ type: 'WEEKLY', timeOfDay: '12:00', weekday: 0 });

    expect(await screen.findByText('Sun')).toBeTruthy();
  });
});
