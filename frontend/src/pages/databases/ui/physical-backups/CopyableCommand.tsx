import { CheckOutlined, CopyOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { CommandToCopy } from './useCopiedCommand';

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
      <div className="mb-1 flex items-center justify-between">
        <div className="text-xs font-medium text-muted">{title}</div>
        <Button
          size="small"
          type="text"
          icon={isCopied ? <CheckOutlined /> : <CopyOutlined />}
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
