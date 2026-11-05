// Spaces at the edges do not count, because the form trims them when the field loses focus.
export const isWebhookHeaderNameValid = (name: string): boolean =>
  /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/.test(name.trim());
