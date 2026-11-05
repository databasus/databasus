import { useCallback, useEffect, useRef } from 'react';

import { getSessionSnapshot, isCurrentSession } from './session';

export const useSessionGuard = () => {
  const session = useRef(getSessionSnapshot());
  const isMounted = useRef(false);
  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
  }, []);
  return useCallback(() => isMounted.current && isCurrentSession(session.current), []);
};
