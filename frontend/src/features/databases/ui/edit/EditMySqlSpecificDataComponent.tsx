import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  type Database,
  DatabaseType,
  type MysqlSettings,
  databaseApi,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '../../../../entities/databases';
import { MySqlConnectionStringParser } from '../../../../entities/databases/model/mysql/MySqlConnectionStringParser';
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
  settings: MysqlSettings;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void | Promise<void>;

  isShowDbName?: boolean;
  isRestoreMode?: boolean;
}

export const EditMySqlSpecificDataComponent = ({
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

  const [editingSettings, setEditingSettings] = useState<MysqlSettings>(settings);
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const hasAdvancedValues = !!settings.excludeTables?.length || !!settings.sshTunnel?.isEnabled;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error(t('databases.edit.clipboardEmpty'));
      return;
    }

    const result = MySqlConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(translateLocalizedText(result.error, t));
      return;
    }

    setEditingSettings({
      ...editingSettings,
      host: result.host,
      port: result.port,
      username: result.username,
      password: result.password,
      database: result.database,
      isHttps: result.isHttps,
    });
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
        onSaved(await databaseApi.updateDatabase(database.id, editedDatabase));
      } else {
        await onSaved(editedDatabase);
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsConnectionTested(false);
    setIsTestingConnection(false);
    setIsConnectionFailed(false);

    setEditingSettings(settings);
  }, [settings]);

  const engineName = t(DATABASE_TYPE_LABEL_KEYS[DatabaseType.MYSQL]);

  const hasStoredSshSecrets = hasStoredSshTunnelSecretsForAuthType(
    settings.sshTunnel,
    editingSettings.sshTunnel?.authType,
    database.id,
  );

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

  // Behind a bastion a loopback address names the database as the bastion sees it, so the hint
  // about reaching a local database would be wrong.
  const isTunnelEnabled = !!editingSettings.sshTunnel?.isEnabled;
  const isLocalhostDb =
    !isTunnelEnabled &&
    (editingSettings.host?.includes('localhost') || editingSettings.host?.includes('127.0.0.1'));

  return (
    <div>
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
            setEditingSettings({
              ...editingSettings,
              host: e.target.value.trim().replace('https://', '').replace('http://', ''),
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.edit.placeholders.host', { engine: engineName })}
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
          placeholder={t('databases.edit.placeholders.port', { engine: engineName })}
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
        <Input
          value={editingSettings.username}
          onChange={(e) => {
            setEditingSettings({ ...editingSettings, username: e.target.value.trim() });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.edit.placeholders.username', { engine: engineName })}
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
          placeholder={t('databases.edit.placeholders.password', { engine: engineName })}
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
            placeholder={t('databases.edit.placeholders.databaseName', { engine: engineName })}
          />
        </div>
      )}

      <div className="mb-3 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.useHttps')}</div>
        <Switch
          checked={editingSettings.isHttps}
          onChange={(checked) => {
            setEditingSettings({ ...editingSettings, isHttps: checked });
            setIsConnectionTested(false);
          }}
          size="small"
        />
      </div>

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

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTables')}</div>
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

            <Tooltip className="cursor-pointer" title={t('databases.edit.excludeTablesTooltip')}>
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </>
      )}

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
      </div>

      {isConnectionFailed && (
        <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
          {t('databases.edit.ipWhitelistHint')}
        </div>
      )}

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
