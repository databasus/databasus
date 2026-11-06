import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import { useEffect, useEffectEvent, useState } from 'react';

import { type PendingSignIn, userApi } from '@/entities/users';
import { AuthNavbarComponent } from '@/pages/auth/ui/AuthNavbarComponent';
import { RequestResetPasswordComponent } from '@/pages/auth/ui/RequestResetPasswordComponent';
import { ResetPasswordComponent } from '@/pages/auth/ui/ResetPasswordComponent';
import { SignInCodeComponent } from '@/pages/auth/ui/SignInCodeComponent';
import { SignInComponent } from '@/pages/auth/ui/SignInComponent';
import { SignUpComponent } from '@/pages/auth/ui/SignUpComponent';
import { useSessionGuard } from '@/shared/auth';
import { useRequestRevision, useScreenHeight } from '@/shared/hooks';
import { LoadFailure } from '@/shared/ui';

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
    const requestRevision = revision.Invalidate();
    setLoading(true);
    setFailure(undefined);
    try {
      const hasUsers = await userApi.isAnyUserExists();
      if (!isCurrent() || !revision.IsCurrent(requestRevision)) return;
      setHasAnyUser(hasUsers);
      setAuthMode(hasUsers ? 'signIn' : 'signUp');
    } catch (error) {
      if (isCurrent() && revision.IsCurrent(requestRevision)) setFailure(error);
    } finally {
      if (isCurrent() && revision.IsCurrent(requestRevision)) setLoading(false);
    }
  };
  const screenHeight = useScreenHeight();
  const revision = useRequestRevision();
  const isCurrent = useSessionGuard();
  const startOptionsLoad = useEffectEvent(() => {
    void loadOptions();
  });

  useEffect(() => {
    startOptionsLoad();
    return () => {
      revision.Invalidate();
    };
  }, [revision]);

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
        <LoadFailure className="m-4" error={failure} onRetry={() => void loadOptions()} />
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
