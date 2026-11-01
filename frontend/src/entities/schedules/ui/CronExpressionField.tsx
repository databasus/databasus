import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import { useLocale } from '../../../shared/i18n';
import { getUserTimeFormat } from '../../../shared/time';
import { getNextCronRun } from '../model/getNextCronRun';

interface Props {
  cronExpression: string;
  onChange: (cronExpression: string) => void;
}

export const CronExpressionField = ({ cronExpression, onChange }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const controlId = useId();

  const nextRun = getNextCronRun(cronExpression);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <label htmlFor={controlId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('schedules.cronExpression')}
        </label>
        <div className="flex items-center">
          <Input
            id={controlId}
            value={cronExpression}
            onChange={(e) => onChange(e.target.value)}
            placeholder="0 2 * * *"
            size="small"
            className="w-full max-w-[200px] grow"
          />
          <Tooltip
            className="cursor-pointer"
            title={
              <div>
                <div className="font-bold">{t('schedules.cronHelp.format')}</div>
                <div className="mt-1">{t('schedules.cronHelp.examples')}</div>
                <div>0 2 * * * - {t('schedules.cronHelp.dailyAt2am')}</div>
                <div>0 */6 * * * - {t('schedules.cronHelp.every6Hours')}</div>
                <div>0 3 * * 1 - {t('schedules.cronHelp.mondaysAt3am')}</div>
                <div>30 4 1,15 * * - {t('schedules.cronHelp.twiceAMonth')}</div>
              </div>
            }
          >
            <InfoCircleOutlined className="ml-2 text-muted" />
          </Tooltip>
        </div>
      </div>

      {cronExpression && nextRun && (
        <div className="mb-1 flex w-full flex-col items-start text-xs text-muted sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
          <div>
            {t('schedules.nextRun', {
              time: dayjs(nextRun).local().format(getUserTimeFormat().format),
            })}
            <br />({formatRelativeTime(nextRun)})
          </div>
        </div>
      )}

      {cronExpression && !nextRun && (
        <div className="mb-1 flex w-full flex-col items-start text-danger sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
          <div>{t('schedules.invalidCron')}</div>
        </div>
      )}
    </>
  );
};
