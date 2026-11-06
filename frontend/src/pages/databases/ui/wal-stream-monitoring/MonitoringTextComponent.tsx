import { useTranslation } from 'react-i18next';

import { getMonitoringTextValues } from '@/pages/databases/lib/wal-stream-monitoring/monitoringTextValues';
import type { MonitoringText } from '@/pages/databases/model/wal-stream-monitoring/MonitoringText';
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
