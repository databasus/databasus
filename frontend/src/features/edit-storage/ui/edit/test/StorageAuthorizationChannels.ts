import { vi } from 'vitest';

export const installStorageAuthorizationChannels = () => {
  const channels: Channel[] = [];
  const messages: unknown[] = [];
  class Channel {
    onmessage: ((event: MessageEvent) => void) | null = null;
    isClosed = false;
    constructor(readonly name: string) {
      channels.push(this);
    }
    postMessage(message: unknown) {
      messages.push(message);

      for (const channel of channels) {
        if (channel !== this && channel.name === this.name && !channel.isClosed) {
          channel.onmessage?.(
            new MessageEvent('message', { data: message, origin: window.location.origin }),
          );
        }
      }
    }
    close() {
      this.isClosed = true;
      this.onmessage = null;
    }
  }
  // eslint-disable-next-line i18next/no-literal-string -- browser API boundary
  vi.stubGlobal('BroadcastChannel', Channel);

  return {
    channels,
    messages,
    send: (state: string, message: unknown, origin = window.location.origin) => {
      for (const channel of channels) {
        if (channel.name === `databasus-storage-authorization:${state}` && !channel.isClosed)
          channel.onmessage?.(new MessageEvent('message', { data: message, origin }));
      }
    },
  };
};
