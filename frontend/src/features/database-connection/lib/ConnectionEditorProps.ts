import type { DatabaseConnectionDraft, SshTunnelSettings } from '@/entities/databases';

export interface ConnectionEditorProps<Settings> {
  scenario: DatabaseConnectionDraft['scenario'];
  settings: Settings;
  savedSshTunnel?: SshTunnelSettings;
  isAddressChanged: boolean;
  onConnectionChange: (settings: Settings, hasUserChosenSslMode?: boolean) => void;
  onNonConnectionChange: (settings: Settings) => void;
}
