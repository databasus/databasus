// Empty UI fields omit a value; each provider owns whether an omitted secret is retained.
export const toOptionalValue = (value: string | undefined): string | undefined =>
  value || undefined;
