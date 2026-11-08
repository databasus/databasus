// @vitest-environment happy-dom
import { useMantineTheme } from '@mantine/core';
import { DatePickerInput } from '@mantine/dates';
import { useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { fireEvent, screen, within } from '@testing-library/react';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

import { useSessionMutation } from '@/shared/api';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { waitForCondition } from '@/shared/lib/test/conditionWait';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { useTheme } from '@/shared/theme';

const ContextScreen = () => {
  const { t } = useTranslation();
  const { locale, setLocale } = useLocale();
  const { resolvedTheme } = useTheme();
  const location = useLocation();
  const controlsTheme = useMantineTheme();
  const notify = useFeedback();
  const [draft, setDraft] = useState('');

  return (
    <>
      <div>{`${locale}:${resolvedTheme}:${location.pathname}${location.search}`}</div>
      <label>
        {t('common.actions.save')}
        <input value={draft} onChange={(event) => setDraft(event.target.value)} />
      </label>
      <div>{controlsTheme.fontFamily}</div>
      <DatePickerInput aria-label="Calendar" defaultDate="2026-10-18" />
      <button onClick={() => notify({ kind: 'info', title: t('common.actions.save') })}>
        Notify
      </button>
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
    async (theme) => {
      const { user } = renderScreen(<ContextScreen />, {
        locale: 'fr',
        theme,
        route: '/settings?tab=profile',
      });

      expect(screen.getByText(`fr:${theme}:/settings?tab=profile`)).toBeTruthy();
      expect(document.documentElement.classList.contains('dark')).toBe(theme === 'dark');
      expect(document.documentElement.lang).toBe('fr');
      expect(screen.getByText(/Jost, system-ui/)).toBeTruthy();
      expect(document.documentElement.style.getPropertyValue('--app-surface')).toBe(
        theme === 'dark' ? '#141414' : '#ffffff',
      );
      expect(
        screen.getByRole('textbox', { name: DICTIONARIES.fr.common.actions.save }).closest('label')
          ?.textContent,
      ).toBe(DICTIONARIES.fr.common.actions.save);
      await user.click(screen.getByRole('button', { name: 'Calendar' }));
      expect(await screen.findByRole('button', { name: 'octobre 2026' })).toBeTruthy();
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
      within(screen.getByRole('status')).getByText(DICTIONARIES.es.common.actions.save),
    ).toBeTruthy();
  });

  it('retains a notice language and deadline when the interface language changes', async () => {
    vi.useFakeTimers();
    renderScreen(<ContextScreen />, { locale: 'en' });

    fireEvent.click(screen.getByRole('button', { name: 'Notify' }));

    const notice = within(screen.getByRole('status'));
    expect(notice.getByText(DICTIONARIES.en.common.actions.save)).toBeTruthy();

    await advanceFakeClock(1000);

    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

    expect(notice.getByText(DICTIONARIES.en.common.actions.save)).toBeTruthy();

    await advanceFakeClock(1999);

    expect(screen.getByRole('status')).toBeTruthy();

    await advanceFakeClock(1);

    await waitForCondition(() => expect(screen.queryByRole('status')).toBeNull());
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

  it.each([false, true])(
    'isolates and disposes request clients with Strict Mode=%s',
    async (isStrictMode) => {
      const clients: QueryClient[] = [];
      const RequestContext = () => {
        const client = useQueryClient();
        useQuery({
          queryKey: ['renderer-state'],
          queryFn: async () => 'accepted response',
          enabled: false,
        });
        useEffect(() => {
          if (!clients.includes(client)) clients.push(client);
        }, [client]);
        const action = useSessionMutation({ mutationFn: async () => 'completed action' });

        return (
          <button
            onClick={() => {
              client.setQueryData(['renderer-state'], 'accepted response');
              void action.mutateAsync(undefined);
            }}
          >
            Run action
          </button>
        );
      };
      const first = renderScreen(<RequestContext />, { isStrictMode });

      await first.user.click(screen.getByRole('button', { name: 'Run action' }));

      const firstClient = clients[0];
      expect(firstClient.getQueryData(['renderer-state'])).toBe('accepted response');
      expect(firstClient.getMutationCache().getAll()).toHaveLength(1);

      first.unmount();

      expect(firstClient.getQueryCache().getAll()).toHaveLength(0);
      expect(firstClient.getMutationCache().getAll()).toHaveLength(0);

      const second = renderScreen(<RequestContext />, { isStrictMode });

      expect(clients).toHaveLength(2);
      expect(clients[1]).not.toBe(firstClient);
      expect(clients[1].getQueryData(['renderer-state'])).toBeUndefined();
      expect(clients[1].getMutationCache().getAll()).toHaveLength(0);

      second.unmount();
    },
  );
});
