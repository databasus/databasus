import { Alert, Button, Modal, Select } from 'antd';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, getStorageLogoFromType, storageApi } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';

interface Props {
  workspaceId: string;
  selectedStorageId?: string;
  canClear: boolean;
  onSelect: (storageId?: string) => void;
}

export function BackupStorageSelection({
  workspaceId,
  selectedStorageId,
  canClear,
  onSelect,
}: Props) {
  const [storages, setStorages] = useState<Storage[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<unknown>();
  const [isCreating, setIsCreating] = useState(false);

  const acceptStorage = (storage: Storage) => {
    if (!isCurrent() || currentWorkspace.current !== storage.workspaceId) return;

    acknowledgedStorages.current = [
      ...acknowledgedStorages.current.filter((item) => item.id !== storage.id),
      storage,
    ];
    setStorages((current) => [...current.filter((item) => item.id !== storage.id), storage]);
    void load();
  };

  const load = async () => {
    const revision = requests.Invalidate();
    setIsLoading(true);
    setError(undefined);

    try {
      const choices = await storageApi.listAllStorages(workspaceId);
      if (!isCurrent() || !requests.IsCurrent(revision)) return;

      const acknowledged = acknowledgedStorages.current;
      setStorages([
        ...choices.filter((storage) => !acknowledged.some((item) => item.id === storage.id)),
        ...acknowledged,
      ]);
    } catch (failure) {
      if (isCurrent() && requests.IsCurrent(revision)) setError(failure);
    } finally {
      if (isCurrent() && requests.IsCurrent(revision)) setIsLoading(false);
    }
  };

  const { t } = useTranslation();
  const controlId = useId();
  const isCurrent = useSessionGuard();
  const requests = useRequestRevision();
  const acknowledgedStorages = useRef<Storage[]>([]);
  const createButtonRef = useRef<HTMLButtonElement>(null);
  const currentWorkspace = useLatestRef(workspaceId);

  const startStoragesLoad = useEffectEvent(() => {
    void load();
  });

  useEffect(() => {
    acknowledgedStorages.current = [];
    setStorages([]);
    setIsCreating(false);
    startStoragesLoad();

    return () => {
      requests.Invalidate();
    };
  }, [workspaceId, requests]);

  // Read when the form closes, because the button can appear after the form opened.
  const returnFocusToButton = useEffectEvent(() => {
    createButtonRef.current?.focus();
  });

  useEffect(() => {
    if (!isCreating) return;

    return () => {
      returnFocusToButton();
    };
  }, [isCreating]);

  const selectedStorage = storages.find((storage) => storage.id === selectedStorageId);

  return (
    <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
      {error !== undefined && (
        <Alert
          type="error"
          className="w-full"
          message={translateApiError(error, t)}
          action={<Button onClick={() => void load()}>{t('common.actions.retry')}</Button>}
        />
      )}
      <Select
        id={controlId}
        aria-label={t('backups.config.storage')}
        allowClear={canClear}
        loading={isLoading}
        value={selectedStorageId}
        onChange={(storageId) => onSelect(storageId)}
        size="small"
        className="min-w-0 flex-1"
        options={storages.map((storage) => ({ label: storage.name, value: storage.id }))}
        placeholder={t('backups.config.selectStorage')}
      />
      {selectedStorage && (
        <img
          src={getStorageLogoFromType(selectedStorage.settings.type)}
          alt=""
          className="h-4 w-4"
        />
      )}
      <Button ref={createButtonRef} size="small" onClick={() => setIsCreating(true)}>
        {t('backups.config.createNewStorage')}
      </Button>
      {isCreating && (
        <Modal
          title={t('backups.config.addStorageTitle')}
          open
          footer={null}
          maskClosable={false}
          onCancel={() => setIsCreating(false)}
        >
          <p className="my-3 text-muted">{t('backups.config.storageDescription')}</p>
          <EditStorageComponent
            key={workspaceId}
            workspaceId={workspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsCreating(false)}
            onPersisted={acceptStorage}
            onChanged={(storage) => {
              if (!isCurrent() || currentWorkspace.current !== storage.workspaceId) return;

              onSelect(storage.id);
              setIsCreating(false);
            }}
          />
        </Modal>
      )}
    </div>
  );
}
