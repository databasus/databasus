import { CopyOutlined } from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
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
      <Tooltip title={t('common.actions.copy')}>
        <Button
          type="text"
          size="small"
          className="!absolute top-2 right-2"
          aria-label={t('common.actions.copy')}
          icon={<CopyOutlined />}
          onClick={() => onCopy(code)}
        />
      </Tooltip>
    </div>
  );
};
