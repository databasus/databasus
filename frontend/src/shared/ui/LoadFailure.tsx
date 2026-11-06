import { Alert, Button } from 'antd';
import { useTranslation } from 'react-i18next';

import { translateApiError } from '@/shared/i18n';

interface Props {
  error: unknown;
  onRetry: () => void;
  className?: string;
}

export const LoadFailure = ({ error, onRetry, className }: Props) => {
  const { t } = useTranslation();

  return (
    <Alert
      className={className}
      type="error"
      showIcon
      message={translateApiError(error, t)}
      action={
        <Button className="!h-auto min-h-8 max-w-full !whitespace-normal" onClick={onRetry}>
          {t('common.actions.retry')}
        </Button>
      }
    />
  );
};
