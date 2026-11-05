import { Button, Modal } from 'antd';
import { useEffect, useRef, useState, type JSX, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

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
  const [invokingControl] = useState(() =>
    document.activeElement instanceof HTMLElement ? document.activeElement : null,
  );
  const isMounted = useRef(false);
  const { t } = useTranslation();

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
      queueMicrotask(() => {
        if (!isMounted.current && invokingControl?.isConnected) invokingControl.focus();
      });
    };
  }, [invokingControl]);

  return (
    <Modal
      title={t('common.confirmation.title')}
      open
      onCancel={() => onDecline()}
      footer={<div />}
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
          danger={isDestructive}
          type="primary"
        >
          {actionText}
        </Button>
      </div>
    </Modal>
  );
}
