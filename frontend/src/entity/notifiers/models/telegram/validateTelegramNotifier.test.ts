import { describe, expect, it } from 'vitest';

import type { TelegramNotifierSettings } from './TelegramNotifierSettings';
import { validateTelegramNotifier } from './validateTelegramNotifier';

const buildTelegram = (change: Partial<TelegramNotifierSettings>): TelegramNotifierSettings => ({
  type: 'TELEGRAM',
  botToken: 'token',
  targetChatId: '123456',
  isProxyEnabled: false,
  ...change,
});

describe('validateTelegramNotifier', () => {
  it('requires proxy URL when proxy is enabled for a new notifier', () => {
    expect(validateTelegramNotifier(true, buildTelegram({ isProxyEnabled: true }))).toBe(false);
  });

  it('allows an empty proxy URL on update (keeps the existing one)', () => {
    expect(
      validateTelegramNotifier(false, buildTelegram({ botToken: undefined, isProxyEnabled: true })),
    ).toBe(true);
  });

  it('rejects proxy URLs with an unsupported scheme', () => {
    expect(
      validateTelegramNotifier(
        true,
        buildTelegram({
          isProxyEnabled: true,
          proxyUrl: 'ftp://proxy.example.com:3128',
        }),
      ),
    ).toBe(false);
  });

  it('accepts http, https, socks5 and socks5h proxy URLs', () => {
    for (const proxyUrl of [
      'http://user:password@proxy.example.com:3128',
      'https://proxy.example.com:8443',
      'socks5://user:password@proxy.example.com:1080',
      'socks5h://proxy.example.com:1080',
    ]) {
      expect(
        validateTelegramNotifier(true, buildTelegram({ isProxyEnabled: true, proxyUrl })),
      ).toBe(true);
    }
  });

  it('accepts an empty Bot API URL and plain http or https addresses', () => {
    for (const botApiUrl of [
      undefined,
      '',
      'http://localhost:8081',
      'http://telegram-bot-api:8081',
      'https://gateway.example.com',
      'https://gateway.example.com/telegram/',
    ]) {
      expect(validateTelegramNotifier(true, buildTelegram({ botApiUrl }))).toBe(true);
    }
  });

  it('rejects Bot API URLs with another scheme, no host, credentials, a query or a fragment', () => {
    for (const botApiUrl of [
      'ftp://telegram.example.com',
      'telegram-bot-api:8081',
      'https://',
      'gateway.example.com/telegram',
      'https://user:password@gateway.example.com',
      'https://user@gateway.example.com',
      'https://gateway.example.com/telegram?key=1',
      'https://gateway.example.com/telegram?',
      'https://gateway.example.com/#bot',
      'https://gateway.example.com/#',
    ]) {
      expect(validateTelegramNotifier(true, buildTelegram({ botApiUrl }))).toBe(false);
    }
  });

  it('requires a positive thread ID once the topic is switched on', () => {
    expect(validateTelegramNotifier(true, buildTelegram({ threadId: 0 }))).toBe(false);
    expect(validateTelegramNotifier(true, buildTelegram({ threadId: 3 }))).toBe(true);
  });
});
