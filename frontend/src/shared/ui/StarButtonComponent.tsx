import { Star } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { useSessionQuery } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function StarButtonComponent() {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const query = useSessionQuery({
    queryKey: ['github-stars'],
    queryFn: async (signal) => {
      const response = await fetch('https://api.github.com/repos/databasus/databasus', { signal });
      if (!response.ok) return null;

      const repository = (await response.json()) as { stargazers_count: number };

      return repository.stargazers_count;
    },
  });
  const starCount = query.data ?? null;
  const isLoading = query.isPending;

  return (
    <a
      href="https://github.com/databasus/databasus"
      target="_blank"
      rel="noopener noreferrer"
      className="flex cursor-pointer items-center rounded-md border !border-border !bg-surface text-sm !text-content transition-colors hover:!bg-hover"
      aria-label={t('app.starButton.ariaLabel')}
    >
      <div className="flex items-center gap-2 border-r border-border px-2.5 py-1 !text-content">
        <Star aria-hidden size={14} />
        <span>{t('app.starButton.label')}</span>
      </div>

      {!isLoading && starCount !== null && (
        <span className="px-2.5 py-1 !text-content">{formatNumber(starCount)}</span>
      )}
    </a>
  );
}
