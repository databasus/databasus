import { useLayoutEffect, useState } from 'react';
import { BrowserRouter, Navigate, Route, Routes, useNavigate, useLocation } from 'react-router';

import { useVersionCheck } from '@/app/model/useVersionCheck';
import { ApplicationScreen } from '@/app/ui/ApplicationScreen';
import { RequestConfiguration } from '@/app/ui/RequestConfiguration';
import { AuthPage } from '@/pages/auth';
import { AuthCallbackPage } from '@/pages/auth-callback';
import { useSession } from '@/shared/auth';
import { LocaleProvider, ControlLocalization } from '@/shared/i18n';
import { FeedbackProvider } from '@/shared/lib/feedback';
import { parseApplicationPath, useNavigationIntent } from '@/shared/navigation';
import { ThemeProvider, ThemeConfiguration } from '@/shared/theme';

function ApplicationEntry() {
  const session = useSession();
  const navigationIntent = useNavigationIntent();
  const location = useLocation();
  const navigate = useNavigate();
  const [entrySession, setEntrySession] = useState(session);
  const shouldResetNavigation =
    !!entrySession.token && entrySession.generation !== session.generation;
  useLayoutEffect(() => {
    if (shouldResetNavigation && location.pathname !== '/') void navigate('/', { replace: true });
    else setEntrySession(session);
  }, [shouldResetNavigation, session, location.pathname, navigate]);

  if (shouldResetNavigation) return null;

  if (!parseApplicationPath(location.pathname)) return <Navigate to="/" replace />;

  if (location.search || location.hash) return <Navigate to={location.pathname} replace />;

  return session.token ? (
    <ApplicationScreen key={session.generation} />
  ) : (
    <AuthPage key={`${session.generation}:${navigationIntent}`} />
  );
}

function AppContent() {
  useVersionCheck();

  return (
    <ControlLocalization>
      <BrowserRouter>
        <Routes>
          <Route path="/auth/callback" element={<AuthCallbackPage />} />
          <Route path="*" element={<ApplicationEntry />} />
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
