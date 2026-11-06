/* eslint-disable i18next/no-literal-string -- a sample HTTP request carrying the English text the backend sends */
import type { WebhookHeader } from '@/entities/notifiers';

const EXAMPLE_HEADING = '✅ Backup completed for database "my-database" (workspace "Production")';

const EXAMPLE_JSON_ESCAPED_MESSAGE =
  'Backup completed successfully in 1m 23s.\\nCompressed backup size: 256.00 MB';

const EXAMPLE_DEFAULT_BODY = `{
  "heading": "✅ Backup completed for database "my-database" (workspace "My workspace")",
  "message": "Backup completed successfully in 1m 23s. Compressed backup size: 256.00 MB"
}`;

export const EXAMPLE_GET_QUERY =
  '?heading=✅ Backup completed for database "my-database" (workspace "Production")&message=Backup completed successfully in 1m 23s.%0ACompressed backup size: 256.00 MB';

export const getExamplePostHeaderLines = (headers: WebhookHeader[]): string => {
  const contentTypeLine = headers.some((header) => header.name.toLowerCase() === 'content-type')
    ? ''
    : 'Content-Type: application/json';

  const customHeaderLines = headers
    .filter((header) => header.name)
    .map((header) => `\n${header.name}: ${header.value ?? ''}`)
    .join('');

  return `${contentTypeLine}${customHeaderLines}`;
};

export const getExamplePostBody = (bodyTemplate: string | undefined): string =>
  bodyTemplate
    ? bodyTemplate
        .replace('{{heading}}', EXAMPLE_HEADING)
        .replace('{{message}}', EXAMPLE_JSON_ESCAPED_MESSAGE)
    : EXAMPLE_DEFAULT_BODY;
