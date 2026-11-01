import { Trans } from 'react-i18next';

import { getWebsitePageUrl, useLocale } from '@/shared/i18n';

interface Props {
  topic: 'localhost' | 'supabase';
}

const renderDocsLink = (href: string) => (
  <a href={href} target="_blank" rel="noreferrer" className="text-accent underline" />
);

export const ConnectionHostHint = ({ topic }: Props) => {
  const { locale } = useLocale();

  return (
    <div className="mb-1 flex">
      <div className="hidden sm:block sm:w-[150px] sm:shrink-0" />
      <div className="max-w-[200px] text-xs text-muted">
        {topic === 'localhost' ? (
          <Trans
            i18nKey="databases.edit.localhostHint"
            components={{ docsLink: renderDocsLink(getWebsitePageUrl('faqLocalhost', locale)) }}
          />
        ) : (
          <Trans
            i18nKey="databases.edit.postgresql.supabaseHint"
            components={{ docsLink: renderDocsLink(getWebsitePageUrl('faqSupabase', locale)) }}
          />
        )}
      </div>
    </div>
  );
};
