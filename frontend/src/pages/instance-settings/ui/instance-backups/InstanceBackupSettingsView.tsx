import { Button } from '@mantine/core';
import { Pencil } from 'lucide-react';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type {
  InstanceBackupDestinationHealth,
  InstanceBackupSettings,
} from '@/entities/instance-backups';
import { InstanceBackupDestinationList } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupDestinationList';

interface Props {
  settings: InstanceBackupSettings;
  destinationHealth: InstanceBackupDestinationHealth[];
  onEdit: () => void;
}

export function InstanceBackupSettingsView({ settings, destinationHealth, onEdit }: Props) {
  const { t } = useTranslation();
  const headingId = useId();

  return (
    <div className="mt-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 id={headingId} className="text-base font-medium">
          {t('instanceBackups.settings.storages')}
        </h3>
        <Button
          size="xs"
          variant="subtle"
          leftSection={<Pencil aria-hidden size={14} />}
          aria-describedby={headingId}
          onClick={onEdit}
        >
          {t('common.actions.edit')}
        </Button>
      </div>

      <InstanceBackupDestinationList
        storageIds={settings.storageIds}
        choices={[]}
        destinationHealth={destinationHealth}
      />
    </div>
  );
}
