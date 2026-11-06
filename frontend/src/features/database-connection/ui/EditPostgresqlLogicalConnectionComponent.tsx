import { Checkbox, InputNumber, Select } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  POSTGRESQL_SHORT_NAME,
  PostgresqlConnectionStringParser,
  type PostgresqlLogicalSettings,
} from '@/entities/databases';
import type { PostgresqlRestoreOptions } from '@/entities/restores';
import type { ConnectionEditorProps } from '@/features/database-connection/lib/ConnectionEditorProps';
import { isLocalhostConnection } from '@/features/database-connection/lib/isLocalhostConnection';
import {
  applyPostgresqlHost,
  applyPostgresqlSslMode,
} from '@/features/database-connection/lib/postgresqlHostSsl';
import { ConnectionCredentialFields } from '@/features/database-connection/ui/ConnectionCredentialFields';
import { ConnectionFieldRow } from '@/features/database-connection/ui/ConnectionFieldRow';
import { ConnectionFieldTooltip } from '@/features/database-connection/ui/ConnectionFieldTooltip';
import { ConnectionHostHint } from '@/features/database-connection/ui/ConnectionHostHint';
import { ConnectionStringImport } from '@/features/database-connection/ui/ConnectionStringImport';
import { EditSshTunnelComponent } from '@/features/database-connection/ui/EditSshTunnelComponent';
import { PostgresqlCertificateFields } from '@/features/database-connection/ui/PostgresqlCertificateFields';
import { PostgresqlSslModeField } from '@/features/database-connection/ui/PostgresqlSslModeField';
import type { TranslationKey } from '@/shared/i18n';
import { NAME_LIST_TOKEN_SEPARATORS, normalizeNameList } from '@/shared/lib';
import { AdvancedSettingsToggleComponent } from '@/shared/ui';

interface Props extends ConnectionEditorProps<PostgresqlLogicalSettings> {
  hasUserChosenSslMode: boolean;
  savedClientCertificate?: string;
  restoreOptions?: PostgresqlRestoreOptions;
  onRestoreOptionsChange?: (restoreOptions: PostgresqlRestoreOptions) => void;
}

interface NameListFieldSpec {
  field: 'includeSchemas' | 'excludeTables' | 'excludeTableData';
  label: string;
  placeholder: string;
  tooltip?: string;
}

const RESTORE_OPTION_FIELDS: {
  option: keyof PostgresqlRestoreOptions;
  labelKey: TranslationKey;
  tooltipKey: TranslationKey;
}[] = [
  {
    option: 'isExcludeExtensions',
    labelKey: 'databases.fields.excludeExtensions',
    tooltipKey: 'databases.edit.postgresql.excludeExtensionsTooltip',
  },
  {
    option: 'isRestoreOwnership',
    labelKey: 'databases.fields.restoreOwnership',
    tooltipKey: 'databases.edit.postgresql.restoreOwnershipTooltip',
  },
  {
    option: 'isRestorePrivileges',
    labelKey: 'databases.fields.restorePrivileges',
    tooltipKey: 'databases.edit.postgresql.restorePrivilegesTooltip',
  },
];

const isSupabaseConnection = ({ host, username }: PostgresqlLogicalSettings): boolean =>
  host.includes('supabase') || username.includes('supabase');

const hasAdvancedValues = (
  settings: PostgresqlLogicalSettings,
  isRestore: boolean,
  restoreOptions: PostgresqlRestoreOptions | undefined,
): boolean => {
  if (
    settings.sslClientCert ||
    settings.sslClientKey ||
    settings.sslRootCert ||
    settings.sshTunnel?.isEnabled
  )
    return true;

  if (isRestore) {
    return (
      !!restoreOptions?.isExcludeExtensions ||
      !!restoreOptions?.isRestoreOwnership ||
      !!restoreOptions?.isRestorePrivileges
    );
  }

  return (
    !!settings.includeSchemas?.length ||
    !!settings.excludeTables?.length ||
    !!settings.excludeTableData?.length ||
    !!settings.isSkipUserMappings
  );
};

