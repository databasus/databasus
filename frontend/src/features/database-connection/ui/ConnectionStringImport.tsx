import { Button } from '@mantine/core';
import { Copy } from 'lucide-react';
import { useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '@/shared/auth';
import { type LocalizedText, translateLocalizedText } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';
import { ClipboardPasteModalComponent } from '@/shared/ui';

interface ParseFailure {
  error: LocalizedText;
}

interface Props<ParseOutcome> {
  parse: (connectionString: string) => ParseOutcome;
  onParsed: (parsed: Exclude<ParseOutcome, ParseFailure>) => void;
}

const isParseFailure = (parseOutcome: object): parseOutcome is ParseFailure =>
  'error' in parseOutcome;
export const ConnectionStringImport = <
  ParseOutcome extends
    | {
        password?: string;
      }
    | ParseFailure,
>({
  parse,
  onParsed,
}: Props<ParseOutcome>) => {
  const { t } = useTranslation();
  const notify = useFeedback();
  const isActive = useSessionGuard();
  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      notify({ kind: 'error', title: t('databases.edit.clipboardEmpty') });
      return;
    }

    const parseOutcome = latest.current.parse(trimmedText);

    if (isParseFailure(parseOutcome)) {
      notify({ kind: 'error', title: translateLocalizedText(parseOutcome.error, t) });
      return;
    }

    const parsed = parseOutcome as Exclude<ParseOutcome, ParseFailure>;
    latest.current.onParsed(parsed);

    if (parsed.password) {
      notify({ kind: 'success', title: t('databases.edit.connectionStringParsed') });
    } else {
      notify({ kind: 'warning', title: t('databases.edit.connectionStringParsedWithoutPassword') });
    }
  };

  const startImport = async () => {
    if (!BrowserClipboard.isClipboardApiAvailable()) {
      setIsShowPasteModal(true);
      return;
    }

    try {
      const text = await BrowserClipboard.readFromClipboard();
      if (isActive()) applyConnectionString(text);
    } catch {
      if (isActive()) notify({ kind: 'error', title: t('databases.edit.clipboardReadFailed') });
    }
  };

  const latest = useRef({ parse, onParsed });

  useLayoutEffect(() => {
    latest.current = { parse, onParsed };
  });

  return (
    <div className="mb-3 flex">
      <div className="hidden sm:block sm:w-[150px] sm:shrink-0" />
      <Button
        variant="transparent"
        size="xs"
        className="!h-auto !px-0 text-left !whitespace-normal"
        leftSection={<Copy aria-hidden size={14} />}
        onClick={startImport}
      >
        {t('databases.edit.parseFromClipboard')}
      </Button>

      <ClipboardPasteModalComponent
        open={isShowPasteModal}
        onSubmit={(text) => {
          setIsShowPasteModal(false);
          applyConnectionString(text);
        }}
        onCancel={() => setIsShowPasteModal(false)}
      />
    </div>
  );
};
