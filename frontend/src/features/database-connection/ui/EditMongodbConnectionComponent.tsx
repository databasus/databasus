import { Input, InputNumber, Select, Switch } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  DatabaseType,
  MongodbConnectionStringParser,
  type MongodbSettings,
  disableSrvWhenTunneled,
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

const hasAdvancedValues = (settings: MongodbSettings, isRestore: boolean): boolean =>
  !!settings.authDatabase ||
  !!settings.isSrv ||
  !!settings.isDirectConnection ||
  !!settings.sshTunnel?.isEnabled ||
  (!isRestore && !!settings.excludeCollections?.length);

export const EditMongodbConnectionComponent = ({
  scenario,
  settings,
  savedSshTunnel,
  isAddressChanged,
  onConnectionChange,
  onNonConnectionChange,
}: ConnectionEditorProps<MongodbSettings>) => {
  const { t } = useTranslation();
  const controlId = useId();
  const isRestore = scenario === 'restore';
  const [isShowAdvanced, setShowAdvanced] = useState(() => hasAdvancedValues(settings, isRestore));

  const engineName = t(DATABASE_TYPE_LABEL_KEYS[DatabaseType.MONGODB]);
  const isTunnelEnabled = !!settings.sshTunnel?.isEnabled;

  return (
    <>
      <ConnectionStringImport
        parse={(connectionString) => MongodbConnectionStringParser.parse(connectionString)}
        onParsed={(parsed) => {
          onConnectionChange(
            disableSrvWhenTunneled({
              ...settings,
              host: parsed.host,
              port: parsed.port,
              username: parsed.username,
              password: parsed.password || '',
              database: parsed.database,
              authDatabase: parsed.authDatabase,
              isHttps: parsed.useTls,
              isSrv: parsed.isSrv,
              isDirectConnection: parsed.isDirectConnection,
              cpuCount: 1,
            }),
          );

          if (parsed.isSrv || parsed.isDirectConnection) setShowAdvanced(true);
        }}
      />

      <ConnectionCredentialFields
        controlId={controlId}
        engineName={engineName}
        values={settings}
        isAddressChanged={isAddressChanged}
        isPortShown={!settings.isSrv}
        // eslint-disable-next-line i18next/no-literal-string -- MongoDB's default port
        portPlaceholder="27017"
        hostHints={isLocalhostConnection(settings) && <ConnectionHostHint topic="localhost" />}
        onHostChange={(rawHost) =>
          onConnectionChange({ ...settings, host: stripHostScheme(rawHost) })
        }
        onValuesChange={(change) =>
          onConnectionChange({
            ...settings,
            ...change,
            database: change.database ?? settings.database,
          })
        }
      />

      <ConnectionFieldRow label={t('databases.fields.useHttps')} controlId={`${controlId}-https`}>
        <Switch
          id={`${controlId}-https`}
          checked={settings.isHttps}
          onChange={(isHttps) => onConnectionChange({ ...settings, isHttps })}
          size="small"
        />
      </ConnectionFieldRow>

      <div className="mb-5">
        <ConnectionFieldRow label={t('databases.fields.cpuCount')} controlId={`${controlId}-cpu`}>
          <div className="flex items-center">
            <InputNumber
              id={`${controlId}-cpu`}
              min={1}
              max={16}
              value={settings.cpuCount}
              onChange={(cpuCount) => onConnectionChange({ ...settings, cpuCount: cpuCount || 1 })}
              size="small"
              className="max-w-[200px]"
            />

            <ConnectionFieldTooltip title={t('databases.edit.cpuCountTooltip')} />
          </div>
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
            onChange={(sshTunnel) =>
              onConnectionChange(disableSrvWhenTunneled({ ...settings, sshTunnel }))
            }
          />

          <ConnectionFieldRow
            label={t('databases.fields.useSrvConnection')}
            controlId={`${controlId}-srv`}
          >
            <div className="flex items-center">
              <Switch
                id={`${controlId}-srv`}
                checked={settings.isSrv || false}
                disabled={isTunnelEnabled || undefined}
                onChange={(isSrv) => onConnectionChange({ ...settings, isSrv })}
                size="small"
              />

              <ConnectionFieldTooltip
                title={
                  isTunnelEnabled
                    ? t('databases.edit.mongodb.srvOverTunnelTooltip')
                    : t('databases.edit.mongodb.srvTooltip')
                }
              />
            </div>
          </ConnectionFieldRow>

          <ConnectionFieldRow
            label={t('databases.fields.directConnection')}
            controlId={`${controlId}-direct`}
          >
            <div className="flex items-center">
              {/* Shown as on but never written: the backend forces it per connection, so the
                  stored value still applies once the tunnel is switched off. */}
              <Switch
                id={`${controlId}-direct`}
                checked={isTunnelEnabled || settings.isDirectConnection || false}
                disabled={isTunnelEnabled || undefined}
                onChange={(isDirectConnection) =>
                  onConnectionChange({ ...settings, isDirectConnection })
                }
                size="small"
              />

              <ConnectionFieldTooltip
                title={
                  isTunnelEnabled
                    ? t('databases.edit.mongodb.directConnectionOverTunnelTooltip')
                    : t('databases.edit.mongodb.directConnectionTooltip')
                }
              />
            </div>
          </ConnectionFieldRow>

          <ConnectionFieldRow
            label={t('databases.fields.authDatabase')}
            controlId={`${controlId}-auth-database`}
          >
            <Input
              id={`${controlId}-auth-database`}
              value={settings.authDatabase}
              onChange={(e) =>
                onConnectionChange({ ...settings, authDatabase: e.target.value.trim() })
              }
              size="small"
              className="w-full max-w-[200px]"
              // eslint-disable-next-line i18next/no-literal-string -- MongoDB's default authentication database
              placeholder="admin"
            />
          </ConnectionFieldRow>

          {!isRestore && (
            <ConnectionFieldRow
              label={t('databases.fields.excludeCollections')}
              controlId={`${controlId}-exclude-collections`}
            >
              <div className="flex w-full items-center">
                <Select
                  id={`${controlId}-exclude-collections`}
                  mode="tags"
                  value={settings.excludeCollections || []}
                  onChange={(values) =>
                    onNonConnectionChange({
                      ...settings,
                      excludeCollections: normalizeNameList(values),
                    })
                  }
                  size="small"
                  className="w-full max-w-[200px]"
                  placeholder={t('databases.edit.mongodb.excludeCollectionsPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />

                <ConnectionFieldTooltip
                  title={t('databases.edit.mongodb.excludeCollectionsTooltip')}
                />
              </div>
            </ConnectionFieldRow>
          )}
        </>
      )}
    </>
  );
};
