import { App as AntdApp, ConfigProvider, Modal } from 'antd';
import { useEffect } from 'react';
import { BrowserRouter, Navigate, Route } from 'react-router';
import { Routes } from 'react-router';

import { useVersionCheck } from '@/app/model/useVersionCheck';
import { ApplicationScreen } from '@/app/ui/ApplicationScreen';
import { AuthPage } from '@/pages/auth';
import { AuthCallbackPage } from '@/pages/auth-callback';
import { StorageOAuthPage } from '@/pages/storage-oauth';
import { useSession } from '@/shared/auth';

import { LocaleProvider, useLocale } from '../../shared/i18n';
import { ThemeProvider, useTheme, getAntdTheme, ThemeCssVariables } from '../../shared/theme';

function AppContent() {
  const session = useSession();
  const isAuthorized = !!session.token;
  const { resolvedTheme } = useTheme();
  const { antdLocale } = useLocale();

  useVersionCheck();
  useEffect(() => {
    Modal.destroyAll();
  }, [session.generation]);

  return (
    <ConfigProvider
      button={{ autoInsertSpace: false }}
      locale={antdLocale}
      theme={getAntdTheme(resolvedTheme)}
    >
      <ThemeCssVariables />
      <AntdApp>
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
      </AntdApp>
    </ConfigProvider>
  );
}

function App() {
  return (
    <ThemeProvider>
      <LocaleProvider>
        <AppContent />
      </LocaleProvider>
    </ThemeProvider>
  );
}

export default App;
