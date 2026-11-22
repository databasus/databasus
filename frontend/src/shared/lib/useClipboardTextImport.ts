import { useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '@/shared/auth';
import { type LocalizedText, type TranslationKey, translateLocalizedText } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';

interface ParseFailure {
  error: LocalizedText;
}

export interface ClipboardTextImportMessages {
  clipboardEmpty: TranslationKey;
  clipboardReadFailed: TranslationKey;
  parsed: TranslationKey;
  // Shown instead of parsed when the text lacked something the member has to enter by hand.
  parsedIncomplete: TranslationKey;
}

interface Options<ParseOutcome extends object> {
  parse: (text: string) => ParseOutcome;
  isIncomplete: (parsed: Exclude<ParseOutcome, ParseFailure>) => boolean;
  onParsed: (parsed: Exclude<ParseOutcome, ParseFailure>) => void;
  messages: ClipboardTextImportMessages;
}

const isParseFailure = (parseOutcome: object): parseOutcome is ParseFailure =>
  'error' in parseOutcome;

// Reads text from the clipboard, parses it and reports the outcome. A browser without the clipboard
// API, such as one on plain HTTP, gets a modal to paste into instead.
export const useClipboardTextImport = <ParseOutcome extends object>({
  parse,
  isIncomplete,
  onParsed,
  messages,
}: Options<ParseOutcome>) => {
  const { t } = useTranslation();
  const notify = useFeedback();
  const isActive = useSessionGuard();
  const [isPasteModalOpen, setIsPasteModalOpen] = useState(false);

  const applyText = (text: string) => {
    const { messages: currentMessages } = latest.current;
    const trimmedText = text.trim();

    if (!trimmedText) {
      notify({ kind: 'error', title: t(currentMessages.clipboardEmpty) });
      return;
    }

    const parseOutcome = latest.current.parse(trimmedText);

    if (isParseFailure(parseOutcome)) {
      notify({ kind: 'error', title: translateLocalizedText(parseOutcome.error, t) });
      return;
    }

    const parsed = parseOutcome as Exclude<ParseOutcome, ParseFailure>;
    latest.current.onParsed(parsed);

    if (latest.current.isIncomplete(parsed)) {
      notify({ kind: 'warning', title: t(currentMessages.parsedIncomplete) });
    } else {
      notify({ kind: 'success', title: t(currentMessages.parsed) });
    }
  };

  const startImport = async () => {
    if (!BrowserClipboard.isClipboardApiAvailable()) {
      setIsPasteModalOpen(true);
      return;
    }

    try {
      const text = await BrowserClipboard.readFromClipboard();
      if (isActive()) applyText(text);
    } catch {
      if (isActive())
        notify({ kind: 'error', title: t(latest.current.messages.clipboardReadFailed) });
    }
  };

  const latest = useRef({ parse, isIncomplete, onParsed, messages });

  useLayoutEffect(() => {
    latest.current = { parse, isIncomplete, onParsed, messages };
  });

  return {
    startImport,
    pasteModal: {
      open: isPasteModalOpen,
      onSubmit: (text: string) => {
        setIsPasteModalOpen(false);
        applyText(text);
      },
      onCancel: () => setIsPasteModalOpen(false),
    },
  };
};
