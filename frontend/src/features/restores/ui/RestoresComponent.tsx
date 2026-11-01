import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { App, Button, Modal, Spin, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { type ReactNode, useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { LogicalBackup } from '../../../entities/backups/logical';
import type { Database, RestoreTargetDraft } from '../../../entities/databases';
import {
  type PostgresqlRestoreOptions,
  RESTORE_STATUS_LABEL_KEYS,
  type Restore,
  RestoreStatus,
  restoreApi,
} from '../../../entities/restores';
import { useSessionGuard } from '../../../shared/auth';
import { translateApiError, useLocale } from '../../../shared/i18n';
import { ClipboardHelper } from '../../../shared/lib/ClipboardHelper';
import { getUserTimeFormat } from '../../../shared/time';
import { ConfirmationComponent } from '../../../shared/ui';
import { createRestoreTargetDraft } from './createRestoreTargetDraft';
import type { RestoreTargetFormOptions } from './RestoreTargetFormOptions';
import { toRestoreTarget } from './toRestoreTarget';

interface Props {
  database: Database;
  backup: LogicalBackup;
  renderRestoreTargetForm: (options: RestoreTargetFormOptions) => ReactNode;
}

const NO_RESTORE_OPTIONS: PostgresqlRestoreOptions = {
  isExcludeExtensions: false,
  isRestoreOwnership: false,
  isRestorePrivileges: false,
};

export const RestoresComponent = ({ database, backup, renderRestoreTargetForm }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const { message } = App.useApp();
  const isActive = useSessionGuard();

  const [targetDraft, setTargetDraft] = useState(() =>
    createRestoreTargetDraft(database.settings.type),
  );
  const [restoreOptions, setRestoreOptions] = useState(NO_RESTORE_OPTIONS);

  const [restores, setRestores] = useState<Restore[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const [showingRestoreError, setShowingRestoreError] = useState<Restore | undefined>();

  const [isShowRestore, setIsShowRestore] = useState(false);

  const [cancellingRestoreId, setCancellingRestoreId] = useState<string | undefined>();
  const [showCancelConfirmation, setShowCancelConfirmation] = useState(false);
  const [restoreToCancelId, setRestoreToCancelId] = useState<string | undefined>();

  const isReloadInProgress = useRef(false);

  const loadRestores = async () => {
    if (isReloadInProgress.current) {
      return;
    }

    isReloadInProgress.current = true;

    try {
      setRestores(await restoreApi.listAllRestores(backup.id));
    } catch (e) {
      alert(translateApiError(e, t));
    }

    isReloadInProgress.current = false;
  };

  // The restore a request answered with is shown at once, before the next poll lists it.
  const showRestore = (answeredRestore: Restore) => {
    setRestores((loadedRestores) => {
      const isLoaded = loadedRestores.some(
        (loadedRestore) => loadedRestore.id === answeredRestore.id,
      );
      if (!isLoaded) {
        return [answeredRestore, ...loadedRestores];
      }

      return loadedRestores.map((loadedRestore) =>
        loadedRestore.id === answeredRestore.id ? answeredRestore : loadedRestore,
      );
    });
  };

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
      showRestore(await restoreApi.cancelRestore(restoreId));
      await loadRestores();
    } catch (e) {
      alert(translateApiError(e, t));
    } finally {
      setCancellingRestoreId(undefined);
    }
  };

  useEffect(() => {
    setIsLoading(true);
    loadRestores().finally(() => setIsLoading(false));

    const interval = setInterval(() => {
      loadRestores();
    }, 1_000);

    return () => clearInterval(interval);
  }, [backup.id]);

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

        {renderRestoreTargetForm({
          draft: targetDraft,
          onDraftChange: setTargetDraft,
          restoreOptions,
          onRestoreOptionsChange: setRestoreOptions,
          submitLabel: t('restores.targetDatabase.submit'),
          onSubmit: startRestore,
        })}
      </>
    );
  }

  return (
    <div className="mt-5">
      {isLoading ? (
        <div className="flex w-full justify-center">
          <Spin />
        </div>
      ) : (
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
            <div className="my-5 text-center text-gray-400">{t('restores.list.empty')}</div>
          )}

          <div className="mt-5">
            {restores.map((restore) => {
              let restoreDurationMs = 0;
              if (restore.status === RestoreStatus.IN_PROGRESS) {
                restoreDurationMs = Date.now() - new Date(restore.createdAt).getTime();
              } else {
                restoreDurationMs = restore.durationMs;
              }

              const minutes = Math.floor(restoreDurationMs / 60000);
              const seconds = Math.floor((restoreDurationMs % 60000) / 1000);
              const milliseconds = restoreDurationMs % 1000;
              // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
              const duration = `${minutes}m ${seconds}s ${milliseconds}ms`;

              const backupDurationMs = backup.durationMs;
              const expectedRestoreDurationMs = backupDurationMs * 5;
              // eslint-disable-next-line i18next/no-literal-string -- unit letters stay English in every language
              const expectedRestoreDuration = `${Math.floor(expectedRestoreDurationMs / 60000)}m ${Math.floor((expectedRestoreDurationMs % 60000) / 1000)}s`;

              return (
                <div key={restore.id} className="mb-1 rounded border border-gray-200 p-3 text-sm">
                  <div className="mb-1 flex items-center justify-between">
                    <div className="flex flex-1">
                      <div className="w-[75px] min-w-[75px] pr-2">{t('common.fields.status')}</div>

                      {restore.status === RestoreStatus.FAILED && (
                        <Tooltip title={t('restores.card.errorDetailsTooltip')}>
                          <div
                            className="flex cursor-pointer items-center text-red-600 underline"
                            onClick={() => setShowingRestoreError(restore)}
                          >
                            <ExclamationCircleOutlined
                              className="mr-2"
                              style={{ fontSize: 16, color: '#ff0000' }}
                            />

                            <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
                          </div>
                        </Tooltip>
                      )}

                      {restore.status === RestoreStatus.COMPLETED && (
                        <div className="flex items-center">
                          <CheckCircleOutlined
                            className="mr-2"
                            style={{ fontSize: 16, color: '#008000' }}
                          />

                          <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
                        </div>
                      )}

                      {restore.status === RestoreStatus.CANCELED && (
                        <div className="flex items-center text-gray-500">
                          <CloseCircleOutlined
                            className="mr-2"
                            style={{ fontSize: 16, color: '#808080' }}
                          />

                          <div>{t(RESTORE_STATUS_LABEL_KEYS[restore.status])}</div>
                        </div>
                      )}

                      {restore.status === RestoreStatus.IN_PROGRESS && (
                        <div className="flex items-center font-bold text-blue-600">
                          <SyncOutlined spin />
                          <span className="ml-2">
                            {t(RESTORE_STATUS_LABEL_KEYS[restore.status])}
                          </span>
                        </div>
                      )}
                    </div>

                    {restore.status === RestoreStatus.IN_PROGRESS && (
                      <div className="ml-2">
                        {cancellingRestoreId === restore.id ? (
                          <SyncOutlined spin style={{ fontSize: 16 }} />
                        ) : (
                          <Tooltip title={t('restores.card.cancelTooltip')}>
                            <CloseCircleOutlined
                              className="cursor-pointer"
                              onClick={() => {
                                if (cancellingRestoreId) return;
                                setRestoreToCancelId(restore.id);
                                setShowCancelConfirmation(true);
                              }}
                              style={{
                                color: '#ff0000',
                                fontSize: 16,
                                opacity: cancellingRestoreId ? 0.2 : 1,
                              }}
                            />
                          </Tooltip>
                        )}
                      </div>
                    )}
                  </div>

                  <div className="mb-1 flex">
                    <div className="w-[75px] min-w-[75px] pr-2">{t('restores.card.startedAt')}</div>
                    <div>
                      {dayjs.utc(restore.createdAt).local().format(getUserTimeFormat().format)} (
                      {formatRelativeTime(dayjs.utc(restore.createdAt).local())})
                    </div>
                  </div>

                  {restore.status === RestoreStatus.IN_PROGRESS && (
                    <div className="flex">
                      <div className="w-[75px] min-w-[75px] pr-2">
                        {t('restores.card.duration')}
                      </div>
                      <div>
                        <div>{duration}</div>
                        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                          {t('restores.card.expectedDurationHint')}
                          <br />
                          <br />
                          {t('restores.card.expectedDurationLimit', {
                            duration: expectedRestoreDuration,
                          })}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </>
      )}

      {showingRestoreError && (
        <Modal
          title={t('restores.errorDetails.title')}
          open={!!showingRestoreError}
          onCancel={() => setShowingRestoreError(undefined)}
          maskClosable={false}
          footer={
            <Button
              icon={<CopyOutlined />}
              onClick={() => {
                ClipboardHelper.copyToClipboard(showingRestoreError.failMessage || '');
                message.success(t('restores.errorDetails.copied'));
              }}
            >
              {t('common.actions.copy')}
            </Button>
          }
        >
          {showingRestoreError.failMessage?.includes('must be owner of extension') && (
            <div className="mb-4 rounded border border-yellow-300 bg-yellow-50 p-3 text-sm dark:border-yellow-600 dark:bg-yellow-900/30">
              <Trans
                i18nKey="restores.errorDetails.excludeExtensionsTip"
                components={{ bold: <strong /> }}
              />
            </div>
          )}
          <div className="overflow-y-auto text-sm whitespace-pre-wrap" style={{ height: '400px' }}>
            {showingRestoreError.failMessage}
          </div>
        </Modal>
      )}

      {showCancelConfirmation && (
        <ConfirmationComponent
          onConfirm={() => {
            setShowCancelConfirmation(false);
            if (restoreToCancelId) {
              cancelRestore(restoreToCancelId);
            }
            setRestoreToCancelId(undefined);
          }}
          onDecline={() => {
            setShowCancelConfirmation(false);
            setRestoreToCancelId(undefined);
          }}
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
