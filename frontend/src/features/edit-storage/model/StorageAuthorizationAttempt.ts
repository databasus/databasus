import type { GoogleDriveStorageSettings } from '@/entities/storages';
import type { SessionSnapshot } from '@/shared/auth';

export interface StorageAuthorizationAttempt {
  state: string;
  createdAt: number;
  redirectUri: string;
  workspaceId: string;
  storageId?: string;
  editorId: symbol;
  session: SessionSnapshot;
  settings: GoogleDriveStorageSettings;
  revision: number;
  isConsumed: boolean;
  channel: BroadcastChannel;
  timer: ReturnType<typeof setTimeout>;
}
