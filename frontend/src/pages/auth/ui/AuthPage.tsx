import { LoadingOutlined } from '@ant-design/icons';
import { Alert, Button, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type PendingSignIn, userApi } from '@/entities/users';
import { useSessionGuard } from '@/shared/auth';
import { useScreenHeight } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';

import { AuthNavbarComponent } from './AuthNavbarComponent';
import { RequestResetPasswordComponent } from './RequestResetPasswordComponent';
import { ResetPasswordComponent } from './ResetPasswordComponent';
import { SignInCodeComponent } from './SignInCodeComponent';
import { SignInComponent } from './SignInComponent';
import { SignUpComponent } from './SignUpComponent';

export function AuthPage() {
  const [hasAnyUser, setHasAnyUser] = useState(true);
  const [authMode, setAuthMode] = useState<
    'signIn' | 'signUp' | 'requestReset' | 'resetPassword' | 'signInCode'
  >('signIn');
  const [resetEmail, setResetEmail] = useState('');
  const [pendingSignIn, setPendingSignIn] = useState<PendingSignIn | undefined>(undefined);
  const [isLoading, setLoading] = useState(true);
  const [failure, setFailure] = useState<unknown>();

  const returnToPasswordStep = () => {
    setPendingSignIn(undefined);
    setAuthMode('signIn');
  };

  const loadOptions = async () => {
    const requestRevision = ++revision.current;
    setLoading(true);
    setFailure(undefined);
    try {
      const hasUsers = await userApi.isAnyUserExists();
      if (!isCurrent() || revision.current !== requestRevision) return;
      setHasAnyUser(hasUsers);
      setAuthMode(hasUsers ? 'signIn' : 'signUp');
    } catch (error) {
      if (isCurrent() && revision.current === requestRevision) setFailure(error);
    } finally {
      if (isCurrent() && revision.current === requestRevision) setLoading(false);
    }
  };
  const { t } = useTranslation();
  const screenHeight = useScreenHeight();
  const revision = useRef(0);
  const isCurrent = useSessionGuard();
  useEffect(() => {
    void loadOptions();
    return () => {
      revision.current++;
    };
  }, []);

  return (
    <div
      className="flex min-h-0 min-w-0 flex-col overflow-y-auto bg-page text-content"
      style={{ height: screenHeight }}
    >
      {isLoading ? (
        <div className="flex h-screen w-screen items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : failure !== undefined ? (
        <Alert
          className="m-4"
          type="error"
          message={translateApiError(failure, t)}
          action={<Button onClick={() => void loadOptions()}>{t('common.actions.retry')}</Button>}
        />
      ) : (
        <div>
          <div>
            <AuthNavbarComponent />

            <div className="mt-10 flex justify-center px-4 pb-8 sm:mt-[10vh]">
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
