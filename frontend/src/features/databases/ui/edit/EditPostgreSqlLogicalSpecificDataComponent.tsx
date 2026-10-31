import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Checkbox, Input, InputNumber, Select, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type Database,
  POSTGRESQL_SHORT_NAME,
  POSTGRES_SSL_MODE_LABEL_KEYS,
  PostgresSslMode,
  type PostgresqlLogicalSettings,
  databaseApi,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '../../../../entities/databases';
import { ConnectionStringParser } from '../../../../entities/databases/model/postgresql/ConnectionStringParser';
import type { PostgresqlRestoreOptions } from '../../../../entities/restores';
import {
  getWebsitePageUrl,
  translateApiError,
  translateLocalizedText,
  useLocale,
} from '../../../../shared/i18n';
import { NAME_LIST_TOKEN_SEPARATORS, normalizeNameList } from '../../../../shared/lib';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { ToastHelper } from '../../../../shared/toast';
import {
  AdvancedSettingsToggleComponent,
  ClipboardPasteModalComponent,
} from '../../../../shared/ui';
import { EditSshTunnelComponent } from './EditSshTunnelComponent';
import { toDatabaseSettingsTarget } from './toDatabaseSettingsTarget';

interface Props {
  database: Database;
  settings: PostgresqlLogicalSettings;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  // The restore options are only shown, and only meaningful, in restore mode.
  onSaved: (database: Database, restoreOptions: PostgresqlRestoreOptions) => void | Promise<void>;

  isShowDbName?: boolean;
  isRestoreMode?: boolean;
}

const DEFAULT_RESTORE_OPTIONS: PostgresqlRestoreOptions = {
  isExcludeExtensions: false,
  isRestoreOwnership: false,
  isRestorePrivileges: false,
};

const IPV4_PATTERN = /^\d{1,3}(\.\d{1,3}){3}$/;

const deriveSslModeFromHost = (rawHost: string): PostgresSslMode | null => {
  const trimmed = rawHost.trim().toLowerCase();

  if (trimmed.startsWith('https://')) return PostgresSslMode.Require;
  if (trimmed.startsWith('http://')) return PostgresSslMode.Disable;

  const bareHost = trimmed.split(':')[0];
  if (bareHost === 'localhost' || IPV4_PATTERN.test(bareHost)) {
    return PostgresSslMode.Disable;
  }

  return null;
};

const applySslMode = (
  postgresqlLogical: PostgresqlLogicalSettings,
  sslMode: PostgresSslMode,
): PostgresqlLogicalSettings => {
  if (sslMode === PostgresSslMode.Disable) {
    return { ...postgresqlLogical, sslMode, sslClientCert: '', sslClientKey: '', sslRootCert: '' };
  }

  return { ...postgresqlLogical, sslMode };
};

