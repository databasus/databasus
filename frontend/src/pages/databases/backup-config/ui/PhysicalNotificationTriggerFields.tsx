import { Checkbox } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import { getShownPhysicalNotificationTriggers } from '@/pages/databases/backup-config/model/getShownPhysicalNotificationTriggers';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/backup-config/model/PhysicalBackupConfigDraft';
import {
  type PhysicalNotificationTrigger,
  PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS,
} from '@/pages/databases/backup-config/model/PhysicalNotificationTrigger';

type PhysicalNotificationTriggers = PhysicalBackupConfigDraft['settings']['notificationTriggers'];

interface Props {
  triggers: PhysicalNotificationTriggers;
  backupType: PhysicalDatabaseBackupType;
  onChange: (triggers: PhysicalNotificationTriggers) => void;
}

export const PhysicalNotificationTriggerFields = ({ triggers, backupType, onChange }: Props) => {
  const { t } = useTranslation();

  const toggleTrigger = (trigger: PhysicalNotificationTrigger, isChecked: boolean) => {
    const otherTriggers = triggers.filter((selectedTrigger) => selectedTrigger !== trigger);

    onChange(isChecked ? [...otherTriggers, trigger] : otherTriggers);
  };

  return (
    <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
      <div className="mt-0 mb-1 min-w-[150px] sm:mt-1 sm:mb-0 sm:pr-2">
        {t('backups.config.notifications')}
      </div>
      <div className="flex flex-col space-y-2">
        {getShownPhysicalNotificationTriggers(backupType).map((trigger) => (
          <Checkbox
            key={trigger}
            checked={triggers.includes(trigger)}
            onChange={(event) => toggleTrigger(trigger, event.target.checked)}
            label={<>{t(PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[trigger])}</>}
          />
        ))}
      </div>
    </div>
  );
};
