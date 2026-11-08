import { Alert, Button } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
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
      color="danger"
      icon={<CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>{translateApiError(error, t)}</div>
        <Button className="!h-auto min-h-8 max-w-full !whitespace-normal" onClick={onRetry}>
          {t('common.actions.retry')}
        </Button>
      </div>
    </Alert>
  );
};
