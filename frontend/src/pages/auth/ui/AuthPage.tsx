import { Loader } from '@mantine/core';
import { useEffect, useState } from 'react';

import { type PendingSignIn, userApi } from '@/entities/users';
import { AuthNavbarComponent } from '@/pages/auth/ui/AuthNavbarComponent';
import { RequestResetPasswordComponent } from '@/pages/auth/ui/RequestResetPasswordComponent';
import { ResetPasswordComponent } from '@/pages/auth/ui/ResetPasswordComponent';
import { SignInCodeComponent } from '@/pages/auth/ui/SignInCodeComponent';
import { SignInComponent } from '@/pages/auth/ui/SignInComponent';
import { SignUpComponent } from '@/pages/auth/ui/SignUpComponent';
import { useSessionQuery } from '@/shared/api';
import { beginAuthIntent, abandonAuthIntent } from '@/shared/auth';
import { useScreenHeight } from '@/shared/hooks';
import { LoadFailure } from '@/shared/ui';

export function AuthPage() {
  useEffect(() => {
    const intent = beginAuthIntent();
    return () => abandonAuthIntent(intent);
  }, []);
  const query = useSessionQuery({
    queryKey: ['registration-options'],
    queryFn: (signal) => userApi.isAnyUserExists({ signal }),
  });
  const hasAnyUser = query.data ?? true;
  const [authMode, setAuthMode] = useState<
    'signIn' | 'signUp' | 'requestReset' | 'resetPassword' | 'signInCode'
  >('signIn');
  const [resetEmail, setResetEmail] = useState('');
  const [pendingSignIn, setPendingSignIn] = useState<PendingSignIn | undefined>(undefined);
  const isLoading = query.isPending;
  const failure = query.error ?? undefined;

  const returnToPasswordStep = () => {
    setPendingSignIn(undefined);
    setAuthMode('signIn');
  };

  const loadOptions = () => query.refetch();
  const screenHeight = useScreenHeight();
  useEffect(() => {
    if (query.data !== undefined) setAuthMode(query.data ? 'signIn' : 'signUp');
  }, [query.data]);

  return (
    <div
      className="flex min-h-0 min-w-0 flex-col overflow-y-auto bg-page text-content"
      style={{ height: screenHeight }}
    >
      {isLoading ? (
        <div className="flex h-screen w-screen items-center justify-center">
          <Loader size="md" role="status" />
        </div>
      ) : failure !== undefined ? (
        <LoadFailure className="m-4" error={failure} onRetry={() => void loadOptions()} />
      ) : (
        <div>
          <div>
            <AuthNavbarComponent />

            <div className="mt-8 flex justify-center px-4 pb-8 sm:mt-[6vh]">
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
