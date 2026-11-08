import { useCallback, useContext, useEffect, useId, useRef } from 'react';

import { getSessionSnapshot, isCurrentSession } from '@/shared/auth';
import { FeedbackContext } from '@/shared/lib/feedback/feedbackContext';
import type { FeedbackNotice } from '@/shared/lib/feedback/FeedbackNotice';

interface Options {
  scope?: 'screen' | 'session';
  contextKey?: string;
}

export const useFeedback = ({ scope = 'session', contextKey }: Options = {}) => {
  const feedback = useContext(FeedbackContext);
  if (!feedback) throw new Error('FeedbackProvider is required');

  const owner = useId();
  const session = useRef(getSessionSnapshot());
  const isMounted = useRef(false);

  useEffect(() => {
    isMounted.current = true;

    return () => {
      isMounted.current = false;
      if (scope === 'screen') feedback.dismissOwner(owner);
    };
  }, [feedback, owner, scope, contextKey]);

  return useCallback(
    (notice: FeedbackNotice) => {
      if (!isMounted.current) return;
      if (!notice.isSessionTransition && !isCurrentSession(session.current)) return;

      feedback.show(notice, scope === 'screen' ? owner : undefined);
    },
    [feedback, owner, scope],
  );
};
