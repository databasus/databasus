import { useEffect, useRef, useState } from 'react';

import { RequestRevision } from '../../../../shared/lib/RequestRevision';
import { type PhysicalRestoreTokenCreate, physicalBackupsApi } from '../../api/physicalBackupsApi';

interface TokenAnswer {
  subjectKey: string;
  token?: string;
  error?: unknown;
}

// A token is requested automatically for each subject: once for a backup, and again whenever the
// target time changes. The token is single-use and the stream is unauthenticated, so it must be a
// fresh capability the user can curl, not a static command. An answer counts only for the subject
// it was requested for, so a late one cannot put another target time's token under the shown time.
export const usePhysicalRestoreToken = (subject: PhysicalRestoreTokenCreate) => {
  const [answer, setAnswer] = useState<TokenAnswer>();

  const requestRevision = useRef(new RequestRevision());
  const subjectKey = JSON.stringify(subject);

  useEffect(() => {
    const revision = requestRevision.current.Invalidate();

    setAnswer(undefined);

    physicalBackupsApi
      .createPhysicalRestoreToken(subject)
      .then((createdToken) => {
        if (requestRevision.current.IsCurrent(revision)) {
          setAnswer({ subjectKey, token: createdToken.token });
        }
      })
      .catch((error: unknown) => {
        if (requestRevision.current.IsCurrent(revision)) setAnswer({ subjectKey, error });
      });

    return () => {
      requestRevision.current.Invalidate();
    };
  }, [subjectKey]);

  const currentAnswer = answer?.subjectKey === subjectKey ? answer : undefined;

  return {
    token: currentAnswer?.token,
    error: currentAnswer?.error,
    isPending: currentAnswer === undefined,
  };
};
