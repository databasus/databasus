import { CopyOutlined } from '@ant-design/icons';
import { App, Button } from 'antd';
import { useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '@/shared/auth';
import { type LocalizedText, translateLocalizedText } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
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

export const ConnectionStringImport = <ParseOutcome extends { password?: string } | ParseFailure>({
  parse,
  onParsed,
}: Props<ParseOutcome>) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const isActive = useSessionGuard();
  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error(t('databases.edit.clipboardEmpty'));
      return;
    }

    const parseOutcome = latest.current.parse(trimmedText);

    if (isParseFailure(parseOutcome)) {
      message.error(translateLocalizedText(parseOutcome.error, t));
      return;
    }

    const parsed = parseOutcome as Exclude<ParseOutcome, ParseFailure>;
    latest.current.onParsed(parsed);

    if (parsed.password) {
      message.success(t('databases.edit.connectionStringParsed'));
    } else {
      message.warning(t('databases.edit.connectionStringParsedWithoutPassword'));
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
      if (isActive()) message.error(t('databases.edit.clipboardReadFailed'));
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
        type="link"
        size="small"
        className="!h-auto !px-0 text-left !whitespace-normal"
        icon={<CopyOutlined aria-hidden />}
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
