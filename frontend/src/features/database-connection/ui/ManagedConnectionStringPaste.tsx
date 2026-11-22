import { Button, Divider } from '@mantine/core';
import { ClipboardPaste } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  MANAGED_POSTGRESQL_PROVIDERS,
  type ManagedPostgresqlProvider,
  PostgresqlConnectionStringParser,
  type PostgresqlConnectionStringParseResult,
} from '@/entities/databases';
import { useConnectionStringImport } from '@/features/database-connection/model/useConnectionStringImport';
import { ClipboardPasteModalComponent } from '@/shared/ui';

interface Props {
  provider: ManagedPostgresqlProvider;
  onParsed: (parsed: PostgresqlConnectionStringParseResult) => void;
}

export const ManagedConnectionStringPaste = ({ provider, onParsed }: Props) => {
  const { t } = useTranslation();
  const { startImport, pasteModal } = useConnectionStringImport({
    parse: (connectionString) => PostgresqlConnectionStringParser.parse(connectionString),
    onParsed,
  });

  const { name: providerName, connectionStringPaste } = MANAGED_POSTGRESQL_PROVIDERS[provider];

  if (!connectionStringPaste) return null;

  return (
    <>
      <div>
        <div className="font-semibold">
          {t('databases.edit.connectionStringPaste.title', { provider: providerName })}
        </div>

        <p className="mt-1 mb-2 text-xs leading-snug text-muted">
          {t(connectionStringPaste.hintKey)}
        </p>

        <pre className="rounded-md bg-code p-2 font-mono text-xs break-all whitespace-pre-wrap text-muted">
          <code>{connectionStringPaste.example}</code>
        </pre>

        <Button
          className="mt-2"
          variant="light"
          size="xs"
          leftSection={<ClipboardPaste aria-hidden size={14} />}
          onClick={() => void startImport()}
        >
          {t('databases.edit.connectionStringPaste.pasteFromClipboard')}
        </Button>

        <ClipboardPasteModalComponent {...pasteModal} />
      </div>

      <Divider
        my="md"
        labelPosition="center"
        label={t('databases.edit.connectionStringPaste.orEnterManually')}
      />
    </>
  );
};
