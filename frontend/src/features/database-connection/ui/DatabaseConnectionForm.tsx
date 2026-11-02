import { Alert, Button, ConfigProvider, notification } from 'antd';
import { type ReactNode, useEffect, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type DatabaseConnectionDraft,
  type DatabaseCreateDraft,
  type DatabaseEditDraft,
  type DatabaseSettings,
  DatabaseType,
  type RestoreTargetDraft,
  toDatabaseSettingsTarget,
} from '@/entities/databases';
import type { PostgresqlRestoreOptions } from '@/entities/restores';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

import { isConnectionAddressChanged } from '../lib/isConnectionAddressChanged';
import { isConnectionDraftFilled } from '../lib/isConnectionDraftFilled';
import { useDatabaseConnectionTest } from '../model/useDatabaseConnectionTest';
import { ConnectionTestFeedback } from './ConnectionTestFeedback';
import { EditMongodbConnectionComponent } from './EditMongodbConnectionComponent';
import { EditMySqlCompatibleConnectionComponent } from './EditMySqlCompatibleConnectionComponent';
import { EditPostgresqlLogicalConnectionComponent } from './EditPostgresqlLogicalConnectionComponent';
import { EditPostgresqlPhysicalConnectionComponent } from './EditPostgresqlPhysicalConnectionComponent';

interface CommonProps<Draft extends DatabaseConnectionDraft> {
  draft: Draft;
  onDraftChange: (draft: Draft) => void;
  onSubmit: (draft: Draft) => Promise<void> | void;
  submitLabel: string;
  secondaryActions?: ReactNode;
  onConnectionErrorChange?: (hasConnectionError: boolean) => void;
}

interface RestoreOptionsProps {
  restoreOptions: PostgresqlRestoreOptions;
  onRestoreOptionsChange: (restoreOptions: PostgresqlRestoreOptions) => void;
}

type Props =
  | CommonProps<DatabaseCreateDraft>
  | CommonProps<DatabaseEditDraft>
  | (CommonProps<RestoreTargetDraft> & RestoreOptionsProps);

const getSavedClientCertificate = (savedSettings: DatabaseSettings | undefined) =>
  savedSettings?.type === DatabaseType.POSTGRES_LOGICAL ||
  savedSettings?.type === DatabaseType.POSTGRES_PHYSICAL
    ? savedSettings.sslClientCert
    : undefined;

