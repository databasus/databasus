import { useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '@/shared/auth';
import { type LocalizedText, translateLocalizedText } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';

interface ParseFailure {
  error: LocalizedText;
}

interface Options<ParseOutcome> {
  parse: (connectionString: string) => ParseOutcome;
  onParsed: (parsed: Exclude<ParseOutcome, ParseFailure>) => void;
}

const isParseFailure = (parseOutcome: object): parseOutcome is ParseFailure =>
  'error' in parseOutcome;

// Reads a connection string from the clipboard and reports the outcome. A browser without the
// clipboard API, such as one on plain HTTP, gets a modal to paste into instead.
export const useConnectionStringImport = <
  ParseOutcome extends { password?: string } | ParseFailure,
>({
  parse,
  onParsed,
}: Options<ParseOutcome>) => {
  const { t } = useTranslation();
  const notify = useFeedback();
  const isActive = useSessionGuard();
  const [isPasteModalOpen, setIsPasteModalOpen] = useState(false);

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
      setIsPasteModalOpen(true);
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

  return {
    startImport,
    pasteModal: {
      open: isPasteModalOpen,
      onSubmit: (text: string) => {
        setIsPasteModalOpen(false);
        applyConnectionString(text);
      },
      onCancel: () => setIsPasteModalOpen(false),
    },
  };
};
