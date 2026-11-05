import { Checkbox } from 'antd';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupSettings } from '../../../../entities/backup-configs';
import { LogicalNotificationTrigger } from '../../model/LogicalNotificationTrigger';
import { LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS } from '../../model/LogicalNotificationTriggerLabelKeys';

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
      <div className="mt-0 mb-1 min-w-[150px] sm:mt-1 sm:mb-0 sm:pr-2">
        {t('backups.config.notifications')}
      </div>
      <div className="flex flex-col space-y-2">
        {SHOWN_TRIGGERS.map((trigger) => (
          <Checkbox
            key={trigger}
            checked={triggers.includes(trigger)}
            onChange={(event) => toggleTrigger(trigger, event.target.checked)}
          >
            {t(LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[trigger])}
          </Checkbox>
        ))}
      </div>
    </div>
  );
};
