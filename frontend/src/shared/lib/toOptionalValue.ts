// An empty field is left out of the request: an optional setting stays unset, and a secret of a
// saved notifier or storage keeps its stored value.
export const toOptionalValue = (value: string): string | undefined => value || undefined;
