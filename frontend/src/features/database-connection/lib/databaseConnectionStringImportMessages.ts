import type { ClipboardTextImportMessages } from '@/shared/lib/useClipboardTextImport';

export const DATABASE_CONNECTION_STRING_IMPORT_MESSAGES: ClipboardTextImportMessages = {
  clipboardEmpty: 'databases.edit.clipboardEmpty',
  clipboardReadFailed: 'databases.edit.clipboardReadFailed',
  parsed: 'databases.edit.connectionStringParsed',
  parsedIncomplete: 'databases.edit.connectionStringParsedWithoutPassword',
};
