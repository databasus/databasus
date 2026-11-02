import { useTranslation } from 'react-i18next';

import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { LanguageThemeControlComponent } from '@/shared/ui';
import { StarButtonComponent } from '@/shared/ui/StarButtonComponent';

// eslint-disable-next-line i18next/no-literal-string -- product name
const PRODUCT_NAME = 'Databasus';

export function AuthNavbarComponent() {
  const { t } = useTranslation();
  const { locale } = useLocale();

  const homePageUrl = getWebsitePageUrl('home', locale);

  return (
    <div className="flex min-h-[65px] flex-wrap items-center justify-between gap-3 px-4 pt-5 sm:justify-start">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href={homePageUrl} target="_blank" rel="noreferrer" aria-label={t('common.websiteLink')}>
          <img className="h-[45px] w-[45px] p-1" src="/logo.svg" alt="" />
        </a>

        <div className="text-xl font-bold">
          <a href={homePageUrl} className="!text-accent" target="_blank" rel="noreferrer">
            {PRODUCT_NAME}
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto flex max-w-full min-w-0 flex-wrap items-center gap-3">
        <a
          className="!text-content hover:opacity-80"
          href={getWebsitePageUrl('installation', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('app.navigation.docs')}
        </a>

        <a
          className="!text-content hover:opacity-80"
          href="https://t.me/databasus_community"
          target="_blank"
          rel="noreferrer"
        >
          {t('app.navigation.community')}
        </a>

        <div className="flex max-w-full min-w-0 flex-wrap items-center gap-2">
          <StarButtonComponent />

          <LanguageThemeControlComponent />
        </div>
      </div>
    </div>
  );
}
