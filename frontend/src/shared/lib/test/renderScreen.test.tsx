// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { App, DatePicker, theme } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

import { DICTIONARIES, useLocale } from '../../i18n';
import { LOCALE_SETTINGS } from '../../i18n/localeSettings';
import { useTheme } from '../../theme';
import { renderScreen } from './renderScreen';

const ContextScreen = () => {
  const { t } = useTranslation();
  const { locale, setLocale } = useLocale();
  const { resolvedTheme } = useTheme();
  const location = useLocation();
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const [draft, setDraft] = useState('');
  return (
    <>
      <output>{`${locale}:${resolvedTheme}:${location.pathname}${location.search}`}</output>
      <label>
        {t('common.actions.save')}
        <input value={draft} onChange={(event) => setDraft(event.target.value)} />
      </label>
      <output>{`${token.colorPrimary}:${token.colorBgContainer}`}</output>
      <DatePicker />
      <button onClick={() => void message.info(t('common.actions.save'))}>Notify</button>
      <button onClick={() => setLocale('es')}>Switch language</button>
    </>
  );
};

const originalState = {
  language: document.documentElement.lang,
  title: document.title,
  classes: document.documentElement.className,
  dayjsLocale: dayjs.locale(),
};

describe('renderScreen', () => {
  it.each(['light', 'dark'] as const)(
    'provides the real locale, %s theme and router entry',
    (theme) => {
      renderScreen(<ContextScreen />, { locale: 'fr', theme, route: '/settings?tab=profile' });

      expect(screen.getByText(`fr:${theme}:/settings?tab=profile`)).toBeTruthy();
      expect(document.documentElement.classList.contains('dark')).toBe(theme === 'dark');
      expect(document.documentElement.lang).toBe('fr');
      expect(
        screen.getByPlaceholderText(LOCALE_SETTINGS.fr.antdLocale.DatePicker!.lang.placeholder),
      ).toBeTruthy();
      expect(
        screen.getByText(theme === 'dark' ? '#1552d9:#141414' : '#155dfc:#ffffff'),
      ).toBeTruthy();
      expect(
        screen.getByRole('textbox', { name: DICTIONARIES.fr.common.actions.save }).closest('label')
          ?.textContent,
      ).toBe(DICTIONARIES.fr.common.actions.save);
    },
  );

  it('retains an unsaved draft when the language changes through a real provider', async () => {
    const { user } = renderScreen(<ContextScreen />, { locale: 'en', theme: 'light' });

    await user.type(
      screen.getByRole('textbox', { name: DICTIONARIES.en.common.actions.save }),
      'Unfinished settings',
    );
    await user.click(screen.getByRole('button', { name: 'Switch language' }));

    expect(screen.getByText('es:light:/')).toBeTruthy();
    expect(
      (
        screen.getByRole('textbox', {
          name: DICTIONARIES.es.common.actions.save,
        }) as HTMLInputElement
      ).value,
    ).toBe('Unfinished settings');
    expect(document.documentElement.lang).toBe('es');
    expect(localStorage.getItem('databasus-locale')).toBe('es');

    await user.click(screen.getByRole('button', { name: 'Notify' }));
    expect(
      screen.getByText(DICTIONARIES.es.common.actions.save, {
        selector: '.ant-message-notice-content span',
      }),
    ).toBeTruthy();
  });

  it('removes media subscriptions and component timers when unmounted', () => {
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const removeListener = vi.spyOn(media, 'removeEventListener');
    vi.spyOn(window, 'matchMedia').mockReturnValue(media);
    vi.useFakeTimers();

    const SubscriptionScreen = () => {
      useEffect(() => {
        const timer = setInterval(() => {}, 1_000);
        return () => clearInterval(timer);
      }, []);
      return <div />;
    };

    const view = renderScreen(<SubscriptionScreen />, { theme: 'dark', locale: 'fr' });
    expect(vi.getTimerCount()).toBeGreaterThan(0);
    view.unmount();
    expect(removeListener).toHaveBeenCalledWith('change', expect.any(Function));
    expect(vi.getTimerCount()).toBe(0);

    sessionStorage.setItem('test-only', 'discard');
    vi.stubGlobal('screenTestGlobal', 'discard');
  });

  it('starts with restored document, storage, globals and timers after another screen', () => {
    expect(localStorage.length).toBe(0);
    expect(sessionStorage.length).toBe(0);
    expect(document.documentElement.lang).toBe(originalState.language);
    expect(document.documentElement.className).toBe(originalState.classes);
    expect(document.title).toBe(originalState.title);
    expect(dayjs.locale()).toBe(originalState.dayjsLocale);
    expect('screenTestGlobal' in globalThis).toBe(false);
    expect(vi.isFakeTimers()).toBe(false);
    expect(document.body.childElementCount).toBe(0);
  });
});
