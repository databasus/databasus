import { ThemeIcon } from '@mantine/core';
import { KeyRound } from 'lucide-react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { MasterKeyExport, MasterKeyHostInstructions } from '@/features/export-master-key';
import { AdvancedSettingsToggleComponent, Dialog } from '@/shared/ui';

interface Props {
  status: InstanceBackupStatus;
  onClose: () => void;
}

// Root admin opens this dialog with Download master key, so opening it is the request for the key.
// The key leaves the page with the dialog, because the owner unmounts it on close.
export function MasterKeyDialog({ status, onClose }: Props) {
  const { t } = useTranslation();
  const hostInstructionsId = useId();
  const [isHostInstructionsShown, setIsHostInstructionsShown] = useState(false);

  return (
    <Dialog opened title={t('masterKey.title')} onClose={onClose}>
      <div className="text-sm">
        <div className="flex flex-col items-center text-center">
          <ThemeIcon variant="light" color="accent" size={48} radius="xl">
            <KeyRound aria-hidden size={24} />
          </ThemeIcon>
          <p className="mt-3 text-balance text-muted">{t('masterKey.description')}</p>
        </div>

        <div className="mt-5">
          <MasterKeyExport isCodeRequired={status.isConfirmationCodeRequired} />
        </div>

        <p className="mt-4 text-center text-xs text-balance text-muted">
          {t('instanceBackups.retainSeparately')}
        </p>

        <div className="mt-2 flex flex-col items-center">
          <AdvancedSettingsToggleComponent
            label={t('masterKey.host.toggle')}
            isShowAdvanced={isHostInstructionsShown}
            regionId={hostInstructionsId}
            onToggle={() => setIsHostInstructionsShown((current) => !current)}
          />
        </div>
        <div
          id={hostInstructionsId}
          className="mt-3 text-center text-balance"
          hidden={!isHostInstructionsShown}
        >
          <MasterKeyHostInstructions />
        </div>
      </div>
    </Dialog>
  );
}