export const EditPostgreSqlLogicalSpecificDataComponent = ({
  database,
  settings,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
  isRestoreMode = false,
}: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();

  const [editingSettings, setEditingSettings] = useState<PostgresqlLogicalSettings>(settings);
  const [restoreOptions, setRestoreOptions] =
    useState<PostgresqlRestoreOptions>(DEFAULT_RESTORE_OPTIONS);
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const hasAdvancedValues =
    !!settings.sslClientCert ||
    !!settings.sslRootCert ||
    !!settings.sshTunnel?.isEnabled ||
    (!isRestoreMode &&
      (!!settings.includeSchemas?.length ||
        !!settings.excludeTables?.length ||
        !!settings.excludeTableData?.length ||
        !!settings.isSkipUserMappings));
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [hasAutoAddedPublicSchema, setHasAutoAddedPublicSchema] = useState(false);
  const [hasUserChosenSslMode, setHasUserChosenSslMode] = useState(!!database.id);
  const [isReplacingCerts, setIsReplacingCerts] = useState(false);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error(t('databases.edit.clipboardEmpty'));
      return;
    }

    const result = ConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(translateLocalizedText(result.error, t));
      return;
    }

    const updatedSettings: PostgresqlLogicalSettings = {
      ...editingSettings,
      host: result.host,
      port: result.port,
      username: result.username,
      password: result.password,
      database: result.database,
      sslMode: result.sslMode,
      cpuCount: 1,
    };

    setHasUserChosenSslMode(true);
    setEditingSettings(autoAddPublicSchemaForSupabase(updatedSettings));
    setIsConnectionTested(false);
    message.success(t('databases.edit.connectionStringParsed'));
  };

  const parseFromClipboard = async () => {
    if (!ClipboardHelper.isClipboardApiAvailable()) {
      setIsShowPasteModal(true);
      return;
    }

    try {
      const text = await ClipboardHelper.readFromClipboard();
      applyConnectionString(text);
    } catch {
      message.error(t('databases.edit.clipboardReadFailed'));
    }
  };

  const autoAddPublicSchemaForSupabase = (
    updatedSettings: PostgresqlLogicalSettings,
  ): PostgresqlLogicalSettings => {
    if (hasAutoAddedPublicSchema) return updatedSettings;

    const isSupabase =
      updatedSettings.host.includes('supabase') || updatedSettings.username.includes('supabase');

    if (isSupabase) {
      setHasAutoAddedPublicSchema(true);

      const currentSchemas = updatedSettings.includeSchemas;
      if (!currentSchemas.includes('public')) {
        return {
          ...updatedSettings,
          // eslint-disable-next-line i18next/no-literal-string -- PostgreSQL schema name
          includeSchemas: ['public', ...currentSchemas],
        };
      }
    }

    return updatedSettings;
  };

  const testConnection = async () => {
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    try {
      const testResult = await databaseApi.testDatabaseSettings(
        toDatabaseSettingsTarget(database, editingSettings),
      );

      if (testResult.isConnected) {
        setIsConnectionTested(true);
        ToastHelper.showToast({
          title: t('databases.edit.connectionTestPassed.title'),
          description: t('databases.edit.connectionTestPassed.description'),
        });
      } else {
        setIsConnectionFailed(true);
        alert(t('databases.edit.connectionTestFailed', { error: testResult.error }));
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsTestingConnection(false);
  };

  const saveDatabase = async () => {
    if (isSaving) return;

    const editedDatabase: Database = { ...database, settings: editingSettings };

    setIsSaving(true);

    try {
      if (isSaveToApi) {
        onSaved(await databaseApi.updateDatabase(database.id, editedDatabase), restoreOptions);
      } else {
        await onSaved(editedDatabase, restoreOptions);
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const updatePostgresqlCert = (
    field: 'sslClientCert' | 'sslClientKey' | 'sslRootCert',
    value: string,
  ) => {
    setEditingSettings({ ...editingSettings, [field]: value });
    setIsConnectionTested(false);
  };

  const startReplacingCerts = () => {
    setIsReplacingCerts(true);
    setEditingSettings({
      ...editingSettings,
      sslClientCert: '',
      sslClientKey: '',
      sslRootCert: '',
    });
    setIsConnectionTested(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsConnectionTested(false);
    setIsTestingConnection(false);
    setIsConnectionFailed(false);
    setIsReplacingCerts(false);
    setHasUserChosenSslMode(!!database.id);

    setEditingSettings(settings);
  }, [settings]);

  const renderFooter = (footerContent?: React.ReactNode) => (
    <div className="mt-5 flex">
      {isShowCancelButton && (
        <Button className="mr-1" danger ghost onClick={() => onCancel()}>
          {t('common.actions.cancel')}
        </Button>
      )}

      {isShowBackButton && (
        <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
          {t('common.actions.back')}
        </Button>
      )}

      {footerContent}
    </div>
  );

  const renderSslCertSection = () => {
    const sslMode = editingSettings.sslMode ?? PostgresSslMode.Disable;
    if (sslMode === PostgresSslMode.Disable) return null;

    const hadSslCert = !!settings.sslClientCert;
    if (hadSslCert && !isReplacingCerts) {
      return (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientCertificate')}</div>
          <div className="flex items-center">
            <span className="mr-3">*************</span>
            <Button size="small" onClick={startReplacingCerts}>
              {t('databases.actions.replace')}
            </Button>
          </div>
        </div>
      );
    }

    return (
      <>
        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientCertificate')}</div>
          <Input.TextArea
            value={editingSettings.sslClientCert || ''}
            onChange={(e) => updatePostgresqlCert('sslClientCert', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            // eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
            placeholder="-----BEGIN CERTIFICATE-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientKey')}</div>
          <Input.TextArea
            value={editingSettings.sslClientKey || ''}
            onChange={(e) => updatePostgresqlCert('sslClientKey', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            // eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
            placeholder="-----BEGIN PRIVATE KEY-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="flex min-w-[150px] items-center pr-2">
            <span>{t('databases.fields.serverCaCertificate')}</span>
            <Tooltip
              className="cursor-pointer"
              title={t('databases.edit.serverCaCertificateTooltip')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
          <Input.TextArea
            value={editingSettings.sslRootCert || ''}
            onChange={(e) => updatePostgresqlCert('sslRootCert', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            // eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
            placeholder="-----BEGIN CERTIFICATE-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>
      </>
    );
  };

  const renderPgDumpForm = () => {
    // A restore start connects to the target itself, so restore mode needs no separate test.
    const isSubmitShown = isConnectionTested || isRestoreMode;

    let isAllFieldsFilled = true;
    if (!editingSettings.host) isAllFieldsFilled = false;
    if (!editingSettings.port) isAllFieldsFilled = false;
    if (!editingSettings.username) isAllFieldsFilled = false;
    if (!database.id && !editingSettings.password) isAllFieldsFilled = false;
    if (!editingSettings.database) isAllFieldsFilled = false;
    if (!isSshTunnelReadyToTest(editingSettings.sshTunnel, hasStoredSshSecrets))
      isAllFieldsFilled = false;

    const isTunnelEnabled = !!editingSettings.sshTunnel?.isEnabled;

    // Behind a tunnel a loopback address means "on the bastion", which is the documented setup
    // rather than the mistake this hint warns about.
    const isLocalhostDb =
      !isTunnelEnabled &&
      (editingSettings.host?.includes('localhost') || editingSettings.host?.includes('127.0.0.1'));

    const isSupabaseDb =
      editingSettings.host?.includes('supabase') || editingSettings.username?.includes('supabase');

    return (
      <>
        <div className="mb-3 flex">
          <div className="min-w-[150px] pr-2" />
          <div
            className="cursor-pointer text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
            onClick={parseFromClipboard}
          >
            <CopyOutlined className="mr-1" />
            {t('databases.edit.parseFromClipboard')}
          </div>
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.host')}</div>
          <Input
            value={editingSettings.host}
            onChange={(e) => {
              const rawHost = e.target.value;
              const basePostgresql = {
                ...editingSettings,
                host: rawHost.trim().replace('https://', '').replace('http://', ''),
              };
              const isHttpsHost = rawHost.trim().toLowerCase().startsWith('https://');
              const currentSslMode = basePostgresql.sslMode ?? PostgresSslMode.Disable;

              let derivedSslMode: PostgresSslMode | null = null;
              if (hasUserChosenSslMode) {
                if (isHttpsHost && currentSslMode === PostgresSslMode.Disable) {
                  derivedSslMode = PostgresSslMode.Require;
                }
              } else {
                derivedSslMode = deriveSslModeFromHost(rawHost);
              }

              const updatedSettings =
                derivedSslMode !== null
                  ? applySslMode(basePostgresql, derivedSslMode)
                  : basePostgresql;
              setEditingSettings(autoAddPublicSchemaForSupabase(updatedSettings));
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder={t('databases.edit.placeholders.host', { engine: POSTGRESQL_SHORT_NAME })}
          />
        </div>

        {isLocalhostDb && (
          <div className="mb-1 flex">
            <div className="min-w-[150px] pr-2" />
            <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
              <Trans
                i18nKey="databases.edit.localhostHint"
                components={{
                  docsLink: (
                    <a
                      href={getWebsitePageUrl('faqLocalhost', locale)}
                      target="_blank"
                      rel="noreferrer"
                      className="!text-blue-600 dark:!text-blue-400"
                    />
                  ),
                }}
              />
            </div>
          </div>
        )}

        {isSupabaseDb && (
          <div className="mb-1 flex">
            <div className="min-w-[150px] pr-2" />
            <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
              <Trans
                i18nKey="databases.edit.postgresql.supabaseHint"
                components={{
                  docsLink: (
                    <a
                      href={getWebsitePageUrl('faqSupabase', locale)}
                      target="_blank"
                      rel="noreferrer"
                      className="!text-blue-600 dark:!text-blue-400"
                    />
                  ),
                }}
              />
            </div>
          </div>
        )}

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
          <InputNumber
            type="number"
            value={editingSettings.port}
            onChange={(e) => {
              if (e === null) return;

              setEditingSettings({ ...editingSettings, port: e });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder={t('databases.edit.placeholders.port', { engine: POSTGRESQL_SHORT_NAME })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
          <Input
            value={editingSettings.username}
            onChange={(e) => {
              setEditingSettings(
                autoAddPublicSchemaForSupabase({
                  ...editingSettings,
                  username: e.target.value.trim(),
                }),
              );
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder={t('databases.edit.placeholders.username', {
              engine: POSTGRESQL_SHORT_NAME,
            })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.password')}</div>
          <Input.Password
            value={editingSettings.password}
            onChange={(e) => {
              setEditingSettings({ ...editingSettings, password: e.target.value });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder={t('databases.edit.placeholders.password', {
              engine: POSTGRESQL_SHORT_NAME,
            })}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
        </div>

        {isShowDbName && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.databaseName')}</div>
            <Input
              value={editingSettings.database}
              onChange={(e) => {
                setEditingSettings({ ...editingSettings, database: e.target.value.trim() });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('databases.edit.placeholders.databaseName', {
                engine: POSTGRESQL_SHORT_NAME,
              })}
            />
          </div>
        )}

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.sslMode')}</div>
          <Select
            value={editingSettings.sslMode ?? PostgresSslMode.Disable}
            onChange={(value: PostgresSslMode) => {
              setHasUserChosenSslMode(true);
              setEditingSettings(applySslMode(editingSettings, value));
              setIsConnectionTested(false);
            }}
            options={Object.values(PostgresSslMode).map((sslMode) => ({
              label: t(POSTGRES_SSL_MODE_LABEL_KEYS[sslMode]),
              value: sslMode,
            }))}
            size="small"
            className="max-w-[200px] grow"
          />
        </div>

        {isRestoreMode && (
          <div className="mb-5 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.cpuCount')}</div>
            <div className="flex items-center">
              <InputNumber
                min={1}
                max={128}
                value={editingSettings.cpuCount}
                onChange={(value) => {
                  setEditingSettings({ ...editingSettings, cpuCount: value || 1 });
                  setIsConnectionTested(false);
                }}
                size="small"
                className="max-w-[75px] grow"
              />

              <Tooltip className="cursor-pointer" title={t('databases.edit.cpuCountTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        )}

        <AdvancedSettingsToggleComponent
          label={t('databases.edit.advancedSettings')}
          isShowAdvanced={isShowAdvanced}
          onToggle={() => setShowAdvanced(!isShowAdvanced)}
        />

        {isShowAdvanced && (
          <>
            <EditSshTunnelComponent
              sshTunnel={editingSettings.sshTunnel}
              hasStoredSecrets={hasStoredSshSecrets}
              onChange={(sshTunnel) => {
                setEditingSettings({ ...editingSettings, sshTunnel });
                setIsConnectionTested(false);
              }}
            />

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('databases.fields.includeSchemas')}</div>
                <Select
                  mode="tags"
                  value={editingSettings.includeSchemas || []}
                  onChange={(values) => {
                    setEditingSettings({
                      ...editingSettings,
                      includeSchemas: normalizeNameList(values),
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder={t('databases.edit.postgresql.includeSchemasPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('databases.fields.excludeWholeTables')}</div>
                <Select
                  mode="tags"
                  value={editingSettings.excludeTables || []}
                  onChange={(values) => {
                    setEditingSettings({
                      ...editingSettings,
                      excludeTables: normalizeNameList(values),
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder={t('databases.edit.excludeTablesPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('databases.edit.postgresql.excludeTablesTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTableData')}</div>
                <Select
                  mode="tags"
                  value={editingSettings.excludeTableData || []}
                  onChange={(values) => {
                    setEditingSettings({
                      ...editingSettings,
                      excludeTableData: normalizeNameList(values),
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder={t('databases.edit.postgresql.excludeTableDataPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('databases.edit.postgresql.excludeTableDataTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.skipUserMappings')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.skipUserMappingsTooltip')}
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={editingSettings.isSkipUserMappings || false}
                  onChange={(e) => {
                    setEditingSettings({
                      ...editingSettings,
                      isSkipUserMappings: e.target.checked,
                    });
                  }}
                />
              </div>
            )}

            {isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.excludeExtensions')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.excludeExtensionsTooltip')}
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={restoreOptions.isExcludeExtensions}
                  onChange={(e) => {
                    setRestoreOptions({ ...restoreOptions, isExcludeExtensions: e.target.checked });
                  }}
                />
              </div>
            )}

            {isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.restoreOwnership')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.restoreOwnershipTooltip')}
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={restoreOptions.isRestoreOwnership}
                  onChange={(e) => {
                    setRestoreOptions({ ...restoreOptions, isRestoreOwnership: e.target.checked });
                  }}
                />
              </div>
            )}

            {isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.restorePrivileges')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.restorePrivilegesTooltip')}
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={restoreOptions.isRestorePrivileges}
                  onChange={(e) => {
                    setRestoreOptions({ ...restoreOptions, isRestorePrivileges: e.target.checked });
                  }}
                />
              </div>
            )}

            {renderSslCertSection()}
          </>
        )}

        {renderFooter(
          <>
            {!isSubmitShown && (
              <Button
                type="primary"
                onClick={() => testConnection()}
                loading={isTestingConnection}
                disabled={!isAllFieldsFilled}
                className="mr-5"
              >
                {t('databases.actions.testConnection')}
              </Button>
            )}

            {isSubmitShown && (
              <Button
                type="primary"
                onClick={() => saveDatabase()}
                loading={isSaving}
                disabled={!isAllFieldsFilled}
                className="mr-5"
              >
                {saveButtonText || t('common.actions.save')}
              </Button>
            )}
          </>,
        )}

        {isConnectionFailed && (
          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            {t('databases.edit.ipWhitelistHint')}
          </div>
        )}
      </>
    );
  };

  const hasStoredSshSecrets = hasStoredSshTunnelSecretsForAuthType(
    settings.sshTunnel,
    editingSettings.sshTunnel?.authType,
    database.id,
  );

  return (
    <div>
      {renderPgDumpForm()}

      <ClipboardPasteModalComponent
        open={isShowPasteModal}
        onSubmit={(text) => {
          setIsShowPasteModal(false);
          applyConnectionString(text);
        }}
        onCancel={() => setIsShowPasteModal(false)}
      />
    </div>
  );
};
