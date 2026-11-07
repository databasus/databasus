// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { TelegramNotifierSettings } from '@/entities/notifiers';
import { EditTelegramNotifierComponent } from '@/features/edit-notifier/ui/edit/notifiers/EditTelegramNotifierComponent';
import { en } from '@/shared/i18n/locales/en';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const createTelegramSettings = (
  change: Partial<TelegramNotifierSettings>,
): TelegramNotifierSettings => ({
  type: 'TELEGRAM',
  targetChatId: '-1001234567890',
  isProxyEnabled: false,
  ...change,
});

const renderTelegramForm = (settings: TelegramNotifierSettings) =>
  renderScreen(
    <EditTelegramNotifierComponent
      settings={settings}
      isSaved
      isAddressChanged={false}
      onChange={() => {}}
    />,
  );

describe('EditTelegramNotifierComponent', () => {
  it('associates advanced URL fields with their help when proxy settings are shown', () => {
    renderTelegramForm(
      createTelegramSettings({
        isProxyEnabled: true,
        botApiUrl: 'https://gateway.example.com/telegram',
        proxyUrl: 'socks5://proxy.example.com:1080',
      }),
    );

    expect(
      screen.getByRole('textbox', {
        name: en.notifiers.telegram.botApiUrl,
        description: en.notifiers.telegram.botApiUrlTooltip,
      }),
    ).toHaveProperty('value', 'https://gateway.example.com/telegram');
    expect(
      screen.getByRole('textbox', {
        name: en.notifiers.telegram.proxyUrl,
        description: en.notifiers.telegram.proxyUrlTooltip,
      }),
    ).toHaveProperty('value', 'socks5://proxy.example.com:1080');
  });

  it('starts with the advanced settings collapsed for a new notifier', () => {
    renderTelegramForm(createTelegramSettings({}));

    expect(
      screen.getByRole('button', { name: 'Advanced settings' }).getAttribute('aria-expanded'),
    ).toBe('false');
    expect(screen.getByRole('textbox', { name: 'Target chat ID' })).toBeTruthy();
    expect(screen.queryByRole('textbox', { name: 'Bot API URL' })).toBeNull();
    expect(screen.queryByRole('switch', { name: 'Use proxy' })).toBeNull();
  });

  it('starts with the advanced settings expanded when a proxy is enabled', () => {
    renderTelegramForm(createTelegramSettings({ isProxyEnabled: true }));

    expect(screen.getByRole('textbox', { name: 'Bot API URL' })).toBeTruthy();
    expect(screen.getByRole('switch', { name: 'Use proxy' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: 'Proxy URL' })).toBeTruthy();
  });

  it('starts with the advanced settings expanded when a Bot API URL is set', () => {
    renderTelegramForm(
      createTelegramSettings({ botApiUrl: 'https://gateway.example.com/telegram' }),
    );

    expect((screen.getByRole('textbox', { name: 'Bot API URL' }) as HTMLInputElement).value).toBe(
      'https://gateway.example.com/telegram',
    );
    expect(screen.getByRole('switch', { name: 'Use proxy' })).toBeTruthy();
    expect(screen.queryByRole('textbox', { name: 'Proxy URL' })).toBeNull();
  });
});
