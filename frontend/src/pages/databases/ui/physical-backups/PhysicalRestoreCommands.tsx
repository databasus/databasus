import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { physicalBackupsApi } from '@/pages/databases/api/physicalBackupsApi';
import {
  type RestoreEnvironment,
  buildDockerScriptCommand,
  buildManualSteps,
  buildScriptCommand,
} from '@/pages/databases/lib/physical-backups/restoreCommands';
import { CopyableCommand } from '@/pages/databases/ui/physical-backups/CopyableCommand';
import type { PhysicalRestoreMethod } from '@/pages/databases/ui/physical-backups/PhysicalRestoreMethod';
import type { CommandToCopy } from '@/pages/databases/ui/physical-backups/useCopiedCommand';

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
  copiedCommandId: string | null;
  onCopy: (command: CommandToCopy) => void;
}

export const PhysicalRestoreCommands = ({
  environment,
  restoreMethod,
  bundleUrl,
  recoveryTargetTime,
  pgVersion,
  hasWal,
  outputDir,
  pgBin,
  dockerImage,
  copiedCommandId,
  onCopy,
}: Props): JSX.Element => {
  const { t } = useTranslation();

  if (restoreMethod === 'script') {
    const scriptUrl = physicalBackupsApi.getPhysicalRecoveryScriptUrl();
    const code =
      environment === 'host'
        ? buildScriptCommand({
            scriptUrl,
            bundleUrl,
            outputDir,
            pgBin,
            targetTime: recoveryTargetTime,
          })
        : buildDockerScriptCommand({
            scriptUrl,
            bundleUrl,
            outputDir,
            image: dockerImage,
            targetTime: recoveryTargetTime,
          });

    return (
      <CopyableCommand
        id={`script-${environment}`}
        title={
          environment === 'host'
            ? t('backups.physical.restore.scriptTitles.host')
            : t('backups.physical.restore.scriptTitles.docker')
        }
        code={code}
        copiedCommandId={copiedCommandId}
        onCopy={onCopy}
      />
    );
  }

  const steps = buildManualSteps({
    bundleUrl,
    outputDir,
    pgVersion,
    pgBin,
    image: dockerImage,
    environment,
    hasWal,
    targetTime: recoveryTargetTime,
  });

  return (
    <>
      {steps.map((step, index) => (
        <CopyableCommand
          key={step.titleKey}
          id={`manual-${environment}-${index}`}
          title={`${index + 1}. ${t(step.titleKey)}`}
          code={step.code}
          copiedCommandId={copiedCommandId}
          onCopy={onCopy}
        />
      ))}
    </>
  );
};
