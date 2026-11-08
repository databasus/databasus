import { useTranslation } from 'react-i18next';

import { getMonitoringTextValues } from '@/pages/databases/wal-monitoring/lib/monitoringTextValues';
import type { MonitoringText } from '@/pages/databases/wal-monitoring/model/MonitoringText';
import { TransByKey, useLocale } from '@/shared/i18n';
import { InlineCodeComponent } from '@/shared/ui';

interface Props {
  text: MonitoringText;
  onOpenConfig?: () => void;
}

export const MonitoringTextComponent = ({ text, onOpenConfig }: Props) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  return (
    <TransByKey
      i18nKey={text.key}
      values={getMonitoringTextValues(text, t, formatNumber)}
      components={{
        code: <InlineCodeComponent />,
        configLink: (
          <button
            type="button"
            className="cursor-pointer underline"
            onClick={() => onOpenConfig?.()}
          />
        ),
      }}
    />
  );
};
