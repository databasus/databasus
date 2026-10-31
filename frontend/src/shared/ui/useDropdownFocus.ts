import { useEffect, useRef } from 'react';

export const useDropdownFocus = (isOpen: boolean) => {
  const popupRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    let frame: number;
    const focusItem = () => {
      const popup = popupRef.current;
      const item =
        popup?.querySelector<HTMLElement>('[role="menuitem"].ant-dropdown-menu-item-selected') ??
        popup?.querySelector<HTMLElement>('[role="menuitem"]');
      item?.focus();
      if (!item || document.activeElement !== item) frame = requestAnimationFrame(focusItem);
    };
    frame = requestAnimationFrame(focusItem);
    return () => cancelAnimationFrame(frame);
  }, [isOpen]);

  return popupRef;
};
