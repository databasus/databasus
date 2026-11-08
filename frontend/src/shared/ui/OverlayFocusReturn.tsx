import { useEffect, useRef, useState } from 'react';

export const OverlayFocusReturn = () => {
  const [invokingControl] = useState(() =>
    document.activeElement instanceof HTMLElement ? document.activeElement : null,
  );
  const isMounted = useRef(false);

  useEffect(() => {
    isMounted.current = true;

    return () => {
      isMounted.current = false;
      queueMicrotask(() => {
        if (!isMounted.current && invokingControl?.isConnected) invokingControl.focus();
      });
    };
  }, [invokingControl]);

  return null;
};
