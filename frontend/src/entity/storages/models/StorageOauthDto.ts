import type { GoogleDriveStorageSettings } from './ProviderStorageSettings';

// The Google Drive form carried through the OAuth redirect in its state, the client secret
// included, so the storage can be saved once the code is exchanged for a token.
export interface StorageOauthDto {
  workspaceId: string;
  // Absent for a storage that is not saved yet.
  storageId?: string;
  name: string;
  settings: GoogleDriveStorageSettings;
  authCode: string;
}
