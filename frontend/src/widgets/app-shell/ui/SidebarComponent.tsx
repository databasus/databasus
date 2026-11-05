import { CloseOutlined } from '@ant-design/icons';
import { Button, Drawer, Tooltip } from 'antd';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { type DiskUsage } from '@/entities/disk';
import { type User, UserRole } from '@/entities/users';
import { useIsMobile } from '@/shared/hooks';
import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { LanguageThemeControlComponent } from '@/shared/ui';
import { StarButtonComponent } from '@/shared/ui/StarButtonComponent';
import { getDiskUsageValues } from '@/widgets/app-shell/ui/getDiskUsageValues';

export type MainTab =
  | 'notifiers'
  | 'storages'
  | 'databases'
  | 'profile'
  | 'databasus-settings'
  | 'users'
  | 'settings';

export interface SidebarTab {
  text: string;
  name: MainTab;
  icon: string;
  selectedIcon: string;
  onClick: () => void;
  isAdminOnly: boolean;
  marginTop: string;
  isVisible: boolean;
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
  selectedTab: MainTab;
  tabs: SidebarTab[];
  user?: User;
  diskUsage?: DiskUsage;
}

export const SidebarComponent = ({
  isOpen,
  onClose,
  selectedTab,
  tabs,
  user,
  diskUsage,
}: Props) => {
  const handleTabClick = (tab: SidebarTab) => {
    tab.onClick();
    if (isMobile) {
      onClose();
    }
  };
  const { t } = useTranslation();
  const { locale, formatNumber } = useLocale();
  const isMobile = useIsMobile();

  useEffect(() => {
    if (!isMobile && isOpen) {
      onClose();
    }
  }, [isMobile, isOpen, onClose]);

  useEffect(() => {
    if (isMobile && isOpen) {
      const previousOverflow = document.body.style.overflowY;
      document.body.style.overflowY = 'hidden';

      return () => {
        document.body.style.overflowY = previousOverflow;
      };
    }
  }, [isMobile, isOpen]);

  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  const filteredTabs = tabs
    .filter((tab) => !tab.isAdminOnly || user?.role === UserRole.ADMIN)
    .filter((tab) => tab.isVisible);

  if (!isMobile) {
    return (
      <div className="h-full min-h-0 max-w-[60px] min-w-[60px] overflow-y-auto rounded bg-surface py-2 shadow">
        <div className="flex h-full flex-col">
          <div className="flex-1">
            {filteredTabs.map((tab) => {
              const isSelected = selectedTab === tab.name;

              return (
                <div key={tab.text} className="flex justify-center">
                  <Button
                    type={isSelected ? 'primary' : 'text'}
                    aria-label={tab.text}
                    aria-current={isSelected ? 'page' : undefined}
                    className="!h-[50px] !w-[50px] select-none"
                    onClick={() => handleTabClick(tab)}
                    style={{ marginTop: tab.marginTop }}
                    icon={
                      <img
                        src={isSelected ? tab.selectedIcon : tab.icon}
                        width={20}
                        alt=""
                        loading="lazy"
                      />
                    }
                  />
                </div>
              );
            })}
          </div>
        </div>
      </div>
    );
  }

  return (
    <Drawer
      title={t('app.navigation.menu')}
      open={isOpen}
      onClose={onClose}
      placement="right"
      width={280}
      styles={{
        body: {
          padding: 0,
        },
      }}
      closable={false}
      mask={false}
    >
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between border-b border-border px-3 py-3">
          <LanguageThemeControlComponent />
          <button
            type="button"
            aria-label={t('common.actions.close')}
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded hover:bg-surface"
          >
            <CloseOutlined />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-3 py-4">
          {filteredTabs.map((tab, index) => {
            const showDivider =
              index < filteredTabs.length - 1 && filteredTabs[index + 1]?.marginTop !== '0px';
            const isSelected = selectedTab === tab.name;

            return (
              <div key={tab.text}>
                <Button
                  block
                  type={isSelected ? 'primary' : 'text'}
                  aria-label={tab.text}
                  aria-current={isSelected ? 'page' : undefined}
                  className="!h-auto !justify-start !gap-3 !px-3 !py-3 text-left !whitespace-normal select-none"
                  onClick={() => handleTabClick(tab)}
                >
                  <img
                    src={isSelected ? tab.selectedIcon : tab.icon}
                    width={24}
                    alt=""
                    loading="lazy"
                  />
                  <span className="text-sm font-medium">{tab.text}</span>
                </Button>
                {showDivider && <div className="my-2 border-t border-border" />}
              </div>
            );
          })}
        </div>
        <div className="border-t border-border bg-surface px-3 py-4">
          {diskUsage && (
            <div className="mb-4">
              <Tooltip title={t('app.diskUsage.hint')}>
                <div
                  className={`cursor-pointer text-xs ${isUsedMoreThan95Percent ? 'text-danger' : 'text-muted'}`}
                >
                  <div className="font-medium">{t('app.diskUsage.title')}</div>
                  <div className="mt-1">
                    {t('app.diskUsage.summary', getDiskUsageValues(diskUsage, formatNumber))}
                  </div>
                </div>
              </Tooltip>
            </div>
          )}

          <div className="space-y-2">
            <a
              className="block rounded text-sm font-medium !text-content hover:bg-surface hover:!text-accent"
              href={getWebsitePageUrl('installation', locale)}
              target="_blank"
              rel="noreferrer"
            >
              {t('app.navigation.documentation')}
            </a>

            <a
              className="block rounded text-sm font-medium !text-content hover:bg-surface hover:!text-accent"
              href="https://t.me/databasus_community"
              target="_blank"
              rel="noreferrer"
            >
              {t('app.navigation.community')}
            </a>

            <div className="flex pt-2">
              <StarButtonComponent />
            </div>
          </div>
        </div>
      </div>
    </Drawer>
  );
};
