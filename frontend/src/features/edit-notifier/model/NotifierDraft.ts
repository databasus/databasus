import type { NotifierSettings } from '../../../entities/notifiers';

export interface NotifierDraft {
  name: string;
  settings: NotifierSettings;
}

const getSignature = (value: unknown): string =>
  JSON.stringify(value, (_key, field) => {
    if (!field || typeof field !== 'object' || Array.isArray(field)) return field;

    return Object.fromEntries(
      Object.entries(field).sort(([first], [second]) => first.localeCompare(second)),
    );
  });

export const isNotifierDraftUnchanged = ({
  draft,
  baseline,
}: {
  draft: NotifierDraft;
  baseline: NotifierDraft;
}): boolean => getSignature(draft) === getSignature(baseline);
