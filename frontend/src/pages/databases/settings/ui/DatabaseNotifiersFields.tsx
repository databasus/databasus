import { Loader } from '@mantine/core';
import {
  type Dispatch,
  type SetStateAction,
  useEffect,
  useEffectEvent,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';

import { type Notifier, notifierApi } from '@/entities/notifiers';
import { EditNotifierComponent } from '@/features/edit-notifier';
import { useSessionGuard } from '@/shared/auth';
import { useResourceCollection } from '@/shared/hooks';
import { ResourcePicker, Dialog, LoadFailure } from '@/shared/ui';

interface Props {
  workspaceId: string;
  notifierIds: string[];
  onNotifierIdsChange: Dispatch<SetStateAction<string[]>>;
}

export const DatabaseNotifiersFields = ({
  workspaceId,
  notifierIds,
  onNotifierIdsChange,
}: Props) => {
  const { t } = useTranslation();

  const collection = useResourceCollection<Notifier>({
    queryKey: ['notifiers', workspaceId],
    read: async (signal) => {
      const listed = await notifierApi.listAllNotifiers(workspaceId, { signal });
      const acknowledged = acknowledgedNotifiers.current;

      return [
        ...listed.filter((notifier) => !acknowledged.some((item) => item.id === notifier.id)),
        ...acknowledged,
      ];
    },
    onError: () => {},
  });
  const notifiers = collection.items;
  const error = collection.loadError;
  const isNotifiersLoading = collection.isLoading;
  const loadNotifiers = collection.reload;
  const [isShowCreateNotifier, setShowCreateNotifier] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const createNotifier = () => {
    setIsDropdownOpen(false);
    notifierSelectRef.current?.focus();
    setShowCreateNotifier(true);
  };

  const acceptNotifier = (notifier: Notifier) => {
    if (!isCurrent() || notifier.workspaceId !== workspaceId) return;

    acknowledgedNotifiers.current = [
      ...acknowledgedNotifiers.current.filter((item) => item.id !== notifier.id),
      notifier,
    ];
    collection.accept(notifier);
    void loadNotifiers(true);
  };

  const isCurrent = useSessionGuard();
  const acknowledgedNotifiers = useRef<Notifier[]>([]);
  const notifierSelectRef = useRef<HTMLButtonElement | HTMLInputElement>(null);

  useEffect(() => {
    acknowledgedNotifiers.current = [];
    setShowCreateNotifier(false);
    setIsDropdownOpen(false);
  }, [workspaceId]);

  const returnFocusToSelect = useEffectEvent(() => {
    notifierSelectRef.current?.focus();
  });

  useEffect(() => {
    if (!isShowCreateNotifier) return;

    return () => {
      returnFocusToSelect();
    };
  }, [isShowCreateNotifier]);

  if (isNotifiersLoading)
    return (
      <div className="mb-5 flex items-center">
        <Loader role="status" />
      </div>
    );

  return (
    <>
      {error !== undefined && <LoadFailure error={error} onRetry={() => void loadNotifiers()} />}
      <div className="mb-5 max-w-68.75 text-muted">
        {t('databases.notifiers.description')}
        <br />
        <br />
        {t('databases.notifiers.multipleHint')}
      </div>

      <div className="mb-5 flex w-full flex-wrap items-center gap-2">
        <div className="min-w-37.5 pr-2">{t('databases.notifiers.title')}</div>

        <ResourcePicker
          targetRef={notifierSelectRef}
          label={t('databases.notifiers.title')}
          isMultiple
          isSearchable
          selectedValues={notifierIds}
          onSelect={(id) =>
            onNotifierIdsChange((current) =>
              current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
            )
          }
          onRemove={(id) =>
            onNotifierIdsChange((current) => current.filter((value) => value !== id))
          }
          data={notifiers.map((notifier) => ({ value: notifier.id, label: notifier.name }))}
          className="max-w-50 flex-1 basis-40"
          isOpen={isDropdownOpen}
          onOpenChange={setIsDropdownOpen}
          createLabel={t('databases.notifiers.createNew')}
          onCreate={createNotifier}
          placeholder={t('databases.notifiers.selectPlaceholder')}
        />
      </div>

      {isShowCreateNotifier && (
        <Dialog
          title={t('databases.notifiers.addTitle')}
          opened={isShowCreateNotifier}
          onClose={() => {
            setShowCreateNotifier(false);
          }}
          closeOnClickOutside={false}
        >
          <div className="my-3 max-w-68.75 text-muted">{t('databases.notifiers.description')}</div>

          <EditNotifierComponent
            workspaceId={workspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateNotifier(false)}
            onPersisted={acceptNotifier}
            onChanged={(notifier) => {
              onNotifierIdsChange((current) => [...new Set([...current, notifier.id])]);
              setShowCreateNotifier(false);
            }}
          />
          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <div />
          </div>
        </Dialog>
      )}
    </>
  );
};
