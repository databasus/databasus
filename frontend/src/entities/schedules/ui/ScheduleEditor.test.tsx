// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { renderScreen } from '../../../shared/lib/test/renderScreen';
import type { Schedule } from '../model/Schedule';
import { ScheduleEditor } from './ScheduleEditor';

const renderEditor = (initialSchedule: Schedule | null) => {
  const returnedSchedules: Schedule[] = [];

  const EditorWithState = () => {
    const [schedule, setSchedule] = useState(initialSchedule);

    return (
      <ScheduleEditor
        label="Backup interval"
        schedule={schedule}
        onChange={(nextSchedule) => {
          returnedSchedules.push(nextSchedule);
          setSchedule(nextSchedule);
        }}
      />
    );
  };

  renderScreen(<EditorWithState />);

  return returnedSchedules;
};

const chooseOption = async ({ label, optionTitle }: { label: string; optionTitle: string }) => {
  const control = screen.getByRole('combobox', { name: label });

  fireEvent.mouseDown(control.closest('.ant-select')!.querySelector('.ant-select-selector')!);

  const option = await waitFor(() => {
    const visibleOption = document.querySelector(`.ant-select-item-option[title="${optionTitle}"]`);
    expect(visibleOption).toBeTruthy();
    return visibleOption!;
  });

  fireEvent.click(option);
};

const chooseTime = (time: string) => {
  const timeInput = screen.getByLabelText('Time of day');

  fireEvent.mouseDown(timeInput);
  fireEvent.change(timeInput, { target: { value: time } });
  fireEvent.keyDown(timeInput, { key: 'Enter' });
};

const pinClock = ({ zone, moment }: { zone: string; moment: string }) => {
  vi.stubEnv('TZ', zone);
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(new Date(moment));
};

const getSelectedOption = (label: string) =>
  screen
    .getByRole('combobox', { name: label })
    .closest('.ant-select')!
    .querySelector('.ant-select-selection-item')?.textContent;

