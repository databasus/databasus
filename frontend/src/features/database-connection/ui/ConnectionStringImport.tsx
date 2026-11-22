import { Button } from '@mantine/core';
import { Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { useConnectionStringImport } from '@/features/database-connection/model/useConnectionStringImport';
import type { LocalizedText } from '@/shared/i18n';
import { ClipboardPasteModalComponent } from '@/shared/ui';

interface ParseFailure {
  error: LocalizedText;
}

interface Props<ParseOutcome> {
  parse: (connectionString: string) => ParseOutcome;
  onParsed: (parsed: Exclude<ParseOutcome, ParseFailure>) => void;
}

export const ConnectionStringImport = <ParseOutcome extends { password?: string } | ParseFailure>({
  parse,
  onParsed,
}: Props<ParseOutcome>) => {
  const { t } = useTranslation();
  const { startImport, pasteModal } = useConnectionStringImport({ parse, onParsed });

  return (
    <div className="mb-3 flex">
      <div className="hidden sm:block sm:w-[150px] sm:shrink-0" />
      <Button
        variant="transparent"
        size="xs"
        className="!h-auto !px-0 text-left !whitespace-normal"
        leftSection={<Copy aria-hidden size={14} />}
        onClick={() => void startImport()}
      >
        {t('databases.edit.parseFromClipboard')}
      </Button>

      <ClipboardPasteModalComponent {...pasteModal} />
    </div>
  );
};
