import { Alert, Button, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  S3_COMPATIBLE_SERVICES,
  STORAGE_TYPE_LABEL_KEYS,
  type S3CompatibleService,
  type Storage,
  StorageType,
  getS3CompatibleServiceFromEndpoint,
  getStorageLogoFromType,
} from '@/entities/storages';
import { createDefaultStorageSettings } from '@/features/edit-storage/lib/createDefaultStorageSettings';
import { isStorageAddressChanged } from '@/features/edit-storage/lib/isStorageAddressChanged';
import { isStorageSettingsFilled } from '@/features/edit-storage/lib/isStorageSettingsFilled';
import type { StorageDraft } from '@/features/edit-storage/lib/StorageDraft';
import { useStorageDraft } from '@/features/edit-storage/model/useStorageDraft';
import { EditAzureBlobStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditAzureBlobStorageComponent';
import { EditFTPStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditFTPStorageComponent';
import { EditLocalStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditLocalStorageComponent';
import { EditNASStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditNASStorageComponent';
import { EditS3StorageComponent } from '@/features/edit-storage/ui/edit/storages/EditS3StorageComponent';
import { EditSFTPStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { WrappingSelect } from '@/shared/ui';

type StorageTypeChoice = StorageType | S3CompatibleService;

const isS3CompatibleService = (choice: StorageTypeChoice): choice is S3CompatibleService =>
  choice in S3_COMPATIBLE_SERVICES;

const s3CompatibleServiceOptions = Object.entries(S3_COMPATIBLE_SERVICES).map(
  ([service, { name }]) => ({ label: name, value: service as S3CompatibleService }),
);

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

  // A managed object storage service is a choice of the form only: the storage stays type S3. A
  // saved storage takes it from its saved endpoint, a new one from the type select.
  const [chosenS3CompatibleService, setChosenS3CompatibleService] = useState<
    S3CompatibleService | undefined
  >(() =>
    settings.type === StorageType.S3
      ? getS3CompatibleServiceFromEndpoint(settings.s3Endpoint)
      : undefined,
  );
  const selectedS3CompatibleService =
    settings.type === StorageType.S3 ? chosenS3CompatibleService : undefined;
  const s3CompatibleService = isSaved
    ? savedSettings?.type === StorageType.S3
      ? getS3CompatibleServiceFromEndpoint(savedSettings.s3Endpoint)
      : undefined
    : selectedS3CompatibleService;

  const storageTypeOptions = Object.values(StorageType).flatMap(
    (storageType): { label: string; value: StorageTypeChoice }[] => [
      { label: t(STORAGE_TYPE_LABEL_KEYS[storageType]), value: storageType },
      ...(storageType === StorageType.S3 ? s3CompatibleServiceOptions : []),
    ],
  );

  const chooseStorageType = (choice: StorageTypeChoice) => {
    const service = isS3CompatibleService(choice) ? choice : undefined;
    const type = isS3CompatibleService(choice) ? StorageType.S3 : choice;
    const isSwitchWithinS3 = type === StorageType.S3 && settings.type === StorageType.S3;

    setChosenS3CompatibleService(service);
    if (!isSwitchWithinS3) changeSettings(createDefaultStorageSettings(type));
  };

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
              value={selectedS3CompatibleService ?? settings.type}
              disabled={isSaving}
              data={storageTypeOptions}
              onChange={chooseStorageType}
              size="xs"
              className="w-full max-w-[250px] min-w-0"
            />
          )}

          <img
            alt=""
            src={
              !isSaved && selectedS3CompatibleService
                ? S3_COMPATIBLE_SERVICES[selectedS3CompatibleService].logo
                : getStorageLogoFromType(settings.type)
            }
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
            s3CompatibleService={s3CompatibleService}
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
