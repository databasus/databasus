import { Button } from '@mantine/core';
import { type JSX, type ReactNode, useEffect, useState } from 'react';
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
  // Keeps the action locked while the person reads why it matters, instead of clicking through.
  actionDelaySeconds?: number;
}

export function ConfirmationComponent({
  onConfirm,
  onDecline,
  description,
  isDestructive,
  actionText,
  cancelText,
  hideCancelButton = false,
  actionDelaySeconds = 0,
}: Props): JSX.Element {
  const { t } = useTranslation();
  const [actionSecondsLeft, setActionSecondsLeft] = useState(actionDelaySeconds);

  const isActionLocked = actionSecondsLeft > 0;

  useEffect(() => {
    if (!isActionLocked) return;

    const interval = setInterval(
      () => setActionSecondsLeft((secondsLeft) => Math.max(secondsLeft - 1, 0)),
      1000,
    );

    return () => clearInterval(interval);
  }, [isActionLocked]);

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
          disabled={isActionLocked}
        >
          {isActionLocked
            ? t('common.confirmation.actionCountdown', {
                action: actionText,
                seconds: actionSecondsLeft,
              })
            : actionText}
        </Button>
      </div>
    </Dialog>
  );
}
