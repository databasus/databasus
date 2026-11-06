import type { NotifierSettings } from '@/entities/notifiers/model/NotifierSettings';

export type NotifierType = NotifierSettings['type'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const NotifierType = {
  EMAIL: 'EMAIL',
  TELEGRAM: 'TELEGRAM',
  WEBHOOK: 'WEBHOOK',
  SLACK: 'SLACK',
  DISCORD: 'DISCORD',
  TEAMS: 'TEAMS',
  MATTERMOST: 'MATTERMOST',
} as const satisfies Record<string, NotifierType>;
