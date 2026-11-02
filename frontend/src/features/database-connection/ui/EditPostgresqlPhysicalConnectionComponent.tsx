import { Radio, Space } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS,
  POSTGRESQL_SHORT_NAME,
  PhysicalDatabaseBackupType,
  PostgresqlConnectionStringParser,
  type PostgresqlPhysicalSettings,
} from '@/entities/databases';
import type { TranslationKey } from '@/shared/i18n';
import { AdvancedSettingsToggleComponent } from '@/shared/ui';

import type { ConnectionEditorProps } from '../lib/ConnectionEditorProps';
import { applyPostgresqlHost, applyPostgresqlSslMode } from '../lib/postgresqlHostSsl';
import { ConnectionCredentialFields } from './ConnectionCredentialFields';
import { ConnectionFieldTooltip } from './ConnectionFieldTooltip';
import { ConnectionStringImport } from './ConnectionStringImport';
import { EditSshTunnelComponent } from './EditSshTunnelComponent';
import { PostgresqlCertificateFields } from './PostgresqlCertificateFields';
import { PostgresqlSslModeField } from './PostgresqlSslModeField';

interface Props extends ConnectionEditorProps<PostgresqlPhysicalSettings> {
  hasUserChosenSslMode: boolean;
  savedClientCertificate?: string;
}

const BACKUP_TYPE_TOOLTIP_KEYS: Record<PhysicalDatabaseBackupType, TranslationKey> = {
  [PhysicalDatabaseBackupType.FULL]: 'databases.edit.postgresql.backupTypeTooltips.full',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL]:
    'databases.edit.postgresql.backupTypeTooltips.fullIncremental',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM]:
    'databases.edit.postgresql.backupTypeTooltips.fullIncrementalWalStream',
};

export const EditPostgresqlPhysicalConnectionComponent = ({
  settings,
  savedSshTunnel,
  isAddressChanged,
  hasUserChosenSslMode,
  savedClientCertificate,
  onConnectionChange,
}: Props) => {
  const { t } = useTranslation();
  const controlId = useId();
  const backupTypeLabelId = useId();
  const [isShowAdvanced, setShowAdvanced] = useState(!!settings.sshTunnel?.isEnabled);

  return (
    <>
      <div className="mb-3 flex w-full flex-col items-start sm:flex-row">
        <div
          id={backupTypeLabelId}
          className="mb-1 min-w-0 sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2"
        >
          {t('databases.fields.backupType')}
        </div>
        <Radio.Group
          aria-labelledby={backupTypeLabelId}
          value={settings.backupType ?? PhysicalDatabaseBackupType.FULL}
          onChange={(e) => onConnectionChange({ ...settings, backupType: e.target.value })}
        >
          <Space direction="vertical" size={0}>
            {Object.values(PhysicalDatabaseBackupType).map((backupType) => (
              <Radio key={backupType} value={backupType} className="my-1! leading-[17px]!">
                {t(PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS[backupType])}
                <ConnectionFieldTooltip title={t(BACKUP_TYPE_TOOLTIP_KEYS[backupType])} />
              </Radio>
            ))}
          </Space>
        </Radio.Group>
      </div>

      <ConnectionStringImport
        parse={(connectionString) => PostgresqlConnectionStringParser.parse(connectionString)}
        onParsed={(parsed) =>
          onConnectionChange(
            applyPostgresqlSslMode(
              {
                ...settings,
                host: parsed.host,
                port: parsed.port,
                username: parsed.username,
                password: parsed.password,
              },
              parsed.sslMode,
            ),
            true,
          )
        }
      />

      <ConnectionCredentialFields
        controlId={controlId}
        engineName={POSTGRESQL_SHORT_NAME}
        values={settings}
        isAddressChanged={isAddressChanged}
        isDatabaseNameShown={false}
        onHostChange={(rawHost) =>
          onConnectionChange(applyPostgresqlHost(settings, rawHost, hasUserChosenSslMode))
        }
        onValuesChange={(change) => onConnectionChange({ ...settings, ...change })}
      />

      <PostgresqlSslModeField
        controlId={controlId}
        sslMode={settings.sslMode}
        onChange={(sslMode) => onConnectionChange(applyPostgresqlSslMode(settings, sslMode), true)}
      />

      <PostgresqlCertificateFields
        values={settings}
        isAddressChanged={isAddressChanged}
        savedClientCertificate={savedClientCertificate}
        onChange={(change) => onConnectionChange({ ...settings, ...change })}
      />

      <AdvancedSettingsToggleComponent
        label={t('databases.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <EditSshTunnelComponent
          sshTunnel={settings.sshTunnel}
          savedSshTunnel={savedSshTunnel}
          onChange={(sshTunnel) => onConnectionChange({ ...settings, sshTunnel })}
        />
      )}
    </>
  );
};
