import { StarOutlined } from '@ant-design/icons';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useLocale } from '../i18n';

export function StarButtonComponent() {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const [starCount, setStarCount] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchStarCount = async () => {
    try {
      const response = await fetch('https://api.github.com/repos/databasus/databasus');
      if (response.ok) {
        const data = (await response.json()) as { stargazers_count: number };
        setStarCount(data.stargazers_count);
      }
    } catch (error) {
      console.error('Failed to fetch GitHub star count:', error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchStarCount();
  }, []);

  return (
    <a
      href="https://github.com/databasus/databasus"
      target="_blank"
      rel="noopener noreferrer"
      className="flex cursor-pointer items-center rounded-md border !border-border !bg-surface text-sm !text-content transition-colors hover:!bg-hover"
      aria-label={t('app.starButton.ariaLabel')}
    >
      <div className="flex items-center gap-2 border-r border-border px-2.5 py-1 !text-content">
        <StarOutlined aria-hidden />
        <span>{t('app.starButton.label')}</span>
      </div>

      {!isLoading && starCount !== null && (
        <span className="px-2.5 py-1 !text-content">{formatNumber(starCount)}</span>
      )}
    </a>
  );
}
