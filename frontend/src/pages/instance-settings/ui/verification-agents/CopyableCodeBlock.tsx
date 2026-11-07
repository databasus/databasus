import { ActionIcon, Tooltip } from '@mantine/core';
import { Copy } from 'lucide-react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  code: string;
  onCopy: (text: string) => void;
}

export const CopyableCodeBlock = ({ code, onCopy }: Props): JSX.Element => {
  const { t } = useTranslation();

  return (
    <div className="relative mt-2">
      <pre className="rounded-md bg-code p-4 pr-10 font-mono text-sm break-all whitespace-pre-wrap">
        {code}
      </pre>
      <Tooltip label={t('common.actions.copy')}>
        <ActionIcon
          variant="subtle"
          size="xs"
          className="!absolute top-2 right-2"
          aria-label={t('common.actions.copy')}
          onClick={() => onCopy(code)}
        >
          <Copy size="1em" className="inline-block shrink-0 align-middle" />
        </ActionIcon>
      </Tooltip>
    </div>
  );
};
