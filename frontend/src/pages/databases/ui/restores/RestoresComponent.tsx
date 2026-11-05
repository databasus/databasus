import { App, Button, Spin } from 'antd';
import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { DatabaseConnectionForm } from '@/features/database-connection';

import type { Database, RestoreTargetDraft } from '../../../../entities/databases';
import {
  type PostgresqlRestoreOptions,
  type Restore,
  RestoreStatus,
  restoreApi,
} from '../../../../entities/restores';
import { useSessionGuard } from '../../../../shared/auth';
import { usePolledPagedList } from '../../../../shared/hooks';
import { translateApiError } from '../../../../shared/i18n';
import { ConfirmationComponent, LoadFailure } from '../../../../shared/ui';
import type { LogicalBackup } from '../../model/LogicalBackup';
import { createRestoreTargetDraft } from './createRestoreTargetDraft';
import { RestoreCard } from './RestoreCard';
import { RestoreErrorDialog } from './RestoreErrorDialog';
import { toRestoreTarget } from './toRestoreTarget';

interface Props {
  database: Database;
  backup: LogicalBackup;
}

const RESTORES_REFRESH_MS = 1_000;

const NO_RESTORE_OPTIONS: PostgresqlRestoreOptions = {
  isExcludeExtensions: false,
  isRestoreOwnership: false,
  isRestorePrivileges: false,
};

export const RestoresComponent = ({ database, backup }: Props) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const isActive = useSessionGuard();

  const [targetDraft, setTargetDraft] = useState(() =>
    createRestoreTargetDraft(database.settings.type),
  );
  const [restoreOptions, setRestoreOptions] = useState(NO_RESTORE_OPTIONS);
  const [isShowRestore, setIsShowRestore] = useState(false);

  const [showingRestoreError, setShowingRestoreError] = useState<Restore | undefined>();
  const [cancellingRestoreId, setCancellingRestoreId] = useState<string | undefined>();
  const [restoreToCancelId, setRestoreToCancelId] = useState<string | undefined>();

  const {
    items: restores,
    isLoading,
    firstLoadError,
    reload: reloadRestores,
    refreshFirstPage: refreshRestores,
    replaceOrPrependItem: showRestore,
  } = usePolledPagedList({
    fetchPage: async () => ({ items: await restoreApi.listAllRestores(backup.id) }),
    getItemKey: (restore: Restore) => restore.id,
    refreshIntervalMs: RESTORES_REFRESH_MS,
    resetKey: backup.id,
    // The listing has no next page, and the screen shows a failed first page itself.
    onError: () => {},
  });

  const startRestore = async (submittedDraft: RestoreTargetDraft) => {
    const startedRestore = await restoreApi.startRestore(
      backup.id,
      toRestoreTarget(submittedDraft, restoreOptions),
    );
    if (!isActive()) return;

    showRestore(startedRestore);
    setIsShowRestore(false);
  };

  const cancelRestore = async (restoreId: string) => {
    setCancellingRestoreId(restoreId);

    try {
      const cancelledRestore = await restoreApi.cancelRestore(restoreId);
      if (!isActive()) return;

      showRestore(cancelledRestore);
      await refreshRestores();
    } catch (error) {
      if (isActive()) message.error(translateApiError(error, t));
    }

    setCancellingRestoreId(undefined);
  };

  const isRestoreInProgress = restores.some(
    (restore) => restore.status === RestoreStatus.IN_PROGRESS,
  );

  if (isShowRestore && targetDraft) {
    return (
      <>
        <div className="my-5 text-sm">
          <Trans i18nKey="restores.targetDatabase.description" components={{ underline: <u /> }} />
          <br />
          <br />
          {t('restores.targetDatabase.notInUseHint')}
        </div>

        <DatabaseConnectionForm
          draft={targetDraft}
          onDraftChange={setTargetDraft}
          restoreOptions={restoreOptions}
          onRestoreOptionsChange={setRestoreOptions}
          submitLabel={t('restores.targetDatabase.submit')}
          onSubmit={startRestore}
        />
      </>
    );
  }

  const renderRestoreList = () => {
    if (isLoading) {
      return (
        <div className="flex w-full justify-center">
          <Spin />
        </div>
      );
    }

    if (firstLoadError !== undefined) {
      return <LoadFailure error={firstLoadError} onRetry={() => void reloadRestores()} />;
    }

    return (
      <>
        {targetDraft && (
          <Button
            className="w-full"
            type="primary"
            disabled={isRestoreInProgress}
            loading={isRestoreInProgress}
            onClick={() => setIsShowRestore(true)}
          >
            {t('restores.targetDatabase.select')}
          </Button>
        )}

        {restores.length === 0 && (
          <div className="my-5 text-center text-muted">{t('restores.list.empty')}</div>
        )}

        <div className="mt-5">
          {restores.map((restore) => (
            <RestoreCard
              key={restore.id}
              restore={restore}
              backupDurationMs={backup.durationMs}
              isCancelling={cancellingRestoreId === restore.id}
              isCancelDisabled={Boolean(cancellingRestoreId)}
              onCancel={() => setRestoreToCancelId(restore.id)}
              onShowError={() => setShowingRestoreError(restore)}
            />
          ))}
        </div>
      </>
    );
  };

  return (
    <div className="mt-5">
      {renderRestoreList()}

      {showingRestoreError && (
        <RestoreErrorDialog
          restore={showingRestoreError}
          onClose={() => setShowingRestoreError(undefined)}
        />
      )}

      {restoreToCancelId && (
        <ConfirmationComponent
          onConfirm={() => {
            setRestoreToCancelId(undefined);
            void cancelRestore(restoreToCancelId);
          }}
          onDecline={() => setRestoreToCancelId(undefined)}
          description={
            <>
              <div>
                <Trans
                  i18nKey="restores.cancelConfirmation.warning"
                  components={{ bold: <strong /> }}
                />
              </div>
              <div className="mt-5">{t('restores.cancelConfirmation.question')}</div>
            </>
          }
          actionText={t('restores.cancelConfirmation.confirm')}
          isDestructive
        />
      )}
    </div>
  );
};
