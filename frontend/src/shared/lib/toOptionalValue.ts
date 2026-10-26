// An empty field is left out of the request: an optional setting stays unset, and a secret of a
// saved notifier, storage or database keeps its stored value.
export const toOptionalValue = (value: string | undefined): string | undefined =>
  value || undefined;
