import { MantineThemeProvider, MultiSelect, PasswordInput, TagsInput } from '@mantine/core';
import {
  DateInput,
  DateTimePicker,
  DatesProvider,
  InlineDateTimePicker,
  TimePicker,
} from '@mantine/dates';
import dayjs from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import localeData from 'dayjs/plugin/localeData';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { useLocale } from '@/shared/i18n/useLocale';
import { getUserClockFormat, getUserDateFormat, getUserTimeFormat } from '@/shared/time';

dayjs.extend(customParseFormat);
dayjs.extend(localeData);

export const ControlLocalization = ({ children }: { children: ReactNode }) => {
  const { t } = useTranslation();
  const { dateLocale } = useLocale();
  const dateFormat = getUserDateFormat().dateFormat;
  const clockFormat = getUserClockFormat();
  const clearButtonProps = {
    'aria-label': t('common.actions.clear'),
    'aria-hidden': false as const,
    tabIndex: 0,
  };
  const calendarProps = {
    ariaLabels: {
      monthLevelControl: t('common.calendar.monthLevelControl'),
      yearLevelControl: t('common.calendar.yearLevelControl'),
      nextMonth: t('common.calendar.nextMonth'),
      previousMonth: t('common.calendar.previousMonth'),
      nextYear: t('common.calendar.nextYear'),
      previousYear: t('common.calendar.previousYear'),
      nextDecade: t('common.calendar.nextDecade'),
      previousDecade: t('common.calendar.previousDecade'),
    },
    getDayAriaLabel: (date: string) => dayjs(date).locale(dateLocale).format('dddd, D MMMM YYYY'),
  };
  const timeProps = {
    hoursInputLabel: t('schedules.hours'),
    minutesInputLabel: t('schedules.minutes'),
    secondsInputLabel: t('common.calendar.seconds'),
    amPmInputLabel: t('schedules.timePeriod'),
    format: clockFormat.use12Hours ? ('12h' as const) : ('24h' as const),
    amPmLabels: {
      am: dayjs().locale(dateLocale).hour(9).format('A'),
      pm: dayjs().locale(dateLocale).hour(15).format('A'),
    },
  };

  return (
    <MantineThemeProvider
      theme={{
        components: {
          DateInput: DateInput.extend({
            defaultProps: {
              ...calendarProps,
              clearButtonProps,
              valueFormat: dateFormat,
              dateParser: (value) => {
                const parsed = dayjs(value, [dateFormat, 'YYYY-MM-DD'], true);
                return parsed.isValid() ? parsed.format('YYYY-MM-DD') : null;
              },
            },
          }),
          DateTimePicker: DateTimePicker.extend({
            defaultProps: {
              ...calendarProps,
              clearButtonProps,
              valueFormat: getUserTimeFormat().format,
              timePickerProps: timeProps,
              submitButtonProps: { 'aria-label': t('common.actions.ok') },
            },
          }),
          InlineDateTimePicker: InlineDateTimePicker.extend({
            defaultProps: {
              ...calendarProps,
              timePickerProps: timeProps,
              submitButtonProps: { 'aria-label': t('common.actions.ok') },
            },
          }),
          TimePicker: TimePicker.extend({ defaultProps: { ...timeProps, clearButtonProps } }),
          MultiSelect: MultiSelect.extend({ defaultProps: { clearButtonProps } }),
          TagsInput: TagsInput.extend({ defaultProps: { clearButtonProps } }),
          PasswordInput: PasswordInput.extend({
            defaultProps: {
              visibilityToggleButtonProps: {
                'aria-label': t('common.actions.togglePasswordVisibility'),
              },
            },
          }),
        },
      }}
    >
      <DatesProvider
        settings={{
          locale: dateLocale,
          firstDayOfWeek: dayjs().locale(dateLocale).localeData().firstDayOfWeek() as
            | 0
            | 1
            | 2
            | 3
            | 4
            | 5
            | 6,
        }}
      >
        {children}
      </DatesProvider>
    </MantineThemeProvider>
  );
};
