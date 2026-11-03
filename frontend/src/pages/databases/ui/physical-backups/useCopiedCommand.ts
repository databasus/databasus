import { useEffect, useRef, useState } from 'react';

import { BrowserClipboard } from '../../../../shared/lib/BrowserClipboard';

const COPIED_MARK_MS = 2000;

export interface CommandToCopy {
  commandId: string;
  code: string;
}

export const useCopiedCommand = () => {
  const [copiedCommandId, setCopiedCommandId] = useState<string | null>(null);

  const copiedMarkTimer = useRef<number | undefined>(undefined);

  const copyCommand = async ({ commandId, code }: CommandToCopy) => {
    await BrowserClipboard.copyToClipboard(code);

    window.clearTimeout(copiedMarkTimer.current);
    setCopiedCommandId(commandId);
    copiedMarkTimer.current = window.setTimeout(() => setCopiedCommandId(null), COPIED_MARK_MS);
  };

  useEffect(() => () => window.clearTimeout(copiedMarkTimer.current), []);

  return { copiedCommandId, copyCommand };
};
