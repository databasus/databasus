import { Button } from '@mantine/core';
import { Check, Copy } from 'lucide-react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { CommandToCopy } from '@/pages/databases/physical-backups/ui/useCopiedCommand';

interface Props {
  id: string;
  title: string;
  code: string;
  copiedCommandId: string | null;
  onCopy: (command: CommandToCopy) => void;
}

export const CopyableCommand = ({
  id,
  title,
  code,
  copiedCommandId,
  onCopy,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const isCopied = copiedCommandId === id;

  return (
    <div className="mt-3">
      <div className="mb-1 flex flex-wrap items-center justify-between gap-2">
        <div className="text-xs font-medium text-muted">{title}</div>
        <Button
          size="xs"
          variant="subtle"
          leftSection={
            isCopied ? (
              <Check size="1em" className="inline-block shrink-0 align-middle" />
            ) : (
              <Copy size="1em" className="inline-block shrink-0 align-middle" />
            )
          }
          onClick={() => onCopy({ commandId: id, code })}
        >
          {isCopied ? t('backups.physical.restore.copied') : t('common.actions.copy')}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded bg-code p-3 text-xs whitespace-pre-wrap text-content">
        {code}
      </pre>
    </div>
  );
};