export const EditPostgresqlLogicalConnectionComponent = ({
  scenario,
  settings,
  savedSshTunnel,
  isAddressChanged,
  hasUserChosenSslMode,
  savedClientCertificate,
  restoreOptions,
  onConnectionChange,
  onNonConnectionChange,
  onRestoreOptionsChange,
}: Props) => {
  const { t } = useTranslation();
  const controlId = useId();
  const isRestore = scenario === 'restore';
  const [isShowAdvanced, setShowAdvanced] = useState(() =>
    hasAdvancedValues(settings, isRestore, restoreOptions),
  );
  const [hasAutoAddedPublicSchema, setHasAutoAddedPublicSchema] = useState(false);

  const addPublicSchemaForSupabase = (
    updatedSettings: PostgresqlLogicalSettings,
  ): PostgresqlLogicalSettings => {
    if (hasAutoAddedPublicSchema || !isSupabaseConnection(updatedSettings)) return updatedSettings;

    setHasAutoAddedPublicSchema(true);

    const currentSchemas = updatedSettings.includeSchemas ?? [];
    if (currentSchemas.includes('public')) return updatedSettings;

    // eslint-disable-next-line i18next/no-literal-string -- PostgreSQL schema name
    return { ...updatedSettings, includeSchemas: ['public', ...currentSchemas] };
  };

  const renderNameListField = ({ field, label, placeholder, tooltip }: NameListFieldSpec) => (
    <ConnectionFieldRow label={label} controlId={`${controlId}-${field}`}>
      <div className="flex w-full items-center">
        <Select
          id={`${controlId}-${field}`}
          mode="tags"
          value={settings[field] || []}
          onChange={(values) =>
            onNonConnectionChange({ ...settings, [field]: normalizeNameList(values) })
          }
          size="small"
          className="w-full max-w-[200px]"
          placeholder={placeholder}
          tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
        />

        {tooltip && <ConnectionFieldTooltip title={tooltip} />}
      </div>
    </ConnectionFieldRow>
  );

  return (
    <>
      <ConnectionStringImport
        parse={(connectionString) => PostgresqlConnectionStringParser.parse(connectionString)}
        onParsed={(parsed) =>
          onConnectionChange(
            addPublicSchemaForSupabase(
              applyPostgresqlSslMode(
                {
                  ...settings,
                  host: parsed.host,
                  port: parsed.port,
                  username: parsed.username,
                  password: parsed.password,
                  database: parsed.database,
                  cpuCount: 1,
                },
                parsed.sslMode,
              ),
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
        hostHints={
          <>
            {isLocalhostConnection(settings) && <ConnectionHostHint topic="localhost" />}
            {isSupabaseConnection(settings) && <ConnectionHostHint topic="supabase" />}
          </>
        }
        onHostChange={(rawHost) =>
          onConnectionChange(
            addPublicSchemaForSupabase(
              applyPostgresqlHost(settings, rawHost, hasUserChosenSslMode),
            ),
          )
        }
        onValuesChange={(change) =>
          onConnectionChange(addPublicSchemaForSupabase({ ...settings, ...change }))
        }
      />

      <PostgresqlSslModeField
        controlId={controlId}
        sslMode={settings.sslMode}
        onChange={(sslMode) => onConnectionChange(applyPostgresqlSslMode(settings, sslMode), true)}
      />

      {isRestore && (
        <ConnectionFieldRow label={t('databases.fields.cpuCount')} controlId={`${controlId}-cpu`}>
          <div className="flex items-center">
            <InputNumber
              id={`${controlId}-cpu`}
              min={1}
              max={128}
              value={settings.cpuCount}
              onChange={(cpuCount) => onConnectionChange({ ...settings, cpuCount: cpuCount || 1 })}
              size="small"
              className="max-w-[75px]"
            />

            <ConnectionFieldTooltip title={t('databases.edit.cpuCountTooltip')} />
          </div>
        </ConnectionFieldRow>
      )}

      <AdvancedSettingsToggleComponent
        label={t('databases.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <>
          <EditSshTunnelComponent
            sshTunnel={settings.sshTunnel}
            savedSshTunnel={savedSshTunnel}
            onChange={(sshTunnel) => onConnectionChange({ ...settings, sshTunnel })}
          />

          {!isRestore && (
            <>
              {renderNameListField({
                field: 'includeSchemas',
                label: t('databases.fields.includeSchemas'),
                placeholder: t('databases.edit.postgresql.includeSchemasPlaceholder'),
              })}

              {renderNameListField({
                field: 'excludeTables',
                label: t('databases.fields.excludeWholeTables'),
                placeholder: t('databases.edit.excludeTablesPlaceholder'),
                tooltip: t('databases.edit.postgresql.excludeTablesTooltip'),
              })}

              {renderNameListField({
                field: 'excludeTableData',
                label: t('databases.fields.excludeTableData'),
                placeholder: t('databases.edit.postgresql.excludeTableDataPlaceholder'),
                tooltip: t('databases.edit.postgresql.excludeTableDataTooltip'),
              })}

              <ConnectionFieldRow
                label={t('databases.fields.skipUserMappings')}
                controlId={`${controlId}-skip-user-mappings`}
              >
                <div className="flex items-center">
                  <Checkbox
                    id={`${controlId}-skip-user-mappings`}
                    checked={settings.isSkipUserMappings || false}
                    onChange={(e) =>
                      onNonConnectionChange({ ...settings, isSkipUserMappings: e.target.checked })
                    }
                  />

                  <ConnectionFieldTooltip
                    title={t('databases.edit.postgresql.skipUserMappingsTooltip')}
                  />
                </div>
              </ConnectionFieldRow>
            </>
          )}

          {restoreOptions &&
            RESTORE_OPTION_FIELDS.map(({ option, labelKey, tooltipKey }) => (
              <ConnectionFieldRow
                key={option}
                label={t(labelKey)}
                controlId={`${controlId}-${option}`}
              >
                <div className="flex items-center">
                  <Checkbox
                    id={`${controlId}-${option}`}
                    checked={restoreOptions[option]}
                    onChange={(e) =>
                      onRestoreOptionsChange?.({ ...restoreOptions, [option]: e.target.checked })
                    }
                  />

                  <ConnectionFieldTooltip title={t(tooltipKey)} />
                </div>
              </ConnectionFieldRow>
            ))}

          <PostgresqlCertificateFields
            values={settings}
            isAddressChanged={isAddressChanged}
            savedClientCertificate={savedClientCertificate}
            onChange={(change) => onConnectionChange({ ...settings, ...change })}
          />
        </>
      )}
    </>
  );
};
