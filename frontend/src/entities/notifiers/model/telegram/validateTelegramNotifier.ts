import type { TelegramNotifierSettings } from './TelegramNotifierSettings';

// eslint-disable-next-line i18next/no-literal-string -- URL protocols, not copy
const allowedProxyProtocols = ['http:', 'https:', 'socks5:', 'socks5h:'];

const isValidProxyUrl = (rawProxyUrl: string): boolean => {
  try {
    const proxyUrl = new URL(rawProxyUrl);
    return allowedProxyProtocols.includes(proxyUrl.protocol) && !!proxyUrl.host;
  } catch {
    return false;
  }
};

// eslint-disable-next-line i18next/no-literal-string -- URL protocols, not copy
const allowedBotApiProtocols = ['http:', 'https:'];

// Outside user info, which is rejected, "?" and "#" can only start a query or a fragment, and an
// empty one ("/telegram?") leaves search and hash empty.
const isValidBotApiUrl = (rawBotApiUrl: string): boolean => {
  try {
    const botApiUrl = new URL(rawBotApiUrl);
    return (
      allowedBotApiProtocols.includes(botApiUrl.protocol) &&
      !!botApiUrl.host &&
      !botApiUrl.username &&
      !botApiUrl.password &&
      !/[?#]/.test(rawBotApiUrl)
    );
  } catch {
    return false;
  }
};

// A thread ID of 0 means the topic is switched on and its ID is not entered yet.
export const validateTelegramNotifier = (
  isCreate: boolean,
  settings: TelegramNotifierSettings,
  isAddressChanged = false,
): boolean => {
  if ((isCreate || isAddressChanged) && !settings.botToken) {
    return false;
  }

  if (!settings.targetChatId) {
    return false;
  }

  if (settings.threadId !== undefined && settings.threadId <= 0) {
    return false;
  }

  if (settings.isProxyEnabled) {
    if (isCreate && !settings.proxyUrl) {
      return false;
    }

    if (settings.proxyUrl && !isValidProxyUrl(settings.proxyUrl)) {
      return false;
    }
  }

  if (settings.botApiUrl && !isValidBotApiUrl(settings.botApiUrl)) {
    return false;
  }

  return true;
};
