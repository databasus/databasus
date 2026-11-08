import { ActionIcon } from '@mantine/core';
import { X } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, getStorageLogoFromType, storageApi } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useResourceCollection } from '@/shared/hooks';
import { ResourcePicker, Dialog, LoadFailure } from '@/shared/ui';

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
  const collection = useResourceCollection<Storage>({
    queryKey: ['storages', workspaceId],
    read: async (signal) => {
      const choices = await storageApi.listAllStorages(workspaceId, { signal });
      const acknowledged = acknowledgedStorages.current;

      return [
        ...choices.filter((storage) => !acknowledged.some((item) => item.id === storage.id)),
        ...acknowledged,
      ];
    },
    onError: () => {},
  });
  const storages = collection.items;
  const isLoading = collection.isLoading;
  const error = collection.loadError;
  const load = collection.reload;
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
    collection.accept(storage);
    void load();
  };

  const { t } = useTranslation();
  const controlId = useId();
  const isCurrent = useSessionGuard();
  const acknowledgedStorages = useRef<Storage[]>([]);
  const storageSelectRef = useRef<HTMLButtonElement | HTMLInputElement>(null);
  const currentWorkspace = useLatestRef(workspaceId);

  useEffect(() => {
    acknowledgedStorages.current = [];
    setIsCreating(false);
    setIsDropdownOpen(false);
  }, [workspaceId]);

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
      <ResourcePicker
        targetRef={storageSelectRef}
        id={controlId}
        label={t('backups.config.storage')}
        data={storages.map((storage) => ({ value: storage.id, label: storage.name }))}
        selectedValues={selectedStorageId ? [selectedStorageId] : []}
        onSelect={onSelect}
        isLoading={isLoading}
        className="max-w-[200px] flex-1"
        isOpen={isDropdownOpen}
        onOpenChange={setIsDropdownOpen}
        placeholder={t('backups.config.selectStorage')}
        createLabel={t('backups.config.createNewStorage')}
        onCreate={createStorage}
      />
      {canClear && selectedStorageId && (
        <ActionIcon
          size="xs"
          aria-label={t('common.actions.clear')}
          onClick={() => onSelect(undefined)}
        >
          <X aria-hidden size={12} />
        </ActionIcon>
      )}
      {selectedStorage && (
        <img
          src={getStorageLogoFromType(selectedStorage.settings.type)}
          alt=""
          className="h-4 w-4"
        />
      )}
      {isCreating && (
        <Dialog
          title={t('backups.config.addStorageTitle')}
          opened
          closeOnClickOutside={false}
          onClose={() => setIsCreating(false)}
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
        </Dialog>
      )}
    </div>
  );
}
