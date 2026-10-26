import { CheckOutlined, CopyOutlined } from '@ant-design/icons';
import { Alert, Button, DatePicker, Input, Segmented, Spin, Tabs } from 'antd';
import type { Dayjs } from 'dayjs';
import { type JSX, useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { getApplicationServer } from '../../../../constants';
import {
  type PhysicalBackupListItem,
  physicalBackupsApi,
} from '../../../../entity/backups/physical';
import { type Database, DatabaseType } from '../../../../entity/databases';
import { ApiError } from '../../../../shared/api';
import { type TranslationKey, translateApiError } from '../../../../shared/i18n';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { InlineCodeComponent } from '../../../../shared/ui';
import {
  type RestoreEnvironment,
  buildChownCommand,
  buildDockerRunCommand,
  buildDockerScriptCommand,
  buildManualSteps,
  buildPgCtlStartCommand,
  buildScriptCommand,
  clusterDataDir,
  containerDataDir,
  containerVolumeDir,
  debianConfigDir,
  defaultDockerImage,
  defaultPgBinDir,
} from '../lib/restoreCommands';

interface Props {
  database: Database;
  backup?: PhysicalBackupListItem;
  onClose: () => void;
}

type RestoreMethod = 'script' | 'manual';

// The two restore failures with a known cause get a hint after the message: a concurrent download
// (409) and an unreachable target time or WAL gap (422). Decided by status, never by the text.
const RESTORE_ERROR_HINT_KEYS: Partial<Record<number, TranslationKey>> = {
  409: 'backups.physical.restore.errorHints.downloadInProgress',
  422: 'backups.physical.restore.errorHints.targetTimeUnreachable',
};

const getRestoreErrorHintKey = (error: unknown): TranslationKey | undefined =>
  error instanceof ApiError && error.status ? RESTORE_ERROR_HINT_KEYS[error.status] : undefined;

// Rendered for host and Docker alike: whether the source keeps its configuration outside the data
// directory cannot be known from here, and the resulting startup failure looks identical to a
// wrong-directory mount, so the difference has to be spelled out.
const missingConfigFilesNote = (pgVersion: string): JSX.Element => (
  <li>
    <Trans
      i18nKey="backups.physical.restore.afterItFinishes.missingConfigFiles"
      components={{
        code: <InlineCodeComponent />,
        configDir: <InlineCodeComponent>{debianConfigDir(pgVersion)}</InlineCodeComponent>,
      }}
    />
  </li>
);

interface CopyableCommandProps {
  id: string;
  title: string;
  code: string;
  copiedKey: string | null;
  onCopy: (id: string, code: string) => void;
}

const CopyableCommand = ({
  id,
  title,
  code,
  copiedKey,
  onCopy,
}: CopyableCommandProps): JSX.Element => {
  const { t } = useTranslation();
  const isCopied = copiedKey === id;

  return (
    <div className="mt-3">
      <div className="mb-1 flex items-center justify-between">
        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">{title}</div>
        <Button
          size="small"
          type="text"
          icon={isCopied ? <CheckOutlined /> : <CopyOutlined />}
          onClick={() => onCopy(id, code)}
        >
          {isCopied ? t('backups.physical.restore.copied') : t('common.actions.copy')}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded bg-gray-100 p-3 text-xs whitespace-pre-wrap text-gray-700 dark:bg-gray-700 dark:text-gray-200">
        {code}
      </pre>
    </div>
  );
};

export const PhysicalRestoreComponent = ({ database, backup, onClose }: Props): JSX.Element => {
  const { t } = useTranslation();

  // The source cluster's major version, detected on connect. Physical databases are
  // always PostgreSQL 17 or 18; the fallback is only a defensive guard. Every shown
  // image tag, bin path and container PGDATA derives from this single value.
  const pgVersion =
    (database.settings.type === DatabaseType.POSTGRES_PHYSICAL && database.settings.version) ||
    '17';

  const [targetTime, setTargetTime] = useState<Dayjs | undefined>();
  const [isGenerating, setIsGenerating] = useState(true);
  const [restoreError, setRestoreError] = useState<unknown>();
  const [bundleUrl, setBundleUrl] = useState<string>();
  const [restoreMethod, setRestoreMethod] = useState<RestoreMethod>('script');
  const [environment, setEnvironment] = useState<RestoreEnvironment>('host');
  const [outputDir, setOutputDir] = useState('./databasus-restore');
  const [pgBin, setPgBin] = useState('');
  const [dockerImage, setDockerImage] = useState(defaultDockerImage(pgVersion));
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const generateRestore = async () => {
    setIsGenerating(true);
    setRestoreError(undefined);
    setBundleUrl(undefined);
    setCopiedKey(null);

    try {
      const response = backup
        ? await physicalBackupsApi.generateBackupRestoreToken(backup.id)
        : await physicalBackupsApi.generatePitrRestoreToken(
            database.id,
            targetTime ? targetTime.utc().toISOString() : undefined,
          );

      setBundleUrl(`${getApplicationServer()}${response.url}`);
    } catch (e) {
      setRestoreError(e);
    }

    setIsGenerating(false);
  };

  const copyText = async (id: string, text: string) => {
    await ClipboardHelper.copyToClipboard(text);
    setCopiedKey(id);
    setTimeout(() => setCopiedKey((current) => (current === id ? null : current)), 2000);
  };

  // A token is minted automatically: once on open for a per-backup restore, and
  // again whenever the PITR target changes. The token is single-use and the stream
  // is unauthenticated, so it must be a fresh capability the user can curl - not a
  // static command.
  useEffect(() => {
    generateRestore();
  }, [targetTime]);

  const hasWal = backup === undefined;
  const dataDir = clusterDataDir(outputDir, pgVersion);
  const scriptUrl = `${getApplicationServer()}/api/v1/backups/physical/recovery-script`;
  const recoveryTargetTime = targetTime ? targetTime.utc().format('YYYY-MM-DD HH:mm:ssZ') : '';
  const restoreErrorHintKey = getRestoreErrorHintKey(restoreError);

  const renderConfig = (env: RestoreEnvironment): JSX.Element => (
    <div className="mb-3">
      <div className="mb-2 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 shrink-0 sm:mb-0 sm:w-[190px] sm:pr-2">
          {t('backups.physical.restore.fields.restoreDirectory')}
        </div>
        <Input
          value={outputDir}
          onChange={(e) => setOutputDir(e.target.value)}
          className="w-full max-w-[320px]"
        />
      </div>
      {env === 'host' ? (
        <div className="flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 shrink-0 sm:mb-0 sm:w-[190px] sm:pr-2">
            {t('backups.physical.restore.fields.pgBinPath')}
          </div>
          <Input
            value={pgBin}
            onChange={(e) => setPgBin(e.target.value)}
            placeholder={t('backups.physical.restore.fields.pgBinPathPlaceholder', {
              path: defaultPgBinDir(pgVersion),
            })}
            className="w-full max-w-[320px]"
          />
        </div>
      ) : (
        <div className="flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 shrink-0 sm:mb-0 sm:w-[190px] sm:pr-2">
            {t('backups.physical.restore.fields.pgImage')}
          </div>
          <Input
            value={dockerImage}
            onChange={(e) => setDockerImage(e.target.value)}
            className="w-full max-w-[320px]"
          />
        </div>
      )}
    </div>
  );

  const renderCommands = (env: RestoreEnvironment, url: string): JSX.Element => {
    if (restoreMethod === 'script') {
      const code =
        env === 'host'
          ? buildScriptCommand({
              scriptUrl,
              bundleUrl: url,
              outputDir,
              pgBin,
              targetTime: recoveryTargetTime,
            })
          : buildDockerScriptCommand({
              scriptUrl,
              bundleUrl: url,
              outputDir,
              image: dockerImage,
              targetTime: recoveryTargetTime,
            });

      return (
        <CopyableCommand
          id={`script-${env}`}
          title={
            env === 'host'
              ? t('backups.physical.restore.scriptTitles.host')
              : t('backups.physical.restore.scriptTitles.docker')
          }
          code={code}
          copiedKey={copiedKey}
          onCopy={copyText}
        />
      );
    }

    const steps = buildManualSteps({
      bundleUrl: url,
      outputDir,
      pgVersion,
      pgBin,
      image: dockerImage,
      environment: env,
      hasWal,
      targetTime: recoveryTargetTime,
    });

    return (
      <>
        {steps.map((step, index) => (
          <CopyableCommand
            key={step.titleKey}
            id={`manual-${env}-${index}`}
            title={`${index + 1}. ${t(step.titleKey)}`}
            code={step.code}
            copiedKey={copiedKey}
            onCopy={copyText}
          />
        ))}
      </>
    );
  };

  const renderEnvironmentPanel = (env: RestoreEnvironment, url: string): JSX.Element => (
    <div>
      {renderConfig(env)}
      <Alert
        type="info"
        showIcon
        message={t('backups.physical.restore.beforeYouRun.title')}
        description={
          env === 'host' ? (
            <ul className="ml-4 list-disc">
              <li>
                {t('backups.physical.restore.beforeYouRun.host.clientTools', {
                  version: pgVersion,
                })}
              </li>
              <li>
                {hasWal
                  ? t('backups.physical.restore.beforeYouRun.host.zstdWithWal')
                  : t('backups.physical.restore.beforeYouRun.host.zstd')}
              </li>
              <li>{t('backups.physical.restore.beforeYouRun.host.emptyDirectory')}</li>
            </ul>
          ) : (
            <ul className="ml-4 list-disc">
              <li>
                {t('backups.physical.restore.beforeYouRun.docker.image', { version: pgVersion })}
              </li>
              <li>
                {hasWal
                  ? t('backups.physical.restore.beforeYouRun.docker.hostToolsWithWal')
                  : t('backups.physical.restore.beforeYouRun.docker.hostTools')}
              </li>
            </ul>
          )
        }
      />
      {renderCommands(env, url)}
      <div className="mt-3" />
      <Alert
        type="info"
        showIcon
        message={t('backups.physical.restore.afterItFinishes.title')}
        description={
          env === 'host' ? (
            <ul className="ml-4 list-disc">
              <li>
                <Trans
                  i18nKey="backups.physical.restore.afterItFinishes.host.clusterLocation"
                  components={{ dataDir: <InlineCodeComponent>{dataDir}</InlineCodeComponent> }}
                />
              </li>
              <li>
                <Trans
                  i18nKey="backups.physical.restore.afterItFinishes.host.changeOwner"
                  components={{
                    command: (
                      <InlineCodeComponent>{buildChownCommand(dataDir)}</InlineCodeComponent>
                    ),
                  }}
                />
              </li>
              <li>
                <Trans
                  i18nKey="backups.physical.restore.afterItFinishes.host.start"
                  components={{
                    command: (
                      <InlineCodeComponent>{buildPgCtlStartCommand(dataDir)}</InlineCodeComponent>
                    ),
                  }}
                />
              </li>
              {hasWal && <li>{t('backups.physical.restore.afterItFinishes.host.walReplay')}</li>}
              {missingConfigFilesNote(pgVersion)}
            </ul>
          ) : (
            <ul className="ml-4 list-disc">
              <li>
                <Trans
                  i18nKey="backups.physical.restore.afterItFinishes.docker.clusterLocation"
                  components={{ dataDir: <InlineCodeComponent>{dataDir}</InlineCodeComponent> }}
                />
              </li>
              {Number(pgVersion) >= 18 ? (
                <li>
                  <Trans
                    i18nKey="backups.physical.restore.afterItFinishes.docker.volumeMount"
                    values={{ version: pgVersion }}
                    components={{
                      code: <InlineCodeComponent />,
                      dataDir: (
                        <InlineCodeComponent>{containerDataDir(pgVersion)}</InlineCodeComponent>
                      ),
                      volumeDir: (
                        <InlineCodeComponent>{containerVolumeDir(pgVersion)}</InlineCodeComponent>
                      ),
                    }}
                  />
                  <br />
                  <InlineCodeComponent>
                    {buildDockerRunCommand({
                      hostDir: outputDir,
                      containerDir: containerVolumeDir(pgVersion),
                      pgVersion,
                    })}
                  </InlineCodeComponent>
                </li>
              ) : (
                <li>
                  <Trans
                    i18nKey="backups.physical.restore.afterItFinishes.docker.dataDirectoryMount"
                    components={{
                      dataDir: (
                        <InlineCodeComponent>{containerDataDir(pgVersion)}</InlineCodeComponent>
                      ),
                    }}
                  />
                  <br />
                  <InlineCodeComponent>
                    {buildDockerRunCommand({
                      hostDir: dataDir,
                      containerDir: containerDataDir(pgVersion),
                      pgVersion,
                    })}
                  </InlineCodeComponent>
                </li>
              )}
              <li>{t('backups.physical.restore.afterItFinishes.docker.ownership')}</li>
              {missingConfigFilesNote(pgVersion)}
            </ul>
          )
        }
      />
    </div>
  );

  return (
    <div>
      {backup ? (
        <div className="mb-3 text-sm text-gray-600 dark:text-gray-400">
          {t('backups.physical.restore.backupIntro')}
        </div>
      ) : (
        <>
          <div className="mb-3 text-sm text-gray-600 dark:text-gray-400">
            {t('backups.physical.restore.pointInTimeIntro')}
          </div>
          <div className="mb-3 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[120px] sm:mb-0 sm:pr-2">
              {t('backups.physical.restore.targetTime')}
            </div>
            <DatePicker
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
        <div className="mt-3 rounded border border-red-300/50 bg-red-50 px-3 py-2 text-sm whitespace-pre-line text-red-700 dark:border-red-600/30 dark:bg-red-900/20 dark:text-red-400">
          <div>{translateApiError(restoreError, t)}</div>
          {restoreErrorHintKey && <div className="mt-5">{t(restoreErrorHintKey)}</div>}
        </div>
      )}

      {isGenerating && (
        <div className="mt-5 flex items-center gap-2 border-t border-gray-200 pt-4 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
          <Spin size="small" />
          {t('backups.physical.restore.preparing')}
        </div>
      )}

      {!isGenerating && bundleUrl && (
        <div className="mt-5 border-t border-gray-200 pt-4 dark:border-gray-700">
          <Segmented<RestoreMethod>
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
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {t('backups.physical.restore.linkExpiry')}
          </p>
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
