import { Alert, Button, Input, Select, notification } from 'antd';
import { useEffect, useId } from 'react';
import { useTranslation } from 'react-i18next';

import {
  STORAGE_TYPE_LABEL_KEYS,
  type Storage,
  StorageType,
  getStorageLogoFromType,
} from '../../../../entities/storages';
import { translateApiError } from '../../../../shared/i18n';
import { WRAPPING_SELECT_CLASS_NAME } from '../../../../shared/ui';
import { createDefaultStorageSettings } from '../../lib/createDefaultStorageSettings';
import { isStorageAddressChanged } from '../../lib/isStorageAddressChanged';
import { isStorageSettingsFilled } from '../../lib/isStorageSettingsFilled';
import type { StorageDraft } from '../../lib/StorageDraft';
import { useStorageDraft } from '../../model/useStorageDraft';
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
  onPersisted?: (storage: Storage) => void;
}

export function EditStorageComponent({
  workspaceId,
  isShowClose,
  onClose,
  isShowName,
  editingStorage,
  draft,
  onChanged,
  onPersisted,
}: Props) {
  const { t } = useTranslation();
  const controlId = useId();
  const noticeId = useId();
  const [notices, noticeHolder] = notification.useNotification();
  const form = useStorageDraft({
    workspaceId,
    editingStorage,
    draft,
    onChanged,
    onPersisted,
    onTestSucceeded: () =>
      notices.success({
        key: noticeId,
        message: t('storages.connectionTestSucceeded.title'),
        description: t('storages.connectionTestSucceeded.description'),
        duration: 4,
        placement: 'bottomRight',
      }),
  });

  useEffect(
    () => () => notices.destroy(noticeId),
    [notices, noticeId, workspaceId, editingStorage?.id],
  );

  const {
    name,
    settings,
    storageId,
    savedSettings,
    isSaving,
    isUnsaved,
    changeSettings,
    save,
    testConnection,
  } = form;

  const isTestingConnection = form.isTesting;
  const isTestConnectionSuccess = form.isTested;
  const connectionError = form.error?.kind === 'connection' ? form.error.diagnostic : undefined;
  const errorText =
    form.error?.kind === 'api'
      ? translateApiError(form.error.error, t)
      : form.error?.kind === 'connection'
        ? t('storages.connectionTestFailed', { error: form.error.diagnostic })
        : undefined;

  const isSaved = storageId !== undefined;
  const isAddressChanged = isStorageAddressChanged(settings, savedSettings);
  const isAllDataFilled = !!name && isStorageSettingsFilled(settings, isSaved, savedSettings);

  const storageTypeOptions = Object.values(StorageType).map((storageType) => ({
    label: t(STORAGE_TYPE_LABEL_KEYS[storageType]),
    value: storageType,
  }));

  return (
    <section
      aria-label={t('storages.config.settingsTitle')}
      className="min-w-0 [&_.ant-btn]:!h-auto [&_.ant-btn]:min-h-8 [&_.ant-btn]:leading-normal [&_.ant-btn]:!whitespace-normal"
    >
      {noticeHolder}
      {errorText && <Alert type="error" showIcon message={errorText} />}
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label htmlFor={`${controlId}-name`} className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
            {t('common.fields.name')}
          </label>

          <Input
            id={`${controlId}-name`}
            value={name}
            onChange={(e) => {
              form.changeName(e.target.value);
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

        <div className="flex w-full min-w-0 items-center">
          {isSaved ? (
            t(STORAGE_TYPE_LABEL_KEYS[settings.type])
          ) : (
            <Select
              id={`${controlId}-type`}
              value={settings.type}
              disabled={isSaving}
              options={storageTypeOptions}
              onChange={(type: StorageType) => changeSettings(createDefaultStorageSettings(type))}
              size="small"
              className={`w-full max-w-[250px] min-w-0 ${WRAPPING_SELECT_CLASS_NAME}`}
            />
          )}

          <img
            alt=""
            src={getStorageLogoFromType(settings.type)}
            className="ml-2 h-4 w-4 shrink-0"
          />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {settings.type === StorageType.S3 && (
          <EditS3StorageComponent
            settings={settings}
            isSaved={isSaved}
            isAddressChanged={isAddressChanged}
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
          <EditNASStorageComponent
            settings={settings}
            isAddressChanged={isAddressChanged}
            onChange={changeSettings}
          />
        )}

        {settings.type === StorageType.AZURE_BLOB && (
          <EditAzureBlobStorageComponent
            settings={settings}
            isSaved={isSaved}
            isAddressChanged={isAddressChanged}
            onChange={changeSettings}
          />
        )}

        {settings.type === StorageType.FTP && (
          <EditFTPStorageComponent
            settings={settings}
            isAddressChanged={isAddressChanged}
            onChange={changeSettings}
          />
        )}

        {settings.type === StorageType.SFTP && (
          <EditSFTPStorageComponent
            settings={settings}
            savedAuthMethods={
              savedSettings?.type === StorageType.SFTP ? savedSettings.authMethods : []
            }
            isAddressChanged={isAddressChanged}
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

      <div className="mt-3 flex flex-wrap gap-2">
        {!isTestConnectionSuccess && (
          <Button
            disabled={isTestingConnection || !isAllDataFilled}
            loading={isTestingConnection}
            type="primary"
            aria-label={t('storages.actions.testConnection')}
            onClick={testConnection}
          >
            {t('storages.actions.testConnection')}
          </Button>
        )}

        {isUnsaved && isTestConnectionSuccess && (
          <Button
            disabled={isSaving || !isAllDataFilled}
            loading={isSaving}
            type="primary"
            aria-label={t('common.actions.save')}
            onClick={save}
          >
            {t('common.actions.save')}
          </Button>
        )}

        {isShowClose && (
          <Button disabled={isSaving} type="primary" danger ghost onClick={onClose}>
            {t('common.actions.cancel')}
          </Button>
        )}
      </div>
    </section>
  );
}
