import { type DatabaseConnectionDraft, DatabaseType } from '@/entities/databases';
import { isOptionalValueChanged } from '@/shared/lib';

import { isSshTunnelAddressChanged } from './isSshTunnelAddressChanged';

// Mirrors the resource address each database provider compares on the backend, which stays the
// authority: a stale list here only makes the user see the generic refusal.
export const isConnectionAddressChanged = (draft: DatabaseConnectionDraft): boolean => {
  if (draft.scenario !== 'edit') return false;

  const { settings, savedSettings } = draft;

  return (
    isOptionalValueChanged(settings.host, savedSettings.host) ||
    isOptionalValueChanged(settings.port, savedSettings.port) ||
    isSshTunnelAddressChanged(settings.sshTunnel, savedSettings.sshTunnel) ||
    (settings.type === DatabaseType.MONGODB &&
      savedSettings.type === DatabaseType.MONGODB &&
      !!settings.isSrv !== !!savedSettings.isSrv)
  );
};
