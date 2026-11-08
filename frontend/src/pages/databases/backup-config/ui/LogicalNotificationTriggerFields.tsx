import { Checkbox } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupSettings } from '@/entities/backup-configs';
import {
  LogicalNotificationTrigger,
  LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS,
} from '@/pages/databases/backup-config/model/LogicalNotificationTrigger';

type LogicalNotificationTriggers = LogicalBackupSettings['notificationTriggers'];

interface Props {
  triggers: LogicalNotificationTriggers;
  onChange: (triggers: LogicalNotificationTriggers) => void;
}

const SHOWN_TRIGGERS = [
  LogicalNotificationTrigger.BACKUP_SUCCESS,
  LogicalNotificationTrigger.BACKUP_FAILED,
];

export const LogicalNotificationTriggerFields = ({ triggers, onChange }: Props) => {
  const { t } = useTranslation();

  const toggleTrigger = (trigger: LogicalNotificationTrigger, isChecked: boolean) => {
    const otherTriggers = triggers.filter((selectedTrigger) => selectedTrigger !== trigger);

    onChange(isChecked ? [...otherTriggers, trigger] : otherTriggers);
  };

  return (
    <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
      <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
        {t('backups.config.notifications')}
      </div>
      <div className="flex min-w-0 flex-col space-y-1">
        {SHOWN_TRIGGERS.map((trigger) => (
          <Checkbox
            key={trigger}
            checked={triggers.includes(trigger)}
            onChange={(event) => toggleTrigger(trigger, event.target.checked)}
            label={<>{t(LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[trigger])}</>}
          />
        ))}
      </div>
    </div>
  );
};
