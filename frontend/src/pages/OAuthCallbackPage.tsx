import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router';

import { getOAuthRedirectUri } from '../constants';
import { userApi } from '../entities/users';
import { type TranslationKey, translateApiError } from '../shared/i18n';

type CallbackFailure = { detectedFailureKey: TranslationKey } | { signInError: unknown };

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [failure, setFailure] = useState<CallbackFailure>();

  useEffect(() => {
    const handleOAuthCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');

      if (!code) {
        setFailure({ detectedFailureKey: 'app.oauthCallback.codeNotFound' });
        return;
      }

      if (!state) {
        setFailure({ detectedFailureKey: 'app.oauthCallback.stateMissing' });
        return;
      }

      const redirectUri = getOAuthRedirectUri();

      try {
        if (state === 'github') {
          await userApi.handleGitHubOAuth({ code, redirectUri });
        } else if (state === 'google') {
          await userApi.handleGoogleOAuth({ code, redirectUri });
        } else {
          setFailure({ detectedFailureKey: 'app.oauthCallback.invalidProvider' });
          return;
        }

        navigate('/');
      } catch (e) {
        setFailure({ signInError: e });
      }
    };

    handleOAuthCallback();
  }, [searchParams, navigate]);

  return (
    <div className="flex h-screen w-screen flex-col items-center justify-center">
      {failure ? (
        <div>
          <div className="mb-4 text-center text-xl font-semibold text-red-600">
            {t('app.oauthCallback.failedTitle')}
          </div>
          <div className="text-center text-sm text-gray-600">
            {'detectedFailureKey' in failure
              ? t(failure.detectedFailureKey)
              : translateApiError(failure.signInError, t)}
          </div>
          <div className="mt-6 text-center">
            <button
              type="button"
              onClick={() => navigate('/')}
              className="cursor-pointer font-medium text-blue-600 hover:text-blue-700"
            >
              {t('app.oauthCallback.returnToSignIn')}
            </button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col items-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
          <div className="mt-4 text-gray-600">{t('app.oauthCallback.completing')}</div>
        </div>
      )}
    </div>
  );
}
