import { Button } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

export const GfsRetentionHint = () => {
  const [isExpanded, setIsExpanded] = useState(false);
  const hintId = useId();
  const { t } = useTranslation();

  return (
    <div>
      <Button
        type="link"
        size="small"
        className="!h-auto !justify-start !px-0 !text-left !text-xs !whitespace-normal"
        aria-expanded={isExpanded}
        aria-controls={hintId}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {isExpanded
          ? t('backups.logical.config.retention.hideGfsHint')
          : t('backups.logical.config.retention.showGfsHint')}
      </Button>

      {isExpanded && (
        <div id={hintId} className="mt-1 max-w-[280px] text-xs text-muted">
          {t('backups.logical.config.retention.gfsHint')}
        </div>
      )}
    </div>
  );
};
