import { ActionIcon } from '@mantine/core';
import { MenuIcon } from 'lucide-react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { useIsMobile, useScreenHeight } from '@/shared/hooks';
import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { LanguageThemeControlComponent } from '@/shared/ui';
import { StarButtonComponent } from '@/shared/ui/StarButtonComponent';

interface Props {
  workspaceSelector: ReactNode;
  sidebar: ReactNode;
  auxiliary?: ReactNode;
  children: ReactNode;
  isSidebarOpen: boolean;
  onOpenSidebar: () => void;
}

export const AppShell = ({
  workspaceSelector,
  sidebar,
  auxiliary,
  children,
  isSidebarOpen,
  onOpenSidebar,
}: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const height = useScreenHeight();
  const isMobile = useIsMobile();
  return (
    <div
      className="flex min-h-0 min-w-0 flex-col gap-2 bg-page p-2 text-content md:gap-3 md:p-3"
      style={{ height }}
    >
      <header className="flex min-h-[50px] shrink-0 flex-wrap items-center gap-2 rounded bg-surface px-2 py-1.5 shadow md:min-h-[60px] md:gap-4 md:px-3">
        <a
          href={getWebsitePageUrl('home', locale)}
          target="_blank"
          rel="noreferrer"
          aria-label={t('common.websiteLink')}
        >
          <img className="h-[30px] w-[30px] p-1 md:h-[40px] md:w-[40px]" src="/logo.svg" alt="" />
        </a>
        <div className="min-w-0 flex-1 md:w-[260px] md:flex-none">{workspaceSelector}</div>
        {!isMobile && (
          <div className="ml-auto flex max-w-full min-w-0 flex-wrap items-center justify-end gap-x-5 gap-y-2">
            <a
              className="font-[450] !text-content"
              href={getWebsitePageUrl('installation', locale)}
              target="_blank"
              rel="noreferrer"
            >
              {t('app.navigation.docs')}
            </a>
            <a
              className="font-[450] !text-content"
              href="https://t.me/databasus_community"
              target="_blank"
              rel="noreferrer"
            >
              {t('app.navigation.community')}
            </a>
            {auxiliary}
            <div className="flex max-w-full flex-wrap items-center justify-end gap-2">
              <StarButtonComponent />
              <LanguageThemeControlComponent />
            </div>
          </div>
        )}
        {isMobile && (
          <ActionIcon
            className="ml-auto"
            variant="subtle"
            aria-label={t('app.navigation.menu')}
            aria-expanded={isSidebarOpen}
            onClick={onOpenSidebar}
          >
            <MenuIcon size="1em" className="inline-block shrink-0 align-middle" />
          </ActionIcon>
        )}
      </header>
      <div className="relative flex min-h-0 min-w-0 flex-1 gap-2 md:gap-3">
        {sidebar}
        <main className="flex min-h-0 min-w-0 flex-1 flex-col">{children}</main>
      </div>
    </div>
  );
};
