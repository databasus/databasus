import { useEffect, useRef } from 'react';

import {
  abandonAuthIntent,
  acceptAuthSession,
  beginAuthIntent,
  isCurrentAuthIntent,
  type AuthIntent,
} from './session';

export const useAuthOperation = () => {
  const start = () => {
    const currentIntent = beginAuthIntent();
    intent.current = currentIntent;
    return currentIntent;
  };
  const isCurrent = (operation: AuthIntent) =>
    isMounted.current && intent.current === operation && isCurrentAuthIntent(operation);
  const accept = ({
    operation,
    session,
  }: {
    operation: AuthIntent;
    session: { token: string; userId: string };
  }) => isCurrent(operation) && acceptAuthSession({ intent: operation, session });

  const intent = useRef<AuthIntent | undefined>(undefined);
  const isMounted = useRef(false);
  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
      if (intent.current) abandonAuthIntent(intent.current);
    };
  }, []);

  return { start, isCurrent, accept };
};
