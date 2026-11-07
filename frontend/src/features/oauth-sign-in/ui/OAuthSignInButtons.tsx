import { Alert, Button } from '@mantine/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  createOAuthSignInAttempt,
  type OAuthProvider,
} from '@/features/oauth-sign-in/model/OAuthSignInAttempt';
import { OAuthProviderIcon } from '@/features/oauth-sign-in/ui/OAuthProviderIcon';
import { GITHUB_CLIENT_ID, GOOGLE_CLIENT_ID } from '@/shared/config';

export function OAuthSignInButtons() {
  const [hasFailure, setHasFailure] = useState(false);
  const start = (provider: OAuthProvider) => {
    try {
      const attempt = createOAuthSignInAttempt(provider);
      const params = new URLSearchParams({
        client_id: provider === 'github' ? GITHUB_CLIENT_ID : GOOGLE_CLIENT_ID,
        redirect_uri: attempt.redirectUri,
        state: attempt.state,
        scope: provider === 'github' ? 'user:email' : 'openid email profile',
      });
      if (provider === 'google') params.set('response_type', 'code');
      window.location.assign(
        `${provider === 'github' ? 'https://github.com/login/oauth/authorize' : 'https://accounts.google.com/o/oauth2/v2/auth'}?${params}`,
      );
    } catch {
      setHasFailure(true);
    }
  };
  const { t } = useTranslation();
  return (
    <div className="space-y-2">
      {GITHUB_CLIENT_ID && (
        <Button
          className="!h-auto min-h-10 w-full max-w-full !whitespace-normal"
          size="md"
          leftSection={<OAuthProviderIcon provider="github" />}
          onClick={() => start('github')}
        >
          {t('users.oauth.continueWithGithub')}
        </Button>
      )}
      {GOOGLE_CLIENT_ID && (
        <Button
          className="!h-auto min-h-10 w-full max-w-full !whitespace-normal"
          size="md"
          leftSection={<OAuthProviderIcon provider="google" />}
          onClick={() => start('google')}
        >
          {t('users.oauth.continueWithGoogle')}
        </Button>
      )}
      {hasFailure && <Alert color="danger" title={t('users.oauth.invalidConfiguration')} />}
    </div>
  );
}
