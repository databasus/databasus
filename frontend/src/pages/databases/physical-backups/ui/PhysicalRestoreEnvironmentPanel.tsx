import { Alert, TextInput } from '@mantine/core';
import { Info } from 'lucide-react';
import { type JSX, useId } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type RestoreEnvironment,
  buildChownCommand,
  buildDockerRunCommand,
  buildPgCtlStartCommand,
  clusterDataDir,
  containerDataDir,
  containerVolumeDir,
  debianConfigDir,
  defaultPgBinDir,
} from '@/pages/databases/physical-backups/lib/restoreCommands';
import { PhysicalRestoreCommands } from '@/pages/databases/physical-backups/ui/PhysicalRestoreCommands';
import type { PhysicalRestoreMethod } from '@/pages/databases/physical-backups/ui/PhysicalRestoreMethod';
import { useCopiedCommand } from '@/pages/databases/physical-backups/ui/useCopiedCommand';
import { InlineCodeComponent } from '@/shared/ui';

interface Props {
  environment: RestoreEnvironment;
  restoreMethod: PhysicalRestoreMethod;
  bundleUrl: string;
  recoveryTargetTime: string;
  pgVersion: string;
  hasWal: boolean;
  outputDir: string;
  pgBin: string;
  dockerImage: string;
  onOutputDirChange: (outputDir: string) => void;
  onPgBinChange: (pgBin: string) => void;
  onDockerImageChange: (dockerImage: string) => void;
}

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

export const PhysicalRestoreEnvironmentPanel = ({
  environment,
  restoreMethod,
  bundleUrl,
  recoveryTargetTime,
  pgVersion,
  hasWal,
  outputDir,
  pgBin,
  dockerImage,
  onOutputDirChange,
  onPgBinChange,
  onDockerImageChange,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const directoryFieldId = useId();
  const pgBinFieldId = useId();
  const imageFieldId = useId();
  const { copiedCommandId, copyCommand } = useCopiedCommand();

  const dataDir = clusterDataDir(outputDir, pgVersion);

  return (
    <div className="pt-3">
      <div className="mb-3">
        <div className="mb-2 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label className="mb-1 shrink-0 sm:mb-0 sm:w-[190px] sm:pr-2" htmlFor={directoryFieldId}>
            {t('backups.physical.restore.fields.restoreDirectory')}
          </label>
          <TextInput
            id={directoryFieldId}
            value={outputDir}
            onChange={(e) => onOutputDirChange(e.target.value)}
            className="w-full max-w-[320px]"
          />
        </div>
        {environment === 'host' ? (
          <div className="flex w-full flex-col items-start sm:flex-row sm:items-center">
            <label className="mb-1 shrink-0 sm:mb-0 sm:w-[190px] sm:pr-2" htmlFor={pgBinFieldId}>
              {t('backups.physical.restore.fields.pgBinPath')}
            </label>
            <TextInput
              id={pgBinFieldId}
              value={pgBin}
              onChange={(e) => onPgBinChange(e.target.value)}
              placeholder={t('backups.physical.restore.fields.pgBinPathPlaceholder', {
                path: defaultPgBinDir(pgVersion),
              })}
              className="w-full max-w-[320px]"
            />
          </div>
        ) : (
          <div className="flex w-full flex-col items-start sm:flex-row sm:items-center">
            <label className="mb-1 shrink-0 sm:mb-0 sm:w-[190px] sm:pr-2" htmlFor={imageFieldId}>
              {t('backups.physical.restore.fields.pgImage')}
            </label>
            <TextInput
              id={imageFieldId}
              value={dockerImage}
              onChange={(e) => onDockerImageChange(e.target.value)}
              className="w-full max-w-[320px]"
            />
          </div>
        )}
      </div>
      <Alert
        color="accent"
        icon={<Info aria-hidden size={16} className="inline-block shrink-0 align-middle" />}
        title={t('backups.physical.restore.beforeYouRun.title')}
      >
        {environment === 'host' ? (
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
        )}
      </Alert>
      <PhysicalRestoreCommands
        environment={environment}
        restoreMethod={restoreMethod}
        bundleUrl={bundleUrl}
        recoveryTargetTime={recoveryTargetTime}
        pgVersion={pgVersion}
        hasWal={hasWal}
        outputDir={outputDir}
        pgBin={pgBin}
        dockerImage={dockerImage}
        copiedCommandId={copiedCommandId}
        onCopy={copyCommand}
      />
      <div className="mt-3" />
      <Alert
        color="accent"
        icon={<Info aria-hidden size={16} className="inline-block shrink-0 align-middle" />}
        title={t('backups.physical.restore.afterItFinishes.title')}
      >
        {environment === 'host' ? (
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
                  command: <InlineCodeComponent>{buildChownCommand(dataDir)}</InlineCodeComponent>,
                }}
              />
            </li>
            <li>
              <Trans
                i18nKey="backups.physical.restore.afterItFinishes.host.start"
                components={{
                  command: (
                    <InlineCodeComponent>
                      {buildPgCtlStartCommand({ dataDir, pgBin })}
                    </InlineCodeComponent>
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
                    image: dockerImage,
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
                    image: dockerImage,
                  })}
                </InlineCodeComponent>
              </li>
            )}
            <li>{t('backups.physical.restore.afterItFinishes.docker.ownership')}</li>
            {missingConfigFilesNote(pgVersion)}
          </ul>
        )}
      </Alert>
    </div>
  );
};
