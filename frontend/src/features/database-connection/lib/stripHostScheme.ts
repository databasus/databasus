export const stripHostScheme = (rawHost: string): string =>
  rawHost.trim().replace('https://', '').replace('http://', '');
