import { CronExpressionParser } from 'cron-parser';
import dayjs from 'dayjs';

const CRON_FIELD_COUNT = 5;
const WEEKDAY_FIELD_INDEX = 4;
const MAX_CRON_WEEKDAY = 6;
const SERVER_SEARCH_YEARS = 5;

// The server's parser takes numbers, wildcards and three-letter names only. Whole names are matched
// because JUL and THU hold the letters of the L and H tokens the server refuses.
const SERVER_FIELD_PATTERN = /^(?:[\d*,/?-]|\b[a-zA-Z]{3}\b)+$/;

const hasWeekdayAboveServerMaximum = (weekdayField: string): boolean => {
  const fieldWithoutSteps = weekdayField.replace(/\/\d+/g, '');

  return (fieldWithoutSteps.match(/\d+/g) ?? []).some(
    (weekdayNumber) => Number(weekdayNumber) > MAX_CRON_WEEKDAY,
  );
};

const parseNextRun = (expression: string): Date | null => {
  try {
    return CronExpressionParser.parse(expression, { tz: 'UTC' }).next().toDate();
  } catch {
    return null;
  }
};

// Answers only for what the server accepts. Past the server's search window the parser returns a false
// date for a day none of the listed months has, such as February or April 31.
export const getNextCronRun = (expression: string): Date | null => {
  const fields = expression.trim().split(/\s+/);

  if (fields.length !== CRON_FIELD_COUNT) return null;
  if (!fields.every((field) => SERVER_FIELD_PATTERN.test(field))) return null;
  if (hasWeekdayAboveServerMaximum(fields[WEEKDAY_FIELD_INDEX])) return null;

  const nextRun = parseNextRun(expression);
  if (!nextRun) return null;

  const isInsideServerSearch = dayjs(nextRun).isBefore(dayjs().add(SERVER_SEARCH_YEARS, 'year'));

  return isInsideServerSearch ? nextRun : null;
};
