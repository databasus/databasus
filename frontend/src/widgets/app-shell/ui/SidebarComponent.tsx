import { ActionIcon, Button, Drawer, Tooltip } from '@mantine/core';
import { X } from 'lucide-react';
import { type MouseEvent, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';

import { type DiskUsage } from '@/entities/disk';
import { type User, UserRole } from '@/entities/users';
import { useIsMobile } from '@/shared/hooks';
import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { closeOverlayOnEscape } from '@/shared/ui';
import { OverlayFocusReturn } from '@/shared/ui';
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
  href: string;
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
  const handleTabClick = (event: MouseEvent<HTMLAnchorElement>) => {
    if (
      event.currentTarget.getAttribute('aria-current') === 'page' &&
      !event.ctrlKey &&
      !event.metaKey &&
      !event.shiftKey &&
      !event.altKey
    )
      event.preventDefault();

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
                  <ActionIcon
                    component={Link}
                    to={tab.href}
                    variant={isSelected ? 'filled' : 'subtle'}
                    aria-label={tab.text}
                    aria-current={isSelected ? 'page' : undefined}
                    className="!h-[50px] !w-[50px] select-none"
                    onClick={handleTabClick}
                    style={{ marginTop: tab.marginTop }}
                  >
                    <img
                      src={isSelected ? tab.selectedIcon : tab.icon}
                      width={20}
                      alt=""
                      loading="lazy"
                    />
                  </ActionIcon>
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
      closeOnEscape={false}
      onKeyDown={(event) => closeOverlayOnEscape(event, { isOpen: isOpen, onClose: onClose })}
      title={t('app.navigation.menu')}
      opened={isOpen}
      onClose={onClose}
      position="right"
      size={280}
      styles={{
        content: { display: 'flex', flexDirection: 'column' },
        header: { flexShrink: 0 },
        body: {
          padding: 0,
          flex: 1,
          minHeight: 0,
        },
      }}
      withCloseButton={false}
      withOverlay={false}
      closeButtonProps={{ 'aria-label': t('common.actions.close') }}
    >
      {isOpen && <OverlayFocusReturn />}
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between border-b border-border px-3 py-3">
          <LanguageThemeControlComponent />
          <button
            type="button"
            aria-label={t('common.actions.close')}
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded hover:bg-surface"
          >
            <X size="1em" className="inline-block shrink-0 align-middle" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-3 py-4">
          {filteredTabs.map((tab, index) => {
            const showDivider =
              index < filteredTabs.length - 1 && filteredTabs[index + 1]?.marginTop !== '0px';
            const isSelected = selectedTab === tab.name;

            return (
              <div key={tab.text}>
                <Button
                  component={Link}
                  to={tab.href}
                  fullWidth
                  variant={isSelected ? 'filled' : 'subtle'}
                  color={isSelected ? 'accent' : 'var(--app-content)'}
                  styles={{ inner: { justifyContent: 'flex-start' } }}
                  leftSection={
                    <img
                      src={isSelected ? tab.selectedIcon : tab.icon}
                      width={24}
                      alt=""
                      loading="lazy"
                    />
                  }
                  aria-label={tab.text}
                  aria-current={isSelected ? 'page' : undefined}
                  className="!h-auto !justify-start !gap-3 !px-3 !py-3 text-left !whitespace-normal select-none"
                  onClick={handleTabClick}
                >
                  <span className="text-sm font-medium">{tab.text}</span>
                </Button>
                {showDivider && <div className="my-2 border-t border-border" />}
              </div>
            );
          })}
        </div>
        <div className="shrink-0 border-t border-border bg-surface px-3 py-4">
          {diskUsage && (
            <div className="mb-4">
              <Tooltip label={t('app.diskUsage.hint')}>
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
