import { describe, expect, it } from 'vitest';

import { MattermostDeliveryMode } from '@/entities/notifiers/model/mattermost/MattermostDeliveryMode';
import type { MattermostNotifierSettings } from '@/entities/notifiers/model/mattermost/MattermostNotifierSettings';
import { validateMattermostNotifier } from '@/entities/notifiers/model/mattermost/validateMattermostNotifier';

const emptyNotifier: MattermostNotifierSettings = {
  type: 'MATTERMOST',
  deliveryMode: MattermostDeliveryMode.WEBHOOK,
  isInsecureSkipVerify: false,
};

const channelId = 'abcdefghijklmnopqrstuvwxyz';

describe('validateMattermostNotifier', () => {
  it('requires a webhook URL on create in webhook mode', () => {
    expect(validateMattermostNotifier(true, emptyNotifier)).toBe(false);
    expect(
      validateMattermostNotifier(true, {
        ...emptyNotifier,
        webhookUrl: 'https://mattermost.example.com/hooks/key',
      }),
    ).toBe(true);
  });

  it('allows a blank webhook URL on edit', () => {
    expect(validateMattermostNotifier(false, emptyNotifier)).toBe(true);
  });

  it('requires server URL, token and channel ID on create in bot mode', () => {
    const botNotifier: MattermostNotifierSettings = {
      ...emptyNotifier,
      deliveryMode: MattermostDeliveryMode.BOT,
      serverUrl: 'https://mattermost.example.com',
      botToken: 'token',
      targetChannelId: channelId,
    };

    expect(validateMattermostNotifier(true, botNotifier)).toBe(true);
    expect(validateMattermostNotifier(true, { ...botNotifier, serverUrl: '' })).toBe(false);
    expect(validateMattermostNotifier(true, { ...botNotifier, botToken: undefined })).toBe(false);
    expect(
      validateMattermostNotifier(false, {
        ...botNotifier,
        botToken: undefined,
      }),
    ).toBe(true);
  });

  it('rejects a channel name pasted into the channel ID field', () => {
    expect(
      validateMattermostNotifier(true, {
        ...emptyNotifier,
        deliveryMode: MattermostDeliveryMode.BOT,
        serverUrl: 'https://mattermost.example.com',
        botToken: 'token',
        targetChannelId: 'town-square',
      }),
    ).toBe(false);
  });
});
