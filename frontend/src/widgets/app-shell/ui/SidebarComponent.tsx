import { CloseOutlined } from '@ant-design/icons';
import { Drawer, Tooltip } from 'antd';
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

  // Close sidebar on desktop when it becomes desktop size
  useEffect(() => {
    if (!isMobile && isOpen) {
      onClose();
    }
  }, [isMobile, isOpen, onClose]);

  // Prevent background scrolling when mobile sidebar is open
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
      <div className="h-full max-w-[60px] min-w-[60px] rounded bg-surface py-2 shadow">
        <div className="flex h-full flex-col">
          <div className="flex-1">
            {filteredTabs.map((tab) => (
              <div key={tab.text} className="flex justify-center">
                <button
                  type="button"
                  aria-label={tab.text}
                  className={`flex h-[50px] w-[50px] cursor-pointer items-center justify-center rounded select-none ${selectedTab === tab.name ? 'bg-hover' : 'hover:bg-hover'}`}
                  onClick={() => handleTabClick(tab)}
                  style={{ marginTop: tab.marginTop }}
                >
                  <div className="mb-1">
                    <div className="flex justify-center">
                      <img src={tab.icon} width={20} alt={tab.text} loading="lazy" />
                    </div>
                  </div>
                </button>
              </div>
            ))}
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

            return (
              <div key={tab.text}>
                <button
                  type="button"
                  aria-label={tab.text}
                  className={`flex w-full cursor-pointer items-center gap-3 rounded px-3 py-3 select-none ${selectedTab === tab.name ? 'bg-hover text-accent' : 'text-content hover:bg-surface'}`}
                  onClick={() => handleTabClick(tab)}
                >
                  <img src={tab.icon} width={24} alt={tab.text} loading="lazy" />
                  <span className="text-sm font-medium">{tab.text}</span>
                </button>
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
