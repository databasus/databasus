import type { components } from '../src/shared/api/schema.gen';

export const notifierProviders: {
  variant?: string;
  settings: components['schemas']['NotifierSettings'];
}[] = [
  {
    settings: {
      type: 'EMAIL',
      targetEmail: 'ops@example.com',
      smtpHost: 'smtp.example.com',
      smtpPort: 587,
      smtpUser: 'mailer',
      from: 'Backups <backups@example.com>',
      security: 'STARTTLS',
      isInsecureSkipVerify: false,
      heloName: 'backups.example.com',
    },
  },
  {
    settings: {
      type: 'TELEGRAM',
      targetChatId: '-100123456',
      isProxyEnabled: true,
      proxyUrl: 'https://proxy.example.com',
      botApiUrl: 'https://telegram.example.com',
      threadId: 7,
    },
  },
  {
    settings: {
      type: 'WEBHOOK',
      webhookUrl: 'https://webhook.example.com/notify',
      webhookMethod: 'POST',
      headers: [{ name: 'Authorization' }],
      acceptedCategories: ['ALL'],
      bodyTemplate: '{"heading":"{{heading}}","message":"{{message}}"}',
    },
  },
  { settings: { type: 'SLACK', targetChatId: 'C123456' } },
  { settings: { type: 'DISCORD' } },
  { settings: { type: 'TEAMS' } },
  {
    settings: {
      type: 'MATTERMOST',
      deliveryMode: 'BOT',
      serverUrl: 'https://mattermost.example.com',
      targetChannelId: 'a'.repeat(26),
      isInsecureSkipVerify: false,
      overrideUsername: 'Backup service',
    },
  },
  {
    variant: 'MATTERMOST incoming webhook',
    settings: {
      type: 'MATTERMOST',
      deliveryMode: 'WEBHOOK',
      targetChannelName: 'backups',
      isInsecureSkipVerify: false,
      overrideUsername: 'Backup service',
    },
  },
];
