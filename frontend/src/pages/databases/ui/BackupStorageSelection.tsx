import { Button, Modal, Select, type GetRef } from 'antd';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, getStorageLogoFromType, storageApi } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { LoadFailure } from '@/shared/ui';

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
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const createStorage = () => {
    setIsDropdownOpen(false);
    storageSelectRef.current?.focus();
    setIsCreating(true);
  };

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
  const storageSelectRef = useRef<GetRef<typeof Select>>(null);
  const createButtonRef = useRef<HTMLButtonElement>(null);
  const currentWorkspace = useLatestRef(workspaceId);

  const startStoragesLoad = useEffectEvent(() => {
    void load();
  });

  useEffect(() => {
    acknowledgedStorages.current = [];
    setStorages([]);
    setIsCreating(false);
    setIsDropdownOpen(false);
    startStoragesLoad();

    return () => {
      requests.Invalidate();
    };
  }, [workspaceId, requests]);

  const returnFocusToSelect = useEffectEvent(() => {
    storageSelectRef.current?.focus();
  });

  useEffect(() => {
    if (!isCreating) return;

    return () => {
      returnFocusToSelect();
    };
  }, [isCreating]);

  const selectedStorage = storages.find((storage) => storage.id === selectedStorageId);

  return (
    <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
      {error !== undefined && (
        <LoadFailure className="w-full" error={error} onRetry={() => void load()} />
      )}
      <Select
        ref={storageSelectRef}
        id={controlId}
        aria-label={t('backups.config.storage')}
        allowClear={canClear}
        loading={isLoading}
        value={selectedStorageId}
        onChange={(storageId) => onSelect(storageId)}
        size="small"
        className="max-w-[200px] min-w-0 flex-1"
        open={isDropdownOpen}
        onOpenChange={setIsDropdownOpen}
        onInputKeyDown={(event) => {
          if (event.key === 'Tab' && !event.shiftKey && isDropdownOpen) {
            event.preventDefault();
            event.stopPropagation();
            createButtonRef.current?.focus();
          }
        }}
        popupRender={(menu) => (
          <>
            {storages.length > 0 && menu}
            <div className="mt-1 border-t border-border pt-1">
              <Button
                ref={createButtonRef}
                type="text"
                block
                className="!h-auto min-h-8 !justify-start !whitespace-normal !text-accent"
                onClick={createStorage}
                onKeyDown={(event) => {
                  event.stopPropagation();

                  if (event.key === 'Escape' || (event.key === 'Tab' && event.shiftKey)) {
                    event.preventDefault();
                    if (event.key === 'Escape') setIsDropdownOpen(false);
                    storageSelectRef.current?.focus();
                  }
                }}
              >
                {t('backups.config.createNewStorage')}
              </Button>
            </div>
          </>
        )}
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
