export const quotePostgresqlIdentifier = (identifier: string): string =>
  `"${identifier.replace(/"/g, '""')}"`;
