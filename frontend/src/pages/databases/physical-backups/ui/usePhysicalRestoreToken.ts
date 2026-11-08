import { useEffect, useEffectEvent, useState } from 'react';

import {
  type PhysicalRestoreTokenCreate,
  physicalBackupsApi,
} from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { useSessionMutation } from '@/shared/api';
import { useRequestRevision } from '@/shared/hooks';

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
  const createPhysicalRestoreTokenMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof physicalBackupsApi.createPhysicalRestoreToken>) =>
      physicalBackupsApi.createPhysicalRestoreToken(...parameters),
  });

  const [answer, setAnswer] = useState<TokenAnswer>();

  const requestRevision = useRequestRevision();
  const subjectKey = JSON.stringify(subject);

  const createToken = useEffectEvent(() =>
    createPhysicalRestoreTokenMutation.mutateAsync([subject]),
  );

  useEffect(() => {
    const revision = requestRevision.Invalidate();

    setAnswer(undefined);

    createToken()
      .then((createdToken) => {
        if (requestRevision.IsCurrent(revision)) {
          setAnswer({ subjectKey, token: createdToken.token });
        }
      })
      .catch((error: unknown) => {
        if (requestRevision.IsCurrent(revision)) setAnswer({ subjectKey, error });
      });

    return () => {
      requestRevision.Invalidate();
    };
  }, [subjectKey, requestRevision]);

  const currentAnswer = answer?.subjectKey === subjectKey ? answer : undefined;

  return {
    token: currentAnswer?.token,
    error: currentAnswer?.error,
    isPending: currentAnswer === undefined,
  };
};
