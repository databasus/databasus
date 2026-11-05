// @vitest-environment happy-dom
import { act, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { StorageOAuthPage } from './StorageOAuthPage';

const mockServer = setUpMockApi();
const state = 'a'.repeat(64);
let previousAddress: string;

beforeEach(() => {
  previousAddress = window.location.href;
});
afterEach(() => {
  window.history.replaceState(null, '', previousAddress);
});

const openCallback = (query: string, isStrictMode = false) => {
  const messages: unknown[] = [];
  const channels: { isClosed: boolean; onmessage?: (event: MessageEvent) => void }[] = [];
  vi.spyOn(window, 'close').mockImplementation(() => {});
  vi.stubGlobal(
    'BroadcastChannel',
    class {
      isClosed = false;
      onmessage?: (event: MessageEvent) => void;
      constructor(readonly name: string) {
        expect(name).toBe(`databasus-storage-authorization:${state}`);
        channels.push(this);
      }
      postMessage(message: unknown) {
        expect(window.location.search).toBe('');
        messages.push(message);
      }
      close() {
        this.isClosed = true;
      }
    },
  );

  window.history.replaceState(null, '', `/storages/google-oauth?${query}`);
  const screenView = renderScreen(<StorageOAuthPage />, { isStrictMode });

  return { ...screenView, messages, channels };
};

describe('StorageOAuthPage', () => {
  it.each([
    '',
    'state=wrong&code=code',
    `state=${encodeURIComponent('{}')}&code=code`,
    `state=${state}&code=code&error=refused`,
    `state=${state}&code=`,
    `state=${state}&state=${state}&code=code`,
    `state=${state}&error=`,
    `state=${state}&code=one&code=two`,
  ])('clears and rejects malformed callback %s without requests', (query) => {
    const callback = openCallback(query);
    expect(window.location.search).toBe('');
    expect(screen.getByRole('alert').textContent).toContain('start authorization again');
    expect(callback.messages).toHaveLength(0);
    expect(mockServer.getRequests()).toHaveLength(0);
  });

  it('relays only code and state across setup replay and waits for a valid acknowledgment', () => {
    vi.useFakeTimers();
    const callback = openCallback(`state=${state}&code=synthetic-code`, true);

    expect(callback.messages).toEqual(
      Array(2).fill({ type: 'code', state, code: 'synthetic-code' }),
    );
    expect(callback.channels[0].isClosed).toBe(true);

    const channel = callback.channels[callback.channels.length - 1];
    act(() => {
      channel.onmessage?.(
        new MessageEvent('message', {
          origin: 'https://other.example',
          data: { type: 'ack', state },
        }),
      );
      channel.onmessage?.(
        new MessageEvent('message', {
          origin: window.location.origin,
          data: { type: 'ack', state: 'wrong' },
        }),
      );
    });

    expect(screen.getByRole('status')).toBeTruthy();
    act(() =>
      channel.onmessage?.(
        new MessageEvent('message', {
          origin: window.location.origin,
          data: { type: 'ack', state },
        }),
      ),
    );

    expect(screen.getByText('Authorization received. Return to the storage editor.')).toBeTruthy();
    expect(window.close).toHaveBeenCalledTimes(1);
    expect(callback.channels.every((channel) => channel.isClosed)).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('omits provider descriptions and stops waiting for a missing editor after five seconds', () => {
    vi.useFakeTimers();
    const callback = openCallback(
      `state=${state}&error=access_denied&error_description=private-provider-text`,
    );

    expect(callback.messages).toEqual([{ type: 'refusal', state }]);

    act(() => vi.advanceTimersByTime(4999));
    expect(screen.getByRole('status')).toBeTruthy();

    act(() => vi.advanceTimersByTime(1));
    expect(screen.getByRole('alert').textContent).toContain('start authorization again');
    expect(screen.queryByText('private-provider-text')).toBeNull();
    expect(callback.channels.every((channel) => channel.isClosed)).toBe(true);
    expect(mockServer.getRequests()).toHaveLength(0);
  });

  it('releases channels and timers on unmount', () => {
    vi.useFakeTimers();
    const callback = openCallback(`state=${state}&code=synthetic-code`);
    callback.unmount();
    expect(callback.channels.every((channel) => channel.isClosed)).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });
});
