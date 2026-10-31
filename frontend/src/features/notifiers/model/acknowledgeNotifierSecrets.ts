import type { NotifierSettings } from '../../../entities/notifiers';

const SECRET_FIELDS = {
  EMAIL: ['smtpPassword'],
  TELEGRAM: ['botToken', 'proxyUrl'],
  SLACK: ['botToken'],
  DISCORD: ['channelWebhookUrl'],
  TEAMS: ['powerAutomateUrl'],
  MATTERMOST: ['webhookUrl', 'botToken'],
  WEBHOOK: [],
} satisfies {
  [Type in NotifierSettings['type']]: readonly (keyof Extract<NotifierSettings, { type: Type }>)[];
};

interface AcknowledgedSettings {
  settings: NotifierSettings;
  submittedSettings: NotifierSettings;
  savedSettings: NotifierSettings;
}

export const acknowledgeNotifierSecrets = ({
  settings,
  submittedSettings,
  savedSettings,
}: AcknowledgedSettings): NotifierSettings => {
  const acknowledged = structuredClone(settings);
  if (settings.type !== submittedSettings.type || settings.type !== savedSettings.type)
    return acknowledged;

  const values = acknowledged as unknown as Record<string, unknown>;
  const submittedValues = submittedSettings as unknown as Record<string, unknown>;
  const savedValues = savedSettings as unknown as Record<string, unknown>;
  for (const field of SECRET_FIELDS[settings.type]) {
    if (values[field] === submittedValues[field] && savedValues[field] === undefined)
      delete values[field];
  }

  if (
    acknowledged.type === 'WEBHOOK' &&
    submittedSettings.type === 'WEBHOOK' &&
    savedSettings.type === 'WEBHOOK'
  ) {
    acknowledged.headers = acknowledged.headers.map((header) => {
      const submittedHeader = submittedSettings.headers.find(
        (candidate) => candidate.name === header.name,
      );
      const savedHeader = savedSettings.headers.find((candidate) => candidate.name === header.name);
      return header.value === submittedHeader?.value && savedHeader?.value === undefined
        ? { name: header.name }
        : header;
    });
  }
  return acknowledged;
};
