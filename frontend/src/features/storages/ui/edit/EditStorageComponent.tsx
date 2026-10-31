import { Button, Input, Select } from 'antd';
import { useEffect, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  STORAGE_TYPE_LABEL_KEYS,
  type Storage,
  type StorageSettings,
  StorageType,
  getStorageLogoFromType,
  storageApi,
} from '../../../../entities/storages';
import { getSessionSnapshot, isCurrentSession, subscribeSession } from '../../../../shared/auth';
import { translateApiError } from '../../../../shared/i18n';
import { RequestRevision } from '../../../../shared/lib/RequestRevision';
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
  const [savedSettings, setSavedSettings] = useState<StorageSettings | undefined>();
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isTestConnectionSuccess, setIsTestConnectionSuccess] = useState(false);
  const [connectionError, setConnectionError] = useState<string | undefined>();

  const invalidateConnectionTest = () => {
    connectionTestRevision.current.Invalidate();
    setIsTestingConnection(false);
    setIsTestConnectionSuccess(false);
    setConnectionError(undefined);
  };

  const invalidatePendingRequests = () => {
    invalidateConnectionTest();
    saveRevision.current.Invalidate();
    setIsSaving(false);
  };

  const changeSettings = (changedSettings: StorageSettings) => {
    invalidateConnectionTest();
    setSettings(changedSettings);
    setIsUnsaved(true);
  };

  const save = async () => {
    if (!settings) return;

    const revision = saveRevision.current.Invalidate();
    const session = getSessionSnapshot();
    const isCurrentSave = () =>
      saveRevision.current.IsCurrent(revision) && isCurrentSession(session);
    setIsSaving(true);

    try {
      const savedStorage = storageId
        ? await storageApi.updateStorage(storageId, name, settings)
        : await storageApi.createStorage(workspaceId, name, settings);

      if (!isCurrentSave()) return;

      setStorageId(savedStorage.id);
      setSavedSettings(savedStorage.settings);
      const currentSettings = currentConnectionContext.current.settings;

      if (settings.type === StorageType.SFTP && currentSettings?.type === StorageType.SFTP) {
        invalidateConnectionTest();
        setSettings({
          ...currentSettings,
          password:
            currentSettings.password === settings.password ? undefined : currentSettings.password,
          privateKey:
            currentSettings.privateKey === settings.privateKey
              ? undefined
              : currentSettings.privateKey,
        });
      }

      onChanged(savedStorage);
      setIsUnsaved(currentSettings !== settings || currentConnectionContext.current.name !== name);
    } catch (e) {
      if (isCurrentSave()) alert(translateApiError(e, t));
    } finally {
      if (isCurrentSave()) setIsSaving(false);
    }
  };

  const testConnection = async () => {
    if (!settings) return;

    const revision = connectionTestRevision.current.Invalidate();
    const session = getSessionSnapshot();
    const isCurrentTest = () =>
      connectionTestRevision.current.IsCurrent(revision) &&
      isCurrentSession(session) &&
      currentConnectionContext.current.workspaceId === workspaceId &&
      currentConnectionContext.current.storageId === storageId &&
      currentConnectionContext.current.settings === settings;

    setIsTestingConnection(true);
    setIsTestConnectionSuccess(false);
    setConnectionError(undefined);

    try {
      const testResult = await storageApi.testStorageSettings(workspaceId, settings, storageId);
      if (!isCurrentTest()) return;

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
      if (!isCurrentTest()) return;
      alert(translateApiError(e, t));
    }

    if (isCurrentTest()) setIsTestingConnection(false);
  };

  const controlId = useId();
  const connectionTestRevision = useRef(new RequestRevision());
  const saveRevision = useRef(new RequestRevision());
  const currentConnectionContext = useRef({ workspaceId, storageId, settings, name });
  currentConnectionContext.current = { workspaceId, storageId, settings, name };

  useEffect(() => subscribeSession(invalidatePendingRequests), []);

  useEffect(() => {
    const initialDraft = draft ?? toDraft(editingStorage);
    invalidatePendingRequests();

    setStorageId(initialDraft.storageId);
    setName(initialDraft.name);
    setSettings(initialDraft.settings);
    setSavedSettings(editingStorage?.settings);
    setIsUnsaved(true);
    return () => {
      connectionTestRevision.current.Invalidate();
      saveRevision.current.Invalidate();
    };
  }, [editingStorage, draft, workspaceId]);

  if (!settings) return <div />;

  const isSaved = storageId !== undefined;
  const isAllDataFilled = !!name && isStorageSettingsFilled(settings, isSaved, savedSettings);

  const storageTypeOptions = Object.values(StorageType).map((storageType) => ({
    label: t(STORAGE_TYPE_LABEL_KEYS[storageType]),
    value: storageType,
  }));

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label htmlFor={`${controlId}-name`} className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
            {t('common.fields.name')}
          </label>

          <Input
            id={`${controlId}-name`}
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
        <label
          htmlFor={isSaved ? undefined : `${controlId}-type`}
          className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2"
        >
          {t('common.fields.type')}
        </label>

        <div className="flex items-center">
          {isSaved ? (
            t(STORAGE_TYPE_LABEL_KEYS[settings.type])
          ) : (
            <Select
              id={`${controlId}-type`}
              value={settings.type}
              options={storageTypeOptions}
              onChange={(type: StorageType) => changeSettings(createDefaultStorageSettings(type))}
              size="small"
              className="w-[250px] max-w-[250px]"
            />
          )}

          <img alt="" src={getStorageLogoFromType(settings.type)} className="ml-2 h-4 w-4" />
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
            workspaceId={workspaceId}
            storageId={storageId}
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
          <EditSFTPStorageComponent
            settings={settings}
            savedAuthMethods={
              savedSettings?.type === StorageType.SFTP ? savedSettings.authMethods : []
            }
            onChange={changeSettings}
          />
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
