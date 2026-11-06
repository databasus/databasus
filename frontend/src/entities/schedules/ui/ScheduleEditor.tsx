import { Select } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { Schedule } from '@/entities/schedules/model/Schedule';
import { ScheduleType } from '@/entities/schedules/model/ScheduleType';
import { SCHEDULE_TYPE_LABEL_KEYS } from '@/entities/schedules/model/ScheduleTypeLabelKeys';
import { ScheduleDetailsFields } from '@/entities/schedules/ui/ScheduleDetailsFields';

interface Props {
  label: string;
  schedule: Schedule | null;
  onChange: (schedule: Schedule) => void;
}

export const ScheduleEditor = ({ label, schedule, onChange }: Props) => {
  const { t } = useTranslation();
  const typeControlId = useId();

  const scheduleTypeOptions = Object.values(ScheduleType).map((scheduleType) => ({
    value: scheduleType,
    label: t(SCHEDULE_TYPE_LABEL_KEYS[scheduleType]),
  }));

  return (
    <div role="group" aria-label={label}>
      <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <label
          htmlFor={typeControlId}
          className="mb-1 max-w-[150px] min-w-[150px] leading-4 sm:mb-0 sm:pr-2"
        >
          {label}
        </label>
        <Select
          id={typeControlId}
          value={schedule?.type}
          onChange={(type) => onChange(schedule ? { ...schedule, type } : { type })}
          size="small"
          className="w-full max-w-[200px] grow"
          options={scheduleTypeOptions}
        />
      </div>

      {schedule && <ScheduleDetailsFields schedule={schedule} onChange={onChange} />}
    </div>
  );
};
