import type { NotifierSettings } from '@/entities/notifiers';

interface AcknowledgedSettings {
  settings: NotifierSettings;
  submittedSettings: NotifierSettings;
  savedSettings: NotifierSettings;
}

export const acknowledgeNotifierSettings = ({
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

  for (const field of new Set([...Object.keys(submittedValues), ...Object.keys(savedValues)])) {
    if (
      field === 'headers' ||
      JSON.stringify(values[field]) !== JSON.stringify(submittedValues[field])
    )
      continue;

    if (savedValues[field] === undefined) delete values[field];
    else values[field] = structuredClone(savedValues[field]);
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
