import { InfoCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { type Notifier, NotifierType, getNotifierLogoFromType } from '../../../entities/notifiers';
import type { TranslationKey } from '../../../shared/i18n';

interface Props {
  notifier: Notifier;
  selectedNotifierId?: string;
  setSelectedNotifierId: (notifierId: string) => void;
}

// One sentence per type rather than one with the type interpolated: some languages put a different
// preposition before a messenger than before email or a webhook.
const NOTIFY_TO_KEYS: Record<NotifierType, TranslationKey> = {
  [NotifierType.EMAIL]: 'notifiers.card.notifyTo.email',
  [NotifierType.TELEGRAM]: 'notifiers.card.notifyTo.telegram',
  [NotifierType.WEBHOOK]: 'notifiers.card.notifyTo.webhook',
  [NotifierType.SLACK]: 'notifiers.card.notifyTo.slack',
  [NotifierType.DISCORD]: 'notifiers.card.notifyTo.discord',
  [NotifierType.TEAMS]: 'notifiers.card.notifyTo.teams',
  [NotifierType.MATTERMOST]: 'notifiers.card.notifyTo.mattermost',
};

export const NotifierCardComponent = ({
  notifier,
  selectedNotifierId,
  setSelectedNotifierId,
}: Props) => {
  const { t } = useTranslation();

  return (
    <div
      className={`mb-3 cursor-pointer rounded p-3 shadow ${selectedNotifierId === notifier.id ? 'bg-blue-100 dark:bg-blue-800' : 'bg-white dark:bg-gray-800'}`}
      onClick={() => setSelectedNotifierId(notifier.id)}
    >
      <div className="mb-1 font-bold">{notifier.name}</div>

      <div className="flex items-center">
        <div className="text-sm text-gray-500 dark:text-gray-400">
          {t(NOTIFY_TO_KEYS[notifier.settings.type])}
        </div>

        <img
          src={getNotifierLogoFromType(notifier.settings.type)}
          alt=""
          className="ml-1 h-4 w-4"
        />
      </div>

      {notifier.lastSendError && (
        <div className="mt-1 flex items-center text-sm text-red-600 underline dark:text-red-400">
          <InfoCircleOutlined className="mr-1" style={{ color: 'red' }} />
          {t('notifiers.card.hasSendError')}
        </div>
      )}
    </div>
  );
};
