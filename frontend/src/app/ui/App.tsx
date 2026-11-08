import { BrowserRouter, Navigate, Route } from 'react-router';
import { Routes } from 'react-router';

import { useVersionCheck } from '@/app/model/useVersionCheck';
import { ApplicationScreen } from '@/app/ui/ApplicationScreen';
import { RequestConfiguration } from '@/app/ui/RequestConfiguration';
import { AuthPage } from '@/pages/auth';
import { AuthCallbackPage } from '@/pages/auth-callback';
import { StorageOAuthPage } from '@/pages/storage-oauth';
import { useSession } from '@/shared/auth';
import { LocaleProvider, ControlLocalization } from '@/shared/i18n';
import { FeedbackProvider } from '@/shared/lib/feedback';
import { ThemeProvider, ThemeConfiguration } from '@/shared/theme';

function AppContent() {
  const session = useSession();
  const isAuthorized = !!session.token;

  useVersionCheck();

  return (
    <ControlLocalization>
      <BrowserRouter>
        <Routes>
          <Route path="/auth/callback" element={<AuthCallbackPage />} />
          <Route path="/storages/google-oauth" element={<StorageOAuthPage />} />
          <Route
            path="/"
            element={
              !isAuthorized ? (
                <AuthPage key={session.generation} />
              ) : (
                <ApplicationScreen key={session.generation} />
              )
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ControlLocalization>
  );
}

function App() {
  return (
    <ThemeProvider>
      <LocaleProvider>
        <ThemeConfiguration isTestEnvironment={import.meta.env.MODE === 'test'}>
          <FeedbackProvider>
            <RequestConfiguration>
              <AppContent />
            </RequestConfiguration>
          </FeedbackProvider>
        </ThemeConfiguration>
      </LocaleProvider>
    </ThemeProvider>
  );
}

export default App;
