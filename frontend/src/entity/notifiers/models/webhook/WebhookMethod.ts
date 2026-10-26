import type { WebhookNotifierSettings } from './WebhookNotifierSettings';

export type WebhookMethod = WebhookNotifierSettings['webhookMethod'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const WebhookMethod = {
  POST: 'POST',
  GET: 'GET',
} as const satisfies Record<string, WebhookMethod>;
