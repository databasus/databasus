import { Alert, Button, Spin } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router';

import { OAuthCallback } from '@/features/oauth-sign-in';
import { ApiError } from '@/shared/api';
import { translateApiError } from '@/shared/i18n';

export function AuthCallbackPage() {
  const [failure, setFailure] = useState<unknown>();
  const [parameters] = useSearchParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const code = parameters.get('code');
  const state = parameters.get('state');
  const error = parameters.get('error');
  const callback = useMemo(() => new OAuthCallback({ code, state, error }), [code, state, error]);
  useEffect(() => {
    let isAttached = true;
    setFailure(undefined);

    const scrubbed = new URL(window.location.href);
    for (const name of ['code', 'state', 'error', 'error_description', 'error_uri'])
      scrubbed.searchParams.delete(name);
    window.history.replaceState(
      window.history.state,
      '',
      scrubbed.pathname + scrubbed.search + scrubbed.hash,
    );

    void callback
      .Start()
      .then((session) => {
        if (isAttached && callback.Accept(session)) navigate('/', { replace: true });
      })
      .catch((failure) => {
        if (isAttached && callback.IsCurrent()) setFailure(failure);
      });

    return () => {
      isAttached = false;
    };
  }, [callback, navigate]);
  return (
    <div className="flex min-h-dvh min-w-0 flex-col items-center justify-center bg-page p-4 text-content">
      {failure !== undefined ? (
        <div className="max-w-md">
          <Alert
            type="error"
            showIcon
            message={t('app.oauthCallback.failedTitle')}
            description={
              failure instanceof ApiError
                ? translateApiError(failure, t)
                : t('app.oauthCallback.notCompleted')
            }
          />
          <Button className="mt-4" onClick={() => navigate('/', { replace: true })}>
            {t('app.oauthCallback.returnToSignIn')}
          </Button>
        </div>
      ) : (
        <>
          <Spin />
          <div className="mt-4">{t('app.oauthCallback.completing')}</div>
        </>
      )}
    </div>
  );
}
