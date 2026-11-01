/* eslint-disable i18next/no-literal-string -- dayjs format tokens, not copy */
// Detect date order (MDY, DMY, YMD) and separator from user's locale
const getLocaleDateFormat = () => {
  const locale = navigator.language || 'en-US';

  // Use a test date where day, month, year are all different: March 15, 2023
  const testDate = new Date(2023, 2, 15);
  const formatted = testDate.toLocaleDateString(locale);

  // Detect separator: . / or -
  let separator = '/';
  if (formatted.includes('.')) separator = '.';
  else if (formatted.includes('-')) separator = '-';

  // Detect order by checking position of 15 (day), 3 (month), 2023/23 (year)
  const parts = formatted.split(/[./-]/);
  const dayIndex = parts.findIndex((p) => p === '15');
  const monthIndex = parts.findIndex((p) => p === '3' || p === '03');
  const yearIndex = parts.findIndex((p) => p === '2023' || p === '23');

  // Default to DMY if detection fails
  let dateFormat = `DD${separator}MM${separator}YYYY`;
  let shortDateFormat = `DD MMM YYYY`;

  if (yearIndex === 0) {
    // YMD (China, Japan, Korea, ISO)
    dateFormat = `YYYY${separator}MM${separator}DD`;
    shortDateFormat = `YYYY MMM DD`;
  } else if (monthIndex === 0 && dayIndex === 1) {
    // MDY (USA)
    dateFormat = `MM${separator}DD${separator}YYYY`;
    shortDateFormat = `MMM DD, YYYY`;
  } else {
    // DMY (Europe, Russia, most of the world)
    dateFormat = `DD${separator}MM${separator}YYYY`;
    shortDateFormat = `DD MMM YYYY`;
  }

  return { dateFormat, shortDateFormat, separator };
};

// Detect if user prefers 12-hour time format
const getIs12HourFormat = () => {
  const locale = navigator.language || 'en-US';
  const testDate = new Date(2023, 0, 1, 13, 0, 0); // 1 PM
  const timeString = testDate.toLocaleTimeString(locale, { hour: 'numeric' });
  return timeString.includes('PM') || timeString.includes('AM');
};

export const getUserTimeFormat = () => {
  const { dateFormat } = getLocaleDateFormat();
  const is12Hour = getIs12HourFormat();

  return {
    use12Hours: is12Hour,
    format: is12Hour ? `${dateFormat} h:mm A` : `${dateFormat} HH:mm`,
  };
};

export const getUserClockFormat = () => {
  const is12Hour = getIs12HourFormat();

  return { use12Hours: is12Hour, format: is12Hour ? 'h:mm A' : 'HH:mm' };
};

export const getUserShortTimeFormat = () => {
  const { shortDateFormat } = getLocaleDateFormat();
  const is12Hour = getIs12HourFormat();

  return {
    use12Hours: is12Hour,
    format: is12Hour ? `${shortDateFormat} h:mm A` : `${shortDateFormat} HH:mm`,
  };
};
