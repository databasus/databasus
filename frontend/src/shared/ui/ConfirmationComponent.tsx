import { Button } from '@mantine/core';
import { type JSX, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { Dialog } from '@/shared/ui/Dialog';

interface Props {
  onConfirm(): void;
  onDecline(): void;

  // Rendered as React content, never as HTML: a sentence with inline markup is a Trans element.
  description: ReactNode;
  isDestructive: boolean;

  actionText: string;
  cancelText?: string;
  hideCancelButton?: boolean;
}

export function ConfirmationComponent({
  onConfirm,
  onDecline,
  description,
  isDestructive,
  actionText,
  cancelText,
  hideCancelButton = false,
}: Props): JSX.Element {
  const { t } = useTranslation();

  return (
    <Dialog
      title={t('common.confirmation.title')}
      opened
      onClose={onDecline}
      closeButtonProps={{ 'aria-label': t('common.actions.close') }}
    >
      <div>{description}</div>

      <div className="mt-5 flex flex-wrap justify-end gap-2">
        {!hideCancelButton && (
          <Button
            className="h-auto min-h-8 max-w-full whitespace-normal"
            onClick={() => onDecline()}
          >
            {cancelText || t('common.actions.cancel')}
          </Button>
        )}

        <Button
          className="h-auto min-h-8 max-w-full whitespace-normal"
          onClick={() => onConfirm()}
          color={isDestructive ? 'danger' : 'accent'}
          variant="filled"
        >
          {actionText}
        </Button>
      </div>
    </Dialog>
  );
}
