import type { StorageSettings } from '@/entities/storages';

export const acknowledgeStorageSettings = (options: {
  current: StorageSettings;
  submitted: StorageSettings;
  saved: StorageSettings;
}): StorageSettings => {
  if (
    options.current.type !== options.submitted.type ||
    options.current.type !== options.saved.type
  )
    return options.current;

  const acknowledged = structuredClone(options.current) as unknown as Record<string, unknown>;
  const submitted = options.submitted as unknown as Record<string, unknown>;
  const saved = options.saved as unknown as Record<string, unknown>;

  for (const field of new Set([...Object.keys(submitted), ...Object.keys(saved)])) {
    if (JSON.stringify(acknowledged[field]) !== JSON.stringify(submitted[field])) continue;

    if (saved[field] === undefined) delete acknowledged[field];
    else acknowledged[field] = structuredClone(saved[field]);
  }

  return acknowledged as unknown as StorageSettings;
};
