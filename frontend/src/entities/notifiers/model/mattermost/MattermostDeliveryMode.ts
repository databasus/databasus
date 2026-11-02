import type { MattermostNotifierSettings } from './MattermostNotifierSettings';

export type MattermostDeliveryMode = MattermostNotifierSettings['deliveryMode'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const MattermostDeliveryMode = {
  WEBHOOK: 'WEBHOOK',
  BOT: 'BOT',
} as const satisfies Record<string, MattermostDeliveryMode>;
