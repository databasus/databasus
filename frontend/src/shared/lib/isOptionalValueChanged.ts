// The API leaves an empty optional setting out, while a form may hold it as an empty string.
export const isOptionalValueChanged = (
  value: string | number | undefined,
  savedValue: string | number | undefined,
): boolean => (value ?? '') !== (savedValue ?? '');
