import type { DateInputProps } from '@mantine/dates';
import { useRef, useState, type FocusEvent, type KeyboardEvent } from 'react';

export const useBackupDatePopover = () => {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const close = () => setIsOpen(false);
  const open = () => setIsOpen(true);
  const closeAndReturnFocus = () => {
    inputRef.current?.focus();
    close();
  };
  const dismissWithKeyboard = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key !== 'Escape' || event.nativeEvent.isComposing) return;

    event.preventDefault();
    event.stopPropagation();
    closeAndReturnFocus();
  };
  const handleBlur = (event: FocusEvent<HTMLInputElement>) => {
    // DateInput closes on blur even when focus moves into its portalled calendar.
    if (
      event.relatedTarget instanceof Element &&
      // eslint-disable-next-line i18next/no-literal-string -- Mantine calendar DOM marker
      event.relatedTarget.closest('[data-dates-dropdown]')
    )
      return;

    close();
  };
  const popoverProps: DateInputProps['popoverProps'] = {
    opened: isOpen,
    onChange: setIsOpen,
    closeOnEscape: false,
    returnFocus: false,
    withRoles: true,
  };

  return {
    closeAndReturnFocus,
    popoverProps,
    onKeyDownCapture: dismissWithKeyboard,
    inputProps: {
      ref: inputRef,
      onClick: open,
      onFocus: open,
      onBlur: handleBlur,
    },
  };
};
