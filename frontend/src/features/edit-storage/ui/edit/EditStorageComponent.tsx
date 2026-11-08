import { Alert, Button, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import {
  STORAGE_TYPE_LABEL_KEYS,
  type Storage,
  StorageType,
  getStorageLogoFromType,
} from '@/entities/storages';
import { createDefaultStorageSettings } from '@/features/edit-storage/lib/createDefaultStorageSettings';
import { isStorageAddressChanged } from '@/features/edit-storage/lib/isStorageAddressChanged';
import { isStorageSettingsFilled } from '@/features/edit-storage/lib/isStorageSettingsFilled';
import type { StorageDraft } from '@/features/edit-storage/lib/StorageDraft';
import { useStorageDraft } from '@/features/edit-storage/model/useStorageDraft';
import { EditAzureBlobStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditAzureBlobStorageComponent';
import { EditFTPStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditFTPStorageComponent';
import { EditGoogleDriveStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditGoogleDriveStorageComponent';
import { EditLocalStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditLocalStorageComponent';
import { EditNASStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditNASStorageComponent';
import { EditRcloneStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditRcloneStorageComponent';
import { EditS3StorageComponent } from '@/features/edit-storage/ui/edit/storages/EditS3StorageComponent';
import { EditSFTPStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { WrappingSelect } from '@/shared/ui';
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
  const notify = useFeedback({
    scope: 'screen',
    contextKey: `${workspaceId}:${editingStorage?.id ?? 'new'}`,
  });
  const form = useStorageDraft({
    workspaceId,
    editingStorage,
    draft,
    onChanged,
    onPersisted,
    onTestSucceeded: () =>
      notify({
        kind: 'success',
        title: t('storages.connectionTestSucceeded.title'),
        description: t('storages.connectionTestSucceeded.description'),
        lifetimeMs: 4 * 1000,
        position: 'bottom-right',
      }),
  });
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
    <section aria-label={t('storages.config.settingsTitle')} className="min-w-0">
      {errorText && (
        <Alert
          color="danger"
          icon={
            <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={errorText}
        />
      )}
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label htmlFor={`${controlId}-name`} className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
            {t('common.fields.name')}
          </label>

          <TextInput
            id={`${controlId}-name`}
            value={name}
            onChange={(e) => {
              form.changeName(e.target.value);
            }}
            size="xs"
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
            <WrappingSelect
              id={`${controlId}-type`}
              value={settings.type}
              disabled={isSaving}
              data={storageTypeOptions}
              onChange={(type: StorageType) => changeSettings(createDefaultStorageSettings(type))}
              size="xs"
              className="w-full max-w-[250px] min-w-0"
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
            variant="filled"
            aria-label={t('storages.actions.testConnection')}
            onClick={testConnection}
            aria-busy={isTestingConnection}
          >
            {t('storages.actions.testConnection')}
          </Button>
        )}

        {isUnsaved && isTestConnectionSuccess && (
          <Button
            disabled={isSaving || !isAllDataFilled}
            loading={isSaving}
            variant="filled"
            aria-label={t('common.actions.save')}
            onClick={save}
            aria-busy={isSaving}
          >
            {t('common.actions.save')}
          </Button>
        )}

        {isShowClose && (
          <Button disabled={isSaving} variant="outline" color="danger" onClick={onClose}>
            {t('common.actions.cancel')}
          </Button>
        )}
      </div>
    </section>
  );
}
