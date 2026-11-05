import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import type { TelegramNotifierSettings } from '../../../../../entities/notifiers';
import { LocaleProvider } from '../../../../../shared/i18n';
import { ShowTelegramNotifierComponent } from './ShowTelegramNotifierComponent';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const createTelegramSettings = (
  change: Partial<TelegramNotifierSettings>,
): TelegramNotifierSettings => ({
  type: 'TELEGRAM',
  targetChatId: '-1001234567890',
  isProxyEnabled: false,
  ...change,
});

const renderTelegramView = (settings: TelegramNotifierSettings) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <ShowTelegramNotifierComponent settings={settings} />
    </LocaleProvider>,
  );

describe('ShowTelegramNotifierComponent', () => {
  it('shows the Bot API URL when it is set', () => {
    const markup = renderTelegramView(
      createTelegramSettings({
        botApiUrl: 'https://gateway.example.com/telegram',
      }),
    );

    expect(markup).toContain('Bot API URL');
    expect(markup).toContain('https://gateway.example.com/telegram');
  });

  it('omits the Bot API URL when the official API is used', () => {
    const markup = renderTelegramView(createTelegramSettings({}));

    expect(markup).not.toContain('Bot API URL');
  });
});
