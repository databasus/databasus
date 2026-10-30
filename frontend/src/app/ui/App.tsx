import { App as AntdApp, ConfigProvider, theme } from 'antd';
import { useEffect, useState } from 'react';
import { BrowserRouter, Navigate, Route } from 'react-router';
import { Routes } from 'react-router';

import { userApi } from '../../entity/users';
import { AuthPageComponent } from '../../pages/AuthPageComponent';
import { OAuthCallbackPage } from '../../pages/OAuthCallbackPage';
import { OauthStorageComponent } from '../../pages/OauthStorageComponent';
import { useVersionCheck } from '../../shared/hooks/useVersionCheck';
import { LocaleProvider, useLocale } from '../../shared/i18n';
import { ThemeProvider, useTheme } from '../../shared/theme';
import { MainScreenComponent } from '../../widgets/main/MainScreenComponent';

function AppContent() {
  const [isAuthorized, setIsAuthorized] = useState(false);
  const { resolvedTheme } = useTheme();
  const { antdLocale } = useLocale();

  useVersionCheck();

  useEffect(() => {
    const isAuthorized = userApi.isAuthorized();
    setIsAuthorized(isAuthorized);

    const onAuthChanged = () => setIsAuthorized(userApi.isAuthorized());
    userApi.addAuthListener(onAuthChanged);

    return () => userApi.removeAuthListener(onAuthChanged);
  }, []);

  return (
    <ConfigProvider
      locale={antdLocale}
      theme={{
        algorithm: resolvedTheme === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: '#155dfc', // Tailwind blue-600
        },
      }}
    >
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route path="/auth/callback" element={<OAuthCallbackPage />} />
            <Route path="/storages/google-oauth" element={<OauthStorageComponent />} />
            <Route
              path="/"
              element={!isAuthorized ? <AuthPageComponent /> : <MainScreenComponent />}
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
