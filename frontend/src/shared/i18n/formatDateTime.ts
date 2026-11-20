// The month is a word in the selected language and the day comes before it where the language
// allows, as in "14 Nov 2025, 13:00". The clock follows the browser's 12- or 24-hour preference.
export const formatDateTime = (
  instant: string,
  dateLanguageTag: string,
  isTwelveHourClock: boolean,
): string =>
  new Intl.DateTimeFormat(dateLanguageTag, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: isTwelveHourClock,
  }).format(new Date(instant));
