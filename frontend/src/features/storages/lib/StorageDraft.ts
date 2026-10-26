import type { StorageSettings } from '../../../entity/storages';

// A storage form that is filled but not saved yet, such as the one the Google Drive OAuth redirect
// brings back. storageId names the saved storage the form edits, absent for a new one.
export interface StorageDraft {
  storageId?: string;
  name: string;
  settings: StorageSettings;
}
