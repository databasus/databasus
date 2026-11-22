import { useTranslation } from 'react-i18next';

import {
  MANAGED_POSTGRESQL_PROVIDERS,
  type ManagedPostgresqlProvider,
  PostgresqlConnectionStringParser,
  type PostgresqlConnectionStringParseResult,
} from '@/entities/databases';
import { DATABASE_CONNECTION_STRING_IMPORT_MESSAGES } from '@/features/database-connection/lib/databaseConnectionStringImportMessages';
import { useClipboardTextImport } from '@/shared/lib/useClipboardTextImport';
import { ClipboardPasteBlockComponent } from '@/shared/ui';

interface Props {
  provider: ManagedPostgresqlProvider;
  onParsed: (parsed: PostgresqlConnectionStringParseResult) => void;
}

export const ManagedConnectionStringPaste = ({ provider, onParsed }: Props) => {
  const { t } = useTranslation();
  const { startImport, pasteModal } = useClipboardTextImport({
    parse: (connectionString) => PostgresqlConnectionStringParser.parse(connectionString),
    isIncomplete: (parsed) => !parsed.password,
    onParsed,
    messages: DATABASE_CONNECTION_STRING_IMPORT_MESSAGES,
  });

  const { name: providerName, connectionStringPaste } = MANAGED_POSTGRESQL_PROVIDERS[provider];

  if (!connectionStringPaste) return null;

  return (
    <ClipboardPasteBlockComponent
      title={t('databases.edit.connectionStringPaste.title', { provider: providerName })}
      hint={t(connectionStringPaste.hintKey)}
      example={connectionStringPaste.example}
      pasteLabel={t('databases.edit.connectionStringPaste.pasteFromClipboard')}
      dividerLabel={t('databases.edit.connectionStringPaste.orEnterManually')}
      onPaste={() => void startImport()}
      pasteModal={pasteModal}
    />
  );
};
