import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  STORAGE_TYPE_LABEL_KEYS,
  type Storage,
  type StorageSettings,
  StorageType,
  getStorageLogoFromType,
  storageApi,
} from '../../../../entities/storages';
import { translateApiError } from '../../../../shared/i18n';
import { ToastHelper } from '../../../../shared/toast';
import { createDefaultStorageSettings } from '../../lib/createDefaultStorageSettings';
import { isStorageSettingsFilled } from '../../lib/isStorageSettingsFilled';
import type { StorageDraft } from '../../lib/StorageDraft';
import { EditAzureBlobStorageComponent } from './storages/EditAzureBlobStorageComponent';
import { EditFTPStorageComponent } from './storages/EditFTPStorageComponent';
import { EditGoogleDriveStorageComponent } from './storages/EditGoogleDriveStorageComponent';
import { EditLocalStorageComponent } from './storages/EditLocalStorageComponent';
import { EditNASStorageComponent } from './storages/EditNASStorageComponent';
import { EditRcloneStorageComponent } from './storages/EditRcloneStorageComponent';
import { EditS3StorageComponent } from './storages/EditS3StorageComponent';
import { EditSFTPStorageComponent } from './storages/EditSFTPStorageComponent';

interface Props {
  workspaceId: string;

  isShowClose: boolean;
  onClose: () => void;

  isShowName: boolean;

  editingStorage?: Storage;
  // A form filled elsewhere, which the component starts from instead of editingStorage.
  draft?: StorageDraft;
  onChanged: (storage: Storage) => void;
}

const toDraft = (editingStorage?: Storage): StorageDraft =>
  editingStorage
    ? {
        storageId: editingStorage.id,
        name: editingStorage.name,
        settings: structuredClone(editingStorage.settings),
      }
    : { name: '', settings: createDefaultStorageSettings(StorageType.LOCAL) };

export function EditStorageComponent({
  workspaceId,
  isShowClose,
  onClose,
  isShowName,
  editingStorage,
  draft,
  onChanged,
}: Props) {
  const { t } = useTranslation();
  const [storageId, setStorageId] = useState<string | undefined>();
  const [name, setName] = useState('');
  const [settings, setSettings] = useState<StorageSettings | undefined>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isTestConnectionSuccess, setIsTestConnectionSuccess] = useState(false);
  const [connectionError, setConnectionError] = useState<string | undefined>();

  const changeSettings = (changedSettings: StorageSettings) => {
    setSettings(changedSettings);
    setIsUnsaved(true);
    setIsTestConnectionSuccess(false);
    setConnectionError(undefined);
  };

  const save = async () => {
    if (!settings) return;

    setIsSaving(true);

    try {
      const savedStorage = storageId
        ? await storageApi.updateStorage(storageId, name, settings)
        : await storageApi.createStorage(workspaceId, name, settings);
      onChanged(savedStorage);
      setIsUnsaved(false);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const testConnection = async () => {
    if (!settings) return;

    setIsTestingConnection(true);
    setConnectionError(undefined);

    try {
      const testResult = await storageApi.testStorageSettings(workspaceId, settings, storageId);

      if (testResult.isConnected) {
        setIsTestConnectionSuccess(true);
        ToastHelper.showToast({
          title: t('storages.connectionTestSucceeded.title'),
          description: t('storages.connectionTestSucceeded.description'),
        });
      } else {
        setConnectionError(testResult.error);
        alert(t('storages.connectionTestFailed', { error: testResult.error }));
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsTestingConnection(false);
  };

  useEffect(() => {
    const initialDraft = draft ?? toDraft(editingStorage);

    setStorageId(initialDraft.storageId);
    setName(initialDraft.name);
    setSettings(initialDraft.settings);
    setIsUnsaved(true);
    setIsTestConnectionSuccess(false);
    setConnectionError(undefined);
  }, [editingStorage, draft]);

  if (!settings) return <div />;

  const isSaved = storageId !== undefined;
  const isAllDataFilled = !!name && isStorageSettingsFilled(settings, isSaved);

  const storageTypeOptions = Object.values(StorageType).map((storageType) => ({
    label: t(STORAGE_TYPE_LABEL_KEYS[storageType]),
    value: storageType,
  }));

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.name')}</div>

          <Input
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.namePlaceholder')}
          />
        </div>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.type')}</div>

        <div className="flex items-center">
          {isSaved ? (
            t(STORAGE_TYPE_LABEL_KEYS[settings.type])
          ) : (
            <Select
              value={settings.type}
              options={storageTypeOptions}
              onChange={(type: StorageType) => changeSettings(createDefaultStorageSettings(type))}
              size="small"
              className="w-[250px] max-w-[250px]"
            />
          )}

          <img src={getStorageLogoFromType(settings.type)} className="ml-2 h-4 w-4" />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {settings.type === StorageType.S3 && (
          <EditS3StorageComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
            connectionError={connectionError}
          />
        )}

        {settings.type === StorageType.GOOGLE_DRIVE && (
          <EditGoogleDriveStorageComponent
            settings={settings}
            onChange={changeSettings}
            oauthForm={{ workspaceId, storageId, name }}
          />
        )}

        {settings.type === StorageType.NAS && (
          <EditNASStorageComponent settings={settings} onChange={changeSettings} />
        )}

        {settings.type === StorageType.AZURE_BLOB && (
          <EditAzureBlobStorageComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === StorageType.FTP && (
          <EditFTPStorageComponent settings={settings} onChange={changeSettings} />
        )}

        {settings.type === StorageType.SFTP && (
          <EditSFTPStorageComponent settings={settings} onChange={changeSettings} />
        )}

        {settings.type === StorageType.RCLONE && (
          <EditRcloneStorageComponent
            settings={settings}
            isSaved={isSaved}
            onChange={changeSettings}
          />
        )}

        {settings.type === StorageType.LOCAL && <EditLocalStorageComponent />}
      </div>

      <div className="mt-3 flex">
        {isUnsaved && !isTestConnectionSuccess ? (
          <Button
            className="mr-1"
            disabled={isTestingConnection || !isAllDataFilled}
            loading={isTestingConnection}
            type="primary"
            onClick={testConnection}
          >
            {t('storages.actions.testConnection')}
          </Button>
        ) : (
          <div />
        )}

        {isUnsaved && isTestConnectionSuccess ? (
          <Button
            className="mr-1"
            disabled={isSaving || !isAllDataFilled}
            loading={isSaving}
            type="primary"
            onClick={save}
          >
            {t('common.actions.save')}
          </Button>
        ) : (
          <div />
        )}

        {isShowClose ? (
          <Button
            className="mr-1"
            disabled={isSaving}
            type="primary"
            danger
            ghost
            onClick={onClose}
          >
            {t('common.actions.cancel')}
          </Button>
        ) : (
          <div />
        )}
      </div>
    </div>
  );
}
