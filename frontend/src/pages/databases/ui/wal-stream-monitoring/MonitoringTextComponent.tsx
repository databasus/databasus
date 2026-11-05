import { useTranslation } from 'react-i18next';

import { TransByKey, useLocale } from '../../../../shared/i18n';
import { InlineCodeComponent } from '../../../../shared/ui';
import { getMonitoringTextValues } from '../../lib/wal-stream-monitoring/monitoringTextValues';
import type { MonitoringText } from '../../model/wal-stream-monitoring/MonitoringText';

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
