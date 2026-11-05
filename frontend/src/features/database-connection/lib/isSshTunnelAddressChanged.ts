import type { SshTunnelSettings } from '@/entities/databases';
import { isOptionalValueChanged } from '@/shared/lib';

// Mirrors IsSSHTunnelAddressChanged on the backend: a disabled tunnel has no address, and turning
// one on or off counts as a change.
export const isSshTunnelAddressChanged = (
  sshTunnel: SshTunnelSettings | undefined,
  savedSshTunnel: SshTunnelSettings | undefined,
): boolean => {
  const isEnabled = !!sshTunnel?.isEnabled;

  if (isEnabled !== !!savedSshTunnel?.isEnabled) return true;
  if (!isEnabled) return false;

  return (
    isOptionalValueChanged(sshTunnel?.host, savedSshTunnel?.host) ||
    isOptionalValueChanged(sshTunnel?.port, savedSshTunnel?.port)
  );
};