export const DatabaseConnectionForm = (props: Props) => {
  const { draft, submitLabel, secondaryActions, onConnectionErrorChange } = props;

  const { t } = useTranslation();
  const noticeId = useId();
  const [notices, noticeHolder] = notification.useNotification();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitFailure, setSubmitFailure] = useState<unknown>();

  // The props pair each draft type with callbacks of the same type, and the editors keep the type
  // of the settings they receive, so a changed draft always matches the callback.
  const reportDraft = (changedDraft: DatabaseConnectionDraft) =>
    (props.onDraftChange as (changedDraft: DatabaseConnectionDraft) => void)(changedDraft);

  const connectionTest = useDatabaseConnectionTest<DatabaseConnectionDraft>({
    onChange: reportDraft,
    onTestSucceeded: () =>
      notices.success({
        key: noticeId,
        message: t('databases.edit.connectionTestPassed.title'),
        description: t('databases.edit.connectionTestPassed.description'),
        duration: 4,
        placement: 'bottomRight',
      }),
  });

  const applySettings = (
    settings: DatabaseSettings,
    hasUserChosenSslMode = draft.hasUserChosenSslMode,
  ) => ({ ...draft, settings, hasUserChosenSslMode }) as DatabaseConnectionDraft;

  const changeConnection = (settings: DatabaseSettings, hasUserChosenSslMode?: boolean) => {
    setSubmitFailure(undefined);
    connectionTest.changeConnection(applySettings(settings, hasUserChosenSslMode));
  };

  const changeNonConnection = (settings: DatabaseSettings) =>
    connectionTest.changeNonConnection(applySettings(settings));

  const submit = async () => {
    if (isSubmitPending.current) return;

    isSubmitPending.current = true;
    setIsSubmitting(true);
    setSubmitFailure(undefined);

    try {
      await (props.onSubmit as (draft: DatabaseConnectionDraft) => Promise<void> | void)(
        structuredClone(draft),
      );
    } catch (e) {
      if (isActive()) setSubmitFailure(e);
    } finally {
      isSubmitPending.current = false;
      if (isActive()) setIsSubmitting(false);
    }
  };

  const renderEditor = () => {
    const savedSettings = draft.scenario === 'edit' ? draft.savedSettings : undefined;
    const editorProps = {
      scenario: draft.scenario,
      savedSshTunnel: savedSettings?.sshTunnel,
      isAddressChanged: isConnectionAddressChanged(draft),
      onConnectionChange: changeConnection,
      onNonConnectionChange: changeNonConnection,
    };
    const { settings } = draft;

    switch (settings.type) {
      case DatabaseType.POSTGRES_LOGICAL:
        return (
          <EditPostgresqlLogicalConnectionComponent
            {...editorProps}
            settings={settings}
            hasUserChosenSslMode={draft.hasUserChosenSslMode}
            savedClientCertificate={getSavedClientCertificate(savedSettings)}
            restoreOptions={'restoreOptions' in props ? props.restoreOptions : undefined}
            onRestoreOptionsChange={
              'onRestoreOptionsChange' in props ? props.onRestoreOptionsChange : undefined
            }
          />
        );
      case DatabaseType.POSTGRES_PHYSICAL:
        return (
          <EditPostgresqlPhysicalConnectionComponent
            {...editorProps}
            settings={settings}
            hasUserChosenSslMode={draft.hasUserChosenSslMode}
            savedClientCertificate={getSavedClientCertificate(savedSettings)}
          />
        );
      case DatabaseType.MYSQL:
      case DatabaseType.MARIADB:
        return <EditMySqlCompatibleConnectionComponent {...editorProps} settings={settings} />;
      case DatabaseType.MONGODB:
        return <EditMongodbConnectionComponent {...editorProps} settings={settings} />;
    }
  };

  const isActive = useSessionGuard();
  const isSubmitPending = useRef(false);

  const hasClassifiedFailure =
    connectionTest.failure?.kind === 'result' && !!connectionTest.failure.result.reason;

  useEffect(() => {
    onConnectionErrorChange?.(hasClassifiedFailure);

    return () => onConnectionErrorChange?.(false);
  }, [hasClassifiedFailure, onConnectionErrorChange]);

  useEffect(() => () => notices.destroy(noticeId), [notices, noticeId]);

  const isFilled = isConnectionDraftFilled(draft);

  return (
    <ConfigProvider componentDisabled={isSubmitting}>
      {noticeHolder}

      {renderEditor()}

      {connectionTest.failure && (
        <ConnectionTestFeedback
          failure={connectionTest.failure}
          isPhysical={draft.settings.type === DatabaseType.POSTGRES_PHYSICAL}
          username={draft.settings.username}
        />
      )}

      {submitFailure !== undefined && (
        <Alert
          type="error"
          showIcon
          className="mt-3"
          message={translateApiError(submitFailure, t)}
        />
      )}

      <div className="mt-5 flex flex-wrap gap-2">
        {secondaryActions}

        {draft.scenario !== 'restore' && !connectionTest.isTested && (
          <Button
            type="primary"
            onClick={() => connectionTest.testConnection(toDatabaseSettingsTarget(draft))}
            loading={connectionTest.isTesting}
            disabled={!isFilled}
          >
            {t('databases.actions.testConnection')}
          </Button>
        )}

        {(draft.scenario === 'restore' || connectionTest.isTested) && (
          <Button type="primary" onClick={submit} loading={isSubmitting} disabled={!isFilled}>
            {submitLabel}
          </Button>
        )}
      </div>
    </ConfigProvider>
  );
};
