import { Checkbox, Select, Switch } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  DatabaseType,
  type MariadbSettings,
  MySqlCompatibleConnectionStringParser,
  type MysqlSettings,
} from '@/entities/databases';
import { NAME_LIST_TOKEN_SEPARATORS, normalizeNameList } from '@/shared/lib';
import { AdvancedSettingsToggleComponent } from '@/shared/ui';

import type { ConnectionEditorProps } from '../lib/ConnectionEditorProps';
import { isLocalhostConnection } from '../lib/isLocalhostConnection';
import { stripHostScheme } from '../lib/stripHostScheme';
import { ConnectionCredentialFields } from './ConnectionCredentialFields';
import { ConnectionFieldRow } from './ConnectionFieldRow';
import { ConnectionFieldTooltip } from './ConnectionFieldTooltip';
import { ConnectionHostHint } from './ConnectionHostHint';
import { ConnectionStringImport } from './ConnectionStringImport';
import { EditSshTunnelComponent } from './EditSshTunnelComponent';

type Settings = MysqlSettings | MariadbSettings;

const hasAdvancedValues = (settings: Settings, isRestore: boolean): boolean => {
  if (settings.sshTunnel?.isEnabled) return true;
  if (settings.type === DatabaseType.MARIADB && settings.isSkipGaleraDisable) return true;
  if (isRestore) return false;

  return (
    !!settings.excludeTables?.length ||
    (settings.type === DatabaseType.MARIADB && !!settings.isExcludeEvents)
  );
};

export const EditMySqlCompatibleConnectionComponent = ({
  scenario,
  settings,
  savedSshTunnel,
  isAddressChanged,
  onConnectionChange,
  onNonConnectionChange,
}: ConnectionEditorProps<Settings>) => {
  const { t } = useTranslation();
  const controlId = useId();
  const isRestore = scenario === 'restore';
  const [isShowAdvanced, setShowAdvanced] = useState(() => hasAdvancedValues(settings, isRestore));

  const engineName = t(DATABASE_TYPE_LABEL_KEYS[settings.type]);

  return (
    <>
      <ConnectionStringImport
        parse={(connectionString) =>
          MySqlCompatibleConnectionStringParser.parse(connectionString, settings.type)
        }
        onParsed={(parsed) =>
          onConnectionChange({
            ...settings,
            host: parsed.host,
            port: parsed.port,
            username: parsed.username,
            password: parsed.password,
            database: parsed.database,
            isHttps: parsed.isHttps,
          })
        }
      />

      <ConnectionCredentialFields
        controlId={controlId}
        engineName={engineName}
        values={settings}
        isAddressChanged={isAddressChanged}
        hostHints={isLocalhostConnection(settings) && <ConnectionHostHint topic="localhost" />}
        onHostChange={(rawHost) =>
          onConnectionChange({ ...settings, host: stripHostScheme(rawHost) })
        }
        onValuesChange={(change) => onConnectionChange({ ...settings, ...change })}
      />

      <div className="mb-3">
        <ConnectionFieldRow label={t('databases.fields.useHttps')} controlId={`${controlId}-https`}>
          <Switch
            id={`${controlId}-https`}
            checked={settings.isHttps}
            onChange={(isHttps) => onConnectionChange({ ...settings, isHttps })}
            size="small"
          />
        </ConnectionFieldRow>
      </div>

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
            <ConnectionFieldRow
              label={t('databases.fields.excludeTables')}
              controlId={`${controlId}-exclude-tables`}
            >
              <div className="flex w-full items-center">
                <Select
                  id={`${controlId}-exclude-tables`}
                  mode="tags"
                  value={settings.excludeTables || []}
                  onChange={(values) =>
                    onNonConnectionChange({ ...settings, excludeTables: normalizeNameList(values) })
                  }
                  size="small"
                  className="w-full max-w-[200px]"
                  placeholder={t('databases.edit.excludeTablesPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />

                <ConnectionFieldTooltip title={t('databases.edit.excludeTablesTooltip')} />
              </div>
            </ConnectionFieldRow>
          )}

          {settings.type === DatabaseType.MARIADB && !isRestore && (
            <ConnectionFieldRow
              label={t('databases.fields.excludeEvents')}
              controlId={`${controlId}-exclude-events`}
            >
              <div className="flex items-center">
                <Checkbox
                  id={`${controlId}-exclude-events`}
                  checked={settings.isExcludeEvents || false}
                  onChange={(e) =>
                    onNonConnectionChange({ ...settings, isExcludeEvents: e.target.checked })
                  }
                >
                  {t('databases.edit.mariadb.skipEvents')}
                </Checkbox>

                <ConnectionFieldTooltip title={t('databases.edit.mariadb.skipEventsTooltip')} />
              </div>
            </ConnectionFieldRow>
          )}

          {settings.type === DatabaseType.MARIADB && (
            <ConnectionFieldRow
              label={t('databases.fields.galeraReplication')}
              controlId={`${controlId}-galera`}
            >
              <div className="flex items-center">
                <Checkbox
                  id={`${controlId}-galera`}
                  checked={settings.isSkipGaleraDisable || false}
                  onChange={(e) =>
                    onNonConnectionChange({ ...settings, isSkipGaleraDisable: e.target.checked })
                  }
                >
                  {t('databases.edit.mariadb.skipGaleraDisable')}
                </Checkbox>

                <ConnectionFieldTooltip
                  title={t('databases.edit.mariadb.skipGaleraDisableTooltip')}
                />
              </div>
            </ConnectionFieldRow>
          )}
        </>
      )}
    </>
  );
};