describe('ScheduleEditor', () => {
  beforeEach(() => {
    pinClock({ zone: 'UTC', moment: '2026-11-15T12:00:00Z' });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  describe('at UTC+3 with the clock at 01:00 local', () => {
    beforeEach(() => {
      pinClock({ zone: 'Europe/Moscow', moment: '2026-10-27T22:00:00Z' });
      vi.spyOn(navigator, 'language', 'get').mockReturnValue('en-GB');
    });

    it('keeps a weekday chosen before a time and converts it with the time', async () => {
      const returnedSchedules = renderEditor({ type: 'HOURLY' });

      await chooseOption({ label: 'Backup interval', optionTitle: 'Weekly' });
      await chooseOption({ label: 'Weekday', optionTitle: 'Mon' });

      expect(getSelectedOption('Weekday')).toBe('Mon');

      chooseTime('12:00');

      expect(returnedSchedules[returnedSchedules.length - 1]).toEqual({
        type: 'WEEKLY',
        weekday: 1,
        timeOfDay: '09:00',
      });
    });

    it('keeps a day of the month chosen before a time and converts it with the time', async () => {
      const returnedSchedules = renderEditor({ type: 'HOURLY' });

      await chooseOption({ label: 'Backup interval', optionTitle: 'Monthly' });
      fireEvent.change(screen.getByLabelText('Day of month'), { target: { value: '10' } });

      expect((screen.getByLabelText('Day of month') as HTMLInputElement).value).toBe('10');

      chooseTime('02:00');

      expect(returnedSchedules[returnedSchedules.length - 1]).toEqual({
        type: 'MONTHLY',
        dayOfMonth: 9,
        timeOfDay: '23:00',
      });
    });

    it('returns the converted UTC day when a weekday is chosen on a schedule with a time', async () => {
      const returnedSchedules = renderEditor({ type: 'WEEKLY', timeOfDay: '23:00', weekday: 7 });

      expect(getSelectedOption('Weekday')).toBe('Mon');

      await chooseOption({ label: 'Weekday', optionTitle: 'Wed' });

      expect(returnedSchedules).toEqual([{ type: 'WEEKLY', timeOfDay: '23:00', weekday: 2 }]);
    });
  });

  it('offers only the type choice without a schedule and returns the chosen type', async () => {
    const returnedSchedules = renderEditor(null);

    expect(screen.queryByLabelText('Time of day')).toBeNull();

    await chooseOption({ label: 'Backup interval', optionTitle: 'Daily' });

    expect(returnedSchedules).toEqual([{ type: 'DAILY' }]);
    expect(screen.getByLabelText('Time of day')).toBeTruthy();
  });

  it('keeps the other fields in the returned schedule after a type switch', async () => {
    const returnedSchedules = renderEditor({ type: 'WEEKLY', timeOfDay: '04:00', weekday: 2 });

    await chooseOption({ label: 'Backup interval', optionTitle: 'Daily' });

    expect(returnedSchedules).toEqual([{ type: 'DAILY', timeOfDay: '04:00', weekday: 2 }]);
  });

  it('returns the whole schedule when a day changes', () => {
    const returnedSchedules = renderEditor({ type: 'MONTHLY', timeOfDay: '12:00', dayOfMonth: 5 });

    fireEvent.change(screen.getByLabelText('Day of month'), { target: { value: '31' } });

    expect(returnedSchedules).toEqual([{ type: 'MONTHLY', timeOfDay: '12:00', dayOfMonth: 31 }]);
  });

  it('names the fields of a schedule through a group', () => {
    renderEditor({ type: 'WEEKLY', timeOfDay: '04:00', weekday: 1 });

    const group = screen.getByRole('group', { name: 'Backup interval' });

    expect(within(group).getByLabelText('Weekday')).toBeTruthy();
    expect(within(group).getByLabelText('Time of day')).toBeTruthy();
  });

  it.each([
    '0 2 * * *',
    '0 0 ? * *',
    '0 0 * * MON',
    '0 0 29 2 *',
    '*/5 1-3 1,15 JUL mon-fri',
    '*/5 1-3 1,15 JAN mon-fri',
  ])('shows the next run of %s', (cronExpression) => {
    renderEditor({ type: 'CRON', cronExpression });

    expect(screen.getByText(/^Next run /)).toBeTruthy();
    expect(screen.queryByText('Invalid cron expression')).toBeNull();
  });

  it.each([
    '0 0 2 * * *',
    '@daily',
    '0 2 * *',
    '0 0 L * *',
    '0 0 * * 1#2',
    '0 0 H * *',
    '0 0 * * 7',
    '0 0 31 2 *',
    '0 0 31 2,4 *',
    '0 0 31 FEB,APR *',
    'TZ=Europe/Moscow 0 2 * * *',
    'CRON_TZ=UTC 0 2 * * *',
  ])('shows %s as invalid without a next run', (cronExpression) => {
    renderEditor({ type: 'CRON', cronExpression });

    expect(screen.getByText('Invalid cron expression')).toBeTruthy();
    expect(screen.queryByText(/^Next run /)).toBeNull();
  });

  it('returns the typed cron expression', () => {
    const returnedSchedules = renderEditor({ type: 'CRON' });

    fireEvent.change(screen.getByLabelText('Cron expression (UTC)'), {
      target: { value: '0 3 * * 1' },
    });

    expect(returnedSchedules).toEqual([{ type: 'CRON', cronExpression: '0 3 * * 1' }]);
  });

  it('takes a time in the 12-hour format of the user', () => {
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('en-US');
    const returnedSchedules = renderEditor({ type: 'DAILY', timeOfDay: '00:00' });

    chooseTime('1:00 PM');

    expect(returnedSchedules[returnedSchedules.length - 1].timeOfDay).toBe('13:00');
    expect((screen.getByLabelText('Time of day') as HTMLInputElement).value).toBe('1:00 PM');
  });

  it('takes a time in the 24-hour format of the user', () => {
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('en-GB');
    const returnedSchedules = renderEditor({ type: 'DAILY', timeOfDay: '00:00' });

    chooseTime('13:00');

    expect(returnedSchedules[returnedSchedules.length - 1].timeOfDay).toBe('13:00');
    expect((screen.getByLabelText('Time of day') as HTMLInputElement).value).toBe('13:00');
  });
});
