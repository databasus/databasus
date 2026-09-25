import { useTranslation } from 'react-i18next';

import { HealthStatus } from '../model/HealthStatus';
import { HEALTH_STATUS_LABEL_KEYS } from '../model/HealthStatusLabelKeys';

interface Props {
  healthStatus: HealthStatus;
}

export const HealthStatusBadgeComponent = ({ healthStatus }: Props) => {
  const { t } = useTranslation();

  return (
    <div
      className={`w-fit rounded px-[6px] py-[2px] text-[10px] whitespace-nowrap text-white ${
        healthStatus === HealthStatus.AVAILABLE ? 'bg-green-500' : 'bg-red-500'
      }`}
    >
      {t(HEALTH_STATUS_LABEL_KEYS[healthStatus])}
    </div>
  );
};
