import { Alert, Button, DatePicker, Segmented, Spin, Tabs } from 'antd';
import type { Dayjs } from 'dayjs';
import { type JSX, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType } from '../../../../entities/databases';
import { ApiError } from '../../../../shared/api';
import { type TranslationKey, translateApiError } from '../../../../shared/i18n';
import { physicalBackupsApi } from '../../api/physicalBackupsApi';
import {
  type RestoreEnvironment,
  defaultDockerImage,
} from '../../lib/physical-backups/restoreCommands';
import { type PhysicalBackupListItem } from '../../model/physical-backups/PhysicalBackupListItem';
import { PhysicalRestoreEnvironmentPanel } from './PhysicalRestoreEnvironmentPanel';
import type { PhysicalRestoreMethod } from './PhysicalRestoreMethod';
import { usePhysicalRestoreToken } from './usePhysicalRestoreToken';

interface Props {
  database: Database;
  backup?: PhysicalBackupListItem;
  onClose: () => void;
}

const RESTORE_ERROR_HINT_KEYS: Partial<Record<string, TranslationKey>> = {
  already_running: 'backups.physical.restore.errorHints.downloadInProgress',
  target_time_unreachable: 'backups.physical.restore.errorHints.targetTimeUnreachable',
  no_restorable_backup: 'backups.physical.restore.errorHints.noRestorableBackup',
};

const getRestoreErrorHintKey = (error: unknown): TranslationKey | undefined =>
  error instanceof ApiError && error.code ? RESTORE_ERROR_HINT_KEYS[error.code] : undefined;

export const PhysicalRestoreComponent = ({ database, backup, onClose }: Props): JSX.Element => {
  const { t } = useTranslation();
  const targetTimeFieldId = useId();

  // The source cluster's major version, detected on connect. Physical databases are
  // always PostgreSQL 17 or 18; the fallback is only a defensive guard. Every shown
  // image tag, bin path and container PGDATA derives from this single value.
  const pgVersion =
    (database.settings.type === DatabaseType.POSTGRES_PHYSICAL && database.settings.version) ||
    '17';

  const [targetTime, setTargetTime] = useState<Dayjs | undefined>();
  const [restoreMethod, setRestoreMethod] = useState<PhysicalRestoreMethod>('script');
  const [environment, setEnvironment] = useState<RestoreEnvironment>('host');
  const [outputDir, setOutputDir] = useState('./databasus-restore');
  const [pgBin, setPgBin] = useState('');
  const [dockerImage, setDockerImage] = useState(defaultDockerImage(pgVersion));

  const {
    token,
    error: restoreError,
    isPending: isGenerating,
  } = usePhysicalRestoreToken(
    backup
      ? { type: 'BACKUP', backupId: backup.id }
      : {
          type: 'POINT_IN_TIME',
          databaseId: database.id,
          targetTime: targetTime ? targetTime.utc().toISOString() : undefined,
        },
  );

  const bundleUrl = token ? physicalBackupsApi.getPhysicalRestoreSetUrl(token) : undefined;
  const recoveryTargetTime = targetTime ? targetTime.utc().format('YYYY-MM-DD HH:mm:ssZ') : '';
  const restoreErrorHintKey = getRestoreErrorHintKey(restoreError);

  const renderEnvironmentPanel = (
    panelEnvironment: RestoreEnvironment,
    url: string,
  ): JSX.Element => (
    <PhysicalRestoreEnvironmentPanel
      environment={panelEnvironment}
      restoreMethod={restoreMethod}
      bundleUrl={url}
      recoveryTargetTime={recoveryTargetTime}
      pgVersion={pgVersion}
      hasWal={backup === undefined}
      outputDir={outputDir}
      pgBin={pgBin}
      dockerImage={dockerImage}
      onOutputDirChange={setOutputDir}
      onPgBinChange={setPgBin}
      onDockerImageChange={setDockerImage}
    />
  );

  return (
    <div>
      {backup ? (
        <div className="mb-3 text-sm text-muted">{t('backups.physical.restore.backupIntro')}</div>
      ) : (
        <>
          <div className="mb-3 text-sm text-muted">
            {t('backups.physical.restore.pointInTimeIntro')}
          </div>
          <div className="mb-3 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <label className="mb-1 min-w-[120px] sm:mb-0 sm:pr-2" htmlFor={targetTimeFieldId}>
              {t('backups.physical.restore.targetTime')}
            </label>
            <DatePicker
              id={targetTimeFieldId}
              showTime
              value={targetTime}
              onChange={(value) => setTargetTime(value ?? undefined)}
              className="w-full max-w-[260px] grow"
              placeholder={t('backups.physical.restore.latestAvailable')}
            />
          </div>
        </>
      )}

      {restoreError !== undefined && (
        <Alert
          className="mt-3 whitespace-pre-line"
          type="error"
          showIcon
          message={translateApiError(restoreError, t)}
          description={restoreErrorHintKey && t(restoreErrorHintKey)}
        />
      )}

      {isGenerating && (
        <div className="mt-5 flex items-center gap-2 border-t border-border pt-4 text-sm text-muted">
          <Spin size="small" />
          {t('backups.physical.restore.preparing')}
        </div>
      )}

      {bundleUrl && (
        <div className="mt-5 border-t border-border pt-4">
          <Segmented<PhysicalRestoreMethod>
            aria-label={t('backups.physical.restore.methods.label')}
            value={restoreMethod}
            onChange={setRestoreMethod}
            options={[
              { label: t('backups.physical.restore.methods.script'), value: 'script' },
              { label: t('backups.physical.restore.methods.manual'), value: 'manual' },
            ]}
          />
          <Tabs
            className="mt-2"
            activeKey={environment}
            onChange={(key) => setEnvironment(key as RestoreEnvironment)}
            items={[
              {
                key: 'host',
                label: t('backups.physical.restore.environments.host'),
                children: renderEnvironmentPanel('host', bundleUrl),
              },
              {
                key: 'docker',
                // eslint-disable-next-line i18next/no-literal-string -- product name
                label: 'Docker',
                children: renderEnvironmentPanel('docker', bundleUrl),
              },
            ]}
          />
          <p className="mt-1 text-xs text-muted">{t('backups.physical.restore.linkExpiry')}</p>
        </div>
      )}

      <div className="mt-4 flex">
        <Button className="ml-auto" onClick={onClose}>
          {t('common.actions.close')}
        </Button>
      </div>
    </div>
  );
};
