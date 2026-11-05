import type { StorageSettings } from '@/entities/storages';

export interface StorageDraft {
  storageId?: string;
  name: string;
  settings: StorageSettings;
}

const getSettingsSignature = (value: unknown): string =>
  JSON.stringify(value, (_key, field) => {
    if (!field || typeof field !== 'object' || Array.isArray(field)) return field;

    return Object.fromEntries(
      Object.entries(field).sort(([first], [second]) => first.localeCompare(second)),
    );
  });

export const isStorageDraftUnchanged = ({
  draft,
  baseline,
}: {
  draft: StorageDraft;
  baseline: StorageDraft;
}) =>
  draft.name === baseline.name &&
  getSettingsSignature(draft.settings) === getSettingsSignature(baseline.settings);
