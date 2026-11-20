import { Menu } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { getWebsitePageUrl, useLocale } from '@/shared/i18n';

const TELEGRAM_CHAT_URL = 'https://t.me/databasus_community';

interface Props {
  targetClassName: string;
}

export const CommunityMenuComponent = ({ targetClassName }: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();

  return (
    <Menu trigger="click-hover" position="bottom-start" withinPortal>
      <Menu.Target>
        <button type="button" className={`cursor-pointer ${targetClassName}`}>
          {t('app.navigation.community')}
        </button>
      </Menu.Target>

      <Menu.Dropdown>
        <Menu.Item
          component="a"
          href={getWebsitePageUrl('discord', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('app.navigation.discord')}
        </Menu.Item>
        <Menu.Item component="a" href={TELEGRAM_CHAT_URL} target="_blank" rel="noreferrer">
          {t('app.navigation.telegram')}
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  );
};
