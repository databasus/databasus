import type { KeyboardEvent } from 'react';

interface OverlayEscapeOptions {
  isOpen: boolean;
  canClose?: boolean;
  onClose: () => void;
}

export const closeOverlayOnEscape = (
  event: KeyboardEvent<HTMLElement>,
  { isOpen, canClose = true, onClose }: OverlayEscapeOptions,
) => {
  if (event.key !== 'Escape' || event.nativeEvent.isComposing || !isOpen) return;

  event.stopPropagation();
  if (event.defaultPrevented) return;

  // Combobox closes its popup before bubbling; its trigger still exposes the open state.
  if (
    event.target instanceof Element &&
    // eslint-disable-next-line i18next/no-literal-string -- CSS selector for the ARIA popup contract
    event.target.closest('[aria-haspopup][aria-expanded="true"]')
  )
    return;

  event.preventDefault();
  if (canClose) onClose();
};
