import { Tooltip } from 'antd';
import dayjs from 'dayjs';

import { useLocale } from '../../../shared/i18n';
import { getUserShortTimeFormat } from '../../../shared/time/getUserTimeFormat';
import { HealthStatus } from '../../databases/model/HealthStatus';

interface Props {
  attempts: { status: HealthStatus; createdAt: Date }[];
}

export const HealthcheckAttemptsStripComponent = ({ attempts }: Props) => {
  const { formatRelativeTime } = useLocale();

  return (
    <div className="flex flex-wrap gap-1">
      {attempts.map((attempt) => (
        <Tooltip
          key={attempt.createdAt.toString()}
          title={`${dayjs(attempt.createdAt).format(getUserShortTimeFormat().format)} (${formatRelativeTime(attempt.createdAt)})`}
        >
          <div
            className={`h-[8px] w-[8px] cursor-pointer rounded-[2px] ${
              attempt.status === HealthStatus.AVAILABLE ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
        </Tooltip>
      ))}
    </div>
  );
};
