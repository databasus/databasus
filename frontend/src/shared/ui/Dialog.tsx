import { Modal, type ModalProps } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { closeOverlayOnEscape } from '@/shared/ui/closeOverlayOnEscape';
import { OverlayFocusReturn } from '@/shared/ui/OverlayFocusReturn';

export const Dialog = (props: ModalProps) => {
  const { t } = useTranslation();

  return (
    <Modal
      {...props}
      closeOnEscape={false}
      onKeyDown={(event) => {
        props.onKeyDown?.(event);
        closeOverlayOnEscape(event, {
          isOpen: props.opened,
          canClose: props.closeOnEscape,
          onClose: props.onClose,
        });
      }}
      closeButtonProps={{ 'aria-label': t('common.actions.close'), ...props.closeButtonProps }}
    >
      {props.opened && <OverlayFocusReturn />}
      {props.children}
    </Modal>
  );
};
