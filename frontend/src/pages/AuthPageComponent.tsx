import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type PendingSignIn, userApi } from '../entity/users';
import {
  AuthNavbarComponent,
  RequestResetPasswordComponent,
  ResetPasswordComponent,
  SignInCodeComponent,
  SignInComponent,
  SignUpComponent,
} from '../features/users';
import { useScreenHeight } from '../shared/hooks';
import { translateApiError } from '../shared/i18n';

export function AuthPageComponent() {
  const { t } = useTranslation();
  const [hasAnyUser, setHasAnyUser] = useState(true);
  const [authMode, setAuthMode] = useState<
    'signIn' | 'signUp' | 'requestReset' | 'resetPassword' | 'signInCode'
  >('signIn');
  const [resetEmail, setResetEmail] = useState('');
  const [pendingSignIn, setPendingSignIn] = useState<PendingSignIn | undefined>(undefined);
  const [isLoading, setLoading] = useState(true);
  const screenHeight = useScreenHeight();

  const returnToPasswordStep = () => {
    setPendingSignIn(undefined);
    setAuthMode('signIn');
  };

  useEffect(() => {
    setLoading(true);

    userApi
      .isAnyUserExists()
      .then((isAnyUserFound) => {
        setHasAnyUser(isAnyUserFound);
        setAuthMode(isAnyUserFound ? 'signIn' : 'signUp');
        setLoading(false);
      })
      .catch((e) => {
        alert(t('app.auth.accountsCheckFailed', { error: translateApiError(e, t) }));
        setLoading(false);
      });
  }, []);

  return (
    <div className="flex min-h-full flex-col dark:bg-gray-900" style={{ minHeight: screenHeight }}>
      {isLoading ? (
        <div className="flex h-screen w-screen items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div>
          <div>
            <AuthNavbarComponent />

            <div className="mt-10 flex justify-center sm:mt-[10vh]">
              {authMode === 'signUp' ? (
                <SignUpComponent
                  onSwitchToSignIn={hasAnyUser ? () => setAuthMode('signIn') : undefined}
                  isClaimingInstance={!hasAnyUser}
                />
              ) : authMode === 'signIn' ? (
                <SignInComponent
                  onSwitchToSignUp={() => setAuthMode('signUp')}
                  onSwitchToResetPassword={() => setAuthMode('requestReset')}
                  onCodeRequired={(newPendingSignIn) => {
                    setPendingSignIn(newPendingSignIn);
                    setAuthMode('signInCode');
                  }}
                />
              ) : authMode === 'signInCode' && pendingSignIn ? (
                <SignInCodeComponent
                  pendingSignIn={pendingSignIn}
                  onPendingSignInReplaced={setPendingSignIn}
                  onPendingSignInLost={returnToPasswordStep}
                  onSwitchToSignIn={returnToPasswordStep}
                />
              ) : authMode === 'requestReset' ? (
                <RequestResetPasswordComponent
                  onSwitchToSignIn={() => setAuthMode('signIn')}
                  onSwitchToResetPassword={(email) => {
                    setResetEmail(email);
                    setAuthMode('resetPassword');
                  }}
                />
              ) : (
                <ResetPasswordComponent
                  onSwitchToSignIn={() => setAuthMode('signIn')}
                  onSwitchToRequestCode={() => setAuthMode('requestReset')}
                  initialEmail={resetEmail}
                />
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
